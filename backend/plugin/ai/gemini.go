package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

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

func newGeminiToolCallID(name string) string {
	return fmt.Sprintf("call_%s_%s", name, uuid.NewString())
}

func chatGemini(ctx context.Context, aiSetting *storepb.AISetting, request *v1pb.AIChatRequest) (*v1pb.AIChatResponse, error) {
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
					Id:        newGeminiToolCallID(part.FunctionCall.Name),
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
	return result, nil
}
