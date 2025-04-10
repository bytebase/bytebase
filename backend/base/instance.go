package base

import (
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func EngineSupportSQLReview(engine storepb.Engine) bool {
	switch engine {
	case storepb.Engine_POSTGRES,
		storepb.Engine_MYSQL,
		storepb.Engine_TIDB,
		storepb.Engine_MARIADB,
		storepb.Engine_ORACLE,
		storepb.Engine_OCEANBASE_ORACLE,
		storepb.Engine_OCEANBASE,
		storepb.Engine_SNOWFLAKE,
		storepb.Engine_DM,
		storepb.Engine_MSSQL:
		return true
	default:
		return false
	}
}

func EngineSupportQueryNewACL(engine storepb.Engine) bool {
	switch engine {
	case storepb.Engine_MYSQL,
		storepb.Engine_POSTGRES,
		storepb.Engine_ORACLE,
		storepb.Engine_MSSQL,
		storepb.Engine_TIDB,
		storepb.Engine_SNOWFLAKE,
		storepb.Engine_SPANNER,
		storepb.Engine_BIGQUERY:
		return true
	default:
		return false
	}
}

func EngineSupportMasking(e storepb.Engine) bool {
	switch e {
	case storepb.Engine_MYSQL,
		storepb.Engine_POSTGRES,
		storepb.Engine_ORACLE,
		storepb.Engine_MSSQL,
		storepb.Engine_MARIADB,
		storepb.Engine_OCEANBASE,
		storepb.Engine_TIDB,
		storepb.Engine_BIGQUERY,
		storepb.Engine_SPANNER:
		return true
	default:
		return false
	}
}
