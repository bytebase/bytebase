
CREATE TABLE customer (
	id INT NOT NULL AUTO_INCREMENT,
	name VARCHAR(100) NOT NULL,
	email VARCHAR(255) NOT NULL,
	loyalty_points INT NOT NULL DEFAULT 0,
	PRIMARY KEY (id),
	UNIQUE KEY uk_cust_email (email)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE ord (
	id INT NOT NULL AUTO_INCREMENT,
	customer_id INT NOT NULL,
	total DECIMAL(10,2) NOT NULL DEFAULT 0.00,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY (id),
	KEY idx_ord_cust (customer_id),
	KEY idx_ord_created (created_at),
	CONSTRAINT fk_ord_cust FOREIGN KEY (customer_id) REFERENCES customer (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE product (
	id INT NOT NULL AUTO_INCREMENT,
	sku VARCHAR(40) NOT NULL,
	name VARCHAR(120) NOT NULL,
	price DECIMAL(10,2) NOT NULL DEFAULT 0.00,
	price_with_tax DECIMAL(12,4) AS (price * 1.1) STORED,
	PRIMARY KEY (id),
	UNIQUE KEY uk_prod_sku (sku)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE order_item (
	id INT NOT NULL AUTO_INCREMENT,
	order_id INT NOT NULL,
	product_id INT NOT NULL,
	qty INT NOT NULL DEFAULT 1,
	PRIMARY KEY (id),
	KEY idx_oi_order (order_id),
	KEY idx_oi_product (product_id),
	CONSTRAINT fk_oi_order FOREIGN KEY (order_id) REFERENCES ord (id) ON DELETE CASCADE,
	CONSTRAINT fk_oi_product FOREIGN KEY (product_id) REFERENCES product (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE VIEW v_cust_orders AS
SELECT c.name, o.id AS order_id, o.total
FROM customer c JOIN ord o ON o.customer_id = c.id;
