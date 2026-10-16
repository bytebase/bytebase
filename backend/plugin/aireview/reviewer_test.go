package aireview

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/ai"
)

const (
	roleSystem    = v1pb.AIChatMessageRole_AI_CHAT_MESSAGE_ROLE_SYSTEM
	roleUser      = v1pb.AIChatMessageRole_AI_CHAT_MESSAGE_ROLE_USER
	roleAssistant = v1pb.AIChatMessageRole_AI_CHAT_MESSAGE_ROLE_ASSISTANT
	roleTool      = v1pb.AIChatMessageRole_AI_CHAT_MESSAGE_ROLE_TOOL
)

const (
	stopTurnCompleted     = v1pb.AIChatResponse_TURN_COMPLETED
	stopTokenLimit        = v1pb.AIChatResponse_TOKEN_LIMIT
	stopBlocked           = v1pb.AIChatResponse_BLOCKED
	stopMalformedToolCall = v1pb.AIChatResponse_MALFORMED_TOOL_CALL
	stopOther             = v1pb.AIChatResponse_OTHER
)

const threeLineStatement = "ALTER TABLE orders ADD COLUMN note text;\n\nDELETE FROM order_events;\n"

const validReply = `{"findings": [{"title": "Add a WHERE clause", "severity": "P0", "line": 3, "rule": "an UPDATE or DELETE that touches more rows than the author means", "evidence": "order_events has 9000000 rows", "fix": "Delete in batches by id range"}]}`

type modelStep struct {
	response *v1pb.AIChatResponse
	err      error
	// block makes the step wait until the context ends.
	block bool
	// cancelParent is called when the step starts, before it blocks.
	cancelParent context.CancelFunc
}

// scriptedModel replays steps in order and repeats the last one.
type scriptedModel struct {
	steps    []modelStep
	requests []*v1pb.AIChatRequest
}

func (m *scriptedModel) Chat(ctx context.Context, request *v1pb.AIChatRequest) (*v1pb.AIChatResponse, error) {
	// The loop appends to the same slice after the call, so keep a copy.
	m.requests = append(m.requests, &v1pb.AIChatRequest{Messages: slices.Clone(request.Messages), ToolDefinitions: request.ToolDefinitions})
	step := m.steps[min(len(m.requests), len(m.steps))-1]
	if step.cancelParent != nil {
		step.cancelParent()
	}
	if step.block {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	return step.response, step.err
}

type fakeTools struct {
	outputs map[string]string
	err     error
	calls   []string
}

func (*fakeTools) Definitions() []*v1pb.AIChatToolDefinition {
	return []*v1pb.AIChatToolDefinition{{Name: "read"}, {Name: "search"}}
}

func (f *fakeTools) Call(_ context.Context, name string, _ string) (string, error) {
	f.calls = append(f.calls, name)
	if f.err != nil {
		return "", f.err
	}
	return f.outputs[name], nil
}

func finalReply(content string) modelStep {
	return modelStep{response: &v1pb.AIChatResponse{Content: &content, Usage: &v1pb.AIChatUsage{TotalTokens: 100}}}
}

func toolCallReply(calls ...*v1pb.AIChatToolCall) modelStep {
	return modelStep{response: &v1pb.AIChatResponse{ToolCalls: calls, Usage: &v1pb.AIChatUsage{TotalTokens: 100}}}
}

func (s modelStep) stoppedFor(reason v1pb.AIChatResponse_StopReason) modelStep {
	s.response.StopReason = reason
	return s
}

func TestReviewReturnsFindings(t *testing.T) {
	t.Parallel()

	model := &scriptedModel{steps: []modelStep{finalReply(validReply)}}
	result, err := NewReviewer(model).Review(context.Background(), &Request{Statement: threeLineStatement}, &fakeTools{})
	require.NoError(t, err)
	require.Equal(t, 1, result.Calls)
	require.Equal(t, 100, result.TotalTokens)
	require.Equal(t, []Finding{{
		Title:    "Add a WHERE clause",
		Severity: SeverityP0,
		Line:     3,
		Rule:     "an UPDATE or DELETE that touches more rows than the author means",
		Evidence: "order_events has 9000000 rows",
		Fix:      "Delete in batches by id range",
	}}, result.Findings)

	require.Len(t, model.requests, 1)
	require.Equal(t, []v1pb.AIChatMessageRole{roleSystem, roleUser}, roles(model.requests[0].Messages))
	require.Len(t, model.requests[0].ToolDefinitions, 2)
}

func TestReviewAnswersEveryToolCallAndKeepsMetadata(t *testing.T) {
	t.Parallel()

	signature := "thought-signature"
	calls := []*v1pb.AIChatToolCall{
		{Id: "call-1", Name: "search", Arguments: `{"text": "orders"}`, Metadata: &signature},
		{Id: "call-2", Name: "read", Arguments: `{"objects": [{"name": "orders"}]}`},
	}
	model := &scriptedModel{steps: []modelStep{toolCallReply(calls...), finalReply(`{"findings": [], "notes": ["orders has no row count in the sync"]}`)}}
	tools := &fakeTools{outputs: map[string]string{"search": "orders_summary", "read": "CREATE TABLE orders"}}

	result, err := NewReviewer(model).Review(context.Background(), &Request{Statement: threeLineStatement}, tools)
	require.NoError(t, err)
	require.Empty(t, result.Findings)
	require.Equal(t, []string{"orders has no row count in the sync"}, result.Notes)
	require.Equal(t, 2, result.Calls)
	require.Equal(t, 200, result.TotalTokens)
	require.Equal(t, []string{"search", "read"}, tools.calls)

	history := model.requests[1].Messages
	require.Equal(t, []v1pb.AIChatMessageRole{roleSystem, roleUser, roleAssistant, roleTool, roleTool}, roles(history))
	require.Equal(t, calls, history[2].ToolCalls, "the assistant turn replays the calls with their vendor metadata")
	require.Nil(t, history[2].Content, "an assistant turn without text carries no content")
	requireToolMessage(t, history[3], "call-1", "orders_summary")
	requireToolMessage(t, history[4], "call-2", "CREATE TABLE orders")
}

func TestReviewExitPaths(t *testing.T) {
	t.Parallel()

	toolFailure := errors.New("metadata store timed out")
	tests := []struct {
		name          string
		steps         []modelStep
		tools         *fakeTools
		maxModelCalls int
		wantErr       error
		wantErrText   string
		wantFindings  int
		wantCalls     int
		// wantRequests is checked when it is not zero.
		wantRequests int
		// wantLastUserText must appear in the last user message the model saw.
		wantLastUserText string
		wantToolCalls    []string
	}{
		{
			name:         "reply in code fences",
			steps:        []modelStep{finalReply("```json\n" + validReply + "\n```")},
			wantFindings: 1,
			wantCalls:    1,
		},
		{
			name:             "invalid reply is corrected once",
			steps:            []modelStep{finalReply(`{"findings": [{"title": "t", "severity": "critical", "line": 3, "rule": "r", "evidence": "e", "fix": "f"}]}`), finalReply(validReply)},
			wantFindings:     1,
			wantCalls:        2,
			wantLastUserText: `findings[0].severity: "critical" is not one of P0, P1, P2`,
		},
		{
			name:             "an invented error key gets a correction round",
			steps:            []modelStep{finalReply(`{"findings": [], "error": "could not review"}`), finalReply(`{"findings": [], "notes": ["could not review"]}`)},
			wantCalls:        2,
			wantLastUserText: `unknown object member name "error"`,
		},
		{
			name:        "invalid reply twice fails",
			steps:       []modelStep{finalReply("not json"), finalReply("still not json")},
			wantErr:     ErrInvalidReply,
			wantErrText: "the reply is not a JSON object",
		},
		{
			name: "announcing a lookup is sent back to call the tool",
			steps: []modelStep{
				finalReply("Let me check the orders table first."),
				toolCallReply(&v1pb.AIChatToolCall{Id: "call-1", Name: "read"}),
				finalReply(validReply),
			},
			wantFindings:  1,
			wantCalls:     3,
			wantToolCalls: []string{"read"},
		},
		{
			name:         "empty reply fails without a correction round",
			steps:        []modelStep{finalReply(" \n"), finalReply(validReply)},
			wantErr:      ErrEmptyReply,
			wantRequests: 1,
		},
		{
			name:         "empty reply of a completed turn fails without a correction round",
			steps:        []modelStep{finalReply("").stoppedFor(stopTurnCompleted), finalReply(validReply)},
			wantErr:      ErrEmptyReply,
			wantRequests: 1,
		},
		{
			name:         "empty reply with another stop reason fails without a correction round",
			steps:        []modelStep{finalReply("").stoppedFor(stopOther), finalReply(validReply)},
			wantErr:      ErrUnfinishedReply,
			wantRequests: 1,
		},
		{
			name:         "reply of a completed turn is parsed",
			steps:        []modelStep{finalReply(validReply).stoppedFor(stopTurnCompleted)},
			wantFindings: 1,
			wantCalls:    1,
		},
		{
			name:         "reply without a stop reason is parsed",
			steps:        []modelStep{finalReply(validReply).stoppedFor(v1pb.AIChatResponse_STOP_REASON_UNSPECIFIED)},
			wantFindings: 1,
			wantCalls:    1,
		},
		{
			name:         "reply with another stop reason fails even when it passes",
			steps:        []modelStep{finalReply(`{"findings": []}`).stoppedFor(stopOther), finalReply(validReply)},
			wantErr:      ErrUnfinishedReply,
			wantRequests: 1,
		},
		{
			name:         "turn with another stop reason runs no tool",
			steps:        []modelStep{toolCallReply(&v1pb.AIChatToolCall{Id: "call-1", Name: "read"}).stoppedFor(stopOther), finalReply(validReply)},
			wantErr:      ErrUnfinishedReply,
			wantRequests: 1,
		},
		{
			name:         "reply with a stop reason added later fails",
			steps:        []modelStep{finalReply(`{"findings": []}`).stoppedFor(v1pb.AIChatResponse_StopReason(99)), finalReply(validReply)},
			wantErr:      ErrUnfinishedReply,
			wantRequests: 1,
		},
		{
			name:         "truncated reply fails even when it parses",
			steps:        []modelStep{finalReply(validReply).stoppedFor(stopTokenLimit), finalReply(validReply)},
			wantErr:      ErrTruncatedReply,
			wantRequests: 1,
		},
		{
			name:         "truncated reply that does not parse gets no correction round",
			steps:        []modelStep{finalReply(`{"findings": [{"title": "Add a WHERE`).stoppedFor(stopTokenLimit), finalReply(validReply)},
			wantErr:      ErrTruncatedReply,
			wantRequests: 1,
		},
		{
			name:         "truncated turn with tool calls runs no tool",
			steps:        []modelStep{toolCallReply(&v1pb.AIChatToolCall{Id: "call-1", Name: "read"}).stoppedFor(stopTokenLimit), finalReply(validReply)},
			wantErr:      ErrTruncatedReply,
			wantRequests: 1,
		},
		{
			name:         "blocked reply with text gets no correction round",
			steps:        []modelStep{finalReply("I can't help with that request.").stoppedFor(stopBlocked), finalReply(validReply)},
			wantErr:      ErrBlockedReply,
			wantRequests: 1,
		},
		{
			name:         "blocked reply that parses still fails",
			steps:        []modelStep{finalReply(`{"findings": []}`).stoppedFor(stopBlocked)},
			wantErr:      ErrBlockedReply,
			wantRequests: 1,
		},
		{
			name:         "blocked reply without text gets no correction round",
			steps:        []modelStep{finalReply("").stoppedFor(stopBlocked), finalReply(validReply)},
			wantErr:      ErrBlockedReply,
			wantRequests: 1,
		},
		{
			name:         "blocked turn with tool calls runs no tool",
			steps:        []modelStep{toolCallReply(&v1pb.AIChatToolCall{Id: "call-1", Name: "read"}).stoppedFor(stopBlocked), finalReply(validReply)},
			wantErr:      ErrBlockedReply,
			wantRequests: 1,
		},
		{
			name:             "tool call the vendor could not parse gets one correction round",
			steps:            []modelStep{finalReply("").stoppedFor(stopMalformedToolCall), finalReply(validReply)},
			wantFindings:     1,
			wantCalls:        2,
			wantLastUserText: "Call the tool again, or reply with only the JSON object",
		},
		{
			name: "model calls the tool again after the vendor could not parse the call",
			steps: []modelStep{
				finalReply("").stoppedFor(stopMalformedToolCall),
				toolCallReply(&v1pb.AIChatToolCall{Id: "call-1", Name: "read"}),
				finalReply(validReply),
			},
			wantFindings:  1,
			wantCalls:     3,
			wantToolCalls: []string{"read"},
		},
		{
			name:             "passing text beside an unparsed tool call gets a correction round, not a pass",
			steps:            []modelStep{finalReply(`{"findings": []}`).stoppedFor(stopMalformedToolCall), finalReply(validReply)},
			wantFindings:     1,
			wantCalls:        2,
			wantLastUserText: "Call the tool again, or reply with only the JSON object",
		},
		{
			name:          "tool calls beside an unparsed tool call do not run",
			steps:         []modelStep{toolCallReply(&v1pb.AIChatToolCall{Id: "call-1", Name: "read"}).stoppedFor(stopMalformedToolCall), finalReply(validReply)},
			wantFindings:  1,
			wantCalls:     2,
			wantToolCalls: nil,
		},
		{
			name:         "second tool call the vendor could not parse fails",
			steps:        []modelStep{finalReply("").stoppedFor(stopMalformedToolCall)},
			wantErr:      ErrInvalidReply,
			wantErrText:  "the vendor could not parse the model's tool call",
			wantRequests: 2,
		},
		{
			name:         "second unparsed tool call with passing text fails",
			steps:        []modelStep{finalReply(`{"findings": []}`).stoppedFor(stopMalformedToolCall)},
			wantErr:      ErrInvalidReply,
			wantErrText:  "the vendor could not parse the model's tool call",
			wantRequests: 2,
		},
		{
			name:         "unparsed tool call after an invalid reply has no correction round left",
			steps:        []modelStep{finalReply("not json"), finalReply("").stoppedFor(stopMalformedToolCall), finalReply(validReply)},
			wantErr:      ErrInvalidReply,
			wantErrText:  "the vendor could not parse the model's tool call",
			wantRequests: 2,
		},
		{
			name:         "unparsed tool call with passing text after an invalid reply has no correction round left",
			steps:        []modelStep{finalReply("not json"), finalReply(`{"findings": []}`).stoppedFor(stopMalformedToolCall), finalReply(validReply)},
			wantErr:      ErrInvalidReply,
			wantErrText:  "the vendor could not parse the model's tool call",
			wantRequests: 2,
		},
		{
			name:         "invalid reply after an unparsed tool call has no correction round left",
			steps:        []modelStep{finalReply("").stoppedFor(stopMalformedToolCall), finalReply("not json"), finalReply(validReply)},
			wantErr:      ErrInvalidReply,
			wantErrText:  "the reply is not a JSON object",
			wantRequests: 2,
		},
		{
			name:          "unparsed tool call on the last permitted call fails",
			steps:         []modelStep{finalReply(`{"findings": []}`).stoppedFor(stopMalformedToolCall)},
			maxModelCalls: 1,
			wantErr:       ErrInvalidReply,
			wantErrText:   "the vendor could not parse the model's tool call",
			wantRequests:  1,
		},
		{
			name:        "model error fails",
			steps:       []modelStep{{err: errors.New("Gemini API returned status 401")}},
			wantErrText: "model call 1 failed: Gemini API returned status 401",
		},
		{
			name:          "tool error fails instead of reaching the model as text",
			steps:         []modelStep{toolCallReply(&v1pb.AIChatToolCall{Id: "call-1", Name: "read"}), finalReply(`{"findings": []}`)},
			tools:         &fakeTools{err: toolFailure},
			wantErr:       toolFailure,
			wantToolCalls: []string{"read"},
		},
		{
			name:      "unknown tool goes back to the model as text",
			steps:     []modelStep{toolCallReply(&v1pb.AIChatToolCall{Id: "call-1", Name: "get_table"}), finalReply(`{"findings": []}`)},
			wantCalls: 2,
		},
		{
			// The tools of the last permitted call do not run: no call is left to carry their results.
			name:          "call limit",
			steps:         []modelStep{toolCallReply(&v1pb.AIChatToolCall{Id: "call-1", Name: "search"})},
			maxModelCalls: 3,
			wantErr:       ErrCallLimit,
			wantRequests:  3,
			wantToolCalls: []string{"search", "search"},
		},
		{
			name:          "invalid reply on the last permitted call reports the problems",
			steps:         []modelStep{finalReply("not json")},
			maxModelCalls: 1,
			wantErr:       ErrInvalidReply,
			wantErrText:   "the reply is not a JSON object",
			wantRequests:  1,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			model := &scriptedModel{steps: tc.steps}
			tools := tc.tools
			if tools == nil {
				tools = &fakeTools{}
			}
			reviewer := NewReviewer(model)
			if tc.maxModelCalls > 0 {
				reviewer.maxModelCalls = tc.maxModelCalls
			}

			result, err := reviewer.Review(context.Background(), &Request{Statement: threeLineStatement}, tools)
			require.Equal(t, tc.wantToolCalls, tools.calls)
			if tc.wantRequests > 0 {
				require.Len(t, model.requests, tc.wantRequests)
			}
			if tc.wantErr != nil || tc.wantErrText != "" {
				require.Nil(t, result)
				if tc.wantErr != nil {
					require.ErrorIs(t, err, tc.wantErr)
				}
				require.ErrorContains(t, err, tc.wantErrText)
				return
			}
			require.NoError(t, err)
			require.Len(t, result.Findings, tc.wantFindings)
			require.Equal(t, tc.wantCalls, result.Calls)
			if tc.wantLastUserText != "" {
				last := model.requests[len(model.requests)-1].Messages
				require.Equal(t, roleUser, last[len(last)-1].GetRole())
				require.Contains(t, last[len(last)-1].GetContent(), tc.wantLastUserText)
			}
		})
	}
}

func TestReviewStopReasonFailureOfAnEmptyReplyIsNotErrEmptyReply(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		reason  v1pb.AIChatResponse_StopReason
		wantErr error
	}{
		{name: "truncated", reason: stopTokenLimit, wantErr: ErrTruncatedReply},
		{name: "blocked", reason: stopBlocked, wantErr: ErrBlockedReply},
		{name: "another stop reason", reason: stopOther, wantErr: ErrUnfinishedReply},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			model := &scriptedModel{steps: []modelStep{finalReply("").stoppedFor(tc.reason)}}
			_, err := NewReviewer(model).Review(context.Background(), &Request{Statement: threeLineStatement}, &fakeTools{})
			require.ErrorIs(t, err, tc.wantErr)
			require.NotErrorIs(t, err, ErrEmptyReply)
		})
	}
}

func TestReviewKeepsAnUnparsedToolCallTurnOutOfTheHistory(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		step modelStep
	}{
		{name: "empty", step: finalReply("")},
		{name: "with text", step: finalReply(`{"findings": []}`)},
		{name: "with tool calls", step: toolCallReply(&v1pb.AIChatToolCall{Id: "call-1", Name: "read"})},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			model := &scriptedModel{steps: []modelStep{tc.step.stoppedFor(stopMalformedToolCall), finalReply(validReply)}}
			_, err := NewReviewer(model).Review(context.Background(), &Request{Statement: threeLineStatement}, &fakeTools{})
			require.NoError(t, err)
			history := model.requests[1].Messages
			require.Equal(t, []v1pb.AIChatMessageRole{roleSystem, roleUser, roleUser}, roles(history))
			require.Equal(t, malformedToolCallMessage, history[2].GetContent())
		})
	}
}

func TestReviewUnknownToolNamesTheRealTools(t *testing.T) {
	t.Parallel()

	model := &scriptedModel{steps: []modelStep{toolCallReply(&v1pb.AIChatToolCall{Id: "call-1", Name: "get_table"}), finalReply(`{"findings": []}`)}}
	_, err := NewReviewer(model).Review(context.Background(), &Request{Statement: threeLineStatement}, &fakeTools{})
	require.NoError(t, err)

	history := model.requests[1].Messages
	requireToolMessage(t, history[len(history)-1], "call-1", `unknown tool "get_table"; the tools are: read, search`)
}

func TestReviewRejectsEmptyStatement(t *testing.T) {
	t.Parallel()

	model := &scriptedModel{steps: []modelStep{finalReply(`{"findings": []}`)}}
	_, err := NewReviewer(model).Review(context.Background(), &Request{Statement: " \n"}, &fakeTools{})
	require.ErrorContains(t, err, "the statement is empty")
	require.Empty(t, model.requests)
}

func TestReviewDeadline(t *testing.T) {
	t.Parallel()

	reviewer := NewReviewer(&scriptedModel{steps: []modelStep{{block: true}}})
	reviewer.timeout = 20 * time.Millisecond
	_, err := reviewer.Review(context.Background(), &Request{Statement: threeLineStatement}, &fakeTools{})
	require.ErrorIs(t, err, ErrDeadline)
}

func TestReviewCanceledParentIsNotADeadline(t *testing.T) {
	t.Parallel()

	t.Run("before the first call", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		model := &scriptedModel{steps: []modelStep{{block: true}}}
		_, err := NewReviewer(model).Review(ctx, &Request{Statement: threeLineStatement}, &fakeTools{})
		require.ErrorIs(t, err, context.Canceled)
		require.NotErrorIs(t, err, ErrDeadline)
		require.Empty(t, model.requests)
	})

	t.Run("during a model call", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		model := &scriptedModel{steps: []modelStep{{block: true, cancelParent: cancel}}}
		_, err := NewReviewer(model).Review(ctx, &Request{Statement: threeLineStatement}, &fakeTools{})
		require.ErrorIs(t, err, context.Canceled)
		require.NotErrorIs(t, err, ErrDeadline)
		require.Len(t, model.requests, 1)
	})
}

func roles(messages []*v1pb.AIChatMessage) []v1pb.AIChatMessageRole {
	result := make([]v1pb.AIChatMessageRole, 0, len(messages))
	for _, message := range messages {
		result = append(result, message.GetRole())
	}
	return result
}

func requireToolMessage(t *testing.T, message *v1pb.AIChatMessage, toolCallID string, content string) {
	t.Helper()
	require.Equal(t, roleTool, message.GetRole())
	require.Equal(t, toolCallID, message.GetToolCallId())
	require.Equal(t, content, message.GetContent())
	require.Empty(t, message.GetToolCalls())
}

func TestReviewRunsWithoutToolsWhenTheEndpointRefusesThem(t *testing.T) {
	t.Parallel()

	refused := modelStep{err: &ai.HTTPError{Provider: "OpenAI", StatusCode: 400, RefusesTools: true}}
	blocked := modelStep{err: &ai.HTTPError{Provider: "OpenAI", StatusCode: 400, Body: `{"error": {"code": "content_filter"}}`}}
	unauthorized := modelStep{err: &ai.HTTPError{Provider: "OpenAI", StatusCode: 401}}
	search := toolCallReply(&v1pb.AIChatToolCall{Id: "call-1", Name: "search", Arguments: `{"text": "orders"}`})

	t.Run("the first call refuses tools", func(t *testing.T) {
		t.Parallel()
		model := &scriptedModel{steps: []modelStep{refused, finalReply(validReply)}}
		result, err := NewReviewer(model).Review(context.Background(), &Request{Statement: threeLineStatement}, &fakeTools{})
		require.NoError(t, err)
		require.Len(t, result.Findings, 1)
		require.Equal(t, 2, result.Calls)
		require.Len(t, model.requests[0].ToolDefinitions, 2)
		require.Empty(t, model.requests[1].ToolDefinitions)
		require.Contains(t, model.requests[1].Messages[1].GetContent(), "No tools are available in this review.")
	})
	t.Run("a refusal without tools is reported", func(t *testing.T) {
		t.Parallel()
		model := &scriptedModel{steps: []modelStep{refused}}
		_, err := NewReviewer(model).Review(context.Background(), &Request{Statement: threeLineStatement}, &fakeTools{})
		require.ErrorContains(t, err, "model call 2 failed")
		require.Len(t, model.requests, 2)
	})
	t.Run("a refusal after the tools worked is reported", func(t *testing.T) {
		t.Parallel()
		model := &scriptedModel{steps: []modelStep{search, refused}}
		_, err := NewReviewer(model).Review(context.Background(), &Request{Statement: threeLineStatement}, &fakeTools{})
		require.ErrorContains(t, err, "model call 2 failed")
		require.Len(t, model.requests, 2)
	})
	t.Run("a content filter is reported at once, never retried without tools", func(t *testing.T) {
		t.Parallel()
		model := &scriptedModel{steps: []modelStep{blocked, finalReply(`{"findings": []}`)}}
		_, err := NewReviewer(model).Review(context.Background(), &Request{Statement: threeLineStatement}, &fakeTools{})
		require.ErrorContains(t, err, "model call 1 failed")
		require.Len(t, model.requests, 1)
	})
	t.Run("another failure is reported at once", func(t *testing.T) {
		t.Parallel()
		model := &scriptedModel{steps: []modelStep{unauthorized}}
		_, err := NewReviewer(model).Review(context.Background(), &Request{Statement: threeLineStatement}, &fakeTools{})
		require.ErrorContains(t, err, "model call 1 failed")
		require.Len(t, model.requests, 1)
	})
}
