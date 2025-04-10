package base

// AnomalyType is the type of a task.
type AnomalyType string

const (
	// AnomalyDatabaseConnection is the anomaly type for database connections.
	AnomalyDatabaseConnection AnomalyType = "bb.anomaly.database.connection"
	// AnomalyDatabaseSchemaDrift is the anomaly type for database schema drifts.
	AnomalyDatabaseSchemaDrift AnomalyType = "bb.anomaly.database.schema.drift"
)
