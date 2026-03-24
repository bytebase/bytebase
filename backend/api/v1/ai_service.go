package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"connectrpc.com/connect"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
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
	aiSetting, err := s.store.GetAISetting(ctx, common.GetWorkspaceIDFromContext(ctx))
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
	Role       string            `json:"role"`
	Content    *string           `json:"content"`
	ToolCalls  []json.RawMessage `json:"tool_calls,omitempty"`
	ToolCallID string            `json:"tool_call_id,omitempty"`
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
	ID       string                    `json:"id"`
	Type     string                    `json:"type"`
	Function chatOpenAIFunctionCallRef `json:"function"`
}

type chatOpenAIFunctionCallRef struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type chatOpenAIUsage struct {
	TotalTokens int32 `json:"total_tokens"`
}

type chatOpenAIResponse struct {
	Choices []struct {
		Message struct {
			Role      string            `json:"role"`
			Content   *string           `json:"content"`
			ToolCalls []json.RawMessage `json:"tool_calls"`
		} `json:"message"`
	} `json:"choices"`
	Usage *chatOpenAIUsage `json:"usage,omitempty"`
}

// chatOpenAIToolCallParsed extracts the fields we need from a raw tool call JSON.
type chatOpenAIToolCallParsed struct {
	ID       string `json:"id"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type chatToolCallMetadata struct {
	OpenAIToolCall         json.RawMessage `json:"openAIToolCall,omitempty"`
	GeminiThoughtSignature string          `json:"geminiThoughtSignature,omitempty"`
}

type chatOpenAIToolCallProviderFields struct {
	ExtraContent struct {
		Google struct {
			ThoughtSignature string `json:"thought_signature,omitempty"`
		} `json:"google,omitempty"`
	} `json:"extra_content,omitempty"`
}

func parseChatToolCallMetadata(raw string) (chatToolCallMetadata, bool) {
	metadataBytes := bytes.TrimSpace([]byte(raw))
	if len(metadataBytes) == 0 {
		return chatToolCallMetadata{}, false
	}
	var metadata chatToolCallMetadata
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		return chatToolCallMetadata{}, false
	}
	if len(metadata.OpenAIToolCall) == 0 && metadata.GeminiThoughtSignature == "" {
		return chatToolCallMetadata{}, false
	}
	return metadata, true
}

func buildChatToolCallMetadata(rawOpenAIToolCall json.RawMessage, geminiThoughtSignature string) (*string, error) {
	metadata := chatToolCallMetadata{
		OpenAIToolCall:         rawOpenAIToolCall,
		GeminiThoughtSignature: geminiThoughtSignature,
	}
	if metadata.GeminiThoughtSignature == "" {
		metadata.GeminiThoughtSignature = extractGeminiThoughtSignatureFromOpenAIToolCall(rawOpenAIToolCall)
	}
	if len(metadata.OpenAIToolCall) == 0 && metadata.GeminiThoughtSignature == "" {
		return nil, nil
	}
	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return nil, err
	}
	metadataString := string(metadataBytes)
	return &metadataString, nil
}

func openAIToolCallFromMetadata(raw string) json.RawMessage {
	if metadata, ok := parseChatToolCallMetadata(raw); ok {
		return metadata.OpenAIToolCall
	}
	metadataBytes := bytes.TrimSpace([]byte(raw))
	if len(metadataBytes) == 0 || !json.Valid(metadataBytes) {
		return nil
	}
	return json.RawMessage(metadataBytes)
}

func geminiThoughtSignatureFromMetadata(raw string) string {
	if metadata, ok := parseChatToolCallMetadata(raw); ok {
		if metadata.GeminiThoughtSignature != "" {
			return metadata.GeminiThoughtSignature
		}
		return extractGeminiThoughtSignatureFromOpenAIToolCall(metadata.OpenAIToolCall)
	}
	metadataBytes := bytes.TrimSpace([]byte(raw))
	if len(metadataBytes) == 0 {
		return ""
	}
	if json.Valid(metadataBytes) {
		return extractGeminiThoughtSignatureFromOpenAIToolCall(json.RawMessage(metadataBytes))
	}
	return string(metadataBytes)
}

func extractGeminiThoughtSignatureFromOpenAIToolCall(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var providerFields chatOpenAIToolCallProviderFields
	if err := json.Unmarshal(raw, &providerFields); err != nil {
		return ""
	}
	return providerFields.ExtraContent.Google.ThoughtSignature
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
			if tc.Metadata != nil {
				if rawToolCall := openAIToolCallFromMetadata(*tc.Metadata); len(rawToolCall) > 0 {
					// Replay the raw tool call JSON exactly as received from the provider.
					// This preserves provider-specific fields like Gemini's thought_signature.
					msg.ToolCalls = append(msg.ToolCalls, rawToolCall)
					continue
				}
			}
			// Construct a standard OpenAI tool call.
			raw, _ := json.Marshal(chatOpenAIToolUse{
				ID:   tc.Id,
				Type: "function",
				Function: chatOpenAIFunctionCallRef{
					Name:      tc.Name,
					Arguments: tc.Arguments,
				},
			})
			msg.ToolCalls = append(msg.ToolCalls, json.RawMessage(raw))
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
	if resp.Usage != nil {
		result.Usage = newAIChatUsage(resp.Usage.TotalTokens)
	}
	if len(resp.Choices) > 0 {
		msg := resp.Choices[0].Message
		result.Content = msg.Content
		for _, rawTC := range msg.ToolCalls {
			var parsed chatOpenAIToolCallParsed
			if err := json.Unmarshal(rawTC, &parsed); err != nil {
				return nil, errors.Errorf("failed to parse tool call: %s", err)
			}
			// Store provider-specific tool metadata in a structured envelope so each
			// adapter can replay only the fields it understands.
			metadata, err := buildChatToolCallMetadata(rawTC, "")
			if err != nil {
				return nil, errors.Errorf("failed to encode tool call metadata: %s", err)
			}
			toolCall := &v1pb.AIChatToolCall{
				Id:        parsed.ID,
				Name:      parsed.Function.Name,
				Arguments: parsed.Function.Arguments,
				Metadata:  metadata,
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

type chatClaudeUsage struct {
	InputTokens  int32 `json:"input_tokens"`
	OutputTokens int32 `json:"output_tokens"`
}

type chatClaudeResponse struct {
	Content []chatClaudeContentBlock `json:"content"`
	Usage   *chatClaudeUsage         `json:"usage,omitempty"`
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
	if resp.Usage != nil {
		result.Usage = newAIChatUsage(resp.Usage.InputTokens + resp.Usage.OutputTokens)
	}
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
	ThoughtSignature string                    `json:"thoughtSignature,omitempty"`
}

type chatGeminiFunctionCall struct {
	Name string `json:"name"`
	Args any    `json:"args"`
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

type chatGeminiUsage struct {
	TotalTokenCount int32 `json:"totalTokenCount"`
}

type chatGeminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []chatGeminiResponsePart `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	UsageMetadata *chatGeminiUsage `json:"usageMetadata,omitempty"`
}

type chatGeminiResponsePart struct {
	Text             string                  `json:"text,omitempty"`
	FunctionCall     *chatGeminiFunctionCall `json:"functionCall,omitempty"`
	ThoughtSignature string                  `json:"thoughtSignature,omitempty"`
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
				part := chatGeminiPart{
					FunctionCall: &chatGeminiFunctionCall{
						Name: tc.Name,
						Args: args,
					},
				}
				if tc.Metadata != nil {
					part.ThoughtSignature = geminiThoughtSignatureFromMetadata(*tc.Metadata)
				}
				parts = append(parts, part)
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

	requestURL, err := url.JoinPath(aiSetting.Endpoint, "models", aiSetting.Model+":generateContent")
	if err != nil {
		return nil, errors.Wrap(err, "failed to build Gemini request URL")
	}
	httpReq, err := http.NewRequestWithContext(ctx, "POST", requestURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, errors.Errorf("failed to create HTTP request: %s", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-goog-api-key", aiSetting.ApiKey)

	body, err := doHTTPRequest(ctx, httpReq, "Gemini")
	if err != nil {
		return nil, err
	}

	var resp chatGeminiResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, errors.Errorf("failed to unmarshal Gemini response: %s", err)
	}

	result := &v1pb.AIChatResponse{}
	if resp.UsageMetadata != nil {
		result.Usage = newAIChatUsage(resp.UsageMetadata.TotalTokenCount)
	}
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
				if part.ThoughtSignature != "" {
					metadata, err := buildChatToolCallMetadata(nil, part.ThoughtSignature)
					if err != nil {
						return nil, errors.Errorf("failed to encode Gemini tool metadata: %s", err)
					}
					tc.Metadata = metadata
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
