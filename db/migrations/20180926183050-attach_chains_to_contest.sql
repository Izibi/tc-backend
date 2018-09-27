-- Attaches chains to contests instead of contest_periods
-- Add an explicit timestamp range to each contest period so chains can still
-- be filtered by period if it becomes useful.

-- +migrate Up

ALTER TABLE contest_periods ADD COLUMN starts_at TIMESTAMP NOT NULL;
ALTER TABLE contest_periods ADD COLUMN ends_at TIMESTAMP NOT NULL;

ALTER TABLE chains ADD COLUMN contest_id BIGINT NOT NULL;
UPDATE chains c
  INNER JOIN contest_periods p ON c.period_id = p.id
  SET c.contest_id = p.contest_id;
CREATE INDEX ix_chains__contest_id USING btree ON chains (contest_id);
ALTER TABLE chains ADD CONSTRAINT fk_chains__contest_id
    FOREIGN KEY ix_chains__contest_id (contest_id)
    REFERENCES contests(id) ON DELETE CASCADE;

ALTER TABLE chains DROP FOREIGN KEY fk_chains__period_id;
ALTER TABLE chains DROP INDEX ix_chains__period_id;
ALTER TABLE chains DROP COLUMN period_id;

-- +migrate Down

ALTER TABLE chains ADD COLUMN period_id BIGINT NOT NULL;
UPDATE chains c
  INNER JOIN contest_periods p ON c.contest_id = p.contest_id
  SET c.period_id = p.id
  WHERE c.updated_at >= p.starts_at AND c.updated_at < p.ends_at;

CREATE INDEX ix_chains__period_id USING btree ON chains (owner_id);
ALTER TABLE chains ADD CONSTRAINT fk_chains__period_id
    FOREIGN KEY (period_id) REFERENCES contest_periods(id) ON DELETE CASCADE;

ALTER TABLE chains DROP FOREIGN KEY fk_chains__contest_id;
ALTER TABLE chains DROP INDEX ix_chains__contest_id;
ALTER TABLE chains DROP COLUMN contest_id;

-- Disabled to avoid losing period boundaries:
-- ALTER TABLE contest_periods DROP COLUMN starts_at;
-- ALTER TABLE contest_periods DROP COLUMN ends_at;
