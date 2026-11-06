package store_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

const (
	reviewResultSheet = "0be1f01d6ee8e6f6c6a2ce9b418ba10ea9d16c9b9bfae5548b8fa0e26c04a5e0"
	// reviewResultIssue is the issue id both projects hold, so the keys collide.
	reviewResultIssue = int64(301)
)

func reviewResult(projectID string, message string, rule storepb.ReviewRuleType) *store.IssueCommentMessage {
	return &store.IssueCommentMessage{
		ProjectID: projectID,
		IssueUID:  reviewResultIssue,
		Payload: &storepb.IssueCommentPayload{
			Comment: message,
			StatementAnchor: &storepb.IssueCommentPayload_StatementAnchor{
				SpecId:        "spec-1",
				SheetSha256:   reviewResultSheet,
				StartPosition: &storepb.Position{Line: 1},
				EndPosition:   &storepb.Position{Line: 1},
			},
			ReviewMetadata: &storepb.IssueCommentPayload_ReviewMetadata{
				RunType:  storepb.ReviewRun_RULE,
				RuleType: rule,
				Priority: storepb.IssueCommentPayload_ReviewMetadata_P1,
				Targets:  []string{"instances/i/databases/db"},
			},
		},
	}
}

type reviewComment struct {
	Comment     string
	ThreadState string
	Creator     *string
}

func listReviewComments(t *testing.T, fixture *storePostgresFixture, projectID string) []reviewComment {
	t.Helper()
	rows, err := fixture.db.QueryContext(fixture.ctx, `
		SELECT payload->>'comment', thread_state, creator
		FROM issue_comment
		WHERE project = $1 AND issue_id = $2 AND payload ? 'reviewMetadata'
		ORDER BY created_at, resource_id
	`, projectID, reviewResultIssue)
	require.NoError(t, err)
	defer rows.Close()
	var out []reviewComment
	for rows.Next() {
		var c reviewComment
		require.NoError(t, rows.Scan(&c.Comment, &c.ThreadState, &c.Creator))
		out = append(out, c)
	}
	require.NoError(t, rows.Err())
	return out
}

// TestCompleteReviewRunPostsResults completes runs on two projects whose issue
// ids collide: a DONE completion posts its results as creatorless OPEN roots
// and resolves only the same reviewer's earlier results on its own issue; a
// superseded or FAILED completion writes nothing.
func TestCompleteReviewRunPostsResults(t *testing.T) {
	t.Parallel()
	fixture := newStorePostgresFixture(t, `
		INSERT INTO project (resource_id, workspace, name) VALUES ('other', 'default', 'Other');
		INSERT INTO issue (id, creator, project, name, status, type)
		VALUES (301, 'admin@example.com', 'default', 'Issue Default', 'OPEN', 'DATABASE_CHANGE');
		INSERT INTO issue (id, creator, project, name, status, type)
		VALUES (301, 'admin@example.com', 'other', 'Issue Other', 'OPEN', 'DATABASE_CHANGE');
	`)
	ctx, s := fixture.ctx, fixture.store
	const issueUID = reviewResultIssue

	// A person's anchored thread and an AI review result on the same issue
	// must survive every rule completion below.
	open := store.ThreadStateOpen
	_, err := s.CreateIssueComments(ctx, "dev@example.com", &store.IssueCommentMessage{
		ProjectID: "default", IssueUID: issueUID,
		Payload: &storepb.IssueCommentPayload{
			Comment:         "person",
			StatementAnchor: reviewResult("default", "", storepb.ReviewRuleType_SYNTAX).Payload.StatementAnchor,
		},
	})
	require.NoError(t, err)
	aiReview := reviewResult("default", "ai", storepb.ReviewRuleType_REVIEW_RULE_TYPE_UNSPECIFIED)
	aiReview.Payload.ReviewMetadata.RunType = storepb.ReviewRun_AI
	aiReview.ThreadState = &open

	// Colliding rule runs on both projects.
	for _, projectID := range []string{"default", "other"} {
		_, err := s.CreateReviewRun(ctx, projectID, issueUID, store.ReviewRunTypeRule)
		require.NoError(t, err)
	}
	_, err = s.CreateReviewRun(ctx, "default", issueUID, store.ReviewRunTypeAI)
	require.NoError(t, err)
	claimed, err := s.ClaimAvailableReviewRuns(ctx, "replica-1")
	require.NoError(t, err)
	require.Len(t, claimed, 3)
	byKey := make(map[[2]string]*store.ClaimedReviewRun)
	for _, c := range claimed {
		byKey[[2]string{c.ProjectID, c.Type}] = c
	}
	ruleDefault := byKey[[2]string{"default", store.ReviewRunTypeRule}]
	ruleOther := byKey[[2]string{"other", store.ReviewRunTypeRule}]
	aiReviewDefault := byKey[[2]string{"default", store.ReviewRunTypeAI}]

	updated, err := s.CompleteReviewRun(ctx, aiReviewDefault, "replica-1", storepb.ReviewRun_DONE, nil, []*store.IssueCommentMessage{aiReview})
	require.NoError(t, err)
	require.True(t, updated)

	// Results must belong to the claimed run and name its reviewer.
	_, err = s.CompleteReviewRun(ctx, ruleDefault, "replica-1", storepb.ReviewRun_DONE, nil,
		[]*store.IssueCommentMessage{reviewResult("other", "wrong project", storepb.ReviewRuleType_SYNTAX)})
	require.Error(t, err)
	wrongReviewer := reviewResult("default", "wrong reviewer", storepb.ReviewRuleType_SYNTAX)
	wrongReviewer.Payload.ReviewMetadata.RunType = storepb.ReviewRun_AI
	_, err = s.CompleteReviewRun(ctx, ruleDefault, "replica-1", storepb.ReviewRun_DONE, nil, []*store.IssueCommentMessage{wrongReviewer})
	require.Error(t, err)
	_, err = s.CompleteReviewRun(ctx, ruleDefault, "replica-1", storepb.ReviewRun_FAILED, nil,
		[]*store.IssueCommentMessage{reviewResult("default", "failed with results", storepb.ReviewRuleType_SYNTAX)})
	require.Error(t, err)
	require.Len(t, listReviewComments(t, fixture, "default"), 1, "a rejected completion writes nothing")

	// The first rule completion on each project posts its results.
	updated, err = s.CompleteReviewRun(ctx, ruleDefault, "replica-1", storepb.ReviewRun_DONE, nil, []*store.IssueCommentMessage{
		reviewResult("default", "default first", storepb.ReviewRuleType_REQUIRE_WHERE),
		reviewResult("default", "default second", storepb.ReviewRuleType_DISALLOW_TRUNCATE),
	})
	require.NoError(t, err)
	require.True(t, updated)
	updated, err = s.CompleteReviewRun(ctx, ruleOther, "replica-1", storepb.ReviewRun_DONE, nil, []*store.IssueCommentMessage{
		reviewResult("other", "other first", storepb.ReviewRuleType_REQUIRE_WHERE),
	})
	require.NoError(t, err)
	require.True(t, updated)

	got := listReviewComments(t, fixture, "default")
	require.Equal(t, []reviewComment{
		{Comment: "ai", ThreadState: "OPEN"},
		{Comment: "default first", ThreadState: "OPEN"},
		{Comment: "default second", ThreadState: "OPEN"},
	}, got, "results post in order as OPEN roots with no creator")
	otherBefore := listReviewComments(t, fixture, "other")
	require.Equal(t, []reviewComment{{Comment: "other first", ThreadState: "OPEN"}}, otherBefore)

	// A re-run supersedes the RUNNING execution: its completion posts nothing
	// and resolves nothing.
	_, err = s.CreateReviewRun(ctx, "default", issueUID, store.ReviewRunTypeRule)
	require.NoError(t, err)
	claimed, err = s.ClaimAvailableReviewRuns(ctx, "replica-1")
	require.NoError(t, err)
	require.Len(t, claimed, 1)
	current := claimed[0]
	_, err = s.CreateReviewRun(ctx, "default", issueUID, store.ReviewRunTypeRule)
	require.NoError(t, err)
	updated, err = s.CompleteReviewRun(ctx, current, "replica-1", storepb.ReviewRun_DONE, nil, []*store.IssueCommentMessage{
		reviewResult("default", "superseded", storepb.ReviewRuleType_SYNTAX),
	})
	require.NoError(t, err)
	require.False(t, updated)
	require.Equal(t, got, listReviewComments(t, fixture, "default"), "a superseded completion writes nothing")

	// A FAILED completion keeps the earlier results open.
	claimed, err = s.ClaimAvailableReviewRuns(ctx, "replica-1")
	require.NoError(t, err)
	require.Len(t, claimed, 1)
	updated, err = s.CompleteReviewRun(ctx, claimed[0], "replica-1", storepb.ReviewRun_FAILED, &storepb.ReviewRunPayload{Error: "boom"}, nil)
	require.NoError(t, err)
	require.True(t, updated)
	require.Equal(t, got, listReviewComments(t, fixture, "default"), "a failed run leaves the comments alone")

	// The next DONE resolves the same reviewer's earlier results on this
	// issue, and only those: the AI review result, the person's thread, and
	// the colliding issue in the other project stay open.
	_, err = s.CreateReviewRun(ctx, "default", issueUID, store.ReviewRunTypeRule)
	require.NoError(t, err)
	claimed, err = s.ClaimAvailableReviewRuns(ctx, "replica-1")
	require.NoError(t, err)
	require.Len(t, claimed, 1)
	updated, err = s.CompleteReviewRun(ctx, claimed[0], "replica-1", storepb.ReviewRun_DONE, nil, []*store.IssueCommentMessage{
		reviewResult("default", "default third", storepb.ReviewRuleType_REQUIRE_WHERE),
	})
	require.NoError(t, err)
	require.True(t, updated)
	require.Equal(t, []reviewComment{
		{Comment: "ai", ThreadState: "OPEN"},
		{Comment: "default first", ThreadState: "RESOLVED"},
		{Comment: "default second", ThreadState: "RESOLVED"},
		{Comment: "default third", ThreadState: "OPEN"},
	}, listReviewComments(t, fixture, "default"))
	require.Equal(t, otherBefore, listReviewComments(t, fixture, "other"), "the colliding issue in the other project is untouched")

	openThreads, err := s.ListIssueComment(ctx, &store.FindIssueCommentMessage{ProjectID: "default", IssueUID: &[]int64{issueUID}[0], ThreadState: &open})
	require.NoError(t, err)
	var comments []string
	for _, c := range openThreads {
		comments = append(comments, c.Payload.Comment)
		if c.Payload.ReviewMetadata != nil {
			require.Empty(t, c.CreatorEmail, "a review result reads back with no creator")
		} else {
			require.Equal(t, "dev@example.com", c.CreatorEmail)
		}
	}
	require.ElementsMatch(t, []string{"person", "ai", "default third"}, comments)

	// A review result's text belongs to the reviewer: a person can resolve or
	// reopen it but not rewrite it. A person's own thread stays editable.
	resolved := store.ThreadStateResolved
	for _, c := range openThreads {
		text := "rewritten"
		err := s.UpdateIssueComment(ctx, &store.UpdateIssueCommentMessage{ProjectID: "default", ResourceID: c.ResourceID, Comment: &text})
		if c.Payload.ReviewMetadata != nil {
			require.Equal(t, common.Invalid, common.ErrorCode(err), c.Payload.Comment)
			require.NoError(t, s.UpdateIssueComment(ctx, &store.UpdateIssueCommentMessage{ProjectID: "default", ResourceID: c.ResourceID, ThreadState: &resolved}))
			require.NoError(t, s.UpdateIssueComment(ctx, &store.UpdateIssueCommentMessage{ProjectID: "default", ResourceID: c.ResourceID, ThreadState: &open}))
		} else {
			require.NoError(t, err, c.Payload.Comment)
		}
	}
	for _, c := range listReviewComments(t, fixture, "default") {
		require.NotEqual(t, "rewritten", c.Comment)
	}

	// A clean DONE with no results still resolves the previous ones.
	_, err = s.CreateReviewRun(ctx, "default", issueUID, store.ReviewRunTypeRule)
	require.NoError(t, err)
	claimed, err = s.ClaimAvailableReviewRuns(ctx, "replica-1")
	require.NoError(t, err)
	require.Len(t, claimed, 1)
	updated, err = s.CompleteReviewRun(ctx, claimed[0], "replica-1", storepb.ReviewRun_DONE, nil, nil)
	require.NoError(t, err)
	require.True(t, updated)
	for _, c := range listReviewComments(t, fixture, "default") {
		if c.Comment == "ai" {
			require.Equal(t, "OPEN", c.ThreadState)
			continue
		}
		require.Equal(t, "RESOLVED", c.ThreadState, c.Comment)
	}
}

// TestCompleteReviewRunKeepsResultsReportedAgain pins how a run tells the
// findings it reports again apart: a current root whose fingerprint a result
// repeats keeps its thread state and updated_at, and the result is not posted
// twice. A fingerprint matches only a current root of the same reviewer on
// the same issue: never a superseded root, an AI result, or the colliding
// issue in another project. A result without one matches nothing.
func TestCompleteReviewRunKeepsResultsReportedAgain(t *testing.T) {
	t.Parallel()
	fixture := newStorePostgresFixture(t, `
		INSERT INTO project (resource_id, workspace, name) VALUES ('other', 'default', 'Other');
		INSERT INTO issue (id, creator, project, name, status, type)
		VALUES (301, 'admin@example.com', 'default', 'Issue Default', 'OPEN', 'DATABASE_CHANGE');
		INSERT INTO issue (id, creator, project, name, status, type)
		VALUES (301, 'admin@example.com', 'other', 'Issue Other', 'OPEN', 'DATABASE_CHANGE');
	`)
	result := func(projectID, comment, fingerprint string) *store.IssueCommentMessage {
		r := reviewResult(projectID, comment, storepb.ReviewRuleType_REQUIRE_WHERE)
		r.Payload.ReviewMetadata.Fingerprint = fingerprint
		return r
	}
	complete := func(projectID string, results ...*store.IssueCommentMessage) {
		t.Helper()
		completeReviewRun(t, fixture, projectID, store.ReviewRunTypeRule, storepb.ReviewRun_DONE, results...)
	}

	// The colliding issue and an AI result hold the fingerprint the second
	// run reports first.
	complete("other", result("other", "other", "new"))
	aiResult := result("default", "ai", "new")
	aiResult.Payload.ReviewMetadata.RunType = storepb.ReviewRun_AI
	aiResult.Payload.ReviewMetadata.RuleType = storepb.ReviewRuleType_REVIEW_RULE_TYPE_UNSPECIFIED
	completeReviewRun(t, fixture, "default", store.ReviewRunTypeAI, storepb.ReviewRun_DONE, aiResult)

	// A finding on the whole change has no anchor to move.
	wholeChange := func(comment string) *store.IssueCommentMessage {
		r := result("default", comment, "whole")
		r.Payload.StatementAnchor = nil
		return r
	}
	complete("default",
		result("default", "open", "open"),
		result("default", "accepted", "accepted"),
		result("default", "fixed", "fixed"),
		result("default", "no fingerprint", ""),
		wholeChange("whole"),
	)
	setReviewThreadState(t, fixture, "accepted", store.ThreadStateResolved)
	_, updatedBefore := listSupersession(t, fixture, "default")

	// The SQL changed above "open": its statement moved down two lines.
	movedSheet := strings.Repeat("ab", 32)
	openAgain := result("default", "open again", "open")
	openAgain.Payload.StatementAnchor.SheetSha256 = movedSheet
	openAgain.Payload.StatementAnchor.StartPosition = &storepb.Position{Line: 3}
	openAgain.Payload.StatementAnchor.EndPosition = &storepb.Position{Line: 3}
	complete("default",
		result("default", "new", "new"),
		openAgain,
		result("default", "accepted again", "accepted"),
		result("default", "no fingerprint again", ""),
		wholeChange("whole again"),
	)
	got, updatedAfter := listSupersession(t, fixture, "default")
	require.Equal(t, []supersession{
		{Comment: "ai", ThreadState: "OPEN"},
		{Comment: "open again", ThreadState: "OPEN"},
		{Comment: "accepted again", ThreadState: "RESOLVED"},
		{Comment: "fixed", ThreadState: "RESOLVED", Superseded: true},
		{Comment: "no fingerprint", ThreadState: "RESOLVED", Superseded: true},
		{Comment: "whole again", ThreadState: "OPEN"},
		{Comment: "new", ThreadState: "OPEN"},
		{Comment: "no fingerprint again", ThreadState: "OPEN"},
	}, got, "a kept result takes the text of the result that repeats it")
	for before, after := range map[string]string{"open": "open again", "accepted": "accepted again", "whole": "whole again"} {
		require.Equal(t, updatedBefore[before], updatedAfter[after], "keeping %q is no person's edit", before)
	}
	anchors := reviewResultAnchors(t, fixture, "default")
	require.Equal(t, movedSheet+"/3", anchors["open again"], "a kept result moves to the statement the run found")
	require.Equal(t, reviewResultSheet+"/1", anchors["accepted again"])
	require.Equal(t, "", anchors["whole again"])
	require.Equal(t, []reviewRunCompletion{
		{RunType: storepb.ReviewRun_AI, Posted: 1},
		{RunType: storepb.ReviewRun_RULE, Posted: 5},
		{RunType: storepb.ReviewRun_RULE, Attempt: 1, Posted: 2, Kept: 3, Superseded: 2},
	}, listReviewRunCompletions(t, fixture, "default"))
	other, _ := listSupersession(t, fixture, "other")
	require.Equal(t, []supersession{{Comment: "other", ThreadState: "OPEN"}}, other, "the colliding issue in the other project is untouched")

	// A finding reported again after a run superseded it starts a new thread,
	// and its superseded root is resolved again though someone reopened it.
	setReviewThreadState(t, fixture, "fixed", store.ThreadStateOpen)
	complete("default", result("default", "new again", "new"), result("default", "fixed again", "fixed"))
	final, _ := listSupersession(t, fixture, "default")
	require.Equal(t, []supersession{
		{Comment: "ai", ThreadState: "OPEN"},
		{Comment: "open again", ThreadState: "RESOLVED", Superseded: true},
		{Comment: "accepted again", ThreadState: "RESOLVED", Superseded: true},
		{Comment: "fixed", ThreadState: "RESOLVED", Superseded: true},
		{Comment: "no fingerprint", ThreadState: "RESOLVED", Superseded: true},
		{Comment: "whole again", ThreadState: "RESOLVED", Superseded: true},
		{Comment: "new again", ThreadState: "OPEN"},
		{Comment: "no fingerprint again", ThreadState: "RESOLVED", Superseded: true},
		{Comment: "fixed again", ThreadState: "OPEN"},
	}, final)
	require.Equal(t, map[string]int64{
		"ai": 0, "open again": 0, "accepted again": 0, "fixed": 0, "no fingerprint": 0, "whole again": 0,
		"new again": 1, "no fingerprint again": 1, "fixed again": 2,
	}, reviewResultAttempts(t, fixture, "default"), "a kept result keeps the attempt that posted it")
	completions := listReviewRunCompletions(t, fixture, "default")
	require.Equal(t, reviewRunCompletion{RunType: storepb.ReviewRun_RULE, Attempt: 2, Posted: 1, Kept: 1, Superseded: 4}, completions[len(completions)-1])
	require.Equal(t, []reviewRunCompletion{{RunType: storepb.ReviewRun_RULE, Posted: 1}}, listReviewRunCompletions(t, fixture, "other"))
	other, _ = listSupersession(t, fixture, "other")
	require.Equal(t, []supersession{{Comment: "other", ThreadState: "OPEN"}}, other, "keeping the colliding fingerprint leaves the other project's text")

	// Fingerprints are unique within a run.
	ctx, s := fixture.ctx, fixture.store
	_, err := s.CreateReviewRun(ctx, "default", reviewResultIssue, store.ReviewRunTypeRule)
	require.NoError(t, err)
	claimed, err := s.ClaimAvailableReviewRuns(ctx, "replica-1")
	require.NoError(t, err)
	require.Len(t, claimed, 1)
	_, err = s.CompleteReviewRun(ctx, claimed[0], "replica-1", storepb.ReviewRun_DONE, nil, []*store.IssueCommentMessage{
		result("default", "first", "same"),
		result("default", "second", "same"),
	})
	require.Error(t, err)
	unchanged, _ := listSupersession(t, fixture, "default")
	require.Equal(t, final, unchanged, "a rejected completion writes nothing")
	require.Equal(t, completions, listReviewRunCompletions(t, fixture, "default"))
}

// TestCompleteReviewRunKeptResultTakesDatabasesPriorityAndTitle keeps an AI
// result whose databases, priority, and title change, reopening it when a
// resolved one's priority rises, on two projects whose issue ids collide.
func TestCompleteReviewRunKeptResultTakesDatabasesPriorityAndTitle(t *testing.T) {
	t.Parallel()
	fixture := newStorePostgresFixture(t, `
		INSERT INTO project (resource_id, workspace, name) VALUES ('other', 'default', 'Other');
		INSERT INTO issue (id, creator, project, name, status, type)
		VALUES (301, 'admin@example.com', 'default', 'Issue Default', 'OPEN', 'DATABASE_CHANGE');
		INSERT INTO issue (id, creator, project, name, status, type)
		VALUES (301, 'admin@example.com', 'other', 'Issue Other', 'OPEN', 'DATABASE_CHANGE');
	`)
	result := func(projectID, title string, priority storepb.IssueCommentPayload_ReviewMetadata_Priority, targets ...string) *store.IssueCommentMessage {
		r := reviewResult(projectID, "kept", storepb.ReviewRuleType_REVIEW_RULE_TYPE_UNSPECIFIED)
		r.Payload.ReviewMetadata.RunType = storepb.ReviewRun_AI
		r.Payload.ReviewMetadata.Priority = priority
		r.Payload.ReviewMetadata.Targets = targets
		r.Payload.ReviewMetadata.Fingerprint = "k"
		r.Payload.ReviewMetadata.Title = title
		return r
	}
	complete := func(projectID string, r *store.IssueCommentMessage) {
		t.Helper()
		completeReviewRun(t, fixture, projectID, store.ReviewRunTypeAI, storepb.ReviewRun_DONE, r)
	}
	metadata := func(projectID string) *storepb.IssueCommentPayload_ReviewMetadata {
		t.Helper()
		var raw []byte
		require.NoError(t, fixture.db.QueryRowContext(fixture.ctx, `
			SELECT payload->'reviewMetadata' FROM issue_comment
			WHERE project = $1 AND issue_id = $2 AND payload ? 'reviewMetadata'
		`, projectID, reviewResultIssue).Scan(&raw))
		m := &storepb.IssueCommentPayload_ReviewMetadata{}
		require.NoError(t, common.ProtojsonUnmarshaler.Unmarshal(raw, m))
		return m
	}
	a, b := "instances/i/databases/a", "instances/i/databases/b"

	complete("other", result("other", "Other title", storepb.IssueCommentPayload_ReviewMetadata_P1, "instances/i/databases/z"))
	complete("default", result("default", "First title", storepb.IssueCommentPayload_ReviewMetadata_P1, a, b))
	setReviewThreadState(t, fixture, "kept", store.ThreadStateResolved)

	complete("default", result("default", "Second title", storepb.IssueCommentPayload_ReviewMetadata_P2, a))
	kept := metadata("default")
	require.Equal(t, []string{a}, kept.Targets)
	require.Equal(t, storepb.IssueCommentPayload_ReviewMetadata_P2, kept.Priority)
	require.Equal(t, "Second title", kept.Title)
	require.Equal(t, "k", kept.Fingerprint)
	require.False(t, kept.Superseded)
	state, _ := listSupersession(t, fixture, "default")
	require.Equal(t, []supersession{{Comment: "kept", ThreadState: "RESOLVED"}}, state, "the person's resolution stays")

	complete("default", result("default", "Second title", storepb.IssueCommentPayload_ReviewMetadata_PRIORITY_UNSPECIFIED, a, b))
	kept = metadata("default")
	require.Equal(t, []string{a, b}, kept.Targets)
	require.Equal(t, storepb.IssueCommentPayload_ReviewMetadata_PRIORITY_UNSPECIFIED, kept.Priority, "a result without a priority clears it")
	state, _ = listSupersession(t, fixture, "default")
	require.Equal(t, []supersession{{Comment: "kept", ThreadState: "RESOLVED"}}, state)

	complete("default", result("default", "", storepb.IssueCommentPayload_ReviewMetadata_P1, a))
	require.Empty(t, metadata("default").Title, "a result without a title clears it")
	require.Equal(t, storepb.IssueCommentPayload_ReviewMetadata_P1, metadata("default").Priority)
	state, _ = listSupersession(t, fixture, "default")
	require.Equal(t, []supersession{{Comment: "kept", ThreadState: "OPEN"}}, state, "nobody accepted the higher priority")
	require.Equal(t, []reviewRunCompletion{
		{RunType: storepb.ReviewRun_AI, Posted: 1},
		{RunType: storepb.ReviewRun_AI, Attempt: 1, Kept: 1},
		{RunType: storepb.ReviewRun_AI, Attempt: 2, Kept: 1},
		{RunType: storepb.ReviewRun_AI, Attempt: 3, Kept: 1},
	}, listReviewRunCompletions(t, fixture, "default"))

	other := metadata("other")
	require.Equal(t, []string{"instances/i/databases/z"}, other.Targets, "the colliding issue in the other project is untouched")
	require.Equal(t, storepb.IssueCommentPayload_ReviewMetadata_P1, other.Priority)
	require.Equal(t, "Other title", other.Title)
}

type reviewRunCompletion struct {
	RunType    storepb.ReviewRun_Type
	Attempt    int64
	Posted     int32
	Kept       int32
	Superseded int32
}

// listReviewRunCompletions reads the completion events on the shared issue,
// oldest first, checking each is a creatorless event row.
func listReviewRunCompletions(t *testing.T, fixture *storePostgresFixture, projectID string) []reviewRunCompletion {
	t.Helper()
	rows, err := fixture.db.QueryContext(fixture.ctx, `
		SELECT payload, creator IS NULL AND thread_state IS NULL AND parent_id IS NULL
		FROM issue_comment
		WHERE project = $1 AND issue_id = $2 AND payload ? 'reviewRunCompletion'
		ORDER BY created_at, resource_id
	`, projectID, reviewResultIssue)
	require.NoError(t, err)
	defer rows.Close()
	var out []reviewRunCompletion
	for rows.Next() {
		var raw []byte
		var event bool
		require.NoError(t, rows.Scan(&raw, &event))
		require.True(t, event, "a completion is an event with no creator")
		payload := &storepb.IssueCommentPayload{}
		require.NoError(t, common.ProtojsonUnmarshaler.Unmarshal(raw, payload))
		c := payload.GetReviewRunCompletion()
		out = append(out, reviewRunCompletion{
			RunType: c.GetRunType(), Attempt: c.GetAttempt(),
			Posted: c.GetPostedCount(), Kept: c.GetKeptCount(), Superseded: c.GetSupersededCount(),
		})
	}
	require.NoError(t, rows.Err())
	return out
}

// reviewResultAnchors maps each review result's text to its anchor's sheet and
// start line, or "" without one.
func reviewResultAnchors(t *testing.T, fixture *storePostgresFixture, projectID string) map[string]string {
	t.Helper()
	rows, err := fixture.db.QueryContext(fixture.ctx, `
		SELECT payload->>'comment',
		       COALESCE((payload->'statementAnchor'->>'sheetSha256') || '/' || (payload->'statementAnchor'->'startPosition'->>'line'), '')
		FROM issue_comment
		WHERE project = $1 AND issue_id = $2 AND payload ? 'reviewMetadata'
	`, projectID, reviewResultIssue)
	require.NoError(t, err)
	defer rows.Close()
	out := make(map[string]string)
	for rows.Next() {
		var comment, anchor string
		require.NoError(t, rows.Scan(&comment, &anchor))
		out[comment] = anchor
	}
	require.NoError(t, rows.Err())
	return out
}

// reviewResultAttempts maps each review result's text to its attempt.
func reviewResultAttempts(t *testing.T, fixture *storePostgresFixture, projectID string) map[string]int64 {
	t.Helper()
	rows, err := fixture.db.QueryContext(fixture.ctx, `
		SELECT payload->>'comment', COALESCE((payload->'reviewMetadata'->>'attempt')::BIGINT, 0)
		FROM issue_comment
		WHERE project = $1 AND issue_id = $2 AND payload ? 'reviewMetadata'
	`, projectID, reviewResultIssue)
	require.NoError(t, err)
	defer rows.Close()
	out := make(map[string]int64)
	for rows.Next() {
		var comment string
		var attempt int64
		require.NoError(t, rows.Scan(&comment, &attempt))
		out[comment] = attempt
	}
	require.NoError(t, rows.Err())
	return out
}

// completeReviewRun starts, claims, and completes one run on the shared issue.
func completeReviewRun(t *testing.T, fixture *storePostgresFixture, projectID, reviewType string, status storepb.ReviewRun_Status, results ...*store.IssueCommentMessage) {
	t.Helper()
	ctx, s := fixture.ctx, fixture.store
	_, err := s.CreateReviewRun(ctx, projectID, reviewResultIssue, reviewType)
	require.NoError(t, err)
	claimed, err := s.ClaimAvailableReviewRuns(ctx, "replica-1")
	require.NoError(t, err)
	require.Len(t, claimed, 1)
	updated, err := s.CompleteReviewRun(ctx, claimed[0], "replica-1", status, &storepb.ReviewRunPayload{}, results)
	require.NoError(t, err)
	require.True(t, updated)
}

// setReviewThreadState sets the thread state of the default project's root
// with the given text, as a person would.
func setReviewThreadState(t *testing.T, fixture *storePostgresFixture, comment string, state store.ThreadState) {
	t.Helper()
	ctx, s := fixture.ctx, fixture.store
	roots, err := s.ListIssueComment(ctx, &store.FindIssueCommentMessage{ProjectID: "default", IssueUID: &[]int64{reviewResultIssue}[0]})
	require.NoError(t, err)
	for _, root := range roots {
		if root.Payload.Comment == comment {
			require.NoError(t, s.UpdateIssueComment(ctx, &store.UpdateIssueCommentMessage{ProjectID: "default", ResourceID: root.ResourceID, ThreadState: &state}))
			return
		}
	}
	t.Fatalf("no comment %q", comment)
}

type supersession struct {
	Comment     string
	ThreadState string
	Superseded  bool
}

func listSupersession(t *testing.T, fixture *storePostgresFixture, projectID string) ([]supersession, map[string]time.Time) {
	t.Helper()
	rows, err := fixture.db.QueryContext(fixture.ctx, `
		SELECT payload->>'comment', thread_state,
		       COALESCE((payload->'reviewMetadata'->>'superseded')::boolean, false), updated_at
		FROM issue_comment
		WHERE project = $1 AND issue_id = $2 AND payload ? 'reviewMetadata'
		ORDER BY created_at, resource_id
	`, projectID, reviewResultIssue)
	require.NoError(t, err)
	defer rows.Close()
	var out []supersession
	updatedAt := make(map[string]time.Time)
	for rows.Next() {
		var s supersession
		var at time.Time
		require.NoError(t, rows.Scan(&s.Comment, &s.ThreadState, &s.Superseded, &at))
		out = append(out, s)
		updatedAt[s.Comment] = at
	}
	require.NoError(t, rows.Err())
	return out, updatedAt
}

// TestCompleteReviewRunSupersedesEarlierResults pins what tells the latest
// review apart: a DONE completion marks every earlier result of its reviewer
// on its issue superseded, whatever its thread state, without touching
// updated_at; a FAILED completion marks nothing; another reviewer's results
// and the colliding issue in another project keep theirs.
func TestCompleteReviewRunSupersedesEarlierResults(t *testing.T) {
	t.Parallel()
	fixture := newStorePostgresFixture(t, `
		INSERT INTO project (resource_id, workspace, name) VALUES ('other', 'default', 'Other');
		INSERT INTO issue (id, creator, project, name, status, type)
		VALUES (301, 'admin@example.com', 'default', 'Issue Default', 'OPEN', 'DATABASE_CHANGE');
		INSERT INTO issue (id, creator, project, name, status, type)
		VALUES (301, 'admin@example.com', 'other', 'Issue Other', 'OPEN', 'DATABASE_CHANGE');
	`)
	complete := func(projectID, reviewType string, status storepb.ReviewRun_Status, results ...*store.IssueCommentMessage) {
		t.Helper()
		completeReviewRun(t, fixture, projectID, reviewType, status, results...)
	}
	setThreadState := func(comment string, state store.ThreadState) {
		t.Helper()
		setReviewThreadState(t, fixture, comment, state)
	}
	aiResult := reviewResult("default", "ai", storepb.ReviewRuleType_REVIEW_RULE_TYPE_UNSPECIFIED)
	aiResult.Payload.ReviewMetadata.RunType = storepb.ReviewRun_AI

	complete("default", store.ReviewRunTypeAI, storepb.ReviewRun_DONE, aiResult)
	complete("other", store.ReviewRunTypeRule, storepb.ReviewRun_DONE, reviewResult("other", "other first", storepb.ReviewRuleType_REQUIRE_WHERE))
	complete("default", store.ReviewRunTypeRule, storepb.ReviewRun_DONE,
		reviewResult("default", "first", storepb.ReviewRuleType_REQUIRE_WHERE),
		reviewResult("default", "dismissed", storepb.ReviewRuleType_DISALLOW_TRUNCATE),
	)
	// A person resolves one result of the run, as a false positive.
	setThreadState("dismissed", store.ThreadStateResolved)
	_, updatedBefore := listSupersession(t, fixture, "default")

	complete("default", store.ReviewRunTypeRule, storepb.ReviewRun_DONE, reviewResult("default", "second", storepb.ReviewRuleType_REQUIRE_WHERE))
	got, updatedAfter := listSupersession(t, fixture, "default")
	require.Equal(t, []supersession{
		{Comment: "ai", ThreadState: "OPEN"},
		{Comment: "first", ThreadState: "RESOLVED", Superseded: true},
		{Comment: "dismissed", ThreadState: "RESOLVED", Superseded: true},
		{Comment: "second", ThreadState: "OPEN"},
	}, got)
	for _, comment := range []string{"first", "dismissed"} {
		require.Equal(t, updatedBefore[comment], updatedAfter[comment], "superseding %q is no content edit", comment)
	}
	other, _ := listSupersession(t, fixture, "other")
	require.Equal(t, []supersession{{Comment: "other first", ThreadState: "OPEN"}}, other, "the colliding issue in the other project is untouched")

	// A FAILED run supersedes nothing: the latest review is still the last DONE.
	setThreadState("first", store.ThreadStateOpen)
	completions := listReviewRunCompletions(t, fixture, "default")
	complete("default", store.ReviewRunTypeRule, storepb.ReviewRun_FAILED)
	require.Equal(t, completions, listReviewRunCompletions(t, fixture, "default"), "a FAILED run records no completion")
	afterFailure, _ := listSupersession(t, fixture, "default")
	require.Equal(t, []supersession{
		{Comment: "ai", ThreadState: "OPEN"},
		{Comment: "first", ThreadState: "OPEN", Superseded: true},
		{Comment: "dismissed", ThreadState: "RESOLVED", Superseded: true},
		{Comment: "second", ThreadState: "OPEN"},
	}, afterFailure)

	// The next DONE, here with no findings, supersedes the rest and resolves
	// the reopened result again.
	complete("default", store.ReviewRunTypeRule, storepb.ReviewRun_DONE)
	completions = listReviewRunCompletions(t, fixture, "default")
	require.Equal(t, reviewRunCompletion{RunType: storepb.ReviewRun_RULE, Attempt: 3, Superseded: 1}, completions[len(completions)-1],
		"only the current result counts as superseded")
	final, _ := listSupersession(t, fixture, "default")
	require.Equal(t, []supersession{
		{Comment: "ai", ThreadState: "OPEN"},
		{Comment: "first", ThreadState: "RESOLVED", Superseded: true},
		{Comment: "dismissed", ThreadState: "RESOLVED", Superseded: true},
		{Comment: "second", ThreadState: "RESOLVED", Superseded: true},
	}, final)
}
