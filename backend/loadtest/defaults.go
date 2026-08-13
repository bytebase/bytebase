package loadtest

// DefaultInteractiveQueries returns representative tutorial queries against the
// Bytebase sample HR schema. These encode the assumed "interactive use" pattern;
// adjust them if product traffic defines a different mix.
func DefaultInteractiveQueries() []string {
	return []string{
		"SELECT first_name, last_name, hire_date FROM employee ORDER BY hire_date DESC LIMIT 20",
		"SELECT d.dept_name, COUNT(de.emp_no) AS headcount FROM department d JOIN dept_emp de ON de.dept_no = d.dept_no GROUP BY d.dept_name ORDER BY headcount DESC",
		"SELECT title, COUNT(*) AS n FROM title GROUP BY title ORDER BY n DESC LIMIT 10",
		"SELECT e.first_name, e.last_name, s.amount, s.from_date FROM employee e JOIN salary s ON s.emp_no = e.emp_no ORDER BY s.amount DESC LIMIT 10",
		"SELECT * FROM employee WHERE emp_no = (SELECT MAX(emp_no) FROM employee)",
	}
}

// DefaultDDLStatements returns representative change-ticket DDL. Statements are
// idempotent and non-rewriting so they are safe to replay across runs.
func DefaultDDLStatements() []string {
	return []string{
		"CREATE INDEX IF NOT EXISTS lt_tmp_idx_title ON title (from_date)",
		"ALTER TABLE employee ADD COLUMN IF NOT EXISTS lt_tmp_meta text DEFAULT ''",
		"ANALYZE salary",
	}
}
