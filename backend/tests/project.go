package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/google/jsonapi"
	"github.com/pkg/errors"

	api "github.com/bytebase/bytebase/backend/legacyapi"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// createProject creates an project.
func (ctl *controller) createProject(ctx context.Context) (*v1pb.Project, error) {
	projectID := generateRandomString("project", 10)
	return ctl.projectServiceClient.CreateProject(ctx, &v1pb.CreateProjectRequest{
		Project: &v1pb.Project{
			Name:  fmt.Sprintf("projects/%s", projectID),
			Title: projectID,
			Key:   projectID,
		},
		ProjectId: projectID,
	})
}

// getProjects gets the projects.
func (ctl *controller) getProjects() ([]*api.Project, error) {
	body, err := ctl.get("/project", nil)
	if err != nil {
		return nil, err
	}

	var projects []*api.Project
	ps, err := jsonapi.UnmarshalManyPayload(body, reflect.TypeOf(new(api.Project)))
	if err != nil {
		return nil, errors.Wrap(err, "fail to unmarshal get project response")
	}
	for _, p := range ps {
		project, ok := p.(*api.Project)
		if !ok {
			return nil, errors.Errorf("fail to convert project")
		}
		projects = append(projects, project)
	}
	return projects, nil
}

// createSQLReviewCI set up the SQL review CI.
func (ctl *controller) createSQLReviewCI(projectID, repositoryID int) (*api.SQLReviewCISetup, error) {
	body, err := ctl.post(fmt.Sprintf("/project/%d/repository/%d/sql-review-ci", projectID, repositoryID), new(bytes.Buffer))
	if err != nil {
		return nil, err
	}

	sqlReviewCISetup := new(api.SQLReviewCISetup)
	if err = jsonapi.UnmarshalPayload(body, sqlReviewCISetup); err != nil {
		return nil, errors.Wrap(err, "fail to unmarshal SQL reivew CI response")
	}
	return sqlReviewCISetup, nil
}

// upsertDeploymentConfig upserts the deployment configuration for a project.
func (ctl *controller) upsertDeploymentConfig(deploymentConfigUpsert api.DeploymentConfigUpsert, deploymentSchedule api.DeploymentSchedule) (*api.DeploymentConfig, error) {
	scheduleBuf, err := json.Marshal(&deploymentSchedule)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal deployment schedule")
	}
	deploymentConfigUpsert.Payload = string(scheduleBuf)

	buf := new(bytes.Buffer)
	if err := jsonapi.MarshalPayload(buf, &deploymentConfigUpsert); err != nil {
		return nil, errors.Wrap(err, "failed to marshal deployment config upsert")
	}

	body, err := ctl.patch(fmt.Sprintf("/project/%d/deployment", deploymentConfigUpsert.ProjectID), buf)
	if err != nil {
		return nil, err
	}

	deploymentConfig := new(api.DeploymentConfig)
	if err = jsonapi.UnmarshalPayload(body, deploymentConfig); err != nil {
		return nil, errors.Wrap(err, "fail to unmarshal upsert deployment config response")
	}
	return deploymentConfig, nil
}
