
-- +migrate Up

ALTER TABLE games ALTER COLUMN locked SET DEFAULT false;
ALTER TABLE games ALTER COLUMN next_block_commands SET DEFAULT "";

-- +migrate Down

ALTER TABLE games ALTER COLUMN locked DROP DEFAULT;
ALTER TABLE games ALTER COLUMN next_block_commands DROP DEFAULT;
