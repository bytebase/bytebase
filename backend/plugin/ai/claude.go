package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

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
	Content []json.RawMessage `json:"content"`
	Usage   *chatClaudeUsage  `json:"usage,omitempty"`
}

func claudeContentFromToolCalls(toolCalls []*v1pb.AIChatToolCall) []json.RawMessage {
	for _, toolCall := range toolCalls {
		if toolCall.Metadata == nil {
			continue
		}
		if content := claudeContentFromMetadata(*toolCall.Metadata); len(content) > 0 {
			return content
		}
	}
	return nil
}

type claudeErrorReply struct {
	Error struct {
		Details struct {
			ErrorCode string `json:"error_code"`
		} `json:"details"`
	} `json:"error"`
}

// readClaudeFailure marks the reply for a reached monthly spend cap. Anthropic
// answers it with 429 and the type rate_limit_error, like a rate limit, and
// tells the two apart by details.error_code.
func readClaudeFailure(httpErr *HTTPError) {
	var reply claudeErrorReply
	if err := json.Unmarshal([]byte(httpErr.Body), &reply); err != nil {
		return
	}
	httpErr.QuotaExhausted = reply.Error.Details.ErrorCode == "enforced_spend_limit_reached"
}

// defaultClaudeMaxTokens is the output limit of a call that asks for none.
const defaultClaudeMaxTokens = 4096

// claudeOutputCeilings are the documented output maximums of older models, and
// Anthropic rejects a larger max_tokens with a 400. A model id matches the
// first family it contains, ignoring case, so dated, aliased, Bedrock, and
// Vertex ids all match.
var claudeOutputCeilings = []struct {
	family    *regexp.Regexp
	maxTokens int
}{
	{family: claudeFamily("claude-3-5-sonnet"), maxTokens: 8192},
	{family: claudeFamily("claude-3-5-haiku"), maxTokens: 8192},
	{family: claudeFamily("claude-3-opus"), maxTokens: 4096},
	{family: claudeFamily("claude-3-sonnet"), maxTokens: 4096},
	{family: claudeFamily("claude-3-haiku"), maxTokens: 4096},
	{family: claudeFamily("claude-2"), maxTokens: 4096},
	{family: claudeFamily("claude-v2"), maxTokens: 4096},
	{family: claudeFamily("claude-instant"), maxTokens: 4096},
}

// claudeFamily matches the name anywhere in a model id. A name that ends in a
// version number must not run on into another digit, so claude-2 matches
// claude-2.1 but not claude-20.
func claudeFamily(name string) *regexp.Regexp {
	pattern := regexp.QuoteMeta(name)
	if last := name[len(name)-1]; last >= '0' && last <= '9' {
		pattern += `($|[^0-9])`
	}
	return regexp.MustCompile(pattern)
}

// claudeMaxTokens caps the requested output limit at the model's ceiling. A
// model missing from the table keeps the request.
func claudeMaxTokens(model string, requested int) int {
	if requested <= 0 {
		return defaultClaudeMaxTokens
	}
	model = strings.ToLower(model)
	for _, ceiling := range claudeOutputCeilings {
		if ceiling.family.MatchString(model) {
			return min(requested, ceiling.maxTokens)
		}
	}
	return requested
}

func chatClaude(ctx context.Context, aiSetting *storepb.AISetting, request *v1pb.AIChatRequest, options ...ChatOption) (*v1pb.AIChatResponse, error) {
	payload := chatClaudeRequest{
		Model:     aiSetting.Model,
		MaxTokens: claudeMaxTokens(aiSetting.Model, newChatOptions(options).maxOutputTokens),
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
			if content := claudeContentFromToolCalls(m.ToolCalls); len(content) > 0 {
				payload.Messages = append(payload.Messages, chatClaudeMsg{
					Role:    "assistant",
					Content: content,
				})
				continue
			}
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
			result := chatClaudeContentBlock{
				Type:      "tool_result",
				ToolUseID: m.GetToolCallId(),
				Content:   m.GetContent(),
			}
			// Anthropic asks for the results of one turn's tool calls in a single
			// user message, since separate messages teach Claude to stop calling
			// tools in parallel. A user message holds blocks only for tool results.
			if last := len(payload.Messages) - 1; last >= 0 && payload.Messages[last].Role == "user" {
				if results, ok := payload.Messages[last].Content.([]chatClaudeContentBlock); ok {
					results = append(results, result)
					payload.Messages[last].Content = results
					continue
				}
			}
			payload.Messages = append(payload.Messages, chatClaudeMsg{
				Role:    "user",
				Content: []chatClaudeContentBlock{result},
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

	body, err := doHTTPRequest(ctx, httpReq, "Claude", readClaudeFailure)
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
	for _, rawBlock := range resp.Content {
		var block chatClaudeContentBlock
		if err := json.Unmarshal(rawBlock, &block); err != nil {
			return nil, errors.Errorf("failed to parse Claude response content block: %s", err)
		}
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
	if len(result.ToolCalls) > 0 {
		metadata, err := buildClaudeToolCallMetadata(resp.Content)
		if err != nil {
			return nil, errors.Errorf("failed to encode Claude tool metadata: %s", err)
		}
		for _, toolCall := range result.ToolCalls {
			toolCall.Metadata = metadata
		}
	}
	if textContent != "" {
		result.Content = &textContent
	}
	return result, nil
}
