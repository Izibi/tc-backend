
-- +migrate Up

ALTER TABLE teams MODIFY COLUMN `public_key` VARCHAR(64) NOT NULL DEFAULT "";

-- +migrate Down

ALTER TABLE teams MODIFY COLUMN `public_key` TEXT;
