package v1

import (
	"encoding/base64"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// IsSQLReviewSupported checks the engine type if SQL review supports it.
func IsSQLReviewSupported(dbType storepb.Engine) bool {
	switch dbType {
	case storepb.Engine_POSTGRES, storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_MARIADB, storepb.Engine_ORACLE, storepb.Engine_OCEANBASE_ORACLE, storepb.Engine_OCEANBASE, storepb.Engine_SNOWFLAKE, storepb.Engine_DM, storepb.Engine_MSSQL:
		return true
	default:
		return false
	}
}

// encodeToBase64String encodes the statement to base64 string.
func encodeToBase64String(statement string) string {
	base64Encoded := base64.StdEncoding.EncodeToString([]byte(statement))
	return base64Encoded
}

func convertChangeType(t v1pb.CheckRequest_ChangeType) storepb.PlanCheckRunConfig_ChangeDatabaseType {
	switch t {
	case v1pb.CheckRequest_DDL:
		return storepb.PlanCheckRunConfig_DDL
	case v1pb.CheckRequest_DDL_GHOST:
		return storepb.PlanCheckRunConfig_DDL_GHOST
	case v1pb.CheckRequest_DML:
		return storepb.PlanCheckRunConfig_DML
	default:
		return storepb.PlanCheckRunConfig_CHANGE_DATABASE_TYPE_UNSPECIFIED
	}
}
