
-- +migrate Up

CREATE TABLE contests (
    id BIGINT NOT NULL AUTO_INCREMENT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    logo_url TEXT NOT NULL,
    task_id BIGINT NOT NULL,
    is_registration_open BOOLEAN NOT NULL,
    starts_at DATETIME NOT NULL,
    ends_at DATETIME NOT NULL,
    required_badge_id BIGINT NOT NULL,
    -- XXX current_period_id BIGINT NULL,
    PRIMARY KEY (id)
) CHARACTER SET utf8 ENGINE=InnoDB;
CREATE INDEX ix_contests__task_id USING btree ON contests (task_id);
CREATE INDEX ix_contests__required_badge_id USING btree ON contests (required_badge_id);

ALTER TABLE contests ADD CONSTRAINT fk_contests__task_id
    FOREIGN KEY ix_contests__task_id (task_id) REFERENCES tasks(id) ON DELETE CASCADE;
ALTER TABLE contests ADD CONSTRAINT fk_contests__required_badge_id
    FOREIGN KEY ix_contests__required_badge_id (required_badge_id) REFERENCES badges(id) ON DELETE CASCADE;

-- +migrate Down

DROP TABLE contests;
