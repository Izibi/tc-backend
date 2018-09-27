
-- +migrate Up

ALTER TABLE chains ADD COLUMN protocol_hash VARCHAR(27) NOT NULL DEFAULT "";
ALTER TABLE chains ADD COLUMN new_protocol_hash VARCHAR(27) NOT NULL DEFAULT "";

-- +migrate Down

ALTER TABLE chains DROP COLUMN new_protocol_hash;
ALTER TABLE chains DROP COLUMN protocol_hash;
