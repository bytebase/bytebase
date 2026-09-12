package tests

import (
	"context"
	"fmt"
	"slices"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// allowDatabaseSync names one more database an instance may sync. The syncer
// imports only what the list names, so a database about to be created has to be
// added first. An instance with a server to itself carries no list and skips this.
func (ctl *controller) allowDatabaseSync(ctx context.Context, instance *v1pb.Instance, databaseName string) error {
	if instance.GetSyncDatabases() == nil {
		return nil
	}
	resp, err := ctl.instanceServiceClient.GetInstance(ctx, connect.NewRequest(&v1pb.GetInstanceRequest{Name: instance.Name}))
	if err != nil {
		return err
	}
	databases := resp.Msg.GetSyncDatabases().GetDatabases()
	if slices.Contains(databases, databaseName) {
		return nil
	}
	_, err = ctl.instanceServiceClient.UpdateInstance(ctx, connect.NewRequest(&v1pb.UpdateInstanceRequest{
		Instance: &v1pb.Instance{
			Name:          instance.Name,
			SyncDatabases: &v1pb.SyncDatabases{Databases: append(databases, databaseName)},
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"sync_databases"}},
	}))
	return err
}

func (ctl *controller) createDatabase(ctx context.Context, project *v1pb.Project, instance *v1pb.Instance, environment *v1pb.EnvironmentSetting_Environment, databaseName string, owner string) error {
	if err := ctl.allowDatabaseSync(ctx, instance, databaseName); err != nil {
		return err
	}

	// Database provisioning is test setup, and its plan is approved by the same
	// test actor. Opt into that legacy behavior explicitly; production projects
	// keep the restrictive default when the setting is absent.
	if _, err := ctl.projectServiceClient.UpdateProject(ctx, connect.NewRequest(&v1pb.UpdateProjectRequest{
		Project: &v1pb.Project{
			Name:                        project.Name,
			AllowLastPlanEditorApproval: true,
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"allow_last_plan_editor_approval"}},
	})); err != nil {
		return err
	}

	characterSet, collation := "utf8mb4", "utf8mb4_general_ci"
	if instance.Engine == v1pb.Engine_POSTGRES {
		characterSet = "UTF8"
		collation = "en_US.UTF-8"
		if owner == "" {
			owner = "postgres"
		}
	}
	environmentName := ""
	if environment != nil {
		environmentName = environment.Name
	}

	planResp, err := ctl.planServiceClient.CreatePlan(ctx, connect.NewRequest(&v1pb.CreatePlanRequest{
		Parent: project.Name,
		Plan: &v1pb.Plan{
			Specs: []*v1pb.Plan_Spec{
				{
					Id: uuid.NewString(),
					Config: &v1pb.Plan_Spec_CreateDatabaseConfig{
						CreateDatabaseConfig: &v1pb.Plan_CreateDatabaseConfig{
							Target:       instance.Name,
							Database:     databaseName,
							CharacterSet: characterSet,
							Collation:    collation,
							Owner:        owner,
							Environment:  environmentName,
						},
					},
				},
			},
		},
	}))
	if err != nil {
		return err
	}
	issueResp, err := ctl.issueServiceClient.CreateIssue(ctx, connect.NewRequest(&v1pb.CreateIssueRequest{
		Parent: project.Name,
		Issue: &v1pb.Issue{
			Title:       fmt.Sprintf("create database %q", databaseName),
			Description: fmt.Sprintf("This creates a database %q.", databaseName),
			Plan:        planResp.Msg.Name,
			Type:        v1pb.Issue_DATABASE_CHANGE,
		},
	}))
	if err != nil {
		return err
	}
	rolloutResp, err := ctl.rolloutServiceClient.CreateRollout(ctx, connect.NewRequest(&v1pb.CreateRolloutRequest{Parent: planResp.Msg.Name}))
	if err != nil {
		return err
	}

	return ctl.waitRollout(ctx, issueResp.Msg.Name, rolloutResp.Msg.Name)
}
