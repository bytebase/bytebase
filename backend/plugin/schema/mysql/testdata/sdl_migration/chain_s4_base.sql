
CREATE TABLE customer (
	id INT NOT NULL AUTO_INCREMENT,
	name VARCHAR(200) NOT NULL,
	email VARCHAR(255) NOT NULL,
	loyalty_points INT NOT NULL DEFAULT 100,
	PRIMARY KEY (id),
	UNIQUE KEY uk_cust_email (email)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE ord (
	id INT NOT NULL AUTO_INCREMENT,
	customer_id INT NOT NULL,
	total DECIMAL(14,4) NOT NULL DEFAULT 0.0000,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY (id),
	KEY idx_ord_cust (customer_id),
	KEY idx_ord_created (created_at){{ORD_CHECK}}
	,CONSTRAINT fk_ord_cust FOREIGN KEY (customer_id) REFERENCES customer (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE order_item (
	id INT NOT NULL AUTO_INCREMENT,
	order_id INT NOT NULL,
	qty INT NOT NULL DEFAULT 1,
	PRIMARY KEY (id),
	KEY idx_oi_order (order_id),
	CONSTRAINT fk_oi_order FOREIGN KEY (order_id) REFERENCES ord (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE VIEW v_cust_orders AS
SELECT c.name, c.email, o.id AS order_id, o.total
FROM customer c JOIN ord o ON o.customer_id = c.id;

CREATE TRIGGER trg_ord_ins BEFORE INSERT ON ord FOR EACH ROW
SET NEW.created_at = NOW();
