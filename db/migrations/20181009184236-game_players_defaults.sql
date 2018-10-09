
-- +migrate Up

ALTER TABLE game_players ALTER COLUMN used SET DEFAULT "";
ALTER TABLE game_players ALTER COLUMN unused SET DEFAULT "";

-- +migrate Down

ALTER TABLE game_players ALTER COLUMN used DROP DEFAULT;
ALTER TABLE game_players ALTER COLUMN unused DROP DEFAULT;
