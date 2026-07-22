-- +goose Up
CREATE TABLE tuple (
    store NVARCHAR(26) NOT NULL,
    object_type NVARCHAR(128) NOT NULL,
    object_id NVARCHAR(128) NOT NULL,
    relation NVARCHAR(50) NOT NULL,
    _user NVARCHAR(256) NOT NULL,
    user_type NVARCHAR(7) NOT NULL,
    ulid NVARCHAR(26) NOT NULL,
    inserted_at DATETIME2 NOT NULL,
    CONSTRAINT PK_tuple PRIMARY KEY NONCLUSTERED (store, object_type, object_id, relation, _user)
);

CREATE UNIQUE INDEX idx_tuple_ulid ON tuple (ulid);

CREATE TABLE authorization_model (
    store NVARCHAR(26) NOT NULL,
    authorization_model_id NVARCHAR(26) NOT NULL,
    type NVARCHAR(256) NOT NULL,
    type_definition VARBINARY(MAX),
    CONSTRAINT PK_authorization_model PRIMARY KEY (store, authorization_model_id, type)
);

CREATE TABLE store (
    id NVARCHAR(26) NOT NULL,
    name NVARCHAR(64) NOT NULL,
    created_at DATETIME2 NOT NULL,
    updated_at DATETIME2,
    deleted_at DATETIME2,
    CONSTRAINT PK_store PRIMARY KEY (id)
);

CREATE TABLE assertion (
    store NVARCHAR(26) NOT NULL,
    authorization_model_id NVARCHAR(26) NOT NULL,
    assertions VARBINARY(MAX),
    CONSTRAINT PK_assertion PRIMARY KEY (store, authorization_model_id)
);

CREATE TABLE changelog (
    store NVARCHAR(26) NOT NULL,
    object_type NVARCHAR(256) NOT NULL,
    object_id NVARCHAR(256) NOT NULL,
    relation NVARCHAR(50) NOT NULL,
    _user NVARCHAR(512) NOT NULL,
    operation BIGINT NOT NULL,
    ulid NVARCHAR(26) NOT NULL,
    inserted_at DATETIME2 NOT NULL,
    CONSTRAINT PK_changelog PRIMARY KEY (store, ulid, object_type)
);

-- +goose Down
DROP TABLE tuple;
DROP TABLE authorization_model;
DROP TABLE store;
DROP TABLE assertion;
DROP TABLE changelog;