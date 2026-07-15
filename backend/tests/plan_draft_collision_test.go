package tests

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/bytebase/bytebase/backend/common"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

func TestCollision_PlanDraftMetadataSync(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	require.NoError(t, err)
	defer ctl.Close(ctx)

	fixture := setupCollidingProjects(ctx, t, ctl)
	createDraft := func(project *v1pb.Project, database *v1pb.Database, title string) (*v1pb.Plan, *v1pb.Issue) {
		t.Helper()
		sheet, err := ctl.sheetServiceClient.CreateSheet(ctx, connect.NewRequest(&v1pb.CreateSheetRequest{
			Parent: project.Name,
			Sheet:  &v1pb.Sheet{Content: []byte("SELECT 1;")},
		}))
		require.NoError(t, err)
		plan, err := ctl.planServiceClient.CreatePlan(ctx, connect.NewRequest(&v1pb.CreatePlanRequest{
			Parent: project.Name,
			Plan: &v1pb.Plan{
				Title: title,
				Specs: []*v1pb.Plan_Spec{{
					Id: uuid.NewString(),
					Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{
						ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
							Targets: []string{database.Name},
							Sheet:   sheet.Msg.Name,
						},
					},
				}},
			},
		}))
		require.NoError(t, err)
		issue, err := ctl.issueServiceClient.CreateIssue(ctx, connect.NewRequest(&v1pb.CreateIssueRequest{
			Parent: project.Name,
			Issue: &v1pb.Issue{
				Title: title,
				Type:  v1pb.Issue_DATABASE_CHANGE,
				Plan:  plan.Msg.Name,
				Draft: true,
			},
		}))
		require.NoError(t, err)
		return plan.Msg, issue.Msg
	}

	planA, issueA := createDraft(fixture.ProjectA, fixture.DatabaseA, "Draft A")
	_, issueB := createDraft(fixture.ProjectB, fixture.DatabaseB, "Draft B")
	beforeB := snapshotProject(ctx, t, ctl, fixture.ProjectB)

	const updatedTitle = "Updated draft A"
	_, err = ctl.planServiceClient.UpdatePlan(ctx, connect.NewRequest(&v1pb.UpdatePlanRequest{
		Plan:       &v1pb.Plan{Name: planA.Name, Title: updatedTitle},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"title"}},
	}))
	require.NoError(t, err)

	updatedIssueA, err := ctl.issueServiceClient.GetIssue(ctx, connect.NewRequest(&v1pb.GetIssueRequest{Name: issueA.Name}))
	require.NoError(t, err)
	require.Equal(t, updatedTitle, updatedIssueA.Msg.Title)
	require.True(t, updatedIssueA.Msg.Draft)
	_, issueAUID, err := common.GetProjectIDIssueUID(issueA.Name)
	require.NoError(t, err)
	_, issueBUID, err := common.GetProjectIDIssueUID(issueB.Name)
	require.NoError(t, err)
	require.Equal(t, issueAUID, issueBUID, "draft issue IDs must collide across projects")

	afterB := snapshotProject(ctx, t, ctl, fixture.ProjectB)
	assertProjectUnchanged(t, beforeB, afterB, "project B after Plan A draft metadata sync")
}
