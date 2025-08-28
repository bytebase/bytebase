package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"connectrpc.com/connect"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
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

// geminiRequest represents the payload for Gemini API requests.
type geminiRequest struct {
	Contents         []geminiContent        `json:"contents"`
	GenerationConfig geminiGenerationConfig `json:"generationConfig"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
	Role  string       `json:"role"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiGenerationConfig struct {
	Temperature     float64  `json:"temperature"`
	TopP            float64  `json:"topP"`
	TopK            int      `json:"topK"`
	MaxOutputTokens int      `json:"maxOutputTokens"`
	StopSequences   []string `json:"stopSequences,omitempty"`
}

// geminiResponse represents the response from Gemini API.
type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

// claudeRequest represents the payload for Claude API requests.
type claudeRequest struct {
	Model         string          `json:"model"`
	Messages      []claudeMessage `json:"messages"`
	MaxTokens     int             `json:"max_tokens"`
	Temperature   float64         `json:"temperature"`
	TopP          float64         `json:"top_p"`
	TopK          int             `json:"top_k"`
	StopSequences []string        `json:"stop_sequences,omitempty"`
}

type claudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// claudeResponse represents the response from Claude API.
type claudeResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
}

// AICompletion is the mixer for AI completion.
func (s *SQLService) AICompletion(ctx context.Context, req *connect.Request[v1pb.AICompletionRequest]) (*connect.Response[v1pb.AICompletionResponse], error) {
	request := req.Msg
	aiSetting, err := s.store.GetAISetting(ctx)
	if err != nil {
		return nil, err
	}
	if len(aiSetting.Providers) == 0 {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("AI is not enabled"))
	}

	// Find the provider based on the request
	var provider *storepb.AIProvider
	// Use the specific provider requested
	for _, p := range aiSetting.Providers {
		if p.Type == storepb.AIProvider_Type(request.Provider) {
			provider = p
			break
		}
	}
	if provider == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("requested AI provider %s is not configured", request.Provider))
	}

	switch provider.Type {
	case storepb.AIProvider_OPEN_AI, storepb.AIProvider_AZURE_OPENAI:
		return callOpenAI(ctx, provider, request)
	case storepb.AIProvider_GEMINI:
		return callGemini(ctx, provider, request)
	case storepb.AIProvider_CLAUDE:
		return callClaude(ctx, provider, request)
	default:
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unsupported AI provider %s", provider.Type))
	}
}

func callOpenAI(ctx context.Context, provider *storepb.AIProvider, request *v1pb.AICompletionRequest) (*connect.Response[v1pb.AICompletionResponse], error) {
	payload := openAIRequest{
		Model: provider.Model,
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

	httpReq, err := http.NewRequestWithContext(ctx, "POST", provider.Endpoint, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, errors.Errorf("failed to create HTTP request: %s", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if provider.Type == storepb.AIProvider_AZURE_OPENAI {
		httpReq.Header.Set("api-key", provider.ApiKey)
	} else {
		httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", provider.ApiKey))
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
	return connect.NewResponse(resp), nil
}

// Gemini API docs: https://ai.google.dev/gemini-api/docs
func callGemini(ctx context.Context, provider *storepb.AIProvider, request *v1pb.AICompletionRequest) (*connect.Response[v1pb.AICompletionResponse], error) {
	// Convert messages to Gemini format
	var contents []geminiContent
	for _, m := range request.Messages {
		if m.Content == "" {
			continue
		}
		// Gemini uses "user" and "model" as roles
		role := m.Role
		if role != "user" {
			role = "model"
		}
		contents = append(contents, geminiContent{
			Role: role,
			Parts: []geminiPart{
				{Text: m.Content},
			},
		})
	}

	payload := geminiRequest{
		Contents: contents,
		GenerationConfig: geminiGenerationConfig{
			Temperature:     0.7,
			TopP:            0.95,
			TopK:            40,
			MaxOutputTokens: 2048,
		},
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, errors.Errorf("failed to marshal Gemini request payload: %s", err)
	}

	// Gemini API endpoint format: https://generativelanguage.googleapis.com/v1beta/models/{model}:generateContent?key={apiKey}
	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", provider.Endpoint, provider.Model, provider.ApiKey)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, errors.Errorf("failed to create HTTP request: %s", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	httpResp, err := client.Do(httpReq)
	if err != nil {
		return nil, errors.Errorf("failed to send HTTP request: %s", err)
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, errors.Errorf("failed to read response body: %s", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("Gemini API returned status %d: %s", httpResp.StatusCode, string(body))
	}

	var geminiResp geminiResponse
	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return nil, errors.Errorf("failed to unmarshal Gemini response: %s", err)
	}

	resp := &v1pb.AICompletionResponse{}
	for _, candidate := range geminiResp.Candidates {
		var sb strings.Builder
		for _, part := range candidate.Content.Parts {
			sb.WriteString(part.Text)
		}
		textContent := sb.String()

		// Strip code block markers (```language or ```)
		if strings.HasPrefix(textContent, "```") {
			// Find the end of the first line (after ```language)
			firstNewline := strings.Index(textContent, "\n")
			if firstNewline != -1 {
				textContent = textContent[firstNewline+1:]
			} else {
				textContent = strings.TrimPrefix(textContent, "```")
			}
		}
		textContent = strings.TrimSuffix(textContent, "```")
		textContent = strings.TrimSpace(textContent)

		resp.Candidates = append(resp.Candidates, &v1pb.AICompletionResponse_Candidate{
			Content: &v1pb.AICompletionResponse_Candidate_Content{
				Parts: []*v1pb.AICompletionResponse_Candidate_Content_Part{
					{
						Text: textContent,
					},
				},
			},
		})
	}
	return connect.NewResponse(resp), nil
}

// Claude API docs: https://docs.anthropic.com/en/api/getting-started
func callClaude(ctx context.Context, provider *storepb.AIProvider, request *v1pb.AICompletionRequest) (*connect.Response[v1pb.AICompletionResponse], error) {
	// Convert messages to Claude format
	var messages []claudeMessage
	for _, m := range request.Messages {
		messages = append(messages, claudeMessage{
			Role:    m.Role,
			Content: m.Content,
		})
	}

	payload := claudeRequest{
		Model:         provider.Model,
		Messages:      messages,
		MaxTokens:     4096,
		Temperature:   0.7,
		TopP:          0.95,
		TopK:          0,
		StopSequences: []string{"#", ";"},
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, errors.Errorf("failed to marshal Claude request payload: %s", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", provider.Endpoint, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, errors.Errorf("failed to create HTTP request: %s", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", provider.ApiKey)
	// Claude API requires anthropic-version header
	if provider.Version != "" {
		httpReq.Header.Set("anthropic-version", provider.Version)
	} else {
		httpReq.Header.Set("anthropic-version", "2023-06-01")
	}

	client := &http.Client{}
	httpResp, err := client.Do(httpReq)
	if err != nil {
		return nil, errors.Errorf("failed to send HTTP request: %s", err)
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, errors.Errorf("failed to read response body: %s", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("Claude API returned status %d: %s", httpResp.StatusCode, string(body))
	}

	var claudeResp claudeResponse
	if err := json.Unmarshal(body, &claudeResp); err != nil {
		return nil, errors.Errorf("failed to unmarshal Claude response: %s", err)
	}

	resp := &v1pb.AICompletionResponse{}
	var sb strings.Builder
	for _, content := range claudeResp.Content {
		if content.Type == "text" {
			sb.WriteString(content.Text)
		}
	}

	textContent := sb.String()
	if textContent != "" {
		resp.Candidates = append(resp.Candidates, &v1pb.AICompletionResponse_Candidate{
			Content: &v1pb.AICompletionResponse_Candidate_Content{
				Parts: []*v1pb.AICompletionResponse_Candidate_Content_Part{
					{
						Text: textContent,
					},
				},
			},
		})
	}

	return connect.NewResponse(resp), nil
}
