package mariadb

import (
	"context"
	"database/sql"

	"github.com/bytebase/omni/mariadb/ast"

	mysqldriver "github.com/bytebase/bytebase/backend/plugin/db/mysql"
	mariadbparser "github.com/bytebase/bytebase/backend/plugin/parser/mariadb"
)

// estimateAffectedRows returns the rows stmt modifies according to plan, the ExplainJSON output for
// query. It mirrors the driver's EstimateAffectedRows, which reads an omni MySQL node: MariaDB
// speaks the MySQL wire protocol and keeps that driver, so only the plan reader is its own.
func estimateAffectedRows(ctx context.Context, db *sql.DB, stmt ast.Node, query string, plan string) (int64, error) {
	count, err := mariadbparser.EstimateAffectedRowsFromExplainJSON(stmt, plan)
	if err == nil {
		return count, nil
	}
	rows, tables, ok := mysqldriver.ExplainTabularEstimate(ctx, db, query)
	if !ok {
		return 0, err
	}
	// A multi-table UPDATE or DELETE can change a row of each target for every row its join produces.
	return mariadbparser.CapAffectedRowsByLimit(stmt, rows*float64(mariadbparser.DMLTargetCount(stmt, tables))), nil
}
