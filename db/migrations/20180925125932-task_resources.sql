
-- +migrate Up

CREATE TABLE task_resources (
    id BIGINT NOT NULL AUTO_INCREMENT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    task_id BIGINT NOT NULL,
    rank INT NOT NULL,
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    url TEXT NOT NULL,
    html TEXT NOT NULL,
    PRIMARY KEY (id)
) CHARACTER SET utf8 ENGINE=InnoDB;
CREATE INDEX ix_task_resources__task_id_rank USING btree ON task_resources (task_id, rank);

ALTER TABLE task_resources ADD CONSTRAINT fk_task_resources__task_id
    FOREIGN KEY ix_task_resources__task_id (task_id) REFERENCES tasks(id) ON DELETE CASCADE;

-- +migrate Down

DROP TABLE task_resources;
