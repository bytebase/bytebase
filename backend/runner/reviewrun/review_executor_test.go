package reviewrun

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"sync/atomic"
	"testing"

	"github.com/bytebase/omni/review"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/semaphore"
	"google.golang.org/genproto/googleapis/rpc/code"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/ai"
	"github.com/bytebase/bytebase/backend/plugin/aireview"
	"github.com/bytebase/bytebase/backend/store"
)

// stubRules is the standard rules' part of a review.
type stubRules struct {
	calls   atomic.Int32
	results []*ruleResult
	err     error
}

func (e *stubRules) review(context.Context, string, int64) ([]*ruleResult, error) {
	e.calls.Add(1)
	return e.results, e.err
}

// stubAIReview is the AI review's part of a review. It adds nothing to the
// rules' results, and returns them before its own.
type stubAIReview struct {
	calls   atomic.Int32
	results []*store.IssueCommentMessage
	err     error
	// onRun runs when the part starts.
	onRun func()
}

func (e *stubAIReview) reviewWith(_ context.Context, projectID string, issueUID int64, rules []*ruleResult) ([]*store.IssueCommentMessage, error) {
	e.calls.Add(1)
	if e.onRun != nil {
		e.onRun()
	}
	if e.err != nil {
		return nil, e.err
	}
	return append(ruleResultComments(projectID, issueUID, rules), e.results...), nil
}

// ruleResultAt is a result of the standard rules on a line, which posts with
// the rule's priority.
func ruleResultAt(line int32, rule storepb.ReviewRuleType) *ruleResult {
	return &ruleResult{specID: "spec-1", members: []*ruleFinding{{
		rule:    rule,
		message: "rule finding",
		anchor: &storepb.IssueCommentPayload_StatementAnchor{
			SpecId:        "spec-1",
			StartPosition: &storepb.Position{Line: line},
			EndPosition:   &storepb.Position{Line: line},
		},
	}}}
}

// aiResultAt is a result of the AI review on a line.
func aiResultAt(line int32, priority storepb.IssueCommentPayload_ReviewMetadata_Priority) *store.IssueCommentMessage {
	return &store.IssueCommentMessage{
		ProjectID: "p",
		Payload: &storepb.IssueCommentPayload{
			StatementAnchor: &storepb.IssueCommentPayload_StatementAnchor{
				SpecId:        "spec-1",
				StartPosition: &storepb.Position{Line: line},
				EndPosition:   &storepb.Position{Line: line},
			},
			ReviewMetadata: &storepb.IssueCommentPayload_ReviewMetadata{Priority: priority},
		},
	}
}

// reviewerOf names the reviewer that posted a result: a result of a standard
// rule names its rule, and one of the AI review names none.
func reviewerOf(result *store.IssueCommentMessage) string {
	if result.Payload.GetReviewMetadata().GetRuleType() != storepb.ReviewRuleType_REVIEW_RULE_TYPE_UNSPECIFIED {
		return "RULE"
	}
	return "AI"
}

// places lists results as "line priority reviewer", in order.
func places(results []*store.IssueCommentMessage) []string {
	var out []string
	for _, result := range results {
		metadata := result.Payload.GetReviewMetadata()
		out = append(out, fmt.Sprintf("%d %s %s", result.Payload.GetStatementAnchor().GetStartPosition().GetLine(), metadata.GetPriority(), reviewerOf(result)))
	}
	return out
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
	rule := &stubRules{results: []*ruleResult{ruleResultAt(3, storepb.ReviewRuleType_REQUIRE_WHERE)}}
	aiReview := &stubAIReview{results: []*store.IssueCommentMessage{
		aiResultAt(3, storepb.IssueCommentPayload_ReviewMetadata_P0),
		aiResultAt(1, storepb.IssueCommentPayload_ReviewMetadata_P2),
	}}
	aiReview.onRun = func() {
		require.Equal(t, int32(1), rule.calls.Load(), "the rules run before the AI review")
	}

	results, err := (&ReviewExecutor{store: stores, license: licenseStub{}, rule: rule, ai: aiReview}).RunOnce(ctx, "p", issues[0])
	require.NoError(t, err)
	require.Equal(t, []string{"1 P2 AI", "3 P0 AI", "3 P1 RULE"}, places(results), "by line, then by priority")
	require.Equal(t, int32(1), aiReview.calls.Load())
}

func TestReviewExecutorPostsNothingWhenAReviewerFails(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	stores, issues := setupReviewFixture(ctx, t, 1)

	rule := &stubRules{err: errors.New("the rules failed")}
	aiReview := &stubAIReview{}
	results, err := (&ReviewExecutor{store: stores, license: licenseStub{}, rule: rule, ai: aiReview}).RunOnce(ctx, "p", issues[0])
	require.ErrorContains(t, err, "the rules failed")
	require.Nil(t, results)
	require.Zero(t, aiReview.calls.Load(), "a failure of the rules costs no model call")

	rule = &stubRules{results: []*ruleResult{ruleResultAt(1, storepb.ReviewRuleType_REQUIRE_WHERE)}}
	aiReview = &stubAIReview{err: errors.New("the model failed")}
	results, err = (&ReviewExecutor{store: stores, license: licenseStub{}, rule: rule, ai: aiReview}).RunOnce(ctx, "p", issues[0])
	require.ErrorContains(t, err, "the model failed")
	require.Nil(t, results, "the rules' results do not post without the AI review's")
}

func TestReviewExecutorSkipsAnAIReviewThatIsOff(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	stores, issues := setupReviewFixture(ctx, t, 1)
	rule := &stubRules{results: []*ruleResult{ruleResultAt(1, storepb.ReviewRuleType_REQUIRE_WHERE)}}

	setAIReview(ctx, t, stores, false)
	aiReview := &stubAIReview{}
	results, err := (&ReviewExecutor{store: stores, license: licenseStub{}, rule: rule, ai: aiReview}).RunOnce(ctx, "p", issues[0])
	require.NoError(t, err)
	require.Equal(t, []string{"1 P1 RULE"}, places(results))
	require.Zero(t, aiReview.calls.Load())

	// Switched off while it runs, the AI review fails at its next model call.
	setAIReview(ctx, t, stores, true)
	aiReview = &stubAIReview{err: errors.New("AI SQL review is not enabled")}
	aiReview.onRun = func() { setAIReview(ctx, t, stores, false) }
	results, err = (&ReviewExecutor{store: stores, license: licenseStub{}, rule: rule, ai: aiReview}).RunOnce(ctx, "p", issues[0])
	require.NoError(t, err, "a review switched off in flight is skipped like one that was off from the start")
	require.Equal(t, []string{"1 P1 RULE"}, places(results))
	require.Equal(t, int32(1), aiReview.calls.Load())
}

// licenseStub is a workspace plan: err is why it lacks every feature, and nil
// gives it all of them.
type licenseStub struct{ err error }

func (l licenseStub) IsFeatureEnabled(context.Context, string, v1pb.PlanFeature) error {
	return l.err
}

var withoutEnterprise = licenseStub{err: errors.New("feature FEATURE_AI_SQL_REVIEW is a ENTERPRISE feature, please upgrade to access it")}

func TestReviewExecutorSkipsTheAIReviewWithoutThePlan(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	stores, issues := setupReviewFixture(ctx, t, 1)
	rule := &stubRules{results: []*ruleResult{ruleResultAt(1, storepb.ReviewRuleType_SYNTAX)}}
	aiReview := &stubAIReview{}

	results, err := (&ReviewExecutor{store: stores, license: withoutEnterprise, rule: rule, ai: aiReview}).RunOnce(ctx, "p", issues[0])
	require.NoError(t, err)
	require.Equal(t, []string{"1 P0 RULE"}, places(results))
	require.Zero(t, aiReview.calls.Load())
}

func TestReviewExecutorWithoutTheStandardRules(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	stores, issues := setupReviewFixture(ctx, t, 1)
	aiResults := []*store.IssueCommentMessage{aiResultAt(1, storepb.IssueCommentPayload_ReviewMetadata_P1)}

	results, err := (&ReviewExecutor{store: stores, license: licenseStub{}, ai: &stubAIReview{results: aiResults}}).RunOnce(ctx, "p", issues[0])
	require.NoError(t, err)
	require.Equal(t, aiResults, results)
}

// TestReviewRunKeepsBothReviewersThreadsAcrossRuns drives the real reviewers
// through one job and its completion: a run posts the results of both, a
// re-run keeps both threads, and a run with the AI review off supersedes the
// AI review's result. It does so for rows as they are written now, and for
// rows that still hold run_type, which results carried until it was removed.
func TestReviewRunKeepsBothReviewersThreadsAcrossRuns(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name string
		// withRunType writes the removed key back into the stored results.
		withRunType bool
	}{
		{name: "rows as they are written now"},
		{name: "rows that still hold the removed run_type", withRunType: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			stores, issues := setupReviewFixture(ctx, t, 1)
			model := &findingModel{}
			executor := NewReviewExecutor(stores, licenseStub{},
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
					license:  licenseStub{},
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
			roots := func() map[string]*store.IssueCommentMessage {
				t.Helper()
				comments, err := stores.ListIssueComment(ctx, &store.FindIssueCommentMessage{ProjectID: "p", IssueUID: &issues[0], TopLevelOnly: true})
				require.NoError(t, err)
				comments = slices.DeleteFunc(comments, func(c *store.IssueCommentMessage) bool { return c.Payload.GetReviewMetadata() == nil })
				require.Len(t, comments, 2, "one result of each reviewer, never posted twice")
				byReviewer := make(map[string]*store.IssueCommentMessage)
				for _, c := range comments {
					byReviewer[reviewerOf(c)] = c
				}
				require.Len(t, byReviewer, 2)
				return byReviewer
			}
			// The number of stored results that hold the removed key.
			holdRunType := func() int {
				t.Helper()
				var n int
				require.NoError(t, stores.GetDB().QueryRowContext(ctx, `
					SELECT count(*) FROM issue_comment
					WHERE project = 'p' AND issue_id = $1 AND payload->'reviewMetadata' ? 'runType'`, issues[0]).Scan(&n))
				return n
			}

			run()
			first := roots()
			require.Equal(t, storepb.ReviewRuleType_REQUIRE_WHERE, first["RULE"].Payload.ReviewMetadata.RuleType)
			require.Equal(t, "Backfill note", first["AI"].Payload.ReviewMetadata.Title)
			require.Zero(t, holdRunType(), "a result is no longer written with the key")
			resolved := store.ThreadStateResolved
			require.NoError(t, stores.UpdateIssueComment(ctx, &store.UpdateIssueCommentMessage{ProjectID: "p", ResourceID: first["AI"].ResourceID, ThreadState: &resolved}))
			if test.withRunType {
				_, err := stores.GetDB().ExecContext(ctx, `
					UPDATE issue_comment
					SET payload = jsonb_set(payload, '{reviewMetadata,runType}',
						to_jsonb(CASE WHEN payload->'reviewMetadata' ? 'ruleType' THEN 'RULE' ELSE 'AI' END))
					WHERE project = 'p' AND issue_id = $1 AND payload ? 'reviewMetadata'`, issues[0])
				require.NoError(t, err)
				require.Equal(t, 2, holdRunType())
			}

			run()
			second := roots()
			for reviewer, root := range second {
				require.Equal(t, first[reviewer].ResourceID, root.ResourceID, "%s result keeps its thread", reviewer)
				require.False(t, root.Payload.ReviewMetadata.Superseded)
			}
			require.Equal(t, store.ThreadStateOpen, *second["RULE"].ThreadState)
			require.Equal(t, store.ThreadStateResolved, *second["AI"].ThreadState, "the person's resolution stays")

			setAIReview(ctx, t, stores, false)
			run()
			third := roots()
			require.False(t, third["RULE"].Payload.ReviewMetadata.Superseded)
			require.True(t, third["AI"].Payload.ReviewMetadata.Superseded, "a run without the AI review no longer reports its result")
		})
	}
}

// TestReviewPutsTheAIReviewProblemInTheRulesThread drives the real reviewers
// through runs whose merge puts the AI review's problem on the rule finding's
// operation: one thread posts with both texts, an unchanged re-run keeps the
// comment, a higher priority rewrites it and reopens the thread, and a merge
// that keeps the two apart, or a run with the AI review off, leaves the rule's
// own text.
func TestReviewPutsTheAIReviewProblemInTheRulesThread(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	stores, issues := setupReviewFixture(ctx, t, 1)
	model := &findingModel{}
	model.onOperation.Store(true)
	executor := NewReviewExecutor(stores, licenseStub{},
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
			license:  licenseStub{},
			store:    stores,
			newModel: func(*storepb.AISetting, ...aireview.ModelOption) aireview.Model { return model },
			slots:    semaphore.NewWeighted(aiReviewServerConcurrency),
		},
	)
	// run completes a review run and returns the current review results on
	// the issue, the rules' first.
	run := func() []*store.IssueCommentMessage {
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
		comments, err := stores.ListIssueComment(ctx, &store.FindIssueCommentMessage{ProjectID: "p", IssueUID: &issues[0], TopLevelOnly: true})
		require.NoError(t, err)
		comments = slices.DeleteFunc(comments, func(c *store.IssueCommentMessage) bool {
			return c.Payload.GetReviewMetadata() == nil || c.Payload.GetReviewMetadata().GetSuperseded()
		})
		// The rule's result first.
		slices.SortStableFunc(comments, func(a, b *store.IssueCommentMessage) int {
			return cmp.Compare(reviewerOf(b), reviewerOf(a))
		})
		return comments
	}
	const both = "no WHERE\n\ne\n\nRule: r\n\nSuggested fix: f"
	p0, p1 := storepb.IssueCommentPayload_ReviewMetadata_P0, storepb.IssueCommentPayload_ReviewMetadata_P1

	first := run()
	require.Len(t, first, 1, "one thread for the operation")
	thread := first[0].ResourceID
	require.Equal(t, both, first[0].Payload.Comment)
	require.Equal(t, storepb.ReviewRuleType_REQUIRE_WHERE, first[0].Payload.ReviewMetadata.RuleType)
	require.Equal(t, p1, first[0].Payload.ReviewMetadata.Priority, "a P2 problem leaves the rule's priority")
	require.Len(t, first[0].Payload.ReviewMetadata.Targets, fixtureDatabases)

	model.same.Store(true)
	summaries := model.summaries.Load()
	second := run()
	require.Len(t, second, 1)
	require.Equal(t, thread, second[0].ResourceID)
	require.Equal(t, both, second[0].Payload.Comment)
	require.Equal(t, summaries, model.summaries.Load(), "a comment that stays is not written again")

	resolved := store.ThreadStateResolved
	require.NoError(t, stores.UpdateIssueComment(ctx, &store.UpdateIssueCommentMessage{ProjectID: "p", ResourceID: thread, ThreadState: &resolved}))
	model.severity = "P0"
	third := run()
	require.Len(t, third, 1)
	require.Equal(t, thread, third[0].ResourceID)
	require.Equal(t, p0, third[0].Payload.ReviewMetadata.Priority, "the problem's priority, since it is the higher one")
	require.Equal(t, store.ThreadStateOpen, *third[0].ThreadState, "nobody accepted the thread at P0")
	require.Equal(t, summaries+1, model.summaries.Load(), "the comment was read at P1, so it is written again")

	model.onOperation.Store(false)
	fourth := run()
	require.Len(t, fourth, 2, "kept apart by the merge, each reviewer posts its own")
	require.Equal(t, thread, fourth[0].ResourceID)
	require.Equal(t, "no WHERE", fourth[0].Payload.Comment)
	require.Equal(t, p1, fourth[0].Payload.ReviewMetadata.Priority)
	require.Equal(t, "AI", reviewerOf(fourth[1]))

	model.onOperation.Store(true)
	fifth := run()
	require.Len(t, fifth, 1, "the AI review's own thread is superseded once its problem joins the rule's")
	require.Equal(t, thread, fifth[0].ResourceID)

	setAIReview(ctx, t, stores, false)
	sixth := run()
	require.Len(t, sixth, 1)
	require.Equal(t, thread, sixth[0].ResourceID)
	require.Equal(t, "no WHERE", sixth[0].Payload.Comment, "a run without the AI review posts the rule's text alone")
	require.Equal(t, p1, sixth[0].Payload.ReviewMetadata.Priority)
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
