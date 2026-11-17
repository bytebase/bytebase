//nolint:revive
package util

import (
	"context"
	"database/sql"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/db"
)

var (
	_ db.Prober            = (*SQLProber)(nil)
	_ db.ColumnExtraLookup = (*SQLProber)(nil)
)

// SQLProber is the probe over a *sql.DB: the handle a driver that has not cut
// over to omni still holds, and what a test hands a rule instead of a server.
type SQLProber struct {
	connection *sql.DB
}

// NewSQLProber returns a probe over connection, for tests and for a caller that
// holds only the SQL handle.
func NewSQLProber(connection *sql.DB) db.Prober {
	return &SQLProber{connection: connection}
}

// Probe runs the pre-executions and the statement in one transaction that is
// rolled back. With pCtx.Autocommit it runs the statement alone, because an
// engine whose probe cannot be wrapped in a transaction (Oracle's BEGIN opens a
// PL/SQL block, TiDB's BATCH ... DRY RUN refuses one) still has to answer.
func (p *SQLProber) Probe(ctx context.Context, pCtx db.ProbeContext, statement string) (db.ProbeResult, error) {
	return ProbeSQL(ctx, p.connection, pCtx, statement)
}

// ColumnExtra returns one column's catalogue extra.
func (p *SQLProber) ColumnExtra(ctx context.Context, database, table, column string) (string, error) {
	return QueryColumnExtra(ctx, p.connection, database, table, column)
}

// ProbeSQL is the shared probe over a *sql.DB: it runs the pre-executions and
// the statement in one transaction that is rolled back.
func ProbeSQL(ctx context.Context, connection *sql.DB, pCtx db.ProbeContext, statement string) (db.ProbeResult, error) {
	if pCtx.Autocommit {
		if len(pCtx.PreExecutions) > 0 {
			return db.ProbeResult{}, errors.New("a probe with pre-executions cannot run outside a transaction")
		}
		rows, err := connection.QueryContext(ctx, statement)
		if err != nil {
			return db.ProbeResult{}, err
		}
		defer rows.Close()
		return readProbeRows(rows)
	}

	tx, err := connection.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return db.ProbeResult{}, err
	}
	defer tx.Rollback()

	for _, preExec := range pCtx.PreExecutions {
		if preExec == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, preExec); err != nil {
			return db.ProbeResult{}, errors.Wrapf(err, "failed to execute pre-execution: %s", preExec)
		}
	}

	rows, err := tx.QueryContext(ctx, statement)
	if err != nil {
		return db.ProbeResult{}, err
	}
	defer rows.Close()
	return readProbeRows(rows)
}

// QueryColumnExtra returns one column's catalogue extra, the bound lookup
// ColumnExtraLookup carries.
func QueryColumnExtra(ctx context.Context, connection *sql.DB, database, table, column string) (string, error) {
	var extra string
	err := connection.QueryRowContext(ctx,
		"SELECT EXTRA FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ? AND COLUMN_NAME = ?",
		database,
		table,
		column,
	).Scan(&extra)
	return extra, err
}

// readProbeRows reads rows into the probe result, scanning by the column type
// the server reports.
func readProbeRows(rows *sql.Rows) (db.ProbeResult, error) {
	columnNames, err := rows.Columns()
	if err != nil {
		return db.ProbeResult{}, err
	}

	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return db.ProbeResult{}, err
	}

	colCount := len(columnTypes)

	var columnTypeNames []string
	for _, v := range columnTypes {
		// DatabaseTypeName returns the database system name of the column type.
		// refer: https://pkg.go.dev/database/sql#ColumnType.DatabaseTypeName
		columnTypeNames = append(columnTypeNames, strings.ToUpper(v.DatabaseTypeName()))
	}

	data := [][]any{}
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
			return db.ProbeResult{}, err
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
		return db.ProbeResult{}, err
	}

	return db.ProbeResult{Columns: columnNames, ColumnTypes: columnTypeNames, Rows: data}, nil
}
