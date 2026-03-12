package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"connectrpc.com/connect"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
	"github.com/bytebase/bytebase/backend/store"
)

// AIService implements the AI chat service with tool-calling support.
type AIService struct {
	v1connect.UnimplementedAIServiceHandler
	store *store.Store
}

// NewAIService creates a new AIService.
func NewAIService(store *store.Store) *AIService {
	return &AIService{
		store: store,
	}
}

// Chat sends a conversation with tool definitions to the configured AI provider and returns the response.
func (s *AIService) Chat(ctx context.Context, req *connect.Request[v1pb.AIChatRequest]) (*connect.Response[v1pb.AIChatResponse], error) {
	aiSetting, err := s.store.GetAISetting(ctx)
	if err != nil {
		return nil, err
	}
	if !aiSetting.Enabled {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("AI is not enabled"))
	}

	switch aiSetting.Provider {
	case storepb.AISetting_OPEN_AI, storepb.AISetting_AZURE_OPENAI:
		return s.chatOpenAI(ctx, aiSetting, req.Msg)
	case storepb.AISetting_GEMINI:
		return s.chatGemini(ctx, aiSetting, req.Msg)
	case storepb.AISetting_CLAUDE:
		return s.chatClaude(ctx, aiSetting, req.Msg)
	default:
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unsupported AI provider %s", aiSetting.Provider))
	}
}

// OpenAI chat types with tool-calling support.

type chatOpenAIRequest struct {
	Model    string              `json:"model"`
	Messages []chatOpenAIMessage `json:"messages"`
	Tools    []chatOpenAITool    `json:"tools,omitempty"`
}

type chatOpenAIMessage struct {
	Role       string              `json:"role"`
	Content    *string             `json:"content"`
	ToolCalls  []chatOpenAIToolUse `json:"tool_calls,omitempty"`
	ToolCallID string              `json:"tool_call_id,omitempty"`
}

type chatOpenAITool struct {
	Type     string             `json:"type"`
	Function chatOpenAIFunction `json:"function"`
}

type chatOpenAIFunction struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Parameters  any    `json:"parameters"`
}

type chatOpenAIToolUse struct {
	ID               string                    `json:"id"`
	Type             string                    `json:"type"`
	Function         chatOpenAIFunctionCallRef `json:"function"`
	ThoughtSignature string                    `json:"thought_signature,omitempty"`
}

type chatOpenAIFunctionCallRef struct {
	Name             string `json:"name"`
	Arguments        string `json:"arguments"`
	ThoughtSignature string `json:"thought_signature,omitempty"`
}

type chatOpenAIResponse struct {
	Choices []struct {
		Message struct {
			Role      string              `json:"role"`
			Content   *string             `json:"content"`
			ToolCalls []chatOpenAIToolUse `json:"tool_calls"`
		} `json:"message"`
	} `json:"choices"`
}

func (*AIService) chatOpenAI(ctx context.Context, aiSetting *storepb.AISetting, request *v1pb.AIChatRequest) (*connect.Response[v1pb.AIChatResponse], error) {
	payload := chatOpenAIRequest{
		Model: aiSetting.Model,
	}

	for _, m := range request.Messages {
		msg := chatOpenAIMessage{
			Role: chatMessageRoleToOpenAI(m.Role),
		}
		if m.Content != nil {
			msg.Content = m.Content
		}
		if m.ToolCallId != nil {
			msg.ToolCallID = *m.ToolCallId
		}
		for _, tc := range m.ToolCalls {
			toolUse := chatOpenAIToolUse{
				ID:   tc.Id,
				Type: "function",
				Function: chatOpenAIFunctionCallRef{
					Name:      tc.Name,
					Arguments: tc.Arguments,
				},
			}
			if tc.Metadata != nil && *tc.Metadata != "" {
				toolUse.ThoughtSignature = *tc.Metadata
			}
			msg.ToolCalls = append(msg.ToolCalls, toolUse)
		}
		payload.Messages = append(payload.Messages, msg)
	}

	for _, td := range request.ToolDefinitions {
		var params any
		if err := json.Unmarshal([]byte(td.ParametersSchema), &params); err != nil {
			return nil, errors.Errorf("failed to parse parameters schema for tool %s: %s", td.Name, err)
		}
		payload.Tools = append(payload.Tools, chatOpenAITool{
			Type: "function",
			Function: chatOpenAIFunction{
				Name:        td.Name,
				Description: td.Description,
				Parameters:  params,
			},
		})
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, errors.Errorf("failed to marshal OpenAI request: %s", err)
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

	body, err := doHTTPRequest(ctx, httpReq, "OpenAI")
	if err != nil {
		return nil, err
	}

	var resp chatOpenAIResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, errors.Errorf("failed to unmarshal OpenAI response: %s", err)
	}

	result := &v1pb.AIChatResponse{}
	if len(resp.Choices) > 0 {
		msg := resp.Choices[0].Message
		result.Content = msg.Content
		for _, tc := range msg.ToolCalls {
			toolCall := &v1pb.AIChatToolCall{
				Id:        tc.ID,
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			}
			// Capture thought_signature from Gemini's OpenAI-compatible endpoint.
			sig := tc.ThoughtSignature
			if sig == "" {
				sig = tc.Function.ThoughtSignature
			}
			if sig != "" {
				toolCall.Metadata = &sig
			}
			result.ToolCalls = append(result.ToolCalls, toolCall)
		}
	}
	return connect.NewResponse(result), nil
}

// Claude chat types with tool-calling support.

type chatClaudeRequest struct {
	Model     string           `json:"model"`
	System    string           `json:"system,omitempty"`
	Messages  []chatClaudeMsg  `json:"messages"`
	Tools     []chatClaudeTool `json:"tools,omitempty"`
	MaxTokens int              `json:"max_tokens"`
}

type chatClaudeMsg struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type chatClaudeTool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	InputSchema any    `json:"input_schema"`
}

type chatClaudeContentBlock struct {
	Type      string `json:"type"`
	Text      string `json:"text,omitempty"`
	ID        string `json:"id,omitempty"`
	Name      string `json:"name,omitempty"`
	Input     any    `json:"input,omitempty"`
	ToolUseID string `json:"tool_use_id,omitempty"`
	Content   string `json:"content,omitempty"`
}

type chatClaudeResponse struct {
	Content []chatClaudeContentBlock `json:"content"`
}

func (*AIService) chatClaude(ctx context.Context, aiSetting *storepb.AISetting, request *v1pb.AIChatRequest) (*connect.Response[v1pb.AIChatResponse], error) {
	payload := chatClaudeRequest{
		Model:     aiSetting.Model,
		MaxTokens: 4096,
	}

	for _, m := range request.Messages {
		switch m.Role {
		case v1pb.AIChatMessageRole_AI_CHAT_MESSAGE_ROLE_SYSTEM:
			payload.System = m.GetContent()
			continue
		case v1pb.AIChatMessageRole_AI_CHAT_MESSAGE_ROLE_USER:
			payload.Messages = append(payload.Messages, chatClaudeMsg{
				Role:    "user",
				Content: m.GetContent(),
			})
		case v1pb.AIChatMessageRole_AI_CHAT_MESSAGE_ROLE_ASSISTANT:
			var contentBlocks []chatClaudeContentBlock
			if m.Content != nil && *m.Content != "" {
				contentBlocks = append(contentBlocks, chatClaudeContentBlock{
					Type: "text",
					Text: *m.Content,
				})
			}
			for _, tc := range m.ToolCalls {
				var input any
				if err := json.Unmarshal([]byte(tc.Arguments), &input); err != nil {
					input = tc.Arguments
				}
				contentBlocks = append(contentBlocks, chatClaudeContentBlock{
					Type:  "tool_use",
					ID:    tc.Id,
					Name:  tc.Name,
					Input: input,
				})
			}
			payload.Messages = append(payload.Messages, chatClaudeMsg{
				Role:    "assistant",
				Content: contentBlocks,
			})
		case v1pb.AIChatMessageRole_AI_CHAT_MESSAGE_ROLE_TOOL:
			// Claude uses "user" role with tool_result content block.
			payload.Messages = append(payload.Messages, chatClaudeMsg{
				Role: "user",
				Content: []chatClaudeContentBlock{
					{
						Type:      "tool_result",
						ToolUseID: m.GetToolCallId(),
						Content:   m.GetContent(),
					},
				},
			})
		default:
			continue
		}
	}

	for _, td := range request.ToolDefinitions {
		var schema any
		if err := json.Unmarshal([]byte(td.ParametersSchema), &schema); err != nil {
			return nil, errors.Errorf("failed to parse parameters schema for tool %s: %s", td.Name, err)
		}
		payload.Tools = append(payload.Tools, chatClaudeTool{
			Name:        td.Name,
			Description: td.Description,
			InputSchema: schema,
		})
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, errors.Errorf("failed to marshal Claude request: %s", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", aiSetting.Endpoint, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, errors.Errorf("failed to create HTTP request: %s", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", aiSetting.ApiKey)
	if aiSetting.Version != "" {
		httpReq.Header.Set("anthropic-version", aiSetting.Version)
	} else {
		httpReq.Header.Set("anthropic-version", "2023-06-01")
	}

	body, err := doHTTPRequest(ctx, httpReq, "Claude")
	if err != nil {
		return nil, err
	}

	var resp chatClaudeResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, errors.Errorf("failed to unmarshal Claude response: %s", err)
	}

	result := &v1pb.AIChatResponse{}
	var textContent string
	for _, block := range resp.Content {
		switch block.Type {
		case "text":
			textContent += block.Text
		case "tool_use":
			args, err := json.Marshal(block.Input)
			if err != nil {
				return nil, errors.Errorf("failed to marshal tool call input: %s", err)
			}
			result.ToolCalls = append(result.ToolCalls, &v1pb.AIChatToolCall{
				Id:        block.ID,
				Name:      block.Name,
				Arguments: string(args),
			})
		default:
		}
	}
	if textContent != "" {
		result.Content = &textContent
	}
	return connect.NewResponse(result), nil
}

// Gemini chat types with tool-calling support.

type chatGeminiRequest struct {
	Contents []chatGeminiContent `json:"contents"`
	Tools    []chatGeminiTool    `json:"tools,omitempty"`
}

type chatGeminiContent struct {
	Role  string           `json:"role"`
	Parts []chatGeminiPart `json:"parts"`
}

type chatGeminiPart struct {
	Text             string                    `json:"text,omitempty"`
	FunctionCall     *chatGeminiFunctionCall   `json:"functionCall,omitempty"`
	FunctionResponse *chatGeminiFunctionResult `json:"functionResponse,omitempty"`
}

type chatGeminiFunctionCall struct {
	Name             string `json:"name"`
	Args             any    `json:"args"`
	ThoughtSignature string `json:"thoughtSignature,omitempty"`
}

type chatGeminiFunctionResult struct {
	Name     string `json:"name"`
	Response any    `json:"response"`
}

type chatGeminiTool struct {
	FunctionDeclarations []chatGeminiFunctionDecl `json:"functionDeclarations"`
}

type chatGeminiFunctionDecl struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Parameters  any    `json:"parameters"`
}

type chatGeminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []chatGeminiResponsePart `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

type chatGeminiResponsePart struct {
	Text         string                 `json:"text,omitempty"`
	FunctionCall *chatGeminiFunctionCall `json:"functionCall,omitempty"`
}

func (*AIService) chatGemini(ctx context.Context, aiSetting *storepb.AISetting, request *v1pb.AIChatRequest) (*connect.Response[v1pb.AIChatResponse], error) {
	payload := chatGeminiRequest{}

	for _, m := range request.Messages {
		switch m.Role {
		case v1pb.AIChatMessageRole_AI_CHAT_MESSAGE_ROLE_SYSTEM:
			// Gemini handles system messages as the first user message.
			payload.Contents = append(payload.Contents, chatGeminiContent{
				Role:  "user",
				Parts: []chatGeminiPart{{Text: m.GetContent()}},
			})
		case v1pb.AIChatMessageRole_AI_CHAT_MESSAGE_ROLE_USER:
			payload.Contents = append(payload.Contents, chatGeminiContent{
				Role:  "user",
				Parts: []chatGeminiPart{{Text: m.GetContent()}},
			})
		case v1pb.AIChatMessageRole_AI_CHAT_MESSAGE_ROLE_ASSISTANT:
			var parts []chatGeminiPart
			if m.Content != nil && *m.Content != "" {
				parts = append(parts, chatGeminiPart{Text: *m.Content})
			}
			for _, tc := range m.ToolCalls {
				var args any
				if err := json.Unmarshal([]byte(tc.Arguments), &args); err != nil {
					args = tc.Arguments
				}
				fc := &chatGeminiFunctionCall{
					Name: tc.Name,
					Args: args,
				}
				if tc.Metadata != nil {
					fc.ThoughtSignature = *tc.Metadata
				}
				parts = append(parts, chatGeminiPart{
					FunctionCall: fc,
				})
			}
			payload.Contents = append(payload.Contents, chatGeminiContent{
				Role:  "model",
				Parts: parts,
			})
		case v1pb.AIChatMessageRole_AI_CHAT_MESSAGE_ROLE_TOOL:
			// Gemini uses functionResponse parts in a "user" role message.
			// We need the tool name; get it from the tool call ID by searching prior messages.
			toolName := getToolNameFromMessages(request.Messages, m.GetToolCallId())
			var resultData any
			if err := json.Unmarshal([]byte(m.GetContent()), &resultData); err != nil {
				resultData = map[string]string{"result": m.GetContent()}
			}
			payload.Contents = append(payload.Contents, chatGeminiContent{
				Role: "user",
				Parts: []chatGeminiPart{
					{
						FunctionResponse: &chatGeminiFunctionResult{
							Name:     toolName,
							Response: resultData,
						},
					},
				},
			})
		default:
		}
	}

	if len(request.ToolDefinitions) > 0 {
		var decls []chatGeminiFunctionDecl
		for _, td := range request.ToolDefinitions {
			var params any
			if err := json.Unmarshal([]byte(td.ParametersSchema), &params); err != nil {
				return nil, errors.Errorf("failed to parse parameters schema for tool %s: %s", td.Name, err)
			}
			decls = append(decls, chatGeminiFunctionDecl{
				Name:        td.Name,
				Description: td.Description,
				Parameters:  params,
			})
		}
		payload.Tools = []chatGeminiTool{{FunctionDeclarations: decls}}
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, errors.Errorf("failed to marshal Gemini request: %s", err)
	}

	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", aiSetting.Endpoint, aiSetting.Model, aiSetting.ApiKey)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, errors.Errorf("failed to create HTTP request: %s", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	body, err := doHTTPRequest(ctx, httpReq, "Gemini")
	if err != nil {
		return nil, err
	}

	var resp chatGeminiResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, errors.Errorf("failed to unmarshal Gemini response: %s", err)
	}

	result := &v1pb.AIChatResponse{}
	if len(resp.Candidates) > 0 {
		var textContent string
		for _, part := range resp.Candidates[0].Content.Parts {
			if part.Text != "" {
				textContent += part.Text
			}
			if part.FunctionCall != nil {
				args, err := json.Marshal(part.FunctionCall.Args)
				if err != nil {
					return nil, errors.Errorf("failed to marshal function call args: %s", err)
				}
				tc := &v1pb.AIChatToolCall{
					Id:        fmt.Sprintf("call_%s", part.FunctionCall.Name),
					Name:      part.FunctionCall.Name,
					Arguments: string(args),
				}
				if part.FunctionCall.ThoughtSignature != "" {
					tc.Metadata = &part.FunctionCall.ThoughtSignature
				}
				result.ToolCalls = append(result.ToolCalls, tc)
			}
		}
		if textContent != "" {
			result.Content = &textContent
		}
	}
	return connect.NewResponse(result), nil
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
