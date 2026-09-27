package ai

import (
	"context"
	"io"
	"net/http"

	"github.com/pkg/errors"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

func newAIChatUsage(totalTokens int32) *v1pb.AIChatUsage {
	if totalTokens <= 0 {
		return nil
	}
	return &v1pb.AIChatUsage{
		TotalTokens: totalTokens,
	}
}

// doHTTPRequest executes an HTTP request and returns the response body.
func doHTTPRequest(_ context.Context, httpReq *http.Request, provider string) ([]byte, error) {
	client := &http.Client{}
	httpResp, err := client.Do(httpReq)
	if err != nil {
		return nil, errors.Errorf("failed to send HTTP request to %s: %s", provider, err)
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, errors.Errorf("failed to read %s response body: %s", provider, err)
	}

	if httpResp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("%s API returned status %d: %s", provider, httpResp.StatusCode, string(body))
	}

	return body, nil
}

// chatMessageRoleToOpenAI converts a proto role enum to OpenAI role string.
func chatMessageRoleToOpenAI(role v1pb.AIChatMessageRole) string {
	switch role {
	case v1pb.AIChatMessageRole_AI_CHAT_MESSAGE_ROLE_SYSTEM:
		return "system"
	case v1pb.AIChatMessageRole_AI_CHAT_MESSAGE_ROLE_USER:
		return "user"
	case v1pb.AIChatMessageRole_AI_CHAT_MESSAGE_ROLE_ASSISTANT:
		return "assistant"
	case v1pb.AIChatMessageRole_AI_CHAT_MESSAGE_ROLE_TOOL:
		return "tool"
	default:
		return "user"
	}
}

// getToolNameFromMessages finds the tool name for a given tool call ID by searching assistant messages.
func getToolNameFromMessages(messages []*v1pb.AIChatMessage, toolCallID string) string {
	for _, m := range messages {
		for _, tc := range m.ToolCalls {
			if tc.Id == toolCallID {
				return tc.Name
			}
		}
	}
	return toolCallID
}
