
-- +migrate Up

CREATE TABLE teams (
    id BIGINT NOT NULL AUTO_INCREMENT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    access_code VARCHAR(32) NOT NULL,
    contest_id BIGINT NOT NULL,
    is_open BOOLEAN NOT NULL,
    is_locked BOOLEAN NOT NULL,
    name TEXT NOT NULL,
    public_key TEXT NULL,
    PRIMARY KEY (id)
) CHARACTER SET utf8 ENGINE=InnoDB;
CREATE INDEX ix_teams__contest_id USING btree ON teams (contest_id);
CREATE UNIQUE INDEX ix_teams__access_code USING btree ON teams (access_code);

-- +migrate Down

DROP TABLE teams;
