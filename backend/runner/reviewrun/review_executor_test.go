package reviewrun

import (
	"context"
	"slices"
	"sync/atomic"
	"testing"

	"github.com/bytebase/omni/review"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/semaphore"
	"google.golang.org/genproto/googleapis/rpc/code"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/ai"
	"github.com/bytebase/bytebase/backend/plugin/aireview"
	"github.com/bytebase/bytebase/backend/store"
)

// stubExecutor is one reviewer's part of a review.
type stubExecutor struct {
	calls   atomic.Int32
	results []*store.IssueCommentMessage
	err     error
	// onRun runs when the part starts.
	onRun func()
}

func (e *stubExecutor) RunOnce(context.Context, string, int64) ([]*store.IssueCommentMessage, error) {
	e.calls.Add(1)
	if e.onRun != nil {
		e.onRun()
	}
	return e.results, e.err
}

func resultAt(line int32, priority storepb.IssueCommentPayload_ReviewMetadata_Priority, runType storepb.ReviewRun_Type) *store.IssueCommentMessage {
	return &store.IssueCommentMessage{
		ProjectID: "p",
		Payload: &storepb.IssueCommentPayload{
			StatementAnchor: &storepb.IssueCommentPayload_StatementAnchor{
				SpecId:        "spec-1",
				StartPosition: &storepb.Position{Line: line},
				EndPosition:   &storepb.Position{Line: line},
			},
			ReviewMetadata: &storepb.IssueCommentPayload_ReviewMetadata{RunType: runType, Priority: priority},
		},
	}
}

func setAIReview(ctx context.Context, t *testing.T, stores *store.Store, on bool) {
	t.Helper()
	_, err := stores.UpsertSetting(ctx, &store.SettingMessage{
		Name:      storepb.SettingName_AI,
		Workspace: "ws",
		Value:     &storepb.AISetting{Enabled: true, SqlReviewEnabled: on},
	})
	require.NoError(t, err)
}

func TestReviewExecutorPutsBothReviewersResultsInReadingOrder(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	stores, issues := setupReviewFixture(ctx, t, 1)
	ruleResult := resultAt(3, storepb.IssueCommentPayload_ReviewMetadata_P1, storepb.ReviewRun_RULE)
	firstLine := resultAt(1, storepb.IssueCommentPayload_ReviewMetadata_P2, storepb.ReviewRun_AI)
	thirdLine := resultAt(3, storepb.IssueCommentPayload_ReviewMetadata_P0, storepb.ReviewRun_AI)
	rule := &stubExecutor{results: []*store.IssueCommentMessage{ruleResult}}
	aiReview := &stubExecutor{results: []*store.IssueCommentMessage{thirdLine, firstLine}}
	aiReview.onRun = func() {
		require.Equal(t, int32(1), rule.calls.Load(), "the rules run before the AI review")
	}

	results, err := NewReviewExecutor(stores, rule, aiReview).RunOnce(ctx, "p", issues[0])
	require.NoError(t, err)
	require.Equal(t, []*store.IssueCommentMessage{firstLine, thirdLine, ruleResult}, results, "by line, then by priority")
	require.Equal(t, int32(1), aiReview.calls.Load())
}

func TestReviewExecutorPostsNothingWhenAReviewerFails(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	stores, issues := setupReviewFixture(ctx, t, 1)
	ruleResults := []*store.IssueCommentMessage{resultAt(1, storepb.IssueCommentPayload_ReviewMetadata_P1, storepb.ReviewRun_RULE)}

	rule := &stubExecutor{err: errors.New("the rules failed")}
	aiReview := &stubExecutor{}
	results, err := NewReviewExecutor(stores, rule, aiReview).RunOnce(ctx, "p", issues[0])
	require.ErrorContains(t, err, "the rules failed")
	require.Nil(t, results)
	require.Zero(t, aiReview.calls.Load(), "a failure of the rules costs no model call")

	rule = &stubExecutor{results: ruleResults}
	aiReview = &stubExecutor{err: errors.New("the model failed")}
	results, err = NewReviewExecutor(stores, rule, aiReview).RunOnce(ctx, "p", issues[0])
	require.ErrorContains(t, err, "the model failed")
	require.Nil(t, results, "the rules' results do not post without the AI review's")
}

func TestReviewExecutorSkipsAnAIReviewThatIsOff(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	stores, issues := setupReviewFixture(ctx, t, 1)
	ruleResults := []*store.IssueCommentMessage{resultAt(1, storepb.IssueCommentPayload_ReviewMetadata_P1, storepb.ReviewRun_RULE)}
	rule := &stubExecutor{results: ruleResults}

	setAIReview(ctx, t, stores, false)
	aiReview := &stubExecutor{}
	results, err := NewReviewExecutor(stores, rule, aiReview).RunOnce(ctx, "p", issues[0])
	require.NoError(t, err)
	require.Equal(t, ruleResults, results)
	require.Zero(t, aiReview.calls.Load())

	// Switched off while it runs, the AI review fails at its next model call.
	setAIReview(ctx, t, stores, true)
	aiReview = &stubExecutor{err: errors.New("AI SQL review is not enabled")}
	aiReview.onRun = func() { setAIReview(ctx, t, stores, false) }
	results, err = NewReviewExecutor(stores, rule, aiReview).RunOnce(ctx, "p", issues[0])
	require.NoError(t, err, "a review switched off in flight is skipped like one that was off from the start")
	require.Equal(t, ruleResults, results)
	require.Equal(t, int32(1), aiReview.calls.Load())
}

func TestReviewExecutorWithoutTheStandardRules(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	stores, issues := setupReviewFixture(ctx, t, 1)
	aiResults := []*store.IssueCommentMessage{resultAt(1, storepb.IssueCommentPayload_ReviewMetadata_P1, storepb.ReviewRun_AI)}

	results, err := NewReviewExecutor(stores, nil, &stubExecutor{results: aiResults}).RunOnce(ctx, "p", issues[0])
	require.NoError(t, err)
	require.Equal(t, aiResults, results)
}

// TestReviewRunKeepsBothReviewersThreadsAcrossRuns drives the real reviewers
// through one job and its completion: a run posts the results of both, a
// re-run keeps both threads, and a run with the AI review off supersedes the
// AI review's result.
func TestReviewRunKeepsBothReviewersThreadsAcrossRuns(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	stores, issues := setupReviewFixture(ctx, t, 1)
	model := &findingModel{}
	executor := NewReviewExecutor(stores,
		&RuleExecutor{store: stores, reviewFuncs: map[storepb.Engine]review.ReviewFunc{
			storepb.Engine_POSTGRES: func(_ context.Context, sql string, _ review.Options, targets []review.Target) (*review.Result, error) {
				every := make([]int, len(targets))
				for i := range targets {
					every[i] = i
				}
				whole := review.Range{Start: 0, End: len(sql)}
				return &review.Result{
					Statements: []review.Range{whole},
					Bodies:     []review.Range{whole},
					Findings:   []review.Finding{{Rule: review.RequireWhere, Statement: 0, Message: "no WHERE", Targets: every}},
				}, nil
			},
		}},
		&AIReviewExecutor{
			store:    stores,
			newModel: func(*storepb.AISetting, ...aireview.ModelOption) aireview.Model { return model },
			slots:    semaphore.NewWeighted(aiReviewServerConcurrency),
		},
	)
	run := func() {
		t.Helper()
		results, err := executor.RunOnce(ctx, "p", issues[0])
		require.NoError(t, err)
		_, err = stores.CreateReviewRun(ctx, "p", issues[0])
		require.NoError(t, err)
		claimed, err := stores.ClaimAvailableReviewRuns(ctx, "replica-1")
		require.NoError(t, err)
		require.Len(t, claimed, 1)
		updated, err := stores.CompleteReviewRun(ctx, claimed[0], "replica-1", storepb.ReviewRun_DONE, nil, results)
		require.NoError(t, err)
		require.True(t, updated)
	}
	// The review results on the issue, by the reviewer that found each.
	roots := func() map[storepb.ReviewRun_Type]*store.IssueCommentMessage {
		t.Helper()
		comments, err := stores.ListIssueComment(ctx, &store.FindIssueCommentMessage{ProjectID: "p", IssueUID: &issues[0], TopLevelOnly: true})
		require.NoError(t, err)
		comments = slices.DeleteFunc(comments, func(c *store.IssueCommentMessage) bool { return c.Payload.GetReviewMetadata() == nil })
		require.Len(t, comments, 2, "one result of each reviewer, never posted twice")
		byReviewer := make(map[storepb.ReviewRun_Type]*store.IssueCommentMessage)
		for _, c := range comments {
			byReviewer[c.Payload.ReviewMetadata.RunType] = c
		}
		require.Len(t, byReviewer, 2)
		return byReviewer
	}

	run()
	first := roots()
	require.Equal(t, storepb.ReviewRuleType_REQUIRE_WHERE, first[storepb.ReviewRun_RULE].Payload.ReviewMetadata.RuleType)
	require.Equal(t, "Backfill note", first[storepb.ReviewRun_AI].Payload.ReviewMetadata.Title)
	resolved := store.ThreadStateResolved
	require.NoError(t, stores.UpdateIssueComment(ctx, &store.UpdateIssueCommentMessage{ProjectID: "p", ResourceID: first[storepb.ReviewRun_AI].ResourceID, ThreadState: &resolved}))

	run()
	second := roots()
	for reviewer, root := range second {
		require.Equal(t, first[reviewer].ResourceID, root.ResourceID, "%s result keeps its thread", reviewer)
		require.False(t, root.Payload.ReviewMetadata.Superseded)
	}
	require.Equal(t, store.ThreadStateOpen, *second[storepb.ReviewRun_RULE].ThreadState)
	require.Equal(t, store.ThreadStateResolved, *second[storepb.ReviewRun_AI].ThreadState, "the person's resolution stays")

	setAIReview(ctx, t, stores, false)
	run()
	third := roots()
	require.False(t, third[storepb.ReviewRun_RULE].Payload.ReviewMetadata.Superseded)
	require.True(t, third[storepb.ReviewRun_AI].Payload.ReviewMetadata.Superseded, "a run without the AI review no longer reports its result")
}

func TestErrorCode(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		description string
		err         error
		want        code.Code
	}{
		{"the provider limits the rate", errors.Wrap(&ai.HTTPError{StatusCode: 429}, "model call 4 failed"), code.Code_RESOURCE_EXHAUSTED},
		{"the provider refuses the key", &ai.HTTPError{StatusCode: 401}, code.Code_FAILED_PRECONDITION},
		{"the provider is down", &ai.HTTPError{StatusCode: 503}, code.Code_UNAVAILABLE},
		{"the provider rejects the request", &ai.HTTPError{StatusCode: 400}, code.Code_UNKNOWN},
		{"the provider cannot be reached", &ai.TransportError{}, code.Code_UNAVAILABLE},
		{"the review ran out of time", errors.Wrap(aireview.ErrDeadline, "10m0s, 3 model calls"), code.Code_DEADLINE_EXCEEDED},
		{"a schema is not synced", errors.Wrap(errMetadataNotSynced, "instances/prod/databases/app"), code.Code_FAILED_PRECONDITION},
		{"a failure of another kind", errors.New("boom"), code.Code_UNKNOWN},
		{
			"one unit of several",
			aggregateUnitErrors(3, []error{
				errors.New("boom"),
				errors.Wrap(errors.Wrap(&ai.HTTPError{StatusCode: 429}, "model call 1 failed"), "instances/prod/databases/app"),
			}),
			code.Code_RESOURCE_EXHAUSTED,
		},
	} {
		require.Equal(t, tc.want, errorCode(tc.err), tc.description)
	}
}
