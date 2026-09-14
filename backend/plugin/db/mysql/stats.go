package mysql

import (
	"context"
	"database/sql"
	"log/slog"
	"slices"
	"strconv"
	"strings"

	"github.com/bytebase/omni/mysql/ast"
	"github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

// erUnknownSystemVariable is the MySQL error number for setting a variable the server lacks.
const erUnknownSystemVariable = 1193

// CountAffectedRows returns the planner's estimate of the rows the INSERT, UPDATE, or DELETE
// statement modifies.
func (d *Driver) CountAffectedRows(ctx context.Context, statement string) (int64, error) {
	if d.dbType == storepb.Engine_OCEANBASE {
		rows, err := d.db.QueryContext(ctx, "EXPLAIN FORMAT=JSON "+statement)
		if err != nil {
			return 0, err
		}
		defer rows.Close()
		plan, err := readExplainJSON(rows)
		if err != nil {
			return 0, err
		}
		return mysqlparser.EstimateAffectedRowsFromOceanBaseExplainJSON(plan)
	}

	list, err := mysqlparser.ParseMySQL(statement)
	if err != nil {
		return 0, errors.Wrap(err, "failed to parse statement")
	}
	if list == nil || len(list.Items) != 1 {
		return 0, errors.New("expected exactly one statement")
	}
	query := mysqlparser.AffectedRowsQuery(list.Items[0], statement)
	plan, err := ExplainJSON(ctx, d.db, query)
	if err != nil {
		return 0, err
	}
	return EstimateAffectedRows(ctx, d.db, list.Items[0], query, plan)
}

// EstimateAffectedRows returns the rows stmt modifies according to plan, the ExplainJSON output for
// query, which is the statement's AffectedRowsQuery. When plan has no estimate to read, as MySQL 8.0
// prints no source plan for an INSERT ... SELECT that computes window functions, the estimate of the
// tabular EXPLAIN stands in for it.
func EstimateAffectedRows(ctx context.Context, db *sql.DB, stmt ast.Node, query string, plan string) (int64, error) {
	count, err := mysqlparser.EstimateAffectedRowsFromExplainJSON(stmt, plan)
	if err == nil {
		return count, nil
	}
	rows, tables, ok := explainTabularEstimate(ctx, db, query)
	if !ok {
		return 0, err
	}
	// A multi-table UPDATE or DELETE can change a row of each target for every row its join produces.
	return mysqlparser.CapAffectedRowsByLimit(stmt, rows*float64(mysqlparser.DMLTargetCount(stmt, tables))), nil
}

// explainTabularEstimate returns the rows that the first query block of the tabular EXPLAIN of query
// produces, and how many tables it joins: the product, over those tables, of the rows each reads
// scaled by its filtered percentage where the server prints one.
func explainTabularEstimate(ctx context.Context, db *sql.DB, query string) (float64, int, bool) {
	// MySQL 9 prints a plain EXPLAIN in the TREE format.
	rows, err := db.QueryContext(ctx, "EXPLAIN FORMAT=TRADITIONAL "+query)
	if err != nil {
		return 0, 0, false
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		return 0, 0, false
	}
	column := func(name string) int {
		return slices.IndexFunc(columns, func(column string) bool { return strings.EqualFold(column, name) })
	}
	idIndex, rowsIndex, filteredIndex := column("id"), column("rows"), column("filtered")
	if idIndex < 0 || rowsIndex < 0 {
		return 0, 0, false
	}
	values := make([]sql.NullString, len(columns))
	dest := make([]any, len(columns))
	for i := range values {
		dest[i] = &values[i]
	}
	var blockID string
	estimate, tables := 1.0, 0
	for rows.Next() {
		if err := rows.Scan(dest...); err != nil {
			return 0, 0, false
		}
		tableRows, err := strconv.ParseFloat(values[rowsIndex].String, 64)
		if !values[rowsIndex].Valid || err != nil || (tables > 0 && values[idIndex].String != blockID) {
			continue
		}
		blockID = values[idIndex].String
		tables++
		estimate *= tableRows
		if filteredIndex >= 0 && values[filteredIndex].Valid {
			if filtered, err := strconv.ParseFloat(values[filteredIndex].String, 64); err == nil {
				estimate *= filtered / 100
			}
		}
	}
	return estimate, tables, tables > 0 && rows.Err() == nil
}

// ExplainJSON returns the `EXPLAIN FORMAT=JSON` output for the statement in the JSON format
// version that mysqlparser.EstimateAffectedRowsFromExplainJSON reads.
func ExplainJSON(ctx context.Context, db *sql.DB, statement string) (string, error) {
	// The session variable must be set on the connection that runs the EXPLAIN.
	conn, err := db.Conn(ctx)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	// MySQL 9.5 defaults explain_json_format_version to 2. MySQL before 8.3 and MariaDB lack the
	// variable and only produce version 1.
	if _, err := conn.ExecContext(ctx, "SET SESSION explain_json_format_version = 1"); err != nil {
		var mysqlErr *mysql.MySQLError
		if !errors.As(err, &mysqlErr) || mysqlErr.Number != erUnknownSystemVariable {
			return "", errors.Wrap(err, "failed to set explain_json_format_version")
		}
	} else {
		// The connection returns to the pool, so later statements on it get the server's version back.
		defer func() {
			if _, err := conn.ExecContext(ctx, "SET SESSION explain_json_format_version = DEFAULT"); err != nil {
				slog.Warn("failed to reset explain_json_format_version", log.BBError(err))
			}
		}()
	}

	rows, err := conn.QueryContext(ctx, "EXPLAIN FORMAT=JSON "+statement)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	return readExplainJSON(rows)
}

// readExplainJSON concatenates the plan rows; OceanBase splits its JSON plan across rows.
func readExplainJSON(rows *sql.Rows) (string, error) {
	var plan strings.Builder
	for rows.Next() {
		var line sql.NullString
		if err := rows.Scan(&line); err != nil {
			return "", err
		}
		plan.WriteString(line.String)
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	return plan.String(), nil
}
