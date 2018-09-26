
-- +migrate Up

CREATE TABLE chain_statuses (
  id SMALLINT NOT NULL AUTO_INCREMENT,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  title TEXT NOT NULL,
  is_public BOOLEAN NOT NULL default 0,
  is_candidate BOOLEAN NOT NULL default 0,
  is_main BOOLEAN NOT NULL default 0,
  is_current BOOLEAN NOT NULL default 0,
  is_valid BOOLEAN NOT NULL default 0,
  PRIMARY KEY (id)
);
INSERT INTO chain_statuses
  (title,is_public,is_candidate,is_main,is_current,is_valid)
VALUES
  ("private test",   0, 0, 0, 1, 1),
  ("public test",    1, 0, 0, 1, 1),
  ("candidate",      1, 1, 0, 1, 1),
  ("main",           1, 0, 1, 1, 1),
  ("past",           1, 0, 0, 0, 1),
  ("invalid",        0, 0, 0, 0, 0);

-- +migrate Down

DROP TABLE chain_statuses;
