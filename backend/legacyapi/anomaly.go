package api

// AnomalyType is the type of a task.
type AnomalyType string

const (
	// AnomalyInstanceConnection is the anomaly type for instance connections.
	AnomalyInstanceConnection AnomalyType = "bb.anomaly.instance.connection"
	// AnomalyInstanceMigrationSchema is the anomaly type for schema migrations.
	AnomalyInstanceMigrationSchema AnomalyType = "bb.anomaly.instance.migration-schema"
	// AnomalyDatabaseConnection is the anomaly type for database connections.
	AnomalyDatabaseConnection AnomalyType = "bb.anomaly.database.connection"
	// AnomalyDatabaseSchemaDrift is the anomaly type for database schema drifts.
	AnomalyDatabaseSchemaDrift AnomalyType = "bb.anomaly.database.schema.drift"
)
