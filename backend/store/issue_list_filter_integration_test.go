package store_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/migrator"
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
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(ctx) })
	db := container.GetDB()
	require.NoError(t, migrator.MigrateSchema(ctx, db))
	_, err := db.ExecContext(ctx, `
		INSERT INTO workspace (resource_id) VALUES ('default');
		INSERT INTO principal (name, email, password_hash) VALUES ('c', 'c@example.com', 'x');
		INSERT INTO project (resource_id, workspace, name) VALUES ('p1', 'default', 'P1');
	`)
	require.NoError(t, err)

	pgURL := fmt.Sprintf(
		"host=%s port=%s user=postgres password=root-password database=postgres",
		container.GetHost(), container.GetPort(),
	)
	stores, err := store.New(ctx, pgURL, false)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, stores.Close()) })

	create := func(title string) *store.IssueMessage {
		plan, err := stores.CreatePlan(ctx, &store.PlanMessage{
			ProjectID: "p1", Name: title, Config: &storepb.PlanConfig{},
		}, "c@example.com")
		require.NoError(t, err)
		issue, err := stores.CreateIssue(ctx, &store.IssueMessage{
			ProjectID: "p1", CreatorEmail: "c@example.com", Title: title,
			Type: storepb.Issue_DATABASE_CHANGE, Payload: &storepb.Issue{}, PlanUID: &plan.UID,
		})
		require.NoError(t, err)
		return issue
	}
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
