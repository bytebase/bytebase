
CREATE TABLE department (
	id INT NOT NULL AUTO_INCREMENT,
	name VARCHAR(100) NOT NULL,
	manager_id INT NULL,
	parent_id INT NULL,
	budget DECIMAL(14,2) NOT NULL DEFAULT 0.00,
	PRIMARY KEY (id),
	UNIQUE KEY uk_dept_name (name),
	CONSTRAINT fk_dept_parent FOREIGN KEY (parent_id) REFERENCES department (id) ON DELETE SET NULL,
	CONSTRAINT chk_budget CHECK (budget >= 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE employee (
	id INT NOT NULL AUTO_INCREMENT,
	dept_id INT NOT NULL,
	first_name VARCHAR(50) NOT NULL,
	last_name VARCHAR(50) NOT NULL,
	full_name VARCHAR(101) AS (CONCAT(first_name, ' ', last_name)) VIRTUAL,
	email VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
	active BOOLEAN NOT NULL DEFAULT TRUE,
	salary DECIMAL(10,2) NOT NULL DEFAULT 0.00,
	hired_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
	rank_kind ENUM('junior','mid','senior','staff') NOT NULL DEFAULT 'junior',
	tags SET('remote','contractor','lead') NOT NULL DEFAULT '',
	notes TEXT,
	PRIMARY KEY (id),
	UNIQUE KEY uk_emp_email (email),
	KEY idx_emp_name (last_name, first_name),
	KEY idx_emp_notes_prefix (notes(20)),
	FULLTEXT KEY ft_emp_notes (notes),
	CONSTRAINT chk_salary CHECK (salary >= 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE project (
	id INT NOT NULL AUTO_INCREMENT,
	dept_id INT NOT NULL,
	code VARCHAR(20) NOT NULL,
	title VARCHAR(200) NOT NULL,
	budget DECIMAL(14,2) NOT NULL DEFAULT 0.00,
	spent DECIMAL(14,2) NOT NULL DEFAULT 0.00,
	remaining DECIMAL(14,2) AS (budget - spent) STORED,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY (id),
	UNIQUE KEY uk_proj_code (code),
	KEY idx_proj_dept (dept_id),
	CONSTRAINT fk_proj_dept FOREIGN KEY (dept_id) REFERENCES department (id) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE assignment (
	employee_id INT NOT NULL,
	project_id INT NOT NULL,
	role VARCHAR(50) NOT NULL DEFAULT 'member',
	allocation DECIMAL(5,2) NOT NULL DEFAULT 100.00,
	assigned_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY (employee_id, project_id),
	KEY idx_assign_project (project_id),
	CONSTRAINT fk_assign_emp FOREIGN KEY (employee_id) REFERENCES employee (id) ON DELETE CASCADE,
	CONSTRAINT fk_assign_proj FOREIGN KEY (project_id) REFERENCES project (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE event_log (
	id BIGINT NOT NULL AUTO_INCREMENT,
	occurred_on DATE NOT NULL,
	kind VARCHAR(40) NOT NULL,
	payload JSON,
	PRIMARY KEY (id, occurred_on)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
PARTITION BY RANGE (YEAR(occurred_on)) (
	PARTITION p2023 VALUES LESS THAN (2024),
	PARTITION p2024 VALUES LESS THAN (2025),
	PARTITION p_future VALUES LESS THAN MAXVALUE
);

CREATE TABLE metric (
	id INT NOT NULL AUTO_INCREMENT,
	bucket INT NOT NULL,
	name VARCHAR(60) NOT NULL,
	value DOUBLE NOT NULL DEFAULT 0,
	PRIMARY KEY (id, bucket)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
PARTITION BY HASH (bucket) PARTITIONS 4;

CREATE TABLE audit_trail (
	id BIGINT NOT NULL AUTO_INCREMENT,
	emp_id INT NOT NULL,
	old_salary DECIMAL(10,2),
	new_salary DECIMAL(10,2),
	changed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE category (
	id INT NOT NULL AUTO_INCREMENT,
	parent_id INT NULL,
	name VARCHAR(80) NOT NULL,
	PRIMARY KEY (id),
	KEY idx_cat_parent (parent_id),
	CONSTRAINT fk_cat_parent FOREIGN KEY (parent_id) REFERENCES category (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE doc (
	id INT NOT NULL AUTO_INCREMENT,
	category_id INT NOT NULL,
	owner_id INT NOT NULL,
	body MEDIUMTEXT,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY (id),
	KEY idx_doc_cat (category_id),
	KEY idx_doc_owner (owner_id),
	CONSTRAINT fk_doc_cat FOREIGN KEY (category_id) REFERENCES category (id),
	CONSTRAINT fk_doc_owner FOREIGN KEY (owner_id) REFERENCES employee (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE tag_map (
	doc_id INT NOT NULL,
	tag VARCHAR(40) NOT NULL,
	weight INT NOT NULL DEFAULT 1,
	PRIMARY KEY (doc_id, tag),
	CONSTRAINT fk_tagmap_doc FOREIGN KEY (doc_id) REFERENCES doc (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE VIEW v_active_employees AS
SELECT e.id, e.full_name, e.email, e.salary, d.name AS dept_name
FROM employee e JOIN department d ON e.dept_id = d.id
WHERE e.active = TRUE;

CREATE VIEW v_dept_payroll AS
SELECT dept_name, COUNT(*) AS headcount, SUM(salary) AS payroll
FROM v_active_employees
GROUP BY dept_name;

CREATE FUNCTION emp_annual_salary(monthly DECIMAL(10,2)) RETURNS DECIMAL(12,2) DETERMINISTIC
RETURN monthly * 12;

CREATE FUNCTION dept_headcount(d_id INT) RETURNS INT READS SQL DATA
RETURN (SELECT COUNT(*) FROM employee WHERE dept_id = d_id);

CREATE PROCEDURE give_raise(IN emp INT, IN pct DECIMAL(5,2))
BEGIN
	UPDATE employee SET salary = salary * (1 + pct / 100) WHERE id = emp;
END;

CREATE PROCEDURE close_project(IN proj INT)
BEGIN
	DELETE FROM assignment WHERE project_id = proj;
	UPDATE project SET spent = budget WHERE id = proj;
END;

CREATE TRIGGER trg_emp_before_ins BEFORE INSERT ON employee FOR EACH ROW
SET NEW.email = LOWER(NEW.email);

CREATE TRIGGER trg_emp_after_upd AFTER UPDATE ON employee FOR EACH ROW
BEGIN
	IF OLD.salary <> NEW.salary THEN
		INSERT INTO audit_trail (emp_id, old_salary, new_salary) VALUES (NEW.id, OLD.salary, NEW.salary);
	END IF;
END;

CREATE TRIGGER trg_proj_before_ins BEFORE INSERT ON project FOR EACH ROW
SET NEW.created_at = NOW();

CREATE EVENT IF NOT EXISTS ev_daily_metric ON SCHEDULE EVERY 1 DAY
DO INSERT INTO metric (bucket, name, value) VALUES (0, 'daily', 1);
