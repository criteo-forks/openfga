-- +goose Up
ALTER TABLE tuple ADD condition_name NVARCHAR(256);
ALTER TABLE tuple ADD condition_context VARBINARY(MAX);
ALTER TABLE changelog ADD condition_name NVARCHAR(256);
ALTER TABLE changelog ADD condition_context VARBINARY(MAX);

-- +goose Down
ALTER TABLE tuple DROP COLUMN condition_name;
ALTER TABLE tuple DROP COLUMN condition_context;
ALTER TABLE changelog DROP COLUMN condition_name;
ALTER TABLE changelog DROP COLUMN condition_context;
