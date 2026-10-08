--  Sample employee database 
--  See changelog table for details
--  Copyright (C) 2007,2008, MySQL AB
--  
--  Original data created by Fusheng Wang and Carlo Zaniolo
--  http://www.cs.aau.dk/TimeCenter/software.htm
--  http://www.cs.aau.dk/TimeCenter/Data/employeeTemporalDataSet.zip
-- 
--  Current schema by Giuseppe Maxia 
--  Data conversion from XML to relational by Patrick Crews
-- 
-- This work is licensed under the 
-- Creative Commons Attribution-Share Alike 3.0 Unported License. 
-- To view a copy of this license, visit 
-- http://creativecommons.org/licenses/by-sa/3.0/ or send a letter to 
-- Creative Commons, 171 Second Street, Suite 300, San Francisco, 
-- California, 94105, USA.
-- 
--  DISCLAIMER
--  To the best of our knowledge, this data is fabricated, and
--  it does not correspond to real people. 
--  Any similarity to existing people is purely coincidental.
-- 





/*!50503 set default_storage_engine = InnoDB */;
/*!50503 select CONCAT('storage engine: ', @@default_storage_engine) as INFO */;

CREATE TABLE employees (
    emp_no      INT             NOT NULL,
    birth_date  DATE            NOT NULL,
    first_name  VARCHAR(14)     NOT NULL,
    last_name   VARCHAR(16)     NOT NULL,
    gender      ENUM ('M','F')  NOT NULL,    
    hire_date   DATE            NOT NULL,
    PRIMARY KEY (emp_no)
);

CREATE TABLE departments (
    dept_no     CHAR(4)         NOT NULL,
    dept_name   VARCHAR(40)     NOT NULL,
    PRIMARY KEY (dept_no),
    UNIQUE  KEY (dept_name)
);

CREATE TABLE dept_manager (
   emp_no       INT             NOT NULL,
   dept_no      CHAR(4)         NOT NULL,
   from_date    DATE            NOT NULL,
   to_date      DATE            NOT NULL,
   FOREIGN KEY (emp_no)  REFERENCES employees (emp_no)    ON DELETE CASCADE,
   FOREIGN KEY (dept_no) REFERENCES departments (dept_no) ON DELETE CASCADE,
   PRIMARY KEY (emp_no,dept_no)
); 

CREATE TABLE dept_emp (
    emp_no      INT             NOT NULL,
    dept_no     CHAR(4)         NOT NULL,
    from_date   DATE            NOT NULL,
    to_date     DATE            NOT NULL,
    FOREIGN KEY (emp_no)  REFERENCES employees   (emp_no)  ON DELETE CASCADE,
    FOREIGN KEY (dept_no) REFERENCES departments (dept_no) ON DELETE CASCADE,
    PRIMARY KEY (emp_no,dept_no)
);

CREATE TABLE titles (
    emp_no      INT             NOT NULL,
    title       VARCHAR(50)     NOT NULL,
    from_date   DATE            NOT NULL,
    to_date     DATE,
    # FOREIGN KEY (emp_no) REFERENCES employees (emp_no) ON DELETE CASCADE,
    PRIMARY KEY (emp_no,title, from_date)
); 

/*!50510
ALTER TABLE titles 
partition by range COLUMNS (from_date)
(
    partition p01 values less than ('1985-12-31'),
    partition p02 values less than ('1986-12-31'),
    partition p03 values less than ('1987-12-31'),
    partition p04 values less than ('1988-12-31'),
    partition p05 values less than ('1989-12-31'),
    partition p06 values less than ('1990-12-31'),
    partition p07 values less than ('1991-12-31'),
    partition p08 values less than ('1992-12-31'),
    partition p09 values less than ('1993-12-31'),
    partition p10 values less than ('1994-12-31'),
    partition p11 values less than ('1995-12-31'),
    partition p12 values less than ('1996-12-31'),
    partition p13 values less than ('1997-12-31'),
    partition p14 values less than ('1998-12-31'),
    partition p15 values less than ('1999-12-31'),
    partition p16 values less than ('2000-12-31'),
    partition p17 values less than ('2001-12-31'),
    partition p18 values less than ('2002-12-31'),
    partition p19 values less than (MAXVALUE)
)
*/;

CREATE TABLE salaries (
    emp_no      INT             NOT NULL,
    salary      INT             NOT NULL,
    from_date   DATE            NOT NULL,
    to_date     DATE            NOT NULL,
    # FOREIGN KEY (emp_no) REFERENCES employees (emp_no) ON DELETE CASCADE,
    PRIMARY KEY (emp_no, from_date)
); 

/*!50510
ALTER TABLE salaries
partition by range COLUMNS (from_date)
(
    partition p01 values less than ('1985-12-31'),
    partition p02 values less than ('1986-12-31'),
    partition p03 values less than ('1987-12-31'),
    partition p04 values less than ('1988-12-31'),
    partition p05 values less than ('1989-12-31'),
    partition p06 values less than ('1990-12-31'),
    partition p07 values less than ('1991-12-31'),
    partition p08 values less than ('1992-12-31'),
    partition p09 values less than ('1993-12-31'),
    partition p10 values less than ('1994-12-31'),
    partition p11 values less than ('1995-12-31'),
    partition p12 values less than ('1996-12-31'),
    partition p13 values less than ('1997-12-31'),
    partition p14 values less than ('1998-12-31'),
    partition p15 values less than ('1999-12-31'),
    partition p16 values less than ('2000-12-31'),
    partition p17 values less than ('2001-12-31'),
    partition p18 values less than ('2002-12-31'),
    partition p19 values less than (MAXVALUE)
)
*/;

CREATE OR REPLACE VIEW dept_emp_latest_date AS
    SELECT emp_no, MAX(from_date) AS from_date, MAX(to_date) AS to_date
    FROM dept_emp
    GROUP BY emp_no;

# shows only the current department for each employee
CREATE OR REPLACE VIEW current_dept_emp AS
    SELECT l.emp_no, dept_no, l.from_date, l.to_date
    FROM dept_emp d
        INNER JOIN dept_emp_latest_date l
        ON d.emp_no=l.emp_no AND d.from_date=l.from_date AND l.to_date = d.to_date;

