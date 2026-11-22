package tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
)

// TestCreateDatabasePlanCannotTakeOverAnotherProjectsDatabase drives a
// create-database plan that names a database already live in another project,
// as a Project Owner of the plan's project only. The task fails and the
// existing database keeps its project, environment and labels.
func TestCreateDatabasePlanCannotTakeOverAnotherProjectsDatabase(t *testing.T) {
	t.Parallel()
	ctl, ctx := startWorkspace(context.Background(), t)

	instance := createPgInstance(ctx, t, ctl, "takeover")
	victim := uniqueDB("victim")
	require.NoError(t, ctl.createDatabase(ctx, ctl.project, instance, nil, victim, ""))
	victimName := fmt.Sprintf("%s/databases/%s", instance.Name, victim)
	prod := "environments/prod"
	_, err := ctl.databaseServiceClient.UpdateDatabase(ctx, connect.NewRequest(&v1pb.UpdateDatabaseRequest{
		Database:   &v1pb.Database{Name: victimName, Labels: map[string]string{"tier": "pci"}, Environment: &prod},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"labels", "environment"}},
	}))
	require.NoError(t, err)
	before, err := ctl.databaseServiceClient.GetDatabase(ctx, connect.NewRequest(&v1pb.GetDatabaseRequest{Name: victimName}))
	require.NoError(t, err)

	// A sandbox project whose owner holds no role on the victim's project.
	sandbox, err := ctl.projectServiceClient.CreateProject(ctx, connect.NewRequest(&v1pb.CreateProjectRequest{
		ProjectId: "sandbox",
		Project:   &v1pb.Project{Title: "sandbox"},
	}))
	require.NoError(t, err)
	token := bot35CreateProjectUser(ctx, t, ctl, sandbox.Msg.Name, "roles/projectOwner", "owner")
	auth := connect.WithInterceptors(&authInterceptor{token: token})
	ownerDatabases := v1connect.NewDatabaseServiceClient(ctl.client, ctl.rootURL, auth)
	ownerPlans := v1connect.NewPlanServiceClient(ctl.client, ctl.rootURL, auth)
	ownerRollouts := v1connect.NewRolloutServiceClient(ctl.client, ctl.rootURL, auth)

	_, err = ownerDatabases.GetDatabase(ctx, connect.NewRequest(&v1pb.GetDatabaseRequest{Name: victimName}))
	require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err), "the sandbox owner holds no role on the victim's project")

	// The owner drives the whole change: plan, rollout and task run.
	plan, err := ownerPlans.CreatePlan(ctx, connect.NewRequest(&v1pb.CreatePlanRequest{
		Parent: sandbox.Msg.Name,
		Plan: &v1pb.Plan{Title: "create", Specs: []*v1pb.Plan_Spec{{
			Id: uuid.NewString(),
			Config: &v1pb.Plan_Spec_CreateDatabaseConfig{CreateDatabaseConfig: &v1pb.Plan_CreateDatabaseConfig{
				Target:       instance.Name,
				Database:     victim,
				CharacterSet: "UTF8",
				Collation:    "en_US.UTF-8",
				Owner:        "postgres",
				Environment:  "environments/test",
			}},
		}}},
	}))
	require.NoError(t, err)
	rollout, err := ownerRollouts.CreateRollout(ctx, connect.NewRequest(&v1pb.CreateRolloutRequest{Parent: plan.Msg.Name}))
	require.NoError(t, err)
	task := rollout.Msg.Stages[0].Tasks[0]
	_, err = ownerRollouts.BatchRunTasks(ctx, connect.NewRequest(&v1pb.BatchRunTasksRequest{
		Parent: rollout.Msg.Name + "/stages/-", Tasks: []string{task.Name},
	}))
	require.NoError(t, err)

	var status v1pb.Task_Status
	var detail string
	deadline := time.Now().Add(time.Minute)
	for time.Now().Before(deadline) {
		got, err := ownerRollouts.GetRollout(ctx, connect.NewRequest(&v1pb.GetRolloutRequest{Name: rollout.Msg.Name}))
		require.NoError(t, err)
		status = got.Msg.Stages[0].Tasks[0].Status
		if status == v1pb.Task_DONE || status == v1pb.Task_FAILED {
			runs, err := ownerRollouts.ListTaskRuns(ctx, connect.NewRequest(&v1pb.ListTaskRunsRequest{Parent: got.Msg.Stages[0].Tasks[0].Name}))
			require.NoError(t, err)
			if len(runs.Msg.TaskRuns) > 0 {
				detail = runs.Msg.TaskRuns[0].Detail
			}
			break
		}
		time.Sleep(pollInterval)
	}
	require.Equal(t, v1pb.Task_FAILED, status, "the create-database task must fail rather than take over the existing database")
	require.Contains(t, detail, "already belongs to another project")

	after, err := ctl.databaseServiceClient.GetDatabase(ctx, connect.NewRequest(&v1pb.GetDatabaseRequest{Name: victimName}))
	require.NoError(t, err)
	require.Equal(t, before.Msg.Project, after.Msg.Project, "the existing database must keep its project")
	require.Equal(t, before.Msg.GetEnvironment(), after.Msg.GetEnvironment(), "the existing database must keep its environment")
	require.Equal(t, before.Msg.Labels, after.Msg.Labels, "the existing database must keep its labels")
}
