package mysql

import (
	"context"
	"database/sql"
	"log/slog"
	"strings"

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
	plan, err := ExplainJSON(ctx, d.db, mysqlparser.AffectedRowsQuery(list.Items[0], statement))
	if err != nil {
		return 0, err
	}
	return mysqlparser.EstimateAffectedRowsFromExplainJSON(list.Items[0], plan)
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
