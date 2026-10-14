package ai

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

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

// doHTTPRequest executes an HTTP request and returns the response body. A
// status other than 200 is an *HTTPError, to which readFailure adds what the
// vendor's body format says. A call with no complete reply is a *TransportError.
func doHTTPRequest(_ context.Context, httpReq *http.Request, provider string, readFailure func(*HTTPError)) ([]byte, error) {
	client := &http.Client{}
	httpResp, err := client.Do(httpReq)
	if err != nil {
		return nil, &TransportError{message: fmt.Sprintf("failed to send HTTP request to %s: %s", provider, err)}
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, &TransportError{message: fmt.Sprintf("failed to read %s response body: %s", provider, err)}
	}

	if httpResp.StatusCode != http.StatusOK {
		httpErr := &HTTPError{
			Provider:    provider,
			StatusCode:  httpResp.StatusCode,
			RetryAfter:  providerWait(httpResp.Header, time.Now()),
			ShouldRetry: readShouldRetry(httpResp.Header),
			Body:        string(body),
		}
		readFailure(httpErr)
		return nil, httpErr
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
