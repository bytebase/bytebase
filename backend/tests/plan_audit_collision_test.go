package tests

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// TestCollision_PlanSpecAuditEmission verifies that PlanService.UpdatePlan
// emitting issue_comment audit rows for a spec mutation in project A does
// not touch any rows in project B (which shares colliding plan / issue ids
// under composite PK).
func TestCollision_PlanSpecAuditEmission(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	fixture := setupCollidingProjects(ctx, t, ctl)

	// fixture.PlanA already has a rollout (setupCollidingProjects drives A's
	// rollout to DONE). UpdatePlan rejects spec changes once a plan has a
	// rollout, so we create a fresh, rollout-free plan + issue in project A
	// for the action under test. The collision invariant still holds because
	// both projects share the same plan/issue id allocator — any cross-project
	// leak would still surface as a delta in project B's snapshot.
	planA2, issueA2 := createPlanAndIssue(ctx, t, ctl, fixture.ProjectA, fixture.DatabaseA, "collision-audit-test")

	// Snapshot B before the action under test so we can detect any leak.
	beforeB := snapshotProject(ctx, t, ctl, fixture.ProjectB)

	// In project A, mutate the new plan's spec in a way that produces a
	// PlanUpdate audit row: flip enable_prior_backup from false to true on
	// the existing CDC spec. (Targets/sheet stay identical so the audit
	// helper emits exactly one row carrying only the bool diff.)
	a.NotEmpty(planA2.Specs, "fresh plan A must have at least one spec")
	originalSpec := planA2.Specs[0]
	cdc := originalSpec.GetChangeDatabaseConfig()
	a.NotNil(cdc, "fresh plan A spec is expected to be a ChangeDatabaseConfig")

	_, err = ctl.planServiceClient.UpdatePlan(ctx, connect.NewRequest(&v1pb.UpdatePlanRequest{
		Plan: &v1pb.Plan{
			Name: planA2.Name,
			Specs: []*v1pb.Plan_Spec{{
				Id: originalSpec.Id,
				Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{
					ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
						Targets:           cdc.Targets,
						Sheet:             cdc.Sheet,
						EnablePriorBackup: true, // flip
					},
				},
			}},
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"specs"}},
	}))
	a.NoError(err)

	// Positive sanity: the fresh issue in project A gained the audit row.
	commentsA, err := ctl.issueServiceClient.ListIssueComments(ctx,
		connect.NewRequest(&v1pb.ListIssueCommentsRequest{Parent: issueA2.Name}))
	a.NoError(err)
	var sawUpdateA bool
	for _, c := range commentsA.Msg.IssueComments {
		if c.GetPlanUpdate() != nil {
			sawUpdateA = true
			break
		}
	}
	a.True(sawUpdateA, "project A's fresh issue should have received a PlanUpdate audit row")

	// Isolation: project B's snapshot is unchanged across plans, issues,
	// task_runs, plan_check_runs, and (importantly) issue_comments.
	afterB := snapshotProject(ctx, t, ctl, fixture.ProjectB)
	assertProjectUnchanged(t, beforeB, afterB, "project B after plan A spec audit emission")
}
