package v1

import (
	"context"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/masker"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

const (
	defaultSemanticTypeID        = masker.DefaultSemanticTypeID
	defaultPartialSemanticTypeID = masker.DefaultPartialSemanticTypeID
)

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
		if !masker.IsBuiltinSemanticTypeID(semanticType.GetId()) {
			types = append(types, semanticType)
		}
	}
	types = append(types, masker.BuiltinSemanticTypes()...)
	return &storepb.SemanticTypeSetting{Types: types}
}
