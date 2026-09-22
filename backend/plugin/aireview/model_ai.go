package aireview

import (
	"context"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/ai"
)

type settingModel struct {
	setting *storepb.AISetting
}

// NewModel returns the Model behind the workspace AI setting.
func NewModel(setting *storepb.AISetting) Model {
	return &settingModel{setting: setting}
}

func (m *settingModel) Chat(ctx context.Context, request *v1pb.AIChatRequest) (*v1pb.AIChatResponse, error) {
	return ai.Chat(ctx, m.setting, request)
}
