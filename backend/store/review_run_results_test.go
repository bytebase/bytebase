package store_test

import (
	"testing"

	"github.com/stretchr/testify/require"

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

	// A person's anchored thread and a guideline result on the same issue
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
	guideline := reviewResult("default", "guideline", storepb.ReviewRuleType_REVIEW_RULE_TYPE_UNSPECIFIED)
	guideline.Payload.ReviewMetadata.RunType = storepb.ReviewRun_GUIDELINE
	guideline.ThreadState = &open

	// Colliding rule runs on both projects.
	for _, projectID := range []string{"default", "other"} {
		_, err := s.CreateReviewRun(ctx, projectID, issueUID, store.ReviewRunTypeRule)
		require.NoError(t, err)
	}
	_, err = s.CreateReviewRun(ctx, "default", issueUID, store.ReviewRunTypeGuideline)
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
	guidelineDefault := byKey[[2]string{"default", store.ReviewRunTypeGuideline}]

	updated, err := s.CompleteReviewRun(ctx, guidelineDefault, "replica-1", storepb.ReviewRun_DONE, nil, []*store.IssueCommentMessage{guideline})
	require.NoError(t, err)
	require.True(t, updated)

	// Results must belong to the claimed run and name its reviewer.
	_, err = s.CompleteReviewRun(ctx, ruleDefault, "replica-1", storepb.ReviewRun_DONE, nil,
		[]*store.IssueCommentMessage{reviewResult("other", "wrong project", storepb.ReviewRuleType_SYNTAX)})
	require.Error(t, err)
	wrongReviewer := reviewResult("default", "wrong reviewer", storepb.ReviewRuleType_SYNTAX)
	wrongReviewer.Payload.ReviewMetadata.RunType = storepb.ReviewRun_GUIDELINE
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
		{Comment: "guideline", ThreadState: "OPEN"},
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
	// issue, and only those: the guideline result, the person's thread, and
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
		{Comment: "guideline", ThreadState: "OPEN"},
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
	require.ElementsMatch(t, []string{"person", "guideline", "default third"}, comments)

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
		if c.Comment == "guideline" {
			require.Equal(t, "OPEN", c.ThreadState)
			continue
		}
		require.Equal(t, "RESOLVED", c.ThreadState, c.Comment)
	}
}
