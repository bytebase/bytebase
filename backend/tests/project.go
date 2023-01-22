package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/google/jsonapi"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/api"
)

// createProject creates an project.
func (ctl *controller) createProject(projectCreate api.ProjectCreate) (*api.Project, error) {
	buf := new(bytes.Buffer)
	if err := jsonapi.MarshalPayload(buf, &projectCreate); err != nil {
		return nil, errors.Wrap(err, "failed to marshal project create")
	}

	body, err := ctl.post("/project", buf)
	if err != nil {
		return nil, err
	}

	project := new(api.Project)
	if err = jsonapi.UnmarshalPayload(body, project); err != nil {
		return nil, errors.Wrap(err, "fail to unmarshal post project response")
	}
	return project, nil
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

func (ctl *controller) patchProject(projectPatch api.ProjectPatch) error {
	buf := new(bytes.Buffer)
	if err := jsonapi.MarshalPayload(buf, &projectPatch); err != nil {
		return errors.Wrap(err, "failed to marshal project patch")
	}

	_, err := ctl.patch(fmt.Sprintf("/project/%d", projectPatch.ID), buf)
	if err != nil {
		return err
	}

	return nil
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
