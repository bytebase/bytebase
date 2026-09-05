package store_test

import (
	"context"
	"testing"

	"github.com/bytebase/bytebase/backend/common/testcontainer"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"

	_ "github.com/bytebase/bytebase/backend/plugin/db/pg"
)

// TestIssueListFilterEdgeCases covers three defects in the issue list filters,
// all of which reproduce against the pre-fix code:
//   - search text that is punctuation only used to build an invalid tsquery and
//     fail the whole query with SQLSTATE 42601;
//   - the ILIKE fallback interpolated the search text unescaped, so `%` and `_`
//     matched as wildcards;
//   - create_time is documented as ">=" and "<=" but was implemented as ">" and
//     "<", excluding an issue at the exact boundary.
func TestIssueListFilterEdgeCases(t *testing.T) {
	ctx := context.Background()
	db, stores, _ := testcontainer.NewMetadataDB(t)
	_, err := db.ExecContext(ctx, `
		INSERT INTO workspace (resource_id) VALUES ('default');
		INSERT INTO principal (name, email, password_hash) VALUES ('c', 'c@example.com', 'x');
		INSERT INTO project (resource_id, workspace, name) VALUES ('p1', 'default', 'P1'), ('p2', 'default', 'P2');
	`)
	require.NoError(t, err)

	createIn := func(projectID, title string, approval *storepb.IssuePayloadApproval) *store.IssueMessage {
		plan, err := stores.CreatePlan(ctx, &store.PlanMessage{
			ProjectID: projectID, Name: title, Config: &storepb.PlanConfig{},
		}, "c@example.com")
		require.NoError(t, err)
		issue, err := stores.CreateIssue(ctx, &store.IssueMessage{
			ProjectID: projectID, CreatorEmail: "c@example.com", Title: title,
			Type: storepb.Issue_DATABASE_CHANGE, Payload: &storepb.Issue{Approval: approval}, PlanUID: &plan.UID,
		})
		require.NoError(t, err)
		return issue
	}
	create := func(title string) *store.IssueMessage { return createIn("p1", title, nil) }
	titles := func(find *store.FindIssueMessage) []string {
		find.Workspace, find.ProjectIDs = "default", []string{"p1"}
		issues, err := stores.ListIssues(ctx, find)
		require.NoError(t, err)
		out := []string{}
		for _, issue := range issues {
			out = append(out, issue.Title)
		}
		return out
	}

	create("fix the 100% CPU spike")
	create("cleanup")
	create("release (v2)")

	// Punctuation-only search returns results instead of failing, via ILIKE.
	for _, text := range []string{"&", "!", ":", "'", "|", "<->", "!!!"} {
		query := text
		require.Empty(t, titles(&store.FindIssueMessage{Query: &query}), "query %q", text)
	}
	paren := "("
	require.Equal(t, []string{"release (v2)"}, titles(&store.FindIssueMessage{Query: &paren}))

	// A real word still goes through full text search.
	word := "cleanup"
	require.Equal(t, []string{"cleanup"}, titles(&store.FindIssueMessage{Query: &word}))

	// LIKE metacharacters are matched literally, not as wildcards.
	wildcard := "%"
	require.Equal(t, []string{"fix the 100% CPU spike"}, titles(&store.FindIssueMessage{Query: &wildcard}))

	// create_time bounds include an issue sitting exactly on the boundary.
	boundary := create("boundary")
	stored, err := stores.GetIssue(ctx, &store.FindIssueMessage{ProjectIDs: []string{"p1"}, UID: &boundary.UID})
	require.NoError(t, err)
	at := stored.CreatedAt
	require.Contains(t, titles(&store.FindIssueMessage{CreatedAtAfter: &at}), "boundary")
	require.Contains(t, titles(&store.FindIssueMessage{CreatedAtBefore: &at}), "boundary")
}

// TestIssueListNextApproverIsProjectScoped locks the cross-project half of the
// current_approver predicate. Two projects run the identical approval flow; a
// role the caller holds in only one of them must match only that project's
// issue. A flat role list without the project pairing passes every other
// assertion in this file and fails this one.
func TestIssueListNextApproverIsProjectScoped(t *testing.T) {
	ctx := context.Background()
	db, stores, _ := testcontainer.NewMetadataDB(t)
	_, err := db.ExecContext(ctx, `
		INSERT INTO workspace (resource_id) VALUES ('default');
		INSERT INTO principal (name, email, password_hash) VALUES ('c', 'c@example.com', 'x');
		INSERT INTO project (resource_id, workspace, name) VALUES ('p1', 'default', 'P1'), ('p2', 'default', 'P2');
	`)
	require.NoError(t, err)

	waiting := &storepb.IssuePayloadApproval{
		ApprovalTemplate:    &storepb.ApprovalTemplate{Flow: &storepb.ApprovalFlow{Roles: []string{"roles/projectOwner"}}},
		ApprovalFindingDone: true,
	}
	create := func(projectID, title string) *store.IssueMessage {
		plan, err := stores.CreatePlan(ctx, &store.PlanMessage{
			ProjectID: projectID, Name: title, Config: &storepb.PlanConfig{},
		}, "c@example.com")
		require.NoError(t, err)
		issue, err := stores.CreateIssue(ctx, &store.IssueMessage{
			ProjectID: projectID, CreatorEmail: "c@example.com", Title: title,
			Type: storepb.Issue_DATABASE_CHANGE, Payload: &storepb.Issue{Approval: waiting}, PlanUID: &plan.UID,
		})
		require.NoError(t, err)
		return issue
	}
	inP1 := create("p1", "waiting in p1")
	create("p2", "waiting in p2")

	list := func(roles []store.ProjectRole) []*store.IssueMessage {
		issues, err := stores.ListIssues(ctx, &store.FindIssueMessage{
			Workspace: "default", ProjectIDs: []string{"p1", "p2"}, NextApproverRoles: &roles,
		})
		require.NoError(t, err)
		return issues
	}

	// Holding the role in p1 only must not surface p2's identical issue.
	got := list([]store.ProjectRole{{ProjectID: "p1", Role: "roles/projectOwner"}})
	require.Len(t, got, 1)
	require.Equal(t, "p1", got[0].ProjectID)
	require.Equal(t, inP1.UID, got[0].UID)

	// Holding it in both surfaces both.
	require.Len(t, list([]store.ProjectRole{
		{ProjectID: "p1", Role: "roles/projectOwner"},
		{ProjectID: "p2", Role: "roles/projectOwner"},
	}), 2)

	// A non-nil empty set matches nothing rather than everything.
	require.Empty(t, list([]store.ProjectRole{}))
}
