// Package aireview reviews the SQL of a migration against a plain-language
// policy with an LLM that reads the target database through tools.
package aireview

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/pkg/errors"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/ai"
)

const (
	defaultMaxModelCalls = 20
	defaultTimeout       = 10 * time.Minute
	// maxInvalidReplies is how many times the model is asked to try again, after
	// an invalid final reply or a tool call the vendor could not parse, before
	// the review fails.
	maxInvalidReplies = 1
	// malformedToolCallMessage answers a turn that held a tool call the vendor
	// could not parse. That turn is not in the history, so the message has to
	// say what happened.
	malformedToolCallMessage = "Your last turn held a tool call that could not be parsed, so no tool ran. Call the tool again, or reply with only the JSON object from the answer format."
)

var (
	// ErrCallLimit means the model kept calling tools until the call limit.
	ErrCallLimit = errors.New("the review reached the model call limit")
	// ErrDeadline means the review ran longer than its time limit.
	ErrDeadline = errors.New("the review ran out of time")
	// ErrInvalidReply means an invalid final reply, or a tool call the vendor could not parse, came when no correction round or call was left.
	ErrInvalidReply = errors.New("the model did not return a valid review result")
	// ErrEmptyReply means the vendor returned neither text nor tool calls.
	ErrEmptyReply = errors.New("the model returned an empty reply")
	// ErrTruncatedReply means the reply was truncated at a token limit, so findings may be missing.
	ErrTruncatedReply = errors.New("the model's reply was truncated at a token limit")
	// ErrBlockedReply means the vendor blocked the request or the reply, or the model refused to review.
	ErrBlockedReply = errors.New("the vendor blocked the review or the model refused it")
	// ErrUnfinishedReply means the stop reason was OTHER, such as Gemini's OTHER, or a value the loop does not know.
	ErrUnfinishedReply = errors.New("the vendor ended the turn for a reason the review does not accept")
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
	// Notes name what the model could not check and why, for example a fact
	// the sync does not hold. They are for the backend's telemetry, not for
	// the user, and they do not affect the verdict.
	Notes []string
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
	rules := Rules(request.WorkspacePolicy, request.ProjectPolicy)
	nonce := rand.Text()
	messages := buildMessages(request, numberedStatement, nonce, len(definitions) > 0)
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
			// Some deployments refuse tool definitions, such as an Azure GPT-5.6
			// deployment on chat completions. The review then runs without
			// tools. Any other refusal, a content filter above all, fails the
			// review, so a blocked review cannot pass on a second try.
			if result.Calls == 1 && len(definitions) > 0 && refusesTools(err) {
				slog.Warn("AI review model refused the request with tools, reviewing without them", slog.String("error", err.Error()))
				definitions, toolNames = nil, nil
				messages = buildMessages(request, numberedStatement, nonce, false)
				continue
			}
			return nil, errors.Wrapf(err, "model call %d failed", result.Calls)
		}
		result.TotalTokens += int(response.GetUsage().GetTotalTokens())

		content := response.GetContent()
		toolCalls := response.GetToolCalls()
		// A truncated reply can parse as a whole answer with findings or tool
		// arguments missing, a blocked change was never judged, and OTHER can be
		// a block the vendor does not name. None of the three gets a correction
		// round, which could pass a change the model did not finish judging.
		switch response.GetStopReason() {
		case v1pb.AIChatResponse_TURN_COMPLETED, v1pb.AIChatResponse_STOP_REASON_UNSPECIFIED:
			// A vendor that sends no reason, such as a compatible server without finish_reason, is judged by its reply.
		case v1pb.AIChatResponse_TOKEN_LIMIT:
			return nil, ErrTruncatedReply
		case v1pb.AIChatResponse_BLOCKED:
			return nil, ErrBlockedReply
		case v1pb.AIChatResponse_MALFORMED_TOOL_CALL:
			// The model meant to call a tool, so the turn is not final whatever
			// text or tool calls came with it. None of it runs or enters the
			// history. The turn takes a correction round from the budget of an
			// invalid reply, and fails the review when no round or call is left.
			if invalidReplies >= maxInvalidReplies || result.Calls >= r.maxModelCalls {
				return nil, errors.Wrap(ErrInvalidReply, "the vendor could not parse the model's tool call")
			}
			invalidReplies++
			messages = append(messages, textMessage(v1pb.AIChatMessageRole_AI_CHAT_MESSAGE_ROLE_USER, malformedToolCallMessage))
			continue
		default:
			// OTHER lands here, and so does any value added later.
			return nil, ErrUnfinishedReply
		}
		// An empty reply that no stop reason explains can be a block the vendor
		// did not name, and asking again could let a blocked review pass.
		if strings.TrimSpace(content) == "" && len(toolCalls) == 0 {
			return nil, ErrEmptyReply
		}
		messages = append(messages, &v1pb.AIChatMessage{
			Role:      v1pb.AIChatMessageRole_AI_CHAT_MESSAGE_ROLE_ASSISTANT,
			Content:   response.Content,
			ToolCalls: toolCalls,
		})

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

		lastChance := invalidReplies >= maxInvalidReplies || result.Calls >= r.maxModelCalls
		parsed, problems := parseReply(content, lineCount, rules, lastChance)
		if len(problems) == 0 {
			result.Findings = parsed.Findings
			result.Notes = parsed.Notes
			return result, nil
		}
		if lastChance {
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

func refusesTools(err error) bool {
	var httpErr *ai.HTTPError
	return errors.As(err, &httpErr) && httpErr.RefusesTools
}
