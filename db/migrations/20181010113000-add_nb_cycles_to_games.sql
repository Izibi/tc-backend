
-- +migrate Up

ALTER TABLE games ADD COLUMN nb_cycles_per_round int NOT NULL DEFAULT 1;

-- +migrate Down

ALTER TABLE games DROP COLUMN nb_cycles_per_round;
