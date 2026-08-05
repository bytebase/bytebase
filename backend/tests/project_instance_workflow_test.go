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

func TestProjectInstanceWorkflowTargets(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	pg, err := provisionPgInstance(ctx, t)
	a.NoError(err)
	const databaseID = "bot36_workflow_database"
	createPgDatabase(t, pg, databaseID)

	otherProject := createProjectForProjectInstanceTest(ctx, t, ctl, "bot36-other-project")
	instance := createProjectInstanceTestInstance(ctx, t, ctl, &ctl.project.Name, "bot36-project-instance", "project instance", pg)
	_, err = ctl.instanceServiceClient.SyncInstance(ctx, connect.NewRequest(&v1pb.SyncInstanceRequest{Name: instance.Name}))
	a.NoError(err)

	databaseName := fmt.Sprintf("%s/databases/%s", instance.Name, databaseID)
	database, err := ctl.databaseServiceClient.GetDatabase(ctx, connect.NewRequest(&v1pb.GetDatabaseRequest{Name: databaseName}))
	a.NoError(err)
	a.Equal(databaseName, database.Msg.Name)

	sheet, err := ctl.sheetServiceClient.CreateSheet(ctx, connect.NewRequest(&v1pb.CreateSheetRequest{
		Parent: ctl.project.Name,
		Sheet:  &v1pb.Sheet{Content: []byte("SELECT 1;")},
	}))
	a.NoError(err)

	changeSpec := func(target string) *v1pb.Plan_Spec {
		return &v1pb.Plan_Spec{
			Id: uuid.NewString(),
			Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{
				ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
					Targets: []string{target},
					Sheet:   sheet.Msg.Name,
				},
			},
		}
	}

	plan, err := ctl.planServiceClient.CreatePlan(ctx, connect.NewRequest(&v1pb.CreatePlanRequest{
		Parent: ctl.project.Name,
		Plan:   &v1pb.Plan{Specs: []*v1pb.Plan_Spec{changeSpec(databaseName)}},
	}))
	a.NoError(err)
	a.Equal(databaseName, plan.Msg.Specs[0].GetChangeDatabaseConfig().Targets[0])

	updated, err := ctl.planServiceClient.UpdatePlan(ctx, connect.NewRequest(&v1pb.UpdatePlanRequest{
		Plan: &v1pb.Plan{
			Name:  plan.Msg.Name,
			Specs: []*v1pb.Plan_Spec{changeSpec(databaseName)},
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"specs"}},
	}))
	a.NoError(err)
	a.Equal(databaseName, updated.Msg.Specs[0].GetChangeDatabaseConfig().Targets[0])
	crossProjectDatabaseName := fmt.Sprintf("%s/instances/%s/databases/%s", otherProject.Name, "bot36-project-instance", databaseID)
	_, err = ctl.planServiceClient.UpdatePlan(ctx, connect.NewRequest(&v1pb.UpdatePlanRequest{
		Plan: &v1pb.Plan{
			Name:  plan.Msg.Name,
			Specs: []*v1pb.Plan_Spec{changeSpec(crossProjectDatabaseName)},
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"specs"}},
	}))
	a.Error(err)
	a.Equal(connect.CodeInvalidArgument, connect.CodeOf(err))
	workspaceAliasDatabase := fmt.Sprintf("instances/%s/databases/%s", "bot36-project-instance", databaseID)
	_, err = ctl.planServiceClient.CreatePlan(ctx, connect.NewRequest(&v1pb.CreatePlanRequest{
		Parent: ctl.project.Name,
		Plan:   &v1pb.Plan{Specs: []*v1pb.Plan_Spec{changeSpec(workspaceAliasDatabase)}},
	}))
	a.Error(err)

	_, err = ctl.issueServiceClient.CreateIssue(ctx, connect.NewRequest(&v1pb.CreateIssueRequest{
		Parent: ctl.project.Name,
		Issue: &v1pb.Issue{
			Title: "project instance workflow target",
			Type:  v1pb.Issue_DATABASE_CHANGE,
			Plan:  plan.Msg.Name,
		},
	}))
	a.NoError(err)
	rollout, err := ctl.rolloutServiceClient.CreateRollout(ctx, connect.NewRequest(&v1pb.CreateRolloutRequest{Parent: plan.Msg.Name}))
	a.NoError(err)
	a.Equal(databaseName, rollout.Msg.Stages[0].Tasks[0].Target)

	_, err = ctl.planServiceClient.CreatePlan(ctx, connect.NewRequest(&v1pb.CreatePlanRequest{
		Parent: otherProject.Name,
		Plan:   &v1pb.Plan{Specs: []*v1pb.Plan_Spec{changeSpec(databaseName)}},
	}))
	a.Error(err)
	a.Equal(connect.CodeInvalidArgument, connect.CodeOf(err))

	createSpec := func(target string) *v1pb.Plan_Spec {
		return &v1pb.Plan_Spec{
			Id: uuid.NewString(),
			Config: &v1pb.Plan_Spec_CreateDatabaseConfig{
				CreateDatabaseConfig: &v1pb.Plan_CreateDatabaseConfig{
					Target:       target,
					Database:     "bot36_create_database",
					CharacterSet: "UTF8",
					Collation:    "en_US.UTF-8",
					Owner:        "postgres",
				},
			},
		}
	}
	created, err := ctl.planServiceClient.CreatePlan(ctx, connect.NewRequest(&v1pb.CreatePlanRequest{
		Parent: ctl.project.Name,
		Plan:   &v1pb.Plan{Specs: []*v1pb.Plan_Spec{createSpec(instance.Name)}},
	}))
	a.NoError(err)
	a.Equal(instance.Name, created.Msg.Specs[0].GetCreateDatabaseConfig().Target)
	_, err = ctl.planServiceClient.CreatePlan(ctx, connect.NewRequest(&v1pb.CreatePlanRequest{
		Parent: ctl.project.Name,
		Plan:   &v1pb.Plan{Specs: []*v1pb.Plan_Spec{createSpec("instances/bot36-project-instance")}},
	}))
	a.Error(err)
	_, err = ctl.issueServiceClient.CreateIssue(ctx, connect.NewRequest(&v1pb.CreateIssueRequest{
		Parent: ctl.project.Name,
		Issue: &v1pb.Issue{
			Title: "project instance create database target",
			Type:  v1pb.Issue_DATABASE_CHANGE,
			Plan:  created.Msg.Name,
		},
	}))
	a.NoError(err)
	createRollout, err := ctl.rolloutServiceClient.CreateRollout(ctx, connect.NewRequest(&v1pb.CreateRolloutRequest{Parent: created.Msg.Name}))
	a.NoError(err)
	a.Equal(instance.Name, createRollout.Msg.Stages[0].Tasks[0].Target)

	_, err = ctl.planServiceClient.CreatePlan(ctx, connect.NewRequest(&v1pb.CreatePlanRequest{
		Parent: otherProject.Name,
		Plan:   &v1pb.Plan{Specs: []*v1pb.Plan_Spec{createSpec(instance.Name)}},
	}))
	a.Error(err)
	a.Equal(connect.CodeInvalidArgument, connect.CodeOf(err))
}
