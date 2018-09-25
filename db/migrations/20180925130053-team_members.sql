
-- +migrate Up

CREATE TABLE team_members (
    team_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    joined_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    is_creator BOOLEAN NOT NULL,
    PRIMARY KEY (team_id, user_id)
) CHARACTER SET utf8 ENGINE=InnoDB;
CREATE INDEX ix_team_members__user_id USING btree ON team_members (user_id);

ALTER TABLE team_members ADD CONSTRAINT fk_team_members__team_id
    FOREIGN KEY ix_team_members__team_id (team_id) REFERENCES teams(id) ON DELETE CASCADE;
ALTER TABLE team_members ADD CONSTRAINT fk_team_members__user_id
    FOREIGN KEY ix_team_members__team_id (user_id) REFERENCES users(id) ON DELETE CASCADE;

-- +migrate Down

DROP TABLE team_members;
