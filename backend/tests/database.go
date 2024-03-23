package tests

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	api "github.com/bytebase/bytebase/backend/legacyapi"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func (ctl *controller) createDatabaseV2(ctx context.Context, project *v1pb.Project, instance *v1pb.Instance, environment *v1pb.Environment, databaseName string, owner string, labels map[string]string) error {
	characterSet, collation := "utf8mb4", "utf8mb4_general_ci"
	if instance.Engine == v1pb.Engine_POSTGRES {
		characterSet = "UTF8"
		collation = "en_US.UTF-8"
	}
	environmentName := ""
	if environment != nil {
		environmentName = environment.Name
	}

	plan, err := ctl.rolloutServiceClient.CreatePlan(ctx, &v1pb.CreatePlanRequest{
		Parent: project.Name,
		Plan: &v1pb.Plan{
			Steps: []*v1pb.Plan_Step{
				{
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
									Labels:       labels,
								},
							},
						},
					},
				},
			},
		},
	})
	if err != nil {
		return err
	}
	issue, err := ctl.issueServiceClient.CreateIssue(ctx, &v1pb.CreateIssueRequest{
		Parent: project.Name,
		Issue: &v1pb.Issue{
			Title:       fmt.Sprintf("create database %q", databaseName),
			Description: fmt.Sprintf("This creates a database %q.", databaseName),
			Plan:        plan.Name,
			Type:        v1pb.Issue_DATABASE_CHANGE,
			Assignee:    fmt.Sprintf("users/%s", api.SystemBotEmail),
		},
	})
	if err != nil {
		return err
	}
	rollout, err := ctl.rolloutServiceClient.CreateRollout(ctx, &v1pb.CreateRolloutRequest{Parent: project.Name, Rollout: &v1pb.Rollout{Plan: plan.Name}})
	if err != nil {
		return err
	}

	if err := ctl.waitRollout(ctx, issue.Name, rollout.Name); err != nil {
		return err
	}

	_, err = ctl.issueServiceClient.BatchUpdateIssuesStatus(ctx, &v1pb.BatchUpdateIssuesStatusRequest{
		Parent: project.Name,
		Issues: []string{issue.Name},
		Status: v1pb.IssueStatus_DONE,
	})
	if err != nil {
		return err
	}
	return nil
}
