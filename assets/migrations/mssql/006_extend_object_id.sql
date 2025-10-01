-- +goose Up
ALTER TABLE tuple DROP CONSTRAINT PK_tuple;
ALTER TABLE tuple ALTER COLUMN object_id NVARCHAR(255) NOT NULL;
ALTER TABLE tuple ADD CONSTRAINT PK_tuple PRIMARY KEY NONCLUSTERED (store, object_type, object_id, relation, _user);

-- +goose Down
ALTER TABLE tuple DROP CONSTRAINT PK_tuple;
ALTER TABLE tuple ALTER COLUMN object_id NVARCHAR(128) NOT NULL;
ALTER TABLE tuple ADD CONSTRAINT PK_tuple PRIMARY KEY NONCLUSTERED (store, object_type, object_id, relation, _user);
