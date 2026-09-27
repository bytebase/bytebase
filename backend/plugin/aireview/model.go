package aireview

import (
	"context"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// Model is one model turn. It speaks the AIService.Chat types, the
// conversation vocabulary that backend/plugin/ai and the frontend agent loop
// already share. A vendor's private tool call data travels in
// AIChatToolCall.Metadata and goes back unchanged.
type Model interface {
	Chat(ctx context.Context, request *v1pb.AIChatRequest) (*v1pb.AIChatResponse, error)
}
