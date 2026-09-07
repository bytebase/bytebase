package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

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
	OpenAIToolCall         json.RawMessage   `json:"openAIToolCall,omitempty"`
	GeminiThoughtSignature string            `json:"geminiThoughtSignature,omitempty"`
	ResponsesOutput        []json.RawMessage `json:"responsesOutput,omitempty"`
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
	if len(metadata.OpenAIToolCall) == 0 && metadata.GeminiThoughtSignature == "" && len(metadata.ResponsesOutput) == 0 {
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
	return new(string(metadataBytes)), nil
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

func responsesOutputFromMetadata(raw string) []json.RawMessage {
	if metadata, ok := parseChatToolCallMetadata(raw); ok {
		return metadata.ResponsesOutput
	}
	return nil
}

func buildResponsesToolCallMetadata(output []json.RawMessage) (*string, error) {
	metadataBytes, err := json.Marshal(chatToolCallMetadata{ResponsesOutput: output})
	if err != nil {
		return nil, err
	}
	return new(string(metadataBytes)), nil
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

func chatOpenAI(ctx context.Context, aiSetting *storepb.AISetting, request *v1pb.AIChatRequest) (*v1pb.AIChatResponse, error) {
	if isOpenAIResponsesEndpoint(aiSetting.Endpoint) {
		return chatOpenAIResponses(ctx, aiSetting, request)
	}
	return chatOpenAIChatCompletions(ctx, aiSetting, request)
}

func isOpenAIResponsesEndpoint(endpoint string) bool {
	requestURL, err := url.Parse(endpoint)
	return err == nil && strings.HasSuffix(strings.TrimRight(requestURL.Path, "/"), "/responses")
}

func chatOpenAIChatCompletions(ctx context.Context, aiSetting *storepb.AISetting, request *v1pb.AIChatRequest) (*v1pb.AIChatResponse, error) {
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
	return result, nil
}

type responsesOpenAIRequest struct {
	Model   string                `json:"model"`
	Input   []json.RawMessage     `json:"input"`
	Store   bool                  `json:"store"`
	Include []string              `json:"include,omitempty"`
	Tools   []responsesOpenAITool `json:"tools,omitempty"`
}

type responsesOpenAITool struct {
	Type        string `json:"type"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Parameters  any    `json:"parameters"`
}

type responsesOpenAIResponse struct {
	Output []json.RawMessage `json:"output"`
	Usage  *chatOpenAIUsage  `json:"usage,omitempty"`
	Error  *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
	Status            string `json:"status,omitempty"`
	IncompleteDetails *struct {
		Reason string `json:"reason"`
	} `json:"incomplete_details,omitempty"`
}

type responsesOpenAIOutputItem struct {
	Type      string `json:"type"`
	CallID    string `json:"call_id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
	Content   []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
}

func chatOpenAIResponses(ctx context.Context, aiSetting *storepb.AISetting, request *v1pb.AIChatRequest) (*v1pb.AIChatResponse, error) {
	payload := responsesOpenAIRequest{
		Model:   aiSetting.Model,
		Store:   false,
		Include: []string{"reasoning.encrypted_content"},
	}
	for _, message := range request.Messages {
		items, err := responsesInputFromMessage(message)
		if err != nil {
			return nil, err
		}
		payload.Input = append(payload.Input, items...)
	}

	for _, definition := range request.ToolDefinitions {
		var parameters any
		if err := json.Unmarshal([]byte(definition.ParametersSchema), &parameters); err != nil {
			return nil, errors.Errorf("failed to parse parameters schema for tool %s: %s", definition.Name, err)
		}
		payload.Tools = append(payload.Tools, responsesOpenAITool{
			Type:        "function",
			Name:        definition.Name,
			Description: definition.Description,
			Parameters:  parameters,
		})
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, errors.Errorf("failed to marshal OpenAI Responses request: %s", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, aiSetting.Endpoint, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, errors.Errorf("failed to create OpenAI Responses request: %s", err)
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
	var response responsesOpenAIResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, errors.Errorf("failed to unmarshal OpenAI Responses response: %s", err)
	}
	if response.Error != nil {
		return nil, errors.Errorf("OpenAI Responses API returned error: %s", response.Error.Message)
	}
	if response.Status == "incomplete" {
		reason := ""
		if response.IncompleteDetails != nil {
			reason = response.IncompleteDetails.Reason
		}
		return nil, errors.Errorf("OpenAI Responses API returned an incomplete response: %s", reason)
	}

	result := &v1pb.AIChatResponse{}
	if response.Usage != nil {
		result.Usage = newAIChatUsage(response.Usage.TotalTokens)
	}
	var content strings.Builder
	for _, rawOutput := range response.Output {
		var output responsesOpenAIOutputItem
		if err := json.Unmarshal(rawOutput, &output); err != nil {
			return nil, errors.Errorf("failed to parse OpenAI Responses output item: %s", err)
		}
		switch output.Type {
		case "message":
			for _, part := range output.Content {
				if part.Type == "output_text" {
					content.WriteString(part.Text)
				}
			}
		case "function_call":
			result.ToolCalls = append(result.ToolCalls, &v1pb.AIChatToolCall{
				Id:        output.CallID,
				Name:      output.Name,
				Arguments: output.Arguments,
			})
		default:
			continue
		}
	}
	if content.Len() > 0 {
		text := content.String()
		result.Content = &text
	}
	if len(result.ToolCalls) > 0 {
		metadata, err := buildResponsesToolCallMetadata(response.Output)
		if err != nil {
			return nil, errors.Errorf("failed to encode OpenAI Responses tool metadata: %s", err)
		}
		result.ToolCalls[0].Metadata = metadata
	}
	return result, nil
}

func responsesInputFromMessage(message *v1pb.AIChatMessage) ([]json.RawMessage, error) {
	if message.Role == v1pb.AIChatMessageRole_AI_CHAT_MESSAGE_ROLE_TOOL {
		if message.ToolCallId == nil || *message.ToolCallId == "" {
			return nil, errors.New("OpenAI Responses tool result is missing a tool call ID")
		}
		output := ""
		if message.Content != nil {
			output = *message.Content
		}
		item, err := json.Marshal(map[string]string{
			"type":    "function_call_output",
			"call_id": *message.ToolCallId,
			"output":  output,
		})
		return []json.RawMessage{item}, err
	}

	if message.Role == v1pb.AIChatMessageRole_AI_CHAT_MESSAGE_ROLE_ASSISTANT && len(message.ToolCalls) > 0 {
		for _, toolCall := range message.ToolCalls {
			if toolCall.Metadata == nil {
				continue
			}
			if output := responsesOutputFromMetadata(*toolCall.Metadata); len(output) > 0 {
				return output, nil
			}
		}
	}

	items := make([]json.RawMessage, 0, 1+len(message.ToolCalls))
	if message.Content != nil {
		if message.Role == v1pb.AIChatMessageRole_AI_CHAT_MESSAGE_ROLE_ASSISTANT {
			item, err := json.Marshal(map[string]any{
				"type":   "message",
				"role":   "assistant",
				"status": "completed",
				"content": []map[string]string{{
					"type": "output_text",
					"text": *message.Content,
				}},
			})
			if err != nil {
				return nil, err
			}
			items = append(items, item)
		} else {
			role := chatMessageRoleToOpenAI(message.Role)
			if message.Role == v1pb.AIChatMessageRole_AI_CHAT_MESSAGE_ROLE_SYSTEM {
				role = "developer"
			}
			item, err := json.Marshal(map[string]string{
				"role":    role,
				"content": *message.Content,
			})
			if err != nil {
				return nil, err
			}
			items = append(items, item)
		}
	}
	if message.Role == v1pb.AIChatMessageRole_AI_CHAT_MESSAGE_ROLE_ASSISTANT {
		for _, toolCall := range message.ToolCalls {
			item, err := json.Marshal(map[string]string{
				"type":      "function_call",
				"call_id":   toolCall.Id,
				"name":      toolCall.Name,
				"arguments": toolCall.Arguments,
			})
			if err != nil {
				return nil, err
			}
			items = append(items, item)
		}
	}
	return items, nil
}
