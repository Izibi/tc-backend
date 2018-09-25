
INSERT INTO badges (id, symbol) VALUES
  (1, "https://badges.concours-alkindi.fr/tezos_contests/NTU_2018");

INSERT INTO tasks (id, title, created_at)
  VALUES (1, "race to the center", NOW());
INSERT INTO contests (id, title, description, logo_url, task_id, is_registration_open, starts_at, ends_at, required_badge_id)
  VALUES (1, "Tezos Contest 2018 Part 1", "", "", 1, 1, "2018-09-24 00:00:00", "2018-09-30 23:59:59", 1);

INSERT INTO task_resources (id, task_id, rank, title, description, url, html) VALUES
  (1, 1, 1, "Task description", "This section describes the task", "", "Task description goes <p>here</p>..."),
  (2, 1, 2, "Commands", "", "", "Commands description goes here…"),
  (3, 1, 3, "API", "", "about:blank#2", ""),
  (4, 1, 4, "Examples", "", "about:blank#3", ""),
  (5, 1, 5, "OCaml basics", "", "about:blank#4", "");
