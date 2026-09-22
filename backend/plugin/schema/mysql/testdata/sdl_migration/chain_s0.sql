
CREATE TABLE customer (
	id INT NOT NULL AUTO_INCREMENT,
	name VARCHAR(100) NOT NULL,
	email VARCHAR(255) NOT NULL,
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
	CONSTRAINT fk_ord_cust FOREIGN KEY (customer_id) REFERENCES customer (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE VIEW v_cust_orders AS
SELECT c.name, o.id AS order_id, o.total
FROM customer c JOIN ord o ON o.customer_id = c.id;
