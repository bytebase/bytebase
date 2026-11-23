package advisor

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

// UnsupportedError reports that a rule needed a live capability the connection
// does not offer. A rule returns it instead of no advice: a review that skips a
// rule must not look like one that passed it.
type UnsupportedError struct {
	Engine storepb.Engine
	// Need is what the rule needed, as a verb phrase: "probe the database",
	// "estimate affected rows".
	Need string
}

func (e *UnsupportedError) Error() string {
	return fmt.Sprintf("%v: the live connection cannot %s, so this rule did not run", e.Engine, e.Need)
}

// ProbeFor returns the prober a live rule runs through, or an UnsupportedError.
func ProbeFor(checkCtx Context) (db.Prober, error) {
	if checkCtx.Prober == nil {
		return nil, &UnsupportedError{Engine: checkCtx.DBType, Need: "probe the database"}
	}
	return checkCtx.Prober, nil
}

// ExplainProbe runs query through prober, returning an empty Probe on success
// or Probe{Error: err.Error()} on failure.
func ExplainProbe(ctx context.Context, prober db.Prober, probeCtx db.ProbeContext, query string) v1review.Probe {
	if _, err := prober.Probe(ctx, probeCtx, query); err != nil {
		return v1review.Probe{Error: err.Error()}
	}
	return v1review.Probe{}
}

// Estimate is a row-limit rule's one call: the engine's estimate of the rows
// statement modifies, shaped as the rule reports it ({Rows} or {CountError}).
func Estimate(ctx context.Context, engine storepb.Engine, prober db.Prober, statement string) (v1review.Probe, error) {
	if prober == nil {
		return v1review.Probe{}, &UnsupportedError{Engine: engine, Need: "probe the database"}
	}
	estimator, ok := prober.(db.RowEstimator)
	if !ok {
		return v1review.Probe{}, &UnsupportedError{Engine: engine, Need: "estimate affected rows"}
	}
	rows, err := estimator.CountAffectedRows(ctx, statement)
	if err != nil {
		//nolint:nilerr // the estimate failure is the probe finding, reported as advice.
		return v1review.Probe{CountError: err.Error()}, nil
	}
	return v1review.Probe{Rows: rows}, nil
}

// ExplainPlanRows runs query through checkCtx's prober, extracts the JSON plan,
// and parses the estimated row count with parsePlan.
func ExplainPlanRows(
	ctx context.Context,
	checkCtx Context,
	preExecutions []string,
	query string,
	parsePlan func(string) (int64, error),
) (v1review.Probe, error) {
	prober, err := ProbeFor(checkCtx)
	if err != nil {
		return v1review.Probe{}, err
	}
	res, err := prober.Probe(ctx, db.ProbeContext{PreExecutions: preExecutions}, query)
	if err != nil {
		//nolint:nilerr // a refused statement is the probe finding, reported as advice.
		return v1review.Probe{Error: err.Error()}, nil
	}
	plan, err := res.Plan()
	if err != nil {
		//nolint:nilerr // an unreadable plan is the probe finding, reported as advice.
		return v1review.Probe{CountError: err.Error()}, nil
	}
	rows, err := parsePlan(plan)
	if err != nil {
		//nolint:nilerr // an unreadable plan is the probe finding, reported as advice.
		return v1review.Probe{CountError: err.Error()}, nil
	}
	return v1review.Probe{Rows: rows}, nil
}

// ColumnExtraOf returns the connection's catalogue lookup, or an
// UnsupportedError.
func ColumnExtraOf(checkCtx Context) (db.ColumnExtraLookup, error) {
	if lookup, ok := checkCtx.Prober.(db.ColumnExtraLookup); ok {
		return lookup, nil
	}
	return nil, &UnsupportedError{Engine: checkCtx.DBType, Need: "read a column's catalogue extra"}
}
