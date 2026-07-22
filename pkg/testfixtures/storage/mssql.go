package storage

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	_ "github.com/microsoft/go-mssqldb" // MSSQL Driver.
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
	"github.com/oklog/ulid/v2"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"

	"github.com/openfga/openfga/assets"
	"github.com/openfga/openfga/pkg/testutils"
)

const (
	msSQLImage          = "mcr.microsoft.com/mssql/server:2022-latest"
	msSQLDBPrefix       = "openfga-test-db-"
	msSQLTemplateDB     = msSQLDBPrefix + "template"
	msSQLTemplateDBDump = "/tmp/" + msSQLTemplateDB + ".bak"
	msSQLUsername       = "sa"
	msSQLPassword       = "pKC8mMA_qu5SLeaG"
)

var (
	_ DatastoreTestContainer = (*msSQLTestContainer)(nil)

	mssqlContainerName = "openfga-test-mssql-" + ulid.Make().String()
	mssqlPort          = network.MustParsePort("1433/tcp")
	mssqlDockerCont    atomic.Pointer[container.InspectResponse]

	mssqlBootstrapping bool
	mssqlCond          = sync.NewCond(&sync.Mutex{})
)

type msSQLTestContainer struct {
	host string
	port string

	database string
	username string
	password string

	version int64
}

// GetConnectionURI returns the postgres connection uri for the running postgres test container.
func (m *msSQLTestContainer) GetConnectionURI(includeCredentials bool) string {
	var username, password string
	if includeCredentials {
		username = m.username
		password = m.password
	}

	return msSQLConnectionURI(m.host, m.port, m.database, username, password)
}

func (m *msSQLTestContainer) GetDatabaseSchemaVersion() int64 {
	return m.version
}

func (m *msSQLTestContainer) GetUsername() string {
	return m.username
}

func (m *msSQLTestContainer) GetPassword() string {
	return m.password
}

// Not supported at the moment
func (m *msSQLTestContainer) CreateSecondary(t testing.TB) error {
	return nil
}

// Not supported at the moment
func (m *msSQLTestContainer) GetSecondaryConnectionURI(includeCredentials bool) string {
	return ""
}

// RunMSSQLTestContainer  runs a MSSQL container, connects to it, and returns a
// bootstrapped implementation of the DatastoreTestContainer interface wired up for the
// MSSQL datastore engine.
func RunMSSQLTestContainer(t testing.TB) DatastoreTestContainer {
	docker, err := testutils.NewDockerClient()

	require.NoError(t, err)

	t.Cleanup(func() {
		docker.Close()
	})

	// Shared MSSQL container bootstrap for concurrent tests using sync.Cond.
	// Only one test bootstraps the shared container at a time, while others wait efficiently using sync.Cond.
	// If bootstrap fails, waiting tests are awakened so another test can retry without being affected by the failure.
	mssqlCond.L.Lock()
	for mssqlDockerCont.Load() == nil {
		if !mssqlBootstrapping {
			mssqlBootstrapping = true
			mssqlCond.L.Unlock()

			dockerCont, err := bootstrapMSSQLContainer(t.Context(), docker)
			mssqlCond.L.Lock()
			mssqlBootstrapping = false
			if err == nil {
				mssqlDockerCont.Store(dockerCont)
			}

			mssqlCond.Broadcast()

			if err != nil {
				// Unlock before failing the test to allow waiting tests to proceed with bootstrapping.
				mssqlCond.L.Unlock()
				require.NoError(t, err)
			}

			continue
		}

		mssqlCond.Wait()
	}
	mssqlCond.L.Unlock()

	dockerCont := mssqlDockerCont.Load()
	port, err := docker.GetHostPort(dockerCont, mssqlPort)
	require.NoError(t, err)

	version, err := latestMigrationVersion(assets.MSSQLMigrationDir)
	require.NoError(t, err, "get expected mssql migration version")

	testCont := &msSQLTestContainer{
		host:     "localhost",
		port:     port,
		database: msSQLTemplateDB + ulid.Make().String(),
		username: msSQLUsername,
		password: msSQLPassword,
		version:  version,
	}

	tplURI := msSQLConnectionURI(testCont.host, testCont.port, msSQLTemplateDB, testCont.username, testCont.password)
	require.NoError(t, waitForMigrationVersion("mssql", tplURI, testCont.version))

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		dropQuery := fmt.Sprintf("DROP DATABASE IF EXISTS [%s]", testCont.database)
		if err := execMSSQLQuery(ctx, docker, dockerCont.ID, testCont.host, testCont.username, testCont.password, dropQuery); err != nil {
			t.Errorf("drop test database in the mssql container: %v", err)
		}
	})

	createDBQuery := fmt.Sprintf("CREATE DATABASE [%s] COLLATE Latin1_General_CS_AS", testCont.database)
	require.NoError(t, execMSSQLQuery(t.Context(), docker, dockerCont.ID, testCont.host, testCont.username, testCont.password, createDBQuery))

	restoreQuery := mssqlRestoreQuery(msSQLTemplateDB, msSQLTemplateDBDump, testCont.database)
	require.NoError(t, execMSSQLQuery(t.Context(), docker, dockerCont.ID, testCont.host, testCont.username, testCont.password, restoreQuery))

	require.NoError(t, waitForMigrationVersion("mssql", testCont.GetConnectionURI(true), testCont.version))

	return testCont
}

// CleanupMSSQLContainer removes the shared mssql test container.
// It should be called from TestMain after all tests in a package have finished.
func CleanupMSSQLContainer() {
	_ = cleanupDatastoreTestContainer(mssqlContainerName)
}

func bootstrapMSSQLContainer(ctx context.Context, docker *testutils.DockerClient) (*container.InspectResponse, error) {
	if err := docker.PullImage(ctx, msSQLImage); err != nil {
		return nil, fmt.Errorf("pull mssql image: %w", err)
	}

	contCfg := &container.Config{
		Env: []string{
			"ACCEPT_EULA=Y",
			"MSSQL_SA_PASSWORD=" + msSQLPassword,
		},
		ExposedPorts: network.PortSet{
			mssqlPort: {},
		},
		Image: msSQLImage,
	}

	hostCfg := &container.HostConfig{
		AutoRemove:      true,
		PublishAllPorts: true,
	}

	cont, err := docker.RunContainer(ctx, contCfg, hostCfg, mssqlContainerName)
	if err != nil {
		return nil, fmt.Errorf("run mssql container: %w", err)
	}

	needsCleanup := true
	defer func() {
		if needsCleanup {
			cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			_ = docker.RemoveContainer(cleanupCtx, cont.ID)
		}
	}()

	port, err := docker.GetHostPort(cont, mssqlPort)
	if err != nil {
		return nil, fmt.Errorf("get mssql host port: %w", err)
	}

	dbMasterURI := msSQLConnectionURI("localhost", port, "master", msSQLUsername, msSQLPassword)
	dbTestURI := msSQLConnectionURI("localhost", port, msSQLTemplateDB, msSQLUsername, msSQLPassword)

	if err := waitForDatabase("mssql", dbMasterURI); err != nil {
		return nil, fmt.Errorf("wait for mssql database: %w", err)
	}

	query := fmt.Sprintf("CREATE DATABASE [%s] COLLATE Latin1_General_CS_AS", msSQLTemplateDB)
	if err := execMSSQLQuery(ctx, docker, cont.ID, "localhost", msSQLUsername, msSQLPassword, query); err != nil {
		return nil, fmt.Errorf("create mssql database: %w", err)
	}

	db, err := goose.OpenDBWithDriver("mssql", dbTestURI)
	if err != nil {
		return nil, fmt.Errorf("migrate mssql database: %w", err)
	}
	defer db.Close()

	if err := goose.Up(db, assets.MSSQLMigrationDir); err != nil {
		return nil, fmt.Errorf("apply mssql migrations: %w", err)
	}

	backupQuery := mssqlBackupQuery(msSQLTemplateDB, msSQLTemplateDBDump)
	if err := execMSSQLQuery(ctx, docker, cont.ID, "localhost", msSQLUsername, msSQLPassword, backupQuery); err != nil {
		return nil, fmt.Errorf("backup mssql database: %w", err)
	}

	needsCleanup = false
	return cont, nil
}

func execMSSQLQuery(ctx context.Context, docker *testutils.DockerClient, containerID, host, username, password, query string) error {
	execOpts := client.ExecCreateOptions{
		Cmd: []string{"/opt/mssql-tools18/bin/sqlcmd", "-S", host, "-U", username, "-P", password, "-Q", query, "-C"},
	}

	return docker.ExecCommand(ctx, containerID, execOpts)
}

func mssqlBackupQuery(database, backupPath string) string {
	return fmt.Sprintf("BACKUP DATABASE [%s] TO DISK = '%s' WITH FORMAT, INIT, SKIP, NOREWIND, NOUNLOAD, STATS = 10", database, backupPath)
}

func mssqlRestoreQuery(sourceDatabase, backupPath, targetDatabase string) string {
	return fmt.Sprintf(
		"RESTORE DATABASE [%s] FROM DISK = '%s' WITH REPLACE, MOVE '%s' TO '/var/opt/mssql/data/%s.mdf', MOVE '%s_log' TO '/var/opt/mssql/data/%s_log.ldf', STATS = 10",
		targetDatabase,
		backupPath,
		sourceDatabase,
		targetDatabase,
		sourceDatabase,
		targetDatabase,
	)
}

func msSQLConnectionURI(host, port, database, username, password string) string {
	creds := ""
	if username != "" && password != "" {
		creds = fmt.Sprintf("%s:%s@", username, password)
	}

	return fmt.Sprintf(
		"sqlserver://%s%s:%s?database=%s",
		creds,
		host,
		port,
		database,
	)
}
