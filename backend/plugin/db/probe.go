package db

import (
	"context"
	"strings"

	"github.com/pkg/errors"
)

// ProbeContext is how a rule's probe runs.
type ProbeContext struct {
	// PreExecutions run before the statement in the same session, in order.
	PreExecutions []string
	// Autocommit runs the statement outside a transaction, for an engine whose
	// probe cannot be wrapped in one. PreExecutions is invalid with it.
	Autocommit bool
}

// ProbeResult is one statement's answer.
type ProbeResult struct {
	Columns     []string
	ColumnTypes []string
	Rows        [][]any
}

// Plan is the result read as one JSON plan: every string cell of every row, in
// order, for an engine that splits its plan across rows.
func (r ProbeResult) Plan() (string, error) {
	var plan strings.Builder
	cells := 0
	for _, row := range r.Rows {
		for _, cell := range row {
			if text, ok := cell.(string); ok {
				plan.WriteString(text)
				cells++
			}
		}
	}
	if cells == 0 {
		return "", errors.New("the probe result has no cell to read as a plan")
	}
	return plan.String(), nil
}

// Prober runs a rule's probe statements against the target database.
type Prober interface {
	// Probe runs pCtx.PreExecutions, then statement, in one session whose
	// transaction is rolled back, and returns the statement's columns and rows.
	// With pCtx.Autocommit it runs the statement alone, outside a transaction.
	Probe(ctx context.Context, pCtx ProbeContext, statement string) (ProbeResult, error)
}

// RowEstimator is the capability a rule needs when the estimate is the
// driver's own rewrite of the statement rather than a plan the rule can read.
type RowEstimator interface {
	// CountAffectedRows returns the planner's estimate of the rows statement
	// modifies: exactly one INSERT, UPDATE or DELETE.
	CountAffectedRows(ctx context.Context, statement string) (int64, error)
}

// ColumnExtraLookup is the capability a rule needs when the answer must bind
// values, which the session contract cannot carry.
type ColumnExtraLookup interface {
	// ColumnExtra returns one column's catalogue extra, or the server's error.
	ColumnExtra(ctx context.Context, database, table, column string) (string, error)
}

// ProberOf returns the driver's probe, or nil when the driver has none.
func ProberOf(driver Driver) Prober {
	prober, ok := driver.(Prober)
	if !ok {
		return nil
	}
	return prober
}
