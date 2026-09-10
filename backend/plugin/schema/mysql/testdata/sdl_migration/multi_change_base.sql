
CREATE TABLE category (
	id INT NOT NULL AUTO_INCREMENT,
	name VARCHAR(80) NOT NULL,
	legacy_code VARCHAR(40) NOT NULL DEFAULT '',
	PRIMARY KEY (id),
	KEY idx_cat_legacy (legacy_code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE product (
	id INT NOT NULL AUTO_INCREMENT,
	category_id INT NOT NULL,
	name VARCHAR(120) NOT NULL,
	price DECIMAL(10,2) NOT NULL DEFAULT 0.00,
	PRIMARY KEY (id),
	KEY idx_prod_cat (category_id),
	CONSTRAINT fk_prod_cat FOREIGN KEY (category_id) REFERENCES category (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE legacy_audit (
	id INT NOT NULL AUTO_INCREMENT,
	product_id INT NOT NULL,
	note VARCHAR(200),
	PRIMARY KEY (id),
	KEY idx_legacy_audit_prod (product_id),
	CONSTRAINT fk_legacy_audit_prod FOREIGN KEY (product_id) REFERENCES product (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE daily_stat (
	id INT NOT NULL AUTO_INCREMENT,
	bucket INT NOT NULL,
	val DOUBLE NOT NULL DEFAULT 0,
	PRIMARY KEY (id, bucket)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
PARTITION BY HASH (bucket) PARTITIONS 2;

CREATE VIEW v_catalog AS
SELECT p.id, p.name, p.price, c.name AS category
FROM product p JOIN category c ON p.category_id = c.id;
