
-- +migrate Up

ALTER TABLE chains ADD COLUMN nb_votes_approve INT NOT NULL DEFAULT 0;
ALTER TABLE chains ADD COLUMN nb_votes_reject INT NOT NULL DEFAULT 0;
ALTER TABLE chains ADD COLUMN nb_votes_unknown INT NOT NULL DEFAULT 0;

-- +migrate Down

ALTER TABLE chains DROP COLUMN nb_votes_unknown;
ALTER TABLE chains DROP COLUMN nb_votes_reject;
ALTER TABLE chains DROP COLUMN nb_votes_approve;
