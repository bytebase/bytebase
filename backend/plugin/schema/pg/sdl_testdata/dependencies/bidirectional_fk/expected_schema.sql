CREATE TABLE public.departments (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    manager_id INTEGER,
    CONSTRAINT fk_department_manager FOREIGN KEY (manager_id) REFERENCES employees(id)
);

CREATE TABLE public.employees (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    department_id INTEGER NOT NULL,
    CONSTRAINT fk_employee_department FOREIGN KEY (department_id) REFERENCES departments(id)
);
