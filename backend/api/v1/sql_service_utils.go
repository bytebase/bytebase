package v1

import (
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// isSQLReviewSupported checks the engine type if SQL review supports it.
func isSQLReviewSupported(dbType storepb.Engine) bool {
	switch dbType {
	case storepb.Engine_POSTGRES, storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_MARIADB, storepb.Engine_ORACLE, storepb.Engine_OCEANBASE_ORACLE, storepb.Engine_OCEANBASE, storepb.Engine_SNOWFLAKE, storepb.Engine_DM, storepb.Engine_MSSQL:
		return true
	default:
		return false
	}
}

func convertChangeType(t v1pb.CheckRequest_ChangeType) storepb.PlanCheckRunConfig_ChangeDatabaseType {
	switch t {
	case v1pb.CheckRequest_DDL:
		return storepb.PlanCheckRunConfig_DDL
	case v1pb.CheckRequest_DDL_GHOST:
		return storepb.PlanCheckRunConfig_DDL_GHOST
	case v1pb.CheckRequest_DML:
		return storepb.PlanCheckRunConfig_DML
	case v1pb.CheckRequest_SQL_EDITOR:
		return storepb.PlanCheckRunConfig_SQL_EDITOR
	default:
		return storepb.PlanCheckRunConfig_CHANGE_DATABASE_TYPE_UNSPECIFIED
	}
}
