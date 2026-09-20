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

func (m *settingModel) Chat(ctx context.Context, request *ChatRequest) (*ChatResponse, error) {
	response, err := ai.Chat(ctx, m.setting, toAIChatRequest(request))
	if err != nil {
		return nil, err
	}

	message := Message{Role: RoleAssistant, Content: response.GetContent()}
	for _, call := range response.GetToolCalls() {
		message.ToolCalls = append(message.ToolCalls, ToolCall{
			ID:        call.GetId(),
			Name:      call.GetName(),
			Arguments: call.GetArguments(),
			Replay:    call.GetMetadata(),
		})
	}
	return &ChatResponse{
		Message: message,
		// ai.Chat does not report the stop reason yet.
		StopReason: StopReasonUnknown,
		Usage:      Usage{TotalTokens: int(response.GetUsage().GetTotalTokens())},
	}, nil
}

func toAIChatRequest(request *ChatRequest) *v1pb.AIChatRequest {
	result := &v1pb.AIChatRequest{}
	for _, message := range request.Messages {
		converted := &v1pb.AIChatMessage{Role: toAIChatRole(message.Role)}
		// A nil content goes out as JSON null, which is how OpenAI expects an
		// assistant message that holds only tool calls.
		if message.Content != "" || message.Role != RoleAssistant {
			content := message.Content
			converted.Content = &content
		}
		for _, call := range message.ToolCalls {
			convertedCall := &v1pb.AIChatToolCall{Id: call.ID, Name: call.Name, Arguments: call.Arguments}
			if call.Replay != "" {
				replay := call.Replay
				convertedCall.Metadata = &replay
			}
			converted.ToolCalls = append(converted.ToolCalls, convertedCall)
		}
		if message.ToolCallID != "" {
			toolCallID := message.ToolCallID
			converted.ToolCallId = &toolCallID
		}
		result.Messages = append(result.Messages, converted)
	}
	for _, tool := range request.Tools {
		result.ToolDefinitions = append(result.ToolDefinitions, &v1pb.AIChatToolDefinition{
			Name:             tool.Name,
			Description:      tool.Description,
			ParametersSchema: tool.ParametersSchema,
		})
	}
	return result
}

func toAIChatRole(role Role) v1pb.AIChatMessageRole {
	switch role {
	case RoleSystem:
		return v1pb.AIChatMessageRole_AI_CHAT_MESSAGE_ROLE_SYSTEM
	case RoleUser:
		return v1pb.AIChatMessageRole_AI_CHAT_MESSAGE_ROLE_USER
	case RoleAssistant:
		return v1pb.AIChatMessageRole_AI_CHAT_MESSAGE_ROLE_ASSISTANT
	case RoleTool:
		return v1pb.AIChatMessageRole_AI_CHAT_MESSAGE_ROLE_TOOL
	default:
		return v1pb.AIChatMessageRole_AI_CHAT_MESSAGE_ROLE_UNSPECIFIED
	}
}
