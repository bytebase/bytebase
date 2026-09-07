// Package ai provides adapters for configured AI providers.
package ai

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// Chat sends a chat request to the configured AI provider.
func Chat(ctx context.Context, aiSetting *storepb.AISetting, request *v1pb.AIChatRequest) (*v1pb.AIChatResponse, error) {
	switch aiSetting.Provider {
	case storepb.AISetting_OPEN_AI, storepb.AISetting_AZURE_OPENAI:
		return chatOpenAI(ctx, aiSetting, request)
	case storepb.AISetting_GEMINI:
		return chatGemini(ctx, aiSetting, request)
	case storepb.AISetting_CLAUDE:
		return chatClaude(ctx, aiSetting, request)
	default:
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unsupported AI provider %s", aiSetting.Provider))
	}
}
