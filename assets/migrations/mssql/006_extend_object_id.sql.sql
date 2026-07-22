-- +goose Up

/*
There is an issue with MSSQL and replication: because the primary key is a composite over `(store, object_type,
object_id, relation, _user)`, we can't change a field that belongs to this index (in this case `object_id`) - the
only solution would be to drop the primary key, change the column and recreate it - however, the replication prevents us
from dropping the primary key, so in order to not have the same issue in the future we'll introduce a useless column
which will be the primary key and add a standard constraint + composite index over the columns (so that if tomorrow we
need to change a column again, we'll be able to do it because they're not part of the primary key anymore and the
replication won't prevent us from doing so).

The downside of this approach is that it introduces a new column, `_id`, which is not used by OpenFGA but basically only
exist to ease our life when using the replication with MSSQL.

Note that before running this migration, make sure to disable temporarily the replication so that the migration will be
able to drop the primary key and recreate it.
*/
ALTER TABLE tuple ADD _id INT IDENTITY NOT NULL;

CREATE UNIQUE NONCLUSTERED INDEX idx_object_lookup ON tuple (store, object_type, object_id, relation, _user);

ALTER TABLE tuple DROP CONSTRAINT PK_tuple;
ALTER TABLE tuple ADD CONSTRAINT PK_tuple PRIMARY KEY CLUSTERED (_id);

ALTER TABLE tuple ALTER COLUMN object_id NVARCHAR(255) NOT NULL;

-- +goose Down
ALTER TABLE tuple ALTER COLUMN object_id NVARCHAR(128) NOT NULL;

ALTER TABLE tuple DROP CONSTRAINT PK_tuple;
ALTER TABLE tuple ADD CONSTRAINT PK_tuple PRIMARY KEY NONCLUSTERED (store, object_type, object_id, relation, _user);

DROP INDEX idx_object_lookup ON tuple;

ALTER TABLE tuple DROP COLUMN _id;
