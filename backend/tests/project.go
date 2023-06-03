package tests

import (
	"context"
	"fmt"

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
