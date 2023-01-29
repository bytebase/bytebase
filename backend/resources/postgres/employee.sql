CREATE TABLE employee (
    emp_no      SERIAL,
    name TEXT NOT NULL,
    PRIMARY KEY (emp_no)
);

INSERT INTO employee (name) VALUES ('Alice');
INSERT INTO employee (name) VALUES ('Bob');