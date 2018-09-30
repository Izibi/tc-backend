
-- +migrate Up

CREATE TABLE game_players (
  game_id BIGINT NOT NULL,
  rank INT NOT NULL,
  team_id BIGINT NOT NULL,
  team_player INT NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  commands TEXT NOT NULL,
  PRIMARY KEY (game_id, rank)
) CHARACTER SET utf8 ENGINE=InnoDB;

CREATE INDEX ix_games__game_id USING btree ON game_players (game_id);
CREATE INDEX ix_games__team_id USING btree ON game_players (team_id);

ALTER TABLE game_players ADD CONSTRAINT fk_game_players__game_id
  FOREIGN KEY ix_games__game_id (game_id) REFERENCES games(id) ON DELETE CASCADE;
ALTER TABLE game_players ADD CONSTRAINT fk_game_players__team_id
  FOREIGN KEY ix_games__team_id (team_id) REFERENCES teams(id) ON DELETE CASCADE;

-- +migrate Down

DROP TABLE game_players;
