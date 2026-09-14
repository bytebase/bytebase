
CREATE TABLE category (
	id INT NOT NULL AUTO_INCREMENT,
	name VARCHAR(80) NOT NULL,
	PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE product (
	id INT NOT NULL AUTO_INCREMENT,
	category_id INT NOT NULL,
	name VARCHAR(120) NOT NULL,
	price DECIMAL(12,4) NOT NULL DEFAULT 0.0000,
	PRIMARY KEY (id),
	KEY idx_prod_cat (category_id),
	KEY idx_prod_name (name),
	CONSTRAINT fk_prod_cat FOREIGN KEY (category_id) REFERENCES category (id) ON DELETE CASCADE,
	CONSTRAINT chk_price CHECK (price >= 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE review (
	id INT NOT NULL AUTO_INCREMENT,
	product_id INT NOT NULL,
	stars INT NOT NULL DEFAULT 5,
	body VARCHAR(500),
	PRIMARY KEY (id),
	KEY idx_review_prod (product_id),
	CONSTRAINT fk_review_prod FOREIGN KEY (product_id) REFERENCES product (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE daily_stat (
	id INT NOT NULL AUTO_INCREMENT,
	bucket INT NOT NULL,
	val DOUBLE NOT NULL DEFAULT 0,
	PRIMARY KEY (id, bucket)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
PARTITION BY HASH (bucket) PARTITIONS 4;

CREATE VIEW v_catalog AS
SELECT p.id, p.name, p.price, c.name AS category, p.category_id
FROM product p JOIN category c ON p.category_id = c.id;

CREATE TRIGGER trg_prod_ins BEFORE INSERT ON product FOR EACH ROW
SET NEW.name = TRIM(NEW.name);
