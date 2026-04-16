package tests

import (
	"context"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

func TestUpdatePlanCreatesPlanSpecUpdateIssueComment(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	instanceRootDir := t.TempDir()
	instanceName := "testPlanUpdateInstance"
	instanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, instanceName)
	a.NoError(err)

	instanceResp, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance"),
		Instance: &v1pb.Instance{
			Title:       instanceName,
			Engine:      v1pb.Engine_SQLITE,
			Environment: new("environments/prod"),
			Activation:  true,
			DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: instanceDir, Id: "admin"}},
		},
	}))
	a.NoError(err)
	instance := instanceResp.Msg

	databaseName := "testPlanUpdateDb"
	err = ctl.createDatabase(ctx, ctl.project, instance, nil, databaseName, "")
	a.NoError(err)

	databaseResp, err := ctl.databaseServiceClient.GetDatabase(ctx, connect.NewRequest(&v1pb.GetDatabaseRequest{
		Name: fmt.Sprintf("%s/databases/%s", instance.Name, databaseName),
	}))
	a.NoError(err)
	database := databaseResp.Msg

	sheet1Resp, err := ctl.sheetServiceClient.CreateSheet(ctx, connect.NewRequest(&v1pb.CreateSheetRequest{
		Parent: ctl.project.Name,
		Sheet: &v1pb.Sheet{
			Content: []byte("CREATE TABLE plan_update_test_1 (id INTEGER PRIMARY KEY);"),
		},
	}))
	a.NoError(err)
	sheet1 := sheet1Resp.Msg

	sheet2Resp, err := ctl.sheetServiceClient.CreateSheet(ctx, connect.NewRequest(&v1pb.CreateSheetRequest{
		Parent: ctl.project.Name,
		Sheet: &v1pb.Sheet{
			Content: []byte("CREATE TABLE plan_update_test_2 (id INTEGER PRIMARY KEY);"),
		},
	}))
	a.NoError(err)
	sheet2 := sheet2Resp.Msg

	specID := uuid.NewString()
	planResp, err := ctl.planServiceClient.CreatePlan(ctx, connect.NewRequest(&v1pb.CreatePlanRequest{
		Parent: ctl.project.Name,
		Plan: &v1pb.Plan{
			Specs: []*v1pb.Plan_Spec{
				{
					Id: specID,
					Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{
						ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
							Targets: []string{database.Name},
							Sheet:   sheet1.Name,
						},
					},
				},
			},
		},
	}))
	a.NoError(err)
	plan := planResp.Msg

	issueResp, err := ctl.issueServiceClient.CreateIssue(ctx, connect.NewRequest(&v1pb.CreateIssueRequest{
		Parent: ctl.project.Name,
		Issue: &v1pb.Issue{
			Type:        v1pb.Issue_DATABASE_CHANGE,
			Title:       fmt.Sprintf("change database %s", database.Name),
			Description: fmt.Sprintf("change database %s", database.Name),
			Plan:        plan.Name,
		},
	}))
	a.NoError(err)
	issue := issueResp.Msg

	_, err = ctl.planServiceClient.UpdatePlan(ctx, connect.NewRequest(&v1pb.UpdatePlanRequest{
		Plan: &v1pb.Plan{
			Name: plan.Name,
			Specs: []*v1pb.Plan_Spec{
				{
					Id: specID,
					Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{
						ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
							Targets: []string{database.Name},
							Sheet:   sheet2.Name,
						},
					},
				},
			},
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"specs"}},
	}))
	a.NoError(err)

	commentsResp, err := ctl.issueServiceClient.ListIssueComments(ctx, connect.NewRequest(&v1pb.ListIssueCommentsRequest{
		Parent: issue.Name,
	}))
	a.NoError(err)

	var planSpecUpdates []*v1pb.IssueComment_PlanSpecUpdate
	for _, comment := range commentsResp.Msg.IssueComments {
		if planSpecUpdate := comment.GetPlanSpecUpdate(); planSpecUpdate != nil {
			planSpecUpdates = append(planSpecUpdates, planSpecUpdate)
		}
	}
	a.Len(planSpecUpdates, 1)
	a.Equal(fmt.Sprintf("%s/specs/%s", plan.Name, specID), planSpecUpdates[0].Spec)
	a.Equal(sheet1.Name, planSpecUpdates[0].GetFromSheet())
	a.Equal(sheet2.Name, planSpecUpdates[0].GetToSheet())
}
