
-- +migrate Up

CREATE TABLE games (
  id BIGINT NOT NULL AUTO_INCREMENT,
  game_key VARCHAR(43) NOT NULL DEFAULT "",
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  owner_id BIGINT NOT NULL,
  first_block VARCHAR(27) NOT NULL,
  last_block VARCHAR(27) NOT NULL,
  started_at DATETIME NULL DEFAULT NULL,
  round_ends_at DATETIME NULL DEFAULT NULL,
  current_round INT NOT NULL DEFAULT 0,
  PRIMARY KEY (id)
) CHARACTER SET utf8 ENGINE=InnoDB;

CREATE UNIQUE INDEX ix_games__game_key USING btree ON games (game_key);

CREATE INDEX ix_games__owner_id USING btree ON games (owner_id);
ALTER TABLE games ADD CONSTRAINT fk_games__owner_id
    FOREIGN KEY ix_games__owner_id (owner_id) REFERENCES teams(id) ON DELETE CASCADE;

-- +migrate Down

DROP TABLE games;
