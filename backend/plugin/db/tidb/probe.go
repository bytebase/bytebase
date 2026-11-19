package tidb

import (
	"context"
	"fmt"
	"time"

	omnidriver "github.com/bytebase/omni/driver"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/db"
)

var (
	_ db.Prober       = (*Driver)(nil)
	_ db.RowEstimator = (*Driver)(nil)
)

// Probe runs a live review rule's probe. A BATCH ... DRY RUN statement passes
// Autocommit, because TiDB refuses it inside a transaction.
//
// The probe runs on a session of its own so the pre-executions and the statement
// share one connection: without Autocommit it opens a transaction with the
// "START TRANSACTION" go-sql-driver's BeginTx sends, runs the pre-executions and
// the statement in it, and rolls it back.
//
// Known difference: a statement the contract classifies as a write answers with
// no rows, so an autocommit probe of BATCH ... DRY RUN reports the server's
// verdict but not the split statements the server listed. The dry-run rule reads
// only the verdict. Opening the session also reads CONNECTION_ID(), which the old
// pool handle did not, so the driver can cancel the statement when the caller's
// context ends.
func (d *Driver) Probe(ctx context.Context, pCtx db.ProbeContext, statement string) (db.ProbeResult, error) {
	if pCtx.Autocommit && len(pCtx.PreExecutions) > 0 {
		return db.ProbeResult{}, errors.New("a probe with pre-executions cannot run outside a transaction")
	}

	session, err := d.impl.Session(ctx)
	if err != nil {
		return db.ProbeResult{}, err
	}
	defer func() { _ = session.Close() }()

	if pCtx.Autocommit {
		// Autocommit was asked for precisely because the engine refuses the
		// statement inside a transaction, so no START TRANSACTION here.
		return runProbeStatement(ctx, session, statement)
	}

	if _, err := runProbeStatement(ctx, session, "START TRANSACTION"); err != nil {
		return db.ProbeResult{}, err
	}
	// Roll back whatever the probe did, including the pre-executions that ran
	// before a failing statement, on the session that holds the transaction.
	defer func() {
		_, _ = session.Query(context.WithoutCancel(ctx), "ROLLBACK", omnidriver.QueryOptions{})
	}()

	for _, preExec := range pCtx.PreExecutions {
		if preExec == "" {
			continue
		}
		if _, err := runProbeStatement(ctx, session, preExec); err != nil {
			return db.ProbeResult{}, errors.Wrapf(err, "failed to execute pre-execution: %s", preExec)
		}
	}
	return runProbeStatement(ctx, session, statement)
}

// runProbeStatement runs one statement on session and returns its cells. The
// contract reports a statement's failure on its result rather than as the call's
// error, so a failed statement is this call's error.
func runProbeStatement(ctx context.Context, session omnidriver.Session, statement string) (db.ProbeResult, error) {
	results, err := session.Query(ctx, statement, omnidriver.QueryOptions{})
	if err != nil {
		return db.ProbeResult{}, err
	}
	if len(results) == 0 {
		return db.ProbeResult{}, nil
	}
	result := results[0]
	if result.Err != nil {
		return db.ProbeResult{}, result.Err
	}
	probe := db.ProbeResult{
		Columns:     make([]string, len(result.Columns)),
		ColumnTypes: make([]string, len(result.Columns)),
	}
	for i, column := range result.Columns {
		probe.Columns[i] = column.Name
		probe.ColumnTypes[i] = column.TypeName
	}
	for _, row := range result.Rows {
		cells := make([]any, len(row))
		for i, value := range row {
			cells[i] = probeCell(value)
		}
		probe.Rows = append(probe.Rows, cells)
	}
	return probe, nil
}

// probeCell maps one contract value onto the Go value a probe result carries.
// The old probe scanned by column type name and returned text for every column
// it did not name, so an exact numeric or a date came back as the server's
// spelling; the contract has already typed the cell, so it arrives as its own Go
// value here and a timestamp keeps its wall clock and precision as text.
func probeCell(value omnidriver.Value) any {
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
		return timestampText(value)
	default:
		return nil
	}
}

// timestampText renders a timestamp the way the session spells it, so a probe's
// cell reads the same as it did when the driver scanned the server's text.
func timestampText(value omnidriver.Value) string {
	text := value.Time.Format(time.DateTime)
	if value.Precision > 0 {
		fraction := fmt.Sprintf("%09d", value.Time.Nanosecond())
		if int(value.Precision) < len(fraction) {
			fraction = fraction[:value.Precision]
		}
		text += "." + fraction
	}
	return text
}
