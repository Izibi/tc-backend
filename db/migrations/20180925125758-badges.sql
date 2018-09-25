
-- +migrate Up

CREATE TABLE badges (
    id BIGINT NOT NULL AUTO_INCREMENT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    symbol TEXT NOT NULL,
    PRIMARY KEY (id)
) CHARACTER SET utf8 ENGINE=InnoDB;
CREATE INDEX ix_badges__symbol USING btree ON badges (symbol(63));

-- +migrate Down

DROP TABLE badges;
