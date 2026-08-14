package loadtest

// Default workload concurrency levels for the per-workspace model. These are
// labeled assumptions, not measured traffic:
//
//   - Sync: N independent per-workspace syncs on independent 15-minute
//     schedules. Expected steady-state overlap is roughly
//     N x (per-database sync duration / 15 min), i.e. at most a handful even at
//     1000 databases; 10 is a conservative upper bound.
//   - DDL: change tickets are human-initiated and rare; 5 concurrent is a
//     generous upper bound.
//   - Interactive: 10 steady-state sessions and a 50-session burst, held fixed
//     while database count varies. Database count and connection count are
//     independent.
const (
	defaultSyncConcurrency        = 10
	defaultDDLConcurrency         = 5
	defaultInteractiveConcurrency = 10
	defaultInteractiveBurst       = 50
)

func (c *Config) syncConcurrency() int {
	if c.SyncConcurrency > 0 {
		return c.SyncConcurrency
	}
	return defaultSyncConcurrency
}

func (c *Config) ddlConcurrency() int {
	if c.DDLConcurrency > 0 {
		return c.DDLConcurrency
	}
	return defaultDDLConcurrency
}

func (c *Config) interactiveConcurrency() int {
	if c.InteractiveConcurrency > 0 {
		return c.InteractiveConcurrency
	}
	return defaultInteractiveConcurrency
}

func (c *Config) interactiveBurst() int {
	if c.InteractiveBurst > 0 {
		return c.InteractiveBurst
	}
	return defaultInteractiveBurst
}

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
