package advisor

// DBType is the type of a database.
type DBType string

const (
	// MySQL is the database type for MYSQL.
	MySQL DBType = "MYSQL"
	// Postgres is the database type for POSTGRES.
	Postgres DBType = "POSTGRES"
	// TiDB is the database type for TiDB.
	TiDB DBType = "TIDB"
)
