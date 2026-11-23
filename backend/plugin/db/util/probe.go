//nolint:revive
package util

import (
	"context"
	"database/sql"
	"strings"
	"time"

	omnidriver "github.com/bytebase/omni/driver"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/db"
)

var (
	_ db.Prober            = (*sqlProber)(nil)
	_ db.ColumnExtraLookup = (*sqlProber)(nil)
)

// sqlProber is the probe over a *sql.DB, used by tests that hand a rule a
// *sql.DB directly instead of an omni driver.
type sqlProber struct {
	connection *sql.DB
}

// NewSQLProber returns a probe over connection, for tests and for a caller that
// holds only the SQL handle.
func NewSQLProber(connection *sql.DB) db.Prober {
	return &sqlProber{connection: connection}
}

// Probe runs the pre-executions and the statement in one transaction that is
// rolled back. With pCtx.Autocommit it runs the statement alone, because an
// engine whose probe cannot be wrapped in a transaction (Oracle's BEGIN opens a
// PL/SQL block, TiDB's BATCH ... DRY RUN refuses one) still has to answer.
func (p *sqlProber) Probe(ctx context.Context, pCtx db.ProbeContext, statement string) (db.ProbeResult, error) {
	return probeSQL(ctx, p.connection, pCtx, statement)
}

// ColumnExtra returns one column's catalogue extra.
func (p *sqlProber) ColumnExtra(ctx context.Context, database, table, column string) (string, error) {
	var extra string
	err := p.connection.QueryRowContext(ctx,
		"SELECT EXTRA FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ? AND COLUMN_NAME = ?",
		database,
		table,
		column,
	).Scan(&extra)
	return extra, err
}

// probeSQL runs the pre-executions and the statement in one rolled-back
// transaction over a *sql.DB.
func probeSQL(ctx context.Context, connection *sql.DB, pCtx db.ProbeContext, statement string) (db.ProbeResult, error) {
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

// OmniProbeOptions configures how ProbeOmniSession executes a live-review probe
// on an omni/driver.Session.
type OmniProbeOptions struct {
	// BeginStatement starts the probe transaction (for example "BEGIN;" or
	// "START TRANSACTION").
	BeginStatement string
	// RollbackStatement ends the probe transaction (for example "ROLLBACK;" or
	// "ROLLBACK").
	RollbackStatement string
	// RollbackWithoutCancel detaches the rollback context from caller
	// cancellation so the pooled connection always exits the transaction.
	RollbackWithoutCancel bool
	// RollbackTimeout, when positive, bounds the rollback query with a timeout.
	RollbackTimeout time.Duration
	// AutocommitOnly rejects non-autocommit probes and probes with pre-executions.
	AutocommitOnly bool
	// AllowAutocommitPreExecutions skips rejecting PreExecutions when
	// Autocommit is set.
	AllowAutocommitPreExecutions bool
	// PrepareTx runs before BeginStatement on transactional probes and may
	// return extra pre-executions to prepend before pCtx.PreExecutions.
	PrepareTx func(ctx context.Context, session omnidriver.Session) ([]string, error)
	// SkipEmptyPreExecutions ignores empty strings in pCtx.PreExecutions.
	SkipEmptyPreExecutions bool
	// CheckPreExecutionResults checks each pre-execution's result slice for Err.
	CheckPreExecutionResults bool
	// RequireResult returns an error when the probed statement returns no results.
	RequireResult bool
	// TakeLastResult converts results[len(results)-1] instead of results[0].
	TakeLastResult bool
	// FormatCell optionally converts a driver.Value into the Go value stored in
	// ProbeResult.Rows. When nil, DefaultProbeCell is used.
	FormatCell func(omnidriver.Value) any
}

// ProbeOmniSession runs pCtx and statement on an already-open session according
// to opts.
func ProbeOmniSession(ctx context.Context, session omnidriver.Session, pCtx db.ProbeContext, statement string, opts OmniProbeOptions) (db.ProbeResult, error) {
	if err := validateOmniProbeContext(pCtx, opts); err != nil {
		return db.ProbeResult{}, err
	}
	if pCtx.Autocommit {
		if err := runPreExecutions(ctx, session, pCtx.PreExecutions, opts); err != nil {
			return db.ProbeResult{}, err
		}
		return runProbeQuery(ctx, session, statement, opts)
	}

	var preExecutions []string
	if opts.PrepareTx != nil {
		extra, err := opts.PrepareTx(ctx, session)
		if err != nil {
			return db.ProbeResult{}, err
		}
		preExecutions = append(preExecutions, extra...)
	}
	preExecutions = append(preExecutions, pCtx.PreExecutions...)

	if opts.BeginStatement != "" {
		if err := runProbeStatement(ctx, session, opts.BeginStatement, opts.CheckPreExecutionResults); err != nil {
			return db.ProbeResult{}, errors.Wrap(err, "failed to start the probe transaction")
		}
		if opts.RollbackStatement != "" {
			defer rollbackProbe(ctx, session, opts)
		}
	}

	if err := runPreExecutions(ctx, session, preExecutions, opts); err != nil {
		return db.ProbeResult{}, err
	}
	return runProbeQuery(ctx, session, statement, opts)
}

func validateOmniProbeContext(pCtx db.ProbeContext, opts OmniProbeOptions) error {
	if opts.AutocommitOnly && (!pCtx.Autocommit || len(pCtx.PreExecutions) > 0) {
		return errors.New("an autocommit-only probe runs alone without pre-executions")
	}
	if pCtx.Autocommit && !opts.AllowAutocommitPreExecutions && len(pCtx.PreExecutions) > 0 {
		return errors.New("a probe with pre-executions cannot run outside a transaction")
	}
	return nil
}

func rollbackProbe(ctx context.Context, session omnidriver.Session, opts OmniProbeOptions) {
	rollbackCtx := ctx
	if opts.RollbackWithoutCancel {
		rollbackCtx = context.WithoutCancel(ctx)
	}
	if opts.RollbackTimeout > 0 {
		var cancel context.CancelFunc
		rollbackCtx, cancel = context.WithTimeout(rollbackCtx, opts.RollbackTimeout)
		defer cancel()
	}
	_, _ = session.Query(rollbackCtx, opts.RollbackStatement, omnidriver.QueryOptions{})
}

func runPreExecutions(ctx context.Context, session omnidriver.Session, preExecutions []string, opts OmniProbeOptions) error {
	for _, preExec := range preExecutions {
		if opts.SkipEmptyPreExecutions && preExec == "" {
			continue
		}
		if err := runProbeStatement(ctx, session, preExec, opts.CheckPreExecutionResults); err != nil {
			return errors.Wrapf(err, "failed to execute pre-execution: %s", preExec)
		}
	}
	return nil
}

func runProbeStatement(ctx context.Context, session omnidriver.Session, statement string, checkResults bool) error {
	results, err := session.Query(ctx, statement, omnidriver.QueryOptions{})
	if err != nil {
		return err
	}
	if checkResults {
		for _, result := range results {
			if result.Err != nil {
				return result.Err
			}
		}
	}
	return nil
}

func runProbeQuery(ctx context.Context, session omnidriver.Session, statement string, opts OmniProbeOptions) (db.ProbeResult, error) {
	results, err := session.Query(ctx, statement, omnidriver.QueryOptions{})
	if err != nil {
		return db.ProbeResult{}, err
	}
	if len(results) == 0 {
		if opts.RequireResult {
			return db.ProbeResult{}, errors.New("the probe returned no result")
		}
		return db.ProbeResult{}, nil
	}
	result := results[0]
	if opts.TakeLastResult {
		result = results[len(results)-1]
	}
	if result.Err != nil {
		return db.ProbeResult{}, result.Err
	}
	return omniResultToProbeResult(result, opts.FormatCell), nil
}

// omniResultToProbeResult converts one omni driver Result into a db.ProbeResult.
func omniResultToProbeResult(result *omnidriver.Result, formatCell func(omnidriver.Value) any) db.ProbeResult {
	if result == nil {
		return db.ProbeResult{}
	}
	if formatCell == nil {
		formatCell = DefaultProbeCell
	}
	probeResult := db.ProbeResult{
		Columns:     make([]string, len(result.Columns)),
		ColumnTypes: make([]string, len(result.Columns)),
		Rows:        make([][]any, 0, len(result.Rows)),
	}
	for i, column := range result.Columns {
		probeResult.Columns[i] = column.Name
		probeResult.ColumnTypes[i] = column.TypeName
	}
	for _, row := range result.Rows {
		values := make([]any, len(row))
		for i, value := range row {
			values[i] = formatCell(value)
		}
		probeResult.Rows = append(probeResult.Rows, values)
	}
	return probeResult
}

// DefaultProbeCell converts one omni driver Value to the untyped Go value
// expected by advisor probes.
func DefaultProbeCell(value omnidriver.Value) any {
	switch value.Kind {
	case omnidriver.Bool:
		return value.Bool
	case omnidriver.Bytes:
		return value.Bytes
	case omnidriver.Float32, omnidriver.Float64:
		return value.Float64
	case omnidriver.Int32, omnidriver.Int64:
		return value.Int64
	case omnidriver.Uint32, omnidriver.Uint64:
		return value.Uint64
	case omnidriver.String:
		return value.String
	case omnidriver.Structured:
		return value.Structured
	case omnidriver.Timestamp, omnidriver.TimestampTZ:
		return value.Time
	default:
		return nil
	}
}
