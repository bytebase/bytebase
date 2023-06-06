package tests

import (
	"context"
	"fmt"

	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// getProjects gets the environments.
func (ctl *controller) getEnvironment(ctx context.Context, id string) (*v1pb.Environment, error) {
	environment, err := ctl.environmentServiceClient.GetEnvironment(ctx,
		&v1pb.GetEnvironmentRequest{
			Name: fmt.Sprintf("environments/%s", id),
		},
	)
	if err != nil {
		return nil, err
	}
	return environment, nil
}
