package advisor

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// NormalizeStatement limit the max length of the statements.
func NormalizeStatement(statement string) string {
	maxLength := 1000
	if len(statement) > maxLength {
		return statement[:maxLength] + "..."
	}
	return statement
}

type QueryContext struct {
	UsePostgresDatabaseOwner bool
	PreExecutions            []string
}

// Query runs the EXPLAIN or SELECT statements for advisors.
func Query(ctx context.Context, qCtx QueryContext, connection *sql.DB, engine storepb.Engine, statement string) ([]any, error) {
	tx, err := connection.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if engine == storepb.Engine_POSTGRES && qCtx.UsePostgresDatabaseOwner {
		const query = `
		SELECT
			u.rolname
		FROM
			pg_roles AS u JOIN pg_database AS d ON (d.datdba = u.oid)
		WHERE
			d.datname = current_database();
		`
		var owner string
		if err := tx.QueryRowContext(ctx, query).Scan(&owner); err != nil {
			return nil, err
		}
		if _, err := tx.ExecContext(ctx, fmt.Sprintf("SET ROLE '%s';", owner)); err != nil {
			return nil, err
		}
	}

	for _, preExec := range qCtx.PreExecutions {
		if preExec != "" {
			if _, err := tx.ExecContext(ctx, preExec); err != nil {
				return nil, errors.Wrapf(err, "failed to execute pre-execution: %s", preExec)
			}
		}
	}

	rows, err := tx.QueryContext(ctx, statement)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columnNames, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, err
	}

	colCount := len(columnTypes)

	var columnTypeNames []string
	for _, v := range columnTypes {
		// DatabaseTypeName returns the database system name of the column type.
		// refer: https://pkg.go.dev/database/sql#ColumnType.DatabaseTypeName
		columnTypeNames = append(columnTypeNames, strings.ToUpper(v.DatabaseTypeName()))
	}

	data := []any{}
	for rows.Next() {
		scanArgs := make([]any, colCount)
		for i, v := range columnTypeNames {
			// TODO(steven need help): Consult a common list of data types from database driver documentation. e.g. MySQL,PostgreSQL.
			switch v {
			case "VARCHAR", "TEXT", "UUID", "TIMESTAMP":
				scanArgs[i] = new(sql.NullString)
			case "BOOL":
				scanArgs[i] = new(sql.NullBool)
			case "INT", "INTEGER":
				scanArgs[i] = new(sql.NullInt64)
			case "FLOAT":
				scanArgs[i] = new(sql.NullFloat64)
			default:
				scanArgs[i] = new(sql.NullString)
			}
		}

		if err := rows.Scan(scanArgs...); err != nil {
			return nil, err
		}

		rowData := []any{}
		for i := range columnTypes {
			if v, ok := (scanArgs[i]).(*sql.NullBool); ok && v.Valid {
				rowData = append(rowData, v.Bool)
				continue
			}
			if v, ok := (scanArgs[i]).(*sql.NullString); ok && v.Valid {
				rowData = append(rowData, v.String)
				continue
			}
			if v, ok := (scanArgs[i]).(*sql.NullInt64); ok && v.Valid {
				rowData = append(rowData, v.Int64)
				continue
			}
			if v, ok := (scanArgs[i]).(*sql.NullInt32); ok && v.Valid {
				rowData = append(rowData, v.Int32)
				continue
			}
			if v, ok := (scanArgs[i]).(*sql.NullFloat64); ok && v.Valid {
				rowData = append(rowData, v.Float64)
				continue
			}
			// If none of them match, set nil to its value.
			rowData = append(rowData, nil)
		}

		data = append(data, rowData)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return []any{columnNames, columnTypeNames, data}, nil
}

func DatabaseExists(ctx Context, database string) bool {
	if ctx.ListDatabaseNamesFunc == nil {
		return false
	}

	names, err := ctx.ListDatabaseNamesFunc(ctx.Context, ctx.InstanceID)
	if err != nil {
		slog.Debug("failed to list databases", slog.String("instance", ctx.InstanceID), log.BBError(err))
		return false
	}

	for _, name := range names {
		if name == database {
			return true
		}
	}

	return false
}
