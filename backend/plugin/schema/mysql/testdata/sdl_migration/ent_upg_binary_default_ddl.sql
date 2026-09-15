
CREATE TABLE ent_up_bin (
  id int NOT NULL,
  u binary(16) NOT NULL DEFAULT '',
  v varbinary(24) NOT NULL DEFAULT 'ab',
  PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
