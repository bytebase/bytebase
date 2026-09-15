
CREATE TABLE ent_ranked (
	id bigint unsigned NOT NULL,
	score integer DEFAULT '0' NOT NULL,
	ratio decimal(5,2) DEFAULT '0.00' NOT NULL,
	owner_userid bigint unsigned NULL,
	PRIMARY KEY (id)
) ENGINE=InnoDB;
CREATE TABLE ent_part_log (
	id bigint unsigned NOT NULL,
	bucket integer NOT NULL,
	note varchar(64) DEFAULT '' NOT NULL,
	PRIMARY KEY (id,bucket)
) ENGINE=InnoDB
PARTITION BY RANGE (bucket) (
	PARTITION p0 VALUES LESS THAN (100),
	PARTITION p1 VALUES LESS THAN (200)
);
CREATE VIEW ent_v_users AS SELECT userid, username, name FROM users;
CREATE FUNCTION ent_f_host_count() RETURNS INT READS SQL DATA RETURN (SELECT COUNT(*) FROM hosts);
CREATE PROCEDURE ent_p_touch_user(IN uid BIGINT UNSIGNED) BEGIN UPDATE users SET name = name WHERE userid = uid; END;
CREATE TRIGGER ent_trg_role_bi BEFORE INSERT ON role FOR EACH ROW SET NEW.name = TRIM(NEW.name);
CREATE EVENT ent_ev_hk ON SCHEDULE EVERY 1 DAY DO DELETE FROM changelog WHERE clock < 0;
