package reviewrun

import (
	"context"

	mariadbreview "github.com/bytebase/omni/mariadb/review"
	mssqlreview "github.com/bytebase/omni/mssql/review"
	mysqlreview "github.com/bytebase/omni/mysql/review"
	oceanbasereview "github.com/bytebase/omni/oceanbase/review"
	oraclereview "github.com/bytebase/omni/oracle/review"
	pgreview "github.com/bytebase/omni/pg/review"
	"github.com/bytebase/omni/review"
	snowflakereview "github.com/bytebase/omni/snowflake/review"
	tidbreview "github.com/bytebase/omni/tidb/review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// reviewFuncs maps an engine to its omni Review entry point. An engine absent
// here has no standard rule review: a run skips its targets, and the plan
// checks keep V1's statement advice for it. The map fills in as omni ships
// engine review packages: each one is a line here and nothing else.
var reviewFuncs = map[storepb.Engine]review.ReviewFunc{
	storepb.Engine_POSTGRES:  pgreview.Review,
	storepb.Engine_MYSQL:     mysqlreview.Review,
	storepb.Engine_OCEANBASE: oceanbasereview.ReviewOceanBase,
	storepb.Engine_MARIADB:   mariadbreview.Review,
	storepb.Engine_TIDB:      tidbreview.Review,
	storepb.Engine_MSSQL:     mssqlreview.Review,
	storepb.Engine_ORACLE:    oraclereview.Review,
	storepb.Engine_SNOWFLAKE: snowflakereview.Review,
}

// ReviewsEngine reports whether the standard rule review covers the engine.
func ReviewsEngine(engine storepb.Engine) bool {
	_, ok := reviewFuncs[engine]
	return ok
}

// statementBodies returns the body of each statement of sql as the standard
// rule review reads it, or false for an engine it does not cover. omni gives
// the bodies only with a review, and a review with no rules splits and parses
// and nothing more.
func statementBodies(ctx context.Context, engine storepb.Engine, sql string) ([]review.Range, bool) {
	reviewFunc, ok := reviewFuncs[engine]
	if !ok {
		return nil, false
	}
	result, err := reviewFunc(ctx, sql, review.Options{Rules: []review.Rule{}}, nil)
	if err != nil {
		return nil, false
	}
	return result.Bodies, true
}
