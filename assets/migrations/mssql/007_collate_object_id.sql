-- +goose Up
ALTER TABLE tuple DROP CONSTRAINT PK_tuple;
ALTER TABLE tuple ALTER COLUMN object_id NVARCHAR(255) COLLATE Latin1_General_100_BIN2_UTF8 NOT NULL;
ALTER TABLE tuple ADD CONSTRAINT PK_tuple PRIMARY KEY NONCLUSTERED (store, object_type, object_id, relation, _user);

CREATE INDEX idx_user_lookup ON tuple (store, _user, relation, object_type, object_id);

DROP INDEX idx_reverse_lookup_user ON tuple;

-- +goose Down
DROP INDEX idx_user_lookup ON tuple;

ALTER TABLE tuple DROP CONSTRAINT PK_tuple;
ALTER TABLE tuple ALTER COLUMN object_id NVARCHAR(255) NOT NULL;
ALTER TABLE tuple ADD CONSTRAINT PK_tuple PRIMARY KEY NONCLUSTERED (store, object_type, object_id, relation, _user);

CREATE INDEX idx_reverse_lookup_user on tuple (store, object_type, relation, _user);
