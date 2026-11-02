package reviewrun

import (
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
