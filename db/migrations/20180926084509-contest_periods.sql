
-- +migrate Up

CREATE TABLE contest_periods (
    id BIGINT NOT NULL AUTO_INCREMENT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    contest_id BIGINT NOT NULL,
    sequence INT NOT NULL DEFAULT 1,
    election_at DATETIME NULL DEFAULT NULL,
    PRIMARY KEY (id)
) CHARACTER SET utf8 ENGINE=InnoDB;
CREATE INDEX ix_contest_periods__contest_id_sequence USING btree ON contest_periods (contest_id, sequence);

ALTER TABLE contest_periods ADD CONSTRAINT fk_contest_periods__contest_id
  FOREIGN KEY ix_contest_periods__contest_id_sequence (contest_id)
  REFERENCES contests(id) ON DELETE CASCADE;

-- +migrate Down

DROP TABLE contest_periods;
