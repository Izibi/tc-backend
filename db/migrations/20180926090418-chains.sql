
-- +migrate Up

-- game_key stores base64-encoded 256-bit game keys

CREATE TABLE chains (
    id BIGINT NOT NULL AUTO_INCREMENT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    started_at DATETIME NULL DEFAULT NULL,
    status_id SMALLINT NOT NULL,
    period_id BIGINT NOT NULL,
    owner_id BIGINT NULL DEFAULT NULL,

    title TEXT NOT NULL DEFAULT "",
    description_text TEXT NOT NULL DEFAULT "",
    interface_text TEXT NOT NULL DEFAULT "",
    implementation_text TEXT NOT NULL DEFAULT "",

    game_key VARCHAR(43) NOT NULL DEFAULT "",
    round INT NOT NULL DEFAULT 0,

    PRIMARY KEY (id)
) CHARACTER SET utf8 ENGINE=InnoDB;

CREATE INDEX ix_chains__period_id USING btree ON chains (owner_id);
CREATE INDEX ix_chains__owner_id USING btree ON chains (owner_id);
CREATE INDEX ix_chains__status_id USING btree ON chains (status_id);

ALTER TABLE chains ADD CONSTRAINT fk_chains__owner_id
    FOREIGN KEY (owner_id) REFERENCES teams(id) ON DELETE CASCADE;
ALTER TABLE chains ADD CONSTRAINT fk_chains__period_id
    FOREIGN KEY (period_id) REFERENCES contest_periods(id) ON DELETE CASCADE;
ALTER TABLE chains ADD CONSTRAINT fk_chains__status_id
    FOREIGN KEY (status_id) REFERENCES chain_statuses(id) ON UPDATE CASCADE;

-- +migrate Down

DROP TABLE chains;
