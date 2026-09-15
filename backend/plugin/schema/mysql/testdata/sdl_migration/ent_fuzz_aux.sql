
CREATE TABLE ent_fz_part (
	id bigint unsigned NOT NULL,
	bucket integer NOT NULL,
	note varchar(40) DEFAULT '' NOT NULL,
	PRIMARY KEY (id,bucket)
) ENGINE=InnoDB
PARTITION BY HASH (bucket) PARTITIONS 4;
CREATE VIEW ent_fz_v1 AS SELECT userid, username FROM users;
CREATE VIEW ent_fz_v2 AS SELECT hostid, host FROM hosts;
CREATE FUNCTION ent_fz_fn1() RETURNS INT DETERMINISTIC RETURN 1;
CREATE PROCEDURE ent_fz_pr1() BEGIN SELECT 1; END;
