---

CREATE TABLE users (
    id BIGINT NOT NULL AUTO_INCREMENT,
    foreign_id TEXT NOT NULL,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    is_admin BOOLEAN NOT NULL DEFAULT 0,
    username TEXT NOT NULL,
    firstname TEXT NOT NULL,
    lastname TEXT NOT NULL,
    PRIMARY KEY (id)
) CHARACTER SET utf8 ENGINE=InnoDB;
CREATE UNIQUE INDEX ix_users__foreign_id USING btree ON users (foreign_id(64));
-- ALTER TABLE users MODIFY is_admin BOOLEAN NOT NULL DEFAULT 0;
-- ALTER TABLE users ADD COLUMN updated_at DATETIME NOT NULL;

CREATE TABLE badges (
    id BIGINT NOT NULL AUTO_INCREMENT,
    symbol TEXT NOT NULL,
    PRIMARY KEY (id)
) CHARACTER SET utf8 ENGINE=InnoDB;
CREATE INDEX ix_badges__symbol USING btree ON badges (symbol(63));

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

CREATE TABLE teams (
    id BIGINT NOT NULL AUTO_INCREMENT,
    created_at DATETIME NOT NULL,
    access_code TEXT NOT NULL,
    contest_id BIGINT NOT NULL,
    is_open BOOLEAN NOT NULL,
    is_locked BOOLEAN NOT NULL,
    name TEXT NOT NULL,
    public_key TEXT NULL,
    PRIMARY KEY (id)
) CHARACTER SET utf8 ENGINE=InnoDB;
CREATE INDEX ix_teams__contest_id USING btree ON teams (contest_id);

CREATE TABLE team_members (
    team_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    joined_at DATETIME NOT NULL,
    is_creator BOOLEAN NOT NULL,
    PRIMARY KEY (team_id, user_id)
) CHARACTER SET utf8 ENGINE=InnoDB;
CREATE INDEX ix_team_members__user_id USING btree ON team_members (user_id);

ALTER TABLE team_members ADD CONSTRAINT fk_team_members__team_id
    FOREIGN KEY ix_team_members__team_id (team_id) REFERENCES teams(id) ON DELETE CASCADE;
ALTER TABLE team_members ADD CONSTRAINT fk_team_members__user_id
    FOREIGN KEY ix_team_members__team_id (user_id) REFERENCES users(id) ON DELETE CASCADE;

CREATE TABLE tasks (
    id BIGINT NOT NULL AUTO_INCREMENT,
    title TEXT NOT NULL,
    created_at DATETIME NOT NULL,
    PRIMARY KEY (id)
) CHARACTER SET utf8 ENGINE=InnoDB;

CREATE TABLE contests (
    id BIGINT NOT NULL AUTO_INCREMENT,
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    logo_url TEXT NOT NULL,
    task_id BIGINT NOT NULL,
    is_registration_open BOOLEAN NOT NULL,
    starts_at DATETIME NOT NULL,
    ends_at DATETIME NOT NULL,
    required_badge_id BIGINT NOT NULL,
    PRIMARY KEY (id)
) CHARACTER SET utf8 ENGINE=InnoDB;
CREATE INDEX ix_contests__task_id USING btree ON contests (task_id);
CREATE INDEX ix_contests__required_badge_id USING btree ON contests (required_badge_id);

ALTER TABLE contests ADD CONSTRAINT fk_contests__task_id
    FOREIGN KEY ix_contests__task_id (task_id) REFERENCES tasks(id) ON DELETE CASCADE;
ALTER TABLE contests ADD CONSTRAINT fk_contests__required_badge_id
    FOREIGN KEY ix_contests__required_badge_id (required_badge_id) REFERENCES badges(id) ON DELETE CASCADE;

