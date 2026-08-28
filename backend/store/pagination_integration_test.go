package store_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/migrator"
	"github.com/bytebase/bytebase/backend/store"

	_ "github.com/bytebase/bytebase/backend/plugin/db/pg"
)

// TestPaginationStabilityAcrossProjects pages through a cross-project issue list
// whose rows are deliberately tied on every sort key that is not a full primary
// key, and asserts every issue is returned exactly once.
//
// This is the behavior TestPaginatedListsUseStableOrderBy cannot check: that
// test only sees whether the clause was built with the helper, not whether the
// columns chosen make the order total. Against the pre-fix ordering
// (`ORDER BY issue.id DESC` alone) this fails — issue IDs restart per project,
// so every id below is tied three ways and rows cross the page boundary between
// reads.
func TestPaginationStabilityAcrossProjects(t *testing.T) {
	const (
		workspaceID  = "pagination-ws"
		projectCount = 3
		perProject   = 60
		pageSize     = 7
	)

	ctx := context.Background()
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(ctx) })
	db := container.GetDB()
	require.NoError(t, migrator.MigrateSchema(ctx, db))

	projectIDs := make([]string, 0, projectCount)
	for i := range projectCount {
		projectIDs = append(projectIDs, fmt.Sprintf("pagination-p%d", i))
	}
	seedTiedIssues(ctx, t, db, workspaceID, projectIDs, perProject)

	pgURL := fmt.Sprintf(
		"host=%s port=%s user=postgres password=root-password database=postgres",
		container.GetHost(), container.GetPort(),
	)
	stores, err := store.New(ctx, pgURL, false)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, stores.Close()) })

	// Walk the list the way the API does: a fresh LIMIT/OFFSET query per page.
	seen := map[string]int{}
	total := projectCount * perProject
	for offset := 0; offset < total+pageSize; offset += pageSize {
		limit, off := pageSize, offset
		issues, err := stores.ListIssues(ctx, &store.FindIssueMessage{
			ProjectIDs: projectIDs,
			Limit:      &limit,
			Offset:     &off,
		})
		require.NoErrorf(t, err, "page at offset %d", offset)
		for _, issue := range issues {
			seen[fmt.Sprintf("%s/%d", issue.ProjectID, issue.UID)]++
		}
	}

	var duplicated []string
	for key, count := range seen {
		if count > 1 {
			duplicated = append(duplicated, fmt.Sprintf("%s seen %d times", key, count))
		}
	}
	require.Emptyf(t, duplicated, "offset paging returned the same issue more than once: %v", duplicated)
	require.Lenf(t, seen, total,
		"offset paging returned %d distinct issues, want %d — %d were skipped entirely",
		len(seen), total, total-len(seen))
}

// seedTiedIssues writes projectCount projects each holding perProject issues,
// all in one transaction so every row shares a created_at (now() is the
// transaction timestamp) and IDs collide across projects.
func seedTiedIssues(ctx context.Context, t *testing.T, db *sql.DB, workspaceID string, projectIDs []string, perProject int) {
	t.Helper()

	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback() //nolint:errcheck

	_, err = tx.ExecContext(ctx, `INSERT INTO workspace (resource_id) VALUES ($1)`, workspaceID)
	require.NoError(t, err)

	for _, projectID := range projectIDs {
		_, err = tx.ExecContext(ctx,
			`INSERT INTO project (resource_id, workspace, name) VALUES ($1, $2, $3)`,
			projectID, workspaceID, projectID)
		require.NoError(t, err)

		// Start at 101 exactly as nextProjectID does, so ids collide across
		// projects the way real data does.
		_, err = tx.ExecContext(ctx, `
			INSERT INTO issue (id, project, creator, name, status, type, description)
			SELECT g, $1, 'users/pagination@example.com', 'issue ' || g, 'OPEN', 'DATABASE_CHANGE', ''
			FROM generate_series(101, $2::int) g`,
			projectID, 100+perProject)
		require.NoError(t, err)
	}
	require.NoError(t, tx.Commit())
}

// TestIssueCommentBatchKeepsInsertionOrder pins the ordering of comments written
// by one CreateIssueComments batch — the several activity events a multi-field
// UpdateIssue produces.
//
// created_at is the transaction timestamp, so without the per-row ordinal offset
// in CreateIssueComments the whole batch lands on one instant and
// ListIssueComment falls through to its resource_id tiebreak, a random UUID.
// The feed would then be stably scrambled: "labels changed" above "title
// changed", permanently, for that issue.
func TestIssueCommentBatchKeepsInsertionOrder(t *testing.T) {
	ctx := context.Background()
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(ctx) })
	db := container.GetDB()
	require.NoError(t, migrator.MigrateSchema(ctx, db))

	const (
		workspaceID = "comment-ws"
		projectID   = "comment-p"
		issueUID    = int64(101)
	)
	_, err := db.ExecContext(ctx, `
		INSERT INTO workspace (resource_id) VALUES ($1);
		INSERT INTO project (resource_id, workspace, name) VALUES ($2, $1, $2);
		INSERT INTO issue (id, project, creator, name, status, type, description)
		VALUES ($3, $2, 'users/comment@example.com', 'issue', 'OPEN', 'DATABASE_CHANGE', '')`,
		workspaceID, projectID, issueUID)
	require.NoError(t, err)

	pgURL := fmt.Sprintf(
		"host=%s port=%s user=postgres password=root-password database=postgres",
		container.GetHost(), container.GetPort(),
	)
	stores, err := store.New(ctx, pgURL, false)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, stores.Close()) })

	want := []string{"title", "description", "labels", "status", "assignee"}
	creates := make([]*store.IssueCommentMessage, 0, len(want))
	for _, comment := range want {
		creates = append(creates, &store.IssueCommentMessage{
			ProjectID: projectID,
			IssueUID:  issueUID,
			Payload:   &storepb.IssueCommentPayload{Comment: comment},
		})
	}
	_, err = stores.CreateIssueComments(ctx, "users/comment@example.com", creates...)
	require.NoError(t, err)

	uid := issueUID
	got, err := stores.ListIssueComment(ctx, &store.FindIssueCommentMessage{
		ProjectID: projectID,
		IssueUID:  &uid,
	})
	require.NoError(t, err)

	order := make([]string, 0, len(got))
	for _, comment := range got {
		order = append(order, comment.Payload.Comment)
	}
	require.Equal(t, want, order, "a batch of issue comments must read back in insertion order")
}
