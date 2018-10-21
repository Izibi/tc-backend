
-- +migrate Up

ALTER TABLE chains DROP COLUMN round;

-- +migrate Down

ALTER TABLE chains ADD COLUMN round INT NOT NULL DEFAULT 0;
