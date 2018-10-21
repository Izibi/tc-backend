
-- +migrate Up

ALTER TABLE games DROP COLUMN current_round;

-- +migrate Down

ALTER TABLE games ADD COLUMN current_round INT NOT NULL DEFAULT 0;
