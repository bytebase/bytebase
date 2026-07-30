package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// aiLintResult represents a single lint result from AI.
type aiLintResult struct {
	RuleName string `json:"ruleName"`
	Passed   bool   `json:"passed"`
	Message  string `json:"message"`
	File     string `json:"file,omitempty"`
	Line     *int   `json:"line,omitempty"`
	Column   *int   `json:"column,omitempty"`
}

// fileSchema represents a file and its schema content for batch linting.
type fileSchema struct {
	Path    string
	Content string
}

// runAIPoweredLintBatch performs AI-powered schema linting for multiple files in batch.
func (s *ReleaseService) runAIPoweredLintBatch(ctx context.Context, files []fileSchema, customRules string) (map[string][]*v1pb.Advice, error) {
	if customRules == "" || len(files) == 0 {
		return nil, nil
	}

	slog.Info("Starting batch AI-powered linting", "filesCount", len(files), "rulesLength", len(customRules))

	aiSetting, err := s.store.GetAISetting(ctx, common.GetWorkspaceIDFromContext(ctx))
	if err != nil {
		slog.Error("Failed to get AI setting", "error", err)
		return nil, errors.Wrap(err, "failed to get AI setting")
	}
	if !aiSetting.Enabled {
		slog.Info("AI is not enabled, skipping AI-powered linting")
		return nil, nil
	}

	slog.Info("AI setting loaded", "provider", aiSetting.Provider, "model", aiSetting.Model)

	// Build batch prompt with all files
	prompt := buildBatchLintPrompt(files, customRules)
	slog.Debug("AI batch prompt built", "promptLength", len(prompt))

	// Call AI through the shared chat provider dispatch.
	request := &v1pb.AIChatRequest{
		Messages: []*v1pb.AIChatMessage{
			{
				Role:    v1pb.AIChatMessageRole_AI_CHAT_MESSAGE_ROLE_USER,
				Content: &prompt,
			},
		},
	}

	slog.Info("Calling AI provider for batch schema linting", "provider", aiSetting.Provider, "endpoint", aiSetting.Endpoint, "filesCount", len(files))
	resp, err := chatWithProvider(ctx, aiSetting, request)
	if err != nil {
		slog.Error("AI call failed", "provider", aiSetting.Provider, "error", err)
		return nil, errors.Wrapf(err, "%s API call failed", aiSetting.Provider)
	}
	responseText := resp.GetContent()

	if responseText == "" {
		slog.Warn("AI returned empty response")
		return nil, errors.New("empty response from AI")
	}

	slog.Info("Parsing AI lint response", "responseLength", len(responseText))
	advicesMap, err := parseBatchAILintResponse(responseText, files)
	if err != nil {
		slog.Error("Failed to parse AI response", "error", err, "response", responseText)
		return nil, err
	}

	totalAdvices := 0
	for _, advices := range advicesMap {
		totalAdvices += len(advices)
	}
	slog.Info("Batch AI linting completed", "filesCount", len(files), "totalAdvices", totalAdvices)
	return advicesMap, nil
}

// buildBatchLintPrompt constructs the prompt for batch AI-powered linting.
func buildBatchLintPrompt(files []fileSchema, rulesContent string) string {
	prompt := `You are a schema validator. Please validate the following schema files against the provided lint rules.

Lint rules:
` + rulesContent + `

Schema files:
`
	for i, file := range files {
		prompt += "\n--- File " + fmt.Sprint(i+1) + ": " + file.Path + " ---\n"
		prompt += file.Content + "\n"
	}

	prompt += `
For each rule in the lint rules, check if each schema file complies with it. Return your results in the following JSON format:
[
  {
    "ruleName": "rule name",
    "passed": true/false,
    "message": "explanation of the result",
    "file": "file path (e.g., 'schema1.sql')",
    "line": line number if applicable,
    "column": column number if applicable
  }
]

Important:
- Include the "file" field to indicate which file the issue is in
- Only include results where "passed" is false (violations)
- Return only the JSON array, no other text

Only return the JSON array, no other text.`
	return prompt
}

// stripCodeFence removes a wrapping Markdown code fence (```lang ... ```) if present.
func stripCodeFence(text string) string {
	text = strings.TrimSpace(text)
	if strings.HasPrefix(text, "```") {
		if idx := strings.Index(text, "\n"); idx != -1 {
			text = text[idx+1:]
		} else {
			text = strings.TrimPrefix(text, "```")
		}
	}
	text = strings.TrimSuffix(strings.TrimSpace(text), "```")
	return strings.TrimSpace(text)
}

// parseBatchAILintResponse parses the batch AI response text into a map of file -> Advice messages.
func parseBatchAILintResponse(responseText string, files []fileSchema) (map[string][]*v1pb.Advice, error) {
	var results []aiLintResult
	if err := json.Unmarshal([]byte(stripCodeFence(responseText)), &results); err != nil {
		return nil, errors.Wrap(err, "failed to parse AI lint response")
	}

	// Create a map of file path to file for easy lookup
	fileMap := make(map[string]bool)
	for _, f := range files {
		fileMap[f.Path] = true
	}

	advicesMap := make(map[string][]*v1pb.Advice)
	for _, result := range results {
		if result.Passed {
			continue
		}

		// Use the file field from AI response, or empty string if not specified
		filePath := result.File
		if filePath == "" && len(files) == 1 {
			// If only one file and no file specified, assume it's for that file
			filePath = files[0].Path
		}

		advice := &v1pb.Advice{
			Status:   v1pb.Advice_ERROR,
			Code:     0,
			Title:    result.RuleName,
			Content:  result.Message,
			RuleType: v1pb.Advice_AI_POWERED,
		}

		if result.Line != nil {
			advice.StartPosition = &v1pb.Position{
				Line:   int32(*result.Line),
				Column: 0,
			}
			if result.Column != nil {
				advice.StartPosition.Column = int32(*result.Column)
			}
		}

		advicesMap[filePath] = append(advicesMap[filePath], advice)
	}

	return advicesMap, nil
}
