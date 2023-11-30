package base

import "context"

// GetAffectedRowsCountByQueryFunc is the interface of getting the affected rows by querying a explain statement.
type GetAffectedRowsCountByQueryFunc func(ctx context.Context, explainSQL string) (int64, error)

// GetTableDataSizeFunc is the interface of getting rowCount of tableMetaData.
type GetTableDataSizeFunc func(schemaName, tableName string) int64
