package v1

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
	"github.com/bytebase/bytebase/backend/plugin/ai"
	"github.com/bytebase/bytebase/backend/store"
)

// AIService implements the AI chat service with tool-calling support.
type AIService struct {
	v1connect.UnimplementedAIServiceHandler
	store *store.Store
}

// NewAIService creates a new AIService.
func NewAIService(store *store.Store) *AIService {
	return &AIService{store: store}
}

// Chat sends a conversation with tool definitions to the configured AI provider and returns the response.
func (s *AIService) Chat(ctx context.Context, req *connect.Request[v1pb.AIChatRequest]) (*connect.Response[v1pb.AIChatResponse], error) {
	aiSetting, err := s.store.GetAISetting(ctx, common.GetWorkspaceIDFromContext(ctx))
	if err != nil {
		return nil, err
	}
	if !aiSetting.Enabled {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("AI is not enabled"))
	}

	resp, err := ai.Chat(ctx, aiSetting, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}
