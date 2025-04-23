package v1

import (
	v1pb "github.com/bytebase/bytebase/proto/generated-go/api/v1alpha"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

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
