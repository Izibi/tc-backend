
-- +migrate Up

CREATE TABLE users (
    id BIGINT NOT NULL AUTO_INCREMENT,
    foreign_id VARCHAR(64) NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    is_admin BOOLEAN NOT NULL DEFAULT 0,
    username TEXT NOT NULL,
    firstname TEXT NOT NULL,
    lastname TEXT NOT NULL,
    PRIMARY KEY (id)
) CHARACTER SET utf8 ENGINE=InnoDB;
CREATE UNIQUE INDEX ix_users__foreign_id USING btree ON users (foreign_id);

-- +migrate Down

DROP TABLE users;
