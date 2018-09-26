
-- +migrate Up

ALTER TABLE task_resources DROP FOREIGN KEY fk_task_resources__task_id;
DROP INDEX ix_task_resources__task_id_rank ON task_resources;
CREATE UNIQUE INDEX ix_task_resources__task_id_rank USING btree ON task_resources (task_id, rank);
ALTER TABLE task_resources ADD CONSTRAINT fk_task_resources__task_id
    FOREIGN KEY ix_task_resources__task_id (task_id) REFERENCES tasks(id) ON DELETE CASCADE;

-- +migrate Down

ALTER TABLE task_resources DROP FOREIGN KEY fk_task_resources__task_id;
DROP INDEX ix_task_resources__task_id_rank ON task_resources;
CREATE INDEX ix_task_resources__task_id_rank USING btree ON task_resources (task_id, rank);
ALTER TABLE task_resources ADD CONSTRAINT fk_task_resources__task_id
    FOREIGN KEY ix_task_resources__task_id (task_id) REFERENCES tasks(id) ON DELETE CASCADE;
