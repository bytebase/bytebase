
CREATE TABLE ent_sf_geo (
	id bigint unsigned NOT NULL,
	pt point NOT NULL SRID 4326,
	PRIMARY KEY (id),
	SPATIAL KEY ent_sf_sp_seed (pt)
) ENGINE=InnoDB;
CREATE TABLE ent_sf_vis (
	id bigint unsigned NOT NULL,
	visa int NOT NULL DEFAULT '0',
	visb int INVISIBLE NOT NULL DEFAULT '0',
	visc int NOT NULL DEFAULT '0',
	PRIMARY KEY (id),
	KEY ent_sf_ki_a (visa) INVISIBLE,
	KEY ent_sf_ki_b (visc)
) ENGINE=InnoDB;
CREATE TABLE ent_sf_chk (
	id bigint unsigned NOT NULL,
	score int NOT NULL DEFAULT '0',
	bound int NOT NULL DEFAULT '10',
	PRIMARY KEY (id),
	CONSTRAINT ent_sf_chk_seed CHECK ((score >= 0))
) ENGINE=InnoDB;
