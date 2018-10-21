
-- +migrate Up

ALTER TABLE teams ADD COLUMN deleted_at TIMESTAMP NULL DEFAULT NULL;

-- +migrate Down

ALTER TABLE teams DROP COLUMN deleted_at;
