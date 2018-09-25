
-- +migrate Up

CREATE TABLE user_badges (
    user_id BIGINT NOT NULL,
    badge_id BIGINT NOT NULL,
    PRIMARY KEY (user_id, badge_id)
) CHARACTER SET utf8 ENGINE=InnoDB;
CREATE INDEX ix_user_badges__badge_id USING btree ON user_badges (badge_id);

ALTER TABLE user_badges ADD CONSTRAINT fk_user_badges__user_id
    FOREIGN KEY ix_user_badges__user_id (user_id) REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE user_badges ADD CONSTRAINT fk_user_badges__badge_id
    FOREIGN KEY ix_user_badges__badge_id (badge_id) REFERENCES badges(id) ON DELETE CASCADE;

-- +migrate Down

DROP TABLE user_badges;
