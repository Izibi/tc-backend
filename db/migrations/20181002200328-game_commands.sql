
-- +migrate Up

ALTER TABLE games ADD COLUMN locked BOOLEAN NOT NULL;
ALTER TABLE games ADD COLUMN next_block_commands TEXT NOT NULL;
ALTER TABLE game_players ADD COLUMN locked_at TIMESTAMP NULL DEFAULT NULL;
ALTER TABLE game_players ADD COLUMN used TEXT NOT NULL;
ALTER TABLE game_players ADD COLUMN unused TEXT NOT NULL;

-- +migrate Down

ALTER TABLE games DROP COLUMN locked;
ALTER TABLE games DROP COLUMN next_block_commands;
ALTER TABLE game_players DROP COLUMN locked_at;
ALTER TABLE game_players DROP COLUMN used;
ALTER TABLE game_players DROP COLUMN unused;
