package v1

import (
	"context"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

const (
	defaultSemanticTypeID        = "bb.default"
	defaultPartialSemanticTypeID = "bb.default-partial"
)

func getBuiltinSemanticTypes() []*storepb.SemanticTypeSetting_SemanticType {
	return []*storepb.SemanticTypeSetting_SemanticType{
		{
			Id:    defaultSemanticTypeID,
			Title: "Default",
			Algorithm: &storepb.Algorithm{
				Mask: &storepb.Algorithm_FullMask_{
					FullMask: &storepb.Algorithm_FullMask{},
				},
			},
		},
		{
			Id:    defaultPartialSemanticTypeID,
			Title: "Default Partial",
			Algorithm: &storepb.Algorithm{
				Mask: &storepb.Algorithm_RangeMask_{
					RangeMask: &storepb.Algorithm_RangeMask{},
				},
			},
		},
	}
}

func getSemanticTypesSettingWithBuiltins(ctx context.Context, stores *store.Store) (*storepb.SemanticTypeSetting, error) {
	setting, err := stores.GetSemanticTypesSetting(ctx, common.GetWorkspaceIDFromContext(ctx))
	if err != nil {
		return nil, err
	}
	return appendBuiltinSemanticTypes(setting), nil
}

func appendBuiltinSemanticTypes(setting *storepb.SemanticTypeSetting) *storepb.SemanticTypeSetting {
	types := make([]*storepb.SemanticTypeSetting_SemanticType, 0, len(setting.GetTypes())+2)
	for _, semanticType := range setting.GetTypes() {
		if !isBuiltinSemanticTypeID(semanticType.GetId()) {
			types = append(types, semanticType)
		}
	}
	types = append(types, getBuiltinSemanticTypes()...)
	return &storepb.SemanticTypeSetting{Types: types}
}

func isBuiltinSemanticTypeID(id string) bool {
	return id == defaultSemanticTypeID || id == defaultPartialSemanticTypeID
}
