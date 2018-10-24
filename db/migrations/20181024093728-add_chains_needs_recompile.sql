
-- +migrate Up

ALTER TABLE chains ADD COLUMN `needs_recompile` BOOLEAN NOT NULL DEFAULT 0;

-- +migrate Down

ALTER TABLE chains DROP COLUMN `needs_recompile`;
