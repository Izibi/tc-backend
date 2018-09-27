
-- +migrate Up

ALTER TABLE chains ADD COLUMN parent_id BIGINT NULL;
CREATE INDEX ix_chains__parent_id USING btree ON chains (parent_id);
ALTER TABLE chains ADD CONSTRAINT fk_chains__parent_id
    FOREIGN KEY (parent_id) REFERENCES chains(id) ON DELETE SET NULL;

-- +migrate Down

ALTER TABLE chains DROP FOREIGN KEY fk_chains__parent_id;
ALTER TABLE chains DROP INDEX ix_chains__parent_id;
ALTER TABLE chains DROP COLUMN parent_id;
