package v1

import (
	"context"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/masker"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

func getSemanticTypesSettingWithBuiltins(ctx context.Context, stores *store.Store) (*storepb.SemanticTypeSetting, error) {
	setting, err := stores.GetSemanticTypesSetting(ctx, common.GetWorkspaceIDFromContext(ctx))
	if err != nil {
		return nil, err
	}
	return &storepb.SemanticTypeSetting{Types: masker.WithBuiltinSemanticTypes(setting.GetTypes())}, nil
}
