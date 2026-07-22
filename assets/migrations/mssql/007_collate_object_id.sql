-- +goose Up
DROP INDEX idx_object_lookup on tuple;
ALTER TABLE tuple ALTER COLUMN object_id NVARCHAR(255) COLLATE Latin1_General_100_BIN2_UTF8 NOT NULL;
CREATE UNIQUE NONCLUSTERED INDEX idx_object_lookup ON tuple (store, object_type, object_id, relation, _user);

CREATE INDEX idx_user_lookup ON tuple (store, _user, relation, object_type, object_id);

DROP INDEX idx_reverse_lookup_user ON tuple;

-- +goose Down
CREATE INDEX idx_reverse_lookup_user on tuple (store, object_type, relation, _user);

DROP INDEX idx_user_lookup ON tuple;

DROP INDEX idx_object_lookup on tuple;
ALTER TABLE tuple ALTER COLUMN object_id NVARCHAR(255) NOT NULL;
CREATE UNIQUE NONCLUSTERED INDEX idx_object_lookup ON tuple (store, object_type, object_id, relation, _user);
