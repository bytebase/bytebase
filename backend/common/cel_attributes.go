//nolint:revive
package common

// CEL attribute names for resource scope.
const (
	// CELAttributeResourceEnvironmentID is the environment ID of the resource.
	CELAttributeResourceEnvironmentID = "resource.environment_id"
	// CELAttributeResourceProjectID is the project ID of the resource.
	CELAttributeResourceProjectID = "resource.project_id"
	// CELAttributeResourceInstanceID is the instance ID of the resource.
	CELAttributeResourceInstanceID = "resource.instance_id"
	// CELAttributeResourceDatabaseName is the database name of the resource.
	CELAttributeResourceDatabaseName = "resource.database_name"
	// CELAttributeResourceSchemaName is the schema name of the resource.
	CELAttributeResourceSchemaName = "resource.schema_name"
	// CELAttributeResourceTableName is the table name of the resource.
	CELAttributeResourceTableName = "resource.table_name"
	// CELAttributeResourceColumnName is the column name of the resource.
	CELAttributeResourceColumnName = "resource.column_name"
	// CELAttributeResourceDBEngine is the database engine of the resource.
	CELAttributeResourceDBEngine = "resource.db_engine"
	// CELAttributeResourceDatabase is the full database name of the resource (used in IAM policy conditions).
	CELAttributeResourceDatabase = "resource.database"
	// CELAttributeResourceClassificationLevel is the classification level of the resource.
	CELAttributeResourceClassificationLevel = "resource.classification_level"
	// CELAttributeResourceDatabaseLabels is the database labels of the resource.
	CELAttributeResourceDatabaseLabels = "resource.database_labels"
)

// CEL attribute names for statement scope.
const (
	// CELAttributeStatementAffectedRows is the number of affected rows by the statement.
	CELAttributeStatementAffectedRows = "statement.affected_rows"
	// CELAttributeStatementTableRows is the total number of rows in the table.
	CELAttributeStatementTableRows = "statement.table_rows"
	// CELAttributeStatementSQLType is the SQL statement type (e.g., SELECT, INSERT, UPDATE, DELETE).
	CELAttributeStatementSQLType = "statement.sql_type"
	// CELAttributeStatementText is the full text of the SQL statement.
	CELAttributeStatementText = "statement.text"
)

// CEL attribute names for request scope.
const (
	// CELAttributeRequestExpirationDays is the number of days until the request expires.
	CELAttributeRequestExpirationDays = "request.expiration_days"
	// CELAttributeRequestRole is the requested role.
	CELAttributeRequestRole = "request.role"
	// CELAttributeRequestTime is the timestamp of the request.
	CELAttributeRequestTime = "request.time"
)

// CEL attribute names for approval scope (deprecated, kept for backward compatibility).
const (
	// CELAttributeLevel is the risk level (deprecated).
	CELAttributeLevel = "level"
	// CELAttributeSource is the risk source (deprecated).
	CELAttributeSource = "source"
)
