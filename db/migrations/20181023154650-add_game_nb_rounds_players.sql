
-- +migrate Up

ALTER TABLE games ADD COLUMN `max_nb_rounds` int NOT NULL DEFAULT 0;
ALTER TABLE games ADD COLUMN `max_nb_players` int NOT NULL DEFAULT 0;

-- +migrate Down

ALTER TABLE games DROP COLUMN `max_nb_rounds`;
ALTER TABLE games DROP COLUMN `max_nb_players`;
