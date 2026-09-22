package aireview

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

const (
	roleSystem    = v1pb.AIChatMessageRole_AI_CHAT_MESSAGE_ROLE_SYSTEM
	roleUser      = v1pb.AIChatMessageRole_AI_CHAT_MESSAGE_ROLE_USER
	roleAssistant = v1pb.AIChatMessageRole_AI_CHAT_MESSAGE_ROLE_ASSISTANT
	roleTool      = v1pb.AIChatMessageRole_AI_CHAT_MESSAGE_ROLE_TOOL
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
	model := &scriptedModel{steps: []modelStep{toolCallReply(calls...), finalReply(`{"findings": []}`)}}
	tools := &fakeTools{outputs: map[string]string{"search": "orders_summary", "read": "CREATE TABLE orders"}}

	result, err := NewReviewer(model).Review(context.Background(), &Request{Statement: threeLineStatement}, tools)
	require.NoError(t, err)
	require.Empty(t, result.Findings)
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

func TestReviewKeepsAnEmptyReplyOutOfTheHistory(t *testing.T) {
	t.Parallel()

	model := &scriptedModel{steps: []modelStep{finalReply(" \n"), finalReply(validReply)}}
	result, err := NewReviewer(model).Review(context.Background(), &Request{Statement: threeLineStatement}, &fakeTools{})
	require.NoError(t, err)
	require.Len(t, result.Findings, 1)

	history := model.requests[1].Messages
	require.Equal(t, []v1pb.AIChatMessageRole{roleSystem, roleUser, roleUser}, roles(history))
	require.Contains(t, history[2].GetContent(), "the reply is empty")
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
