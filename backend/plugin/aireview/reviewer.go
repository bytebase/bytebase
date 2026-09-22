// Package aireview reviews the SQL of a migration against a plain-language
// policy with an LLM that reads the target database through tools.
package aireview

import (
	"context"
	"crypto/rand"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

const (
	defaultMaxModelCalls = 20
	defaultTimeout       = 10 * time.Minute
	// maxInvalidReplies is how many times the model is asked to correct a
	// final reply before the review fails.
	maxInvalidReplies = 1
)

var (
	// ErrCallLimit means the model kept calling tools until the call limit.
	ErrCallLimit = errors.New("the review reached the model call limit")
	// ErrDeadline means the review ran longer than its time limit.
	ErrDeadline = errors.New("the review ran out of time")
	// ErrInvalidReply means the final reply was still invalid after the model was asked to correct it.
	ErrInvalidReply = errors.New("the model did not return a valid review result")
)

// Request is one review: one sheet against one target database.
type Request struct {
	WorkspacePolicy string
	ProjectPolicy   string
	Target          Target
	// Statement is the sheet text. Finding lines refer to its lines.
	Statement string
}

// Result is the outcome of a completed review. No findings means the change passes.
type Result struct {
	Findings []Finding
	// Calls is the number of model calls the review made.
	Calls int
	// TotalTokens is the token usage the vendor reported, summed over the calls.
	TotalTokens int
}

// Reviewer runs reviews against one model.
type Reviewer struct {
	model         Model
	maxModelCalls int
	timeout       time.Duration
}

// NewReviewer creates a Reviewer.
func NewReviewer(model Model) *Reviewer {
	return &Reviewer{
		model:         model,
		maxModelCalls: defaultMaxModelCalls,
		timeout:       defaultTimeout,
	}
}

// Review runs the agent loop until the model answers without a tool call. It
// never returns a partial answer: any error, bound, or invalid reply fails the
// review, so a failure is never mistaken for a pass.
func (r *Reviewer) Review(parent context.Context, request *Request, tools Tools) (*Result, error) {
	if strings.TrimSpace(request.Statement) == "" {
		return nil, errors.New("the statement is empty")
	}

	// The timeout covers the whole review, so every model call, retry, and tool
	// call counts against the same clock.
	ctx, cancel := context.WithTimeout(parent, r.timeout)
	defer cancel()

	definitions := tools.Definitions()
	toolNames := make([]string, 0, len(definitions))
	for _, definition := range definitions {
		toolNames = append(toolNames, definition.GetName())
	}

	numberedStatement, lineCount := numberLines(request.Statement)
	messages := buildMessages(request, numberedStatement, rand.Text())
	result := &Result{}
	invalidReplies := 0

	for result.Calls < r.maxModelCalls {
		if err := r.contextError(parent, ctx, result.Calls); err != nil {
			return nil, err
		}
		result.Calls++
		response, err := r.model.Chat(ctx, &v1pb.AIChatRequest{Messages: messages, ToolDefinitions: definitions})
		if err != nil {
			if ctxErr := r.contextError(parent, ctx, result.Calls); ctxErr != nil {
				return nil, ctxErr
			}
			return nil, errors.Wrapf(err, "model call %d failed", result.Calls)
		}
		result.TotalTokens += int(response.GetUsage().GetTotalTokens())

		content := response.GetContent()
		toolCalls := response.GetToolCalls()
		// Vendors reject a history that holds an assistant message with neither
		// text nor tool calls, which would break the correction round.
		if strings.TrimSpace(content) != "" || len(toolCalls) > 0 {
			messages = append(messages, &v1pb.AIChatMessage{
				Role:      v1pb.AIChatMessageRole_AI_CHAT_MESSAGE_ROLE_ASSISTANT,
				Content:   response.Content,
				ToolCalls: toolCalls,
			})
		}

		// A reply with tool calls is not final, whatever else it says.
		if len(toolCalls) > 0 {
			if result.Calls >= r.maxModelCalls {
				// No call is left to carry the results, so do not run the tools.
				break
			}
			// Vendors reject the next request unless every call has a result.
			for _, call := range toolCalls {
				output, err := callTool(ctx, tools, toolNames, call)
				if err != nil {
					if ctxErr := r.contextError(parent, ctx, result.Calls); ctxErr != nil {
						return nil, ctxErr
					}
					return nil, errors.Wrapf(err, "tool %q failed", call.GetName())
				}
				toolCallID := call.GetId()
				messages = append(messages, &v1pb.AIChatMessage{
					Role:       v1pb.AIChatMessageRole_AI_CHAT_MESSAGE_ROLE_TOOL,
					Content:    &output,
					ToolCallId: &toolCallID,
				})
			}
			continue
		}

		findings, problems := parseFindings(content, lineCount)
		if len(problems) == 0 {
			result.Findings = findings
			return result, nil
		}
		if invalidReplies >= maxInvalidReplies || result.Calls >= r.maxModelCalls {
			return nil, errors.Wrap(ErrInvalidReply, strings.Join(problems, "; "))
		}
		invalidReplies++
		messages = append(messages, textMessage(v1pb.AIChatMessageRole_AI_CHAT_MESSAGE_ROLE_USER, correctionMessage(problems)))
	}
	return nil, errors.Wrapf(ErrCallLimit, "%d calls", r.maxModelCalls)
}

// contextError reports why the review must stop, or nil when it can go on. A
// canceled parent is returned as is, so the caller can tell it from a failure.
func (r *Reviewer) contextError(parent context.Context, ctx context.Context, calls int) error {
	if err := parent.Err(); err != nil {
		return err
	}
	if ctx.Err() != nil {
		return errors.Wrapf(ErrDeadline, "%s, %d model calls", r.timeout, calls)
	}
	return nil
}

func callTool(ctx context.Context, tools Tools, toolNames []string, call *v1pb.AIChatToolCall) (string, error) {
	for _, name := range toolNames {
		if name == call.GetName() {
			return tools.Call(ctx, call.GetName(), call.GetArguments())
		}
	}
	return fmt.Sprintf("unknown tool %q; the tools are: %s", call.GetName(), strings.Join(toolNames, ", ")), nil
}

// correctionMessage asks for a corrected reply. It offers a tool call as well
// as the JSON: a model that replied "let me check the table" has not finished,
// and asking it for the JSON alone would make it answer without the fact.
func correctionMessage(problems []string) string {
	var b strings.Builder
	_, _ = b.WriteString("Your reply is not a valid review result:\n")
	for _, problem := range problems {
		_, _ = fmt.Fprintf(&b, "- %s\n", problem)
	}
	_, _ = b.WriteString("If the review is not complete, call a tool now. Otherwise reply with only the JSON object from the answer format.")
	return b.String()
}
