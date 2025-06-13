package tests

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func (ctl *controller) createDatabaseV2(ctx context.Context, project *v1pb.Project, instance *v1pb.Instance, environment *v1pb.EnvironmentSetting_Environment, databaseName string, owner string) error {
	characterSet, collation := "utf8mb4", "utf8mb4_general_ci"
	if instance.Engine == v1pb.Engine_POSTGRES {
		characterSet = "UTF8"
		collation = "en_US.UTF-8"
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
	rolloutResp, err := ctl.rolloutServiceClient.CreateRollout(ctx, connect.NewRequest(&v1pb.CreateRolloutRequest{Parent: project.Name, Rollout: &v1pb.Rollout{Plan: planResp.Msg.Name}}))
	if err != nil {
		return err
	}

	if err := ctl.waitRollout(ctx, issueResp.Msg.Name, rolloutResp.Msg.Name); err != nil {
		return err
	}

	_, err = ctl.issueServiceClient.BatchUpdateIssuesStatus(ctx, connect.NewRequest(&v1pb.BatchUpdateIssuesStatusRequest{
		Parent: project.Name,
		Issues: []string{issueResp.Msg.Name},
		Status: v1pb.IssueStatus_DONE,
	}))
	if err != nil {
		return err
	}
	return nil
}
