package db

import (
	"fmt"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// ExplainStatement returns the statement that produces a query plan for
// statement on engine, and whether engine explains by prefixing the statement
// with EXPLAIN. It is the single source of that wrapping: a driver calls it to
// build the statement it runs, and the Query handler calls it to validate the
// exact statement that will run (an EXPLAIN ANALYZE of a write is not read-only
// and is refused before execution).
//
// The second return is false for engines whose EXPLAIN is not a bare prefix and
// so does not execute the wrapped statement — Oracle (EXPLAIN PLAN FOR), SQL
// Server (SET SHOWPLAN), Spanner/BigQuery (plan APIs) — and for engines with no
// EXPLAIN at all. Those drivers produce a plan on their own and pass "" here.
func ExplainStatement(engine storepb.Engine, statement string, format v1pb.QueryOption_ExplainFormat) (string, bool) {
	switch engine {
	case storepb.Engine_POSTGRES:
		switch format {
		case v1pb.QueryOption_JSON:
			return fmt.Sprintf("EXPLAIN (FORMAT JSON) %s", statement), true
		case v1pb.QueryOption_XML:
			return fmt.Sprintf("EXPLAIN (FORMAT XML) %s", statement), true
		default:
			return fmt.Sprintf("EXPLAIN %s", statement), true
		}
	case storepb.Engine_MYSQL,
		storepb.Engine_MARIADB,
		storepb.Engine_OCEANBASE,
		storepb.Engine_TIDB,
		storepb.Engine_REDSHIFT,
		storepb.Engine_COCKROACHDB,
		storepb.Engine_SNOWFLAKE,
		storepb.Engine_CLICKHOUSE,
		storepb.Engine_STARROCKS,
		storepb.Engine_DORIS,
		storepb.Engine_HIVE,
		storepb.Engine_TRINO:
		return fmt.Sprintf("EXPLAIN %s", statement), true
	default:
		return "", false
	}
}
