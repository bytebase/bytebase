package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// openAIRequest represents the payload for OpenAI API requests.
type openAIRequest struct {
	Model    string          `json:"model"`
	Messages []openAIMessage `json:"messages"`
	TopP     float64         `json:"top_p"`
	Stop     []string        `json:"stop,omitempty"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// openAIResponse represents the response from OpenAI API.
type openAIResponse struct {
	Choices []struct {
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// AICompletion is the mixer for AI completion.
func (s *SQLService) AICompletion(ctx context.Context, request *v1pb.AICompletionRequest) (*v1pb.AICompletionResponse, error) {
	aiSetting, err := s.store.GetAISetting(ctx)
	if err != nil {
		return nil, err
	}
	if !aiSetting.Enabled {
		return nil, status.Errorf(codes.FailedPrecondition, "AI is not enabled")
	}

	switch aiSetting.Provider {
	case storepb.AISetting_OPEN_AI, storepb.AISetting_AZURE_OPENAI:
		return callOpenAI(ctx, aiSetting, request)
	default:
		return nil, status.Errorf(codes.InvalidArgument, "unsupported AI provider %s", aiSetting.Provider)
	}
}

func callOpenAI(ctx context.Context, aiSetting *storepb.AISetting, request *v1pb.AICompletionRequest) (*v1pb.AICompletionResponse, error) {
	payload := openAIRequest{
		Model: aiSetting.Model,
		TopP:  1.0,
		Stop:  []string{"#", ";"},
	}
	for _, m := range request.Messages {
		payload.Messages = append(payload.Messages, openAIMessage{
			Role:    m.Role,
			Content: m.Content,
		})
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, errors.Errorf("failed to marshal OpenAI request payload: %s", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", aiSetting.Endpoint, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, errors.Errorf("failed to create HTTP request: %s", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if aiSetting.Provider == storepb.AISetting_AZURE_OPENAI {
		httpReq.Header.Set("api-key", aiSetting.ApiKey)
	} else {
		httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", aiSetting.ApiKey))
	}

	client := &http.Client{}
	httpResp, err := client.Do(httpReq)
	if err != nil {
		return nil, errors.Errorf("failed to send HTTP request: %s", err)
	}
	defer httpResp.Body.Close()

	// Read and parse the response
	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, errors.Errorf("failed to read response body: %s", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("OpenAI API returned status %d: %s", httpResp.StatusCode, string(body))
	}

	var openAIResponse openAIResponse
	if err := json.Unmarshal(body, &openAIResponse); err != nil {
		return nil, errors.Errorf("failed to unmarshal OpenAI response: %s", err)
	}

	resp := &v1pb.AICompletionResponse{}
	for _, choice := range openAIResponse.Choices {
		resp.Candidates = append(resp.Candidates, &v1pb.AICompletionResponse_Candidate{
			Content: &v1pb.AICompletionResponse_Candidate_Content{
				Parts: []*v1pb.AICompletionResponse_Candidate_Content_Part{
					{
						Text: choice.Message.Content,
					},
				},
			},
		})
	}
	return resp, nil
}
