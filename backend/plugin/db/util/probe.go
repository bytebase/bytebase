//nolint:revive
package util

import (
	"context"
	"time"

	omnidriver "github.com/bytebase/omni/driver"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/db"
)

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
