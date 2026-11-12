package tests

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// reviewRun is one row of the review_run table. The v1 ReviewRun shows
// neither the attempt nor the payload, so this suite reads the row with a
// table-specific direct DB read, the same allowance
// listPlanWebhookDeliveries uses for tables without a public read API.
type reviewRun struct {
	IssueID int64
	Attempt int64
	Status  string
	Payload string
}

// listReviewRuns reads every review_run row for the project directly from
// the test metadata database, ordered by primary key.
func listReviewRuns(ctx context.Context, t *testing.T, ctl *controller, projectID string) []*reviewRun {
	t.Helper()
	a := require.New(t)
	db, err := sql.Open("pgx", ctl.profile.PgURL)
	a.NoError(err, "open metadata DB")
	defer db.Close()

	rows, err := db.QueryContext(ctx, `
		SELECT issue_id, attempt, status, payload::text
		FROM review_run
		WHERE project = $1
		ORDER BY issue_id
	`, projectID)
	a.NoError(err, "query review_run for project %s", projectID)
	defer rows.Close()

	var out []*reviewRun
	for rows.Next() {
		var r reviewRun
		a.NoError(rows.Scan(&r.IssueID, &r.Attempt, &r.Status, &r.Payload))
		out = append(out, &r)
	}
	a.NoError(rows.Err())
	return out
}

// seedReviewRun inserts one review_run row directly into the metadata
// database, so the purge collision test holds a row in the state it names
// without running a review.
func seedReviewRun(ctx context.Context, t *testing.T, ctl *controller, projectID string, issueID int64, attempt int64, status string) {
	t.Helper()
	a := require.New(t)
	db, err := sql.Open("pgx", ctl.profile.PgURL)
	a.NoError(err, "open metadata DB")
	defer db.Close()

	_, err = db.ExecContext(ctx, `
		INSERT INTO review_run (project, issue_id, attempt, status)
		VALUES ($1, $2, $3, $4)
	`, projectID, issueID, attempt, status)
	a.NoError(err, "seed review_run (%s, %d)", projectID, issueID)
}

// TestCollision_ReviewRunProjectPurge verifies that purging project A
// deletes A's review_run rows and leaves project B's rows untouched when
// both projects hold a row under the same issue id.
func TestCollision_ReviewRunProjectPurge(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	ctl, ctx := startProject(ctx, t)

	fixture := setupCollidingProjects(ctx, t, ctl)

	projectAID, issueAUID, err := common.GetProjectIDIssueUID(fixture.IssueA.Name)
	a.NoError(err)
	projectBID, issueBUID, err := common.GetProjectIDIssueUID(fixture.IssueB.Name)
	a.NoError(err)
	// The fixture's per-project id allocation makes the two issues collide;
	// assert it so the test cannot silently lose its collision.
	a.Equal(issueAUID, issueBUID, "fixture issue ids should collide")

	seedReviewRun(ctx, t, ctl, projectAID, issueAUID, 2, "DONE")
	seedReviewRun(ctx, t, ctl, projectBID, issueBUID, 5, "FAILED")

	// Positive preconditions: an empty list would make the isolation
	// assertions below vacuously true.
	beforeA := listReviewRuns(ctx, t, ctl, projectAID)
	beforeB := listReviewRuns(ctx, t, ctl, projectBID)
	a.Greater(len(beforeA), 0, "project A should have review_run rows before purge")
	a.Greater(len(beforeB), 0, "project B should have review_run rows before purge")

	// Project purge is an explicit archive-then-purge lifecycle.
	_, err = ctl.projectServiceClient.DeleteProject(ctx,
		connect.NewRequest(&v1pb.DeleteProjectRequest{Name: fixture.ProjectA.Name}))
	a.NoError(err)
	_, err = ctl.projectServiceClient.DeleteProject(ctx,
		connect.NewRequest(&v1pb.DeleteProjectRequest{
			Name:  fixture.ProjectA.Name,
			Purge: true,
		}))
	a.NoError(err)

	a.Empty(listReviewRuns(ctx, t, ctl, projectAID), "project A review_run rows should be purged")
	a.Equal(beforeB, listReviewRuns(ctx, t, ctl, projectBID), "project B review_run rows should be untouched")
}
