
CREATE TABLE ent_sf_enum (
	id bigint unsigned NOT NULL,
	mood enum('sunny','cloudy') NOT NULL DEFAULT 'sunny',
	grade enum('a','b','c') NOT NULL DEFAULT 'a',
	PRIMARY KEY (id)
) ENGINE=InnoDB;
CREATE TABLE ent_sf_text (
	id bigint unsigned NOT NULL,
	title varchar(120) NOT NULL DEFAULT '',
	body text,
	summary text,
	PRIMARY KEY (id),
	FULLTEXT KEY ent_sf_ft_seed (body)
) ENGINE=InnoDB;
CREATE TABLE ent_sf_gen (
	id bigint unsigned NOT NULL,
	price decimal(8,2) NOT NULL DEFAULT '0.00',
	qty int NOT NULL DEFAULT '0',
	total decimal(10,2) GENERATED ALWAYS AS ((price * 2)) STORED,
	total2 bigint GENERATED ALWAYS AS ((qty + 7)) VIRTUAL,
	PRIMARY KEY (id)
) ENGINE=InnoDB;
