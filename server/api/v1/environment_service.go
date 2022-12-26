package v1

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/pkg/errors"

	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
	"github.com/bytebase/bytebase/store"
)

const environmentNamePrefix = "environments/"

// EnvironmentService implements the environment service.
type EnvironmentService struct {
	v1pb.UnimplementedEnvironmentServiceServer
	store *store.Store
}

// NewEnvironmentService creates a new EnvironmentService.
func NewEnvironmentService(store *store.Store) *EnvironmentService {
	return &EnvironmentService{
		store: store,
	}
}

// GetEnvironment gets an environment.
func (s *EnvironmentService) GetEnvironment(ctx context.Context, request *v1pb.GetEnvironmentRequest) (*v1pb.Environment, error) {
	environmentID, err := getEnvironmentID(request.GetName())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	environment, err := s.store.GetEnvironmentV2(ctx, environmentID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if environment == nil {
		return nil, status.Errorf(codes.InvalidArgument, "environment %q not found", environmentID)
	}
	state := v1pb.State_STATE_ACTIVE
	if environment.Deleted {
		state = v1pb.State_STATE_DELETED
	}
	return &v1pb.Environment{
		Name:  request.GetName(),
		Title: environment.Name,
		Order: int32(environment.Order),
		State: state,
	}, nil
}

// ListEnvironments lists all environments.
func (s *EnvironmentService) ListEnvironments(ctx context.Context, request *v1pb.ListEnvironmentsRequest) (*v1pb.ListEnvironmentsResponse, error) {
	environments, err := s.store.ListEnvironmentV2(ctx, request.GetShowDeleted())
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	response := &v1pb.ListEnvironmentsResponse{}
	for _, environment := range environments {
		state := v1pb.State_STATE_ACTIVE
		if environment.Deleted {
			state = v1pb.State_STATE_DELETED
		}
		response.Environments = append(
			response.Environments,
			&v1pb.Environment{
				Name:  fmt.Sprintf("%s%s", environmentNamePrefix, environment.ResourceID),
				Title: environment.Name,
				Order: int32(environment.Order),
				State: state,
			})
	}
	return response, nil
}

// CreateEnvironment creates an environment.
func (*EnvironmentService) CreateEnvironment(_ context.Context, _ *v1pb.CreateEnvironmentRequest) (*v1pb.Environment, error) {
	return nil, nil
}

// UpdateEnvironment updates an environment.
func (*EnvironmentService) UpdateEnvironment(_ context.Context, _ *v1pb.UpdateEnvironmentRequest) (*v1pb.Environment, error) {
	return nil, nil
}

// DeleteEnvironment deletes an environment.
func (*EnvironmentService) DeleteEnvironment(_ context.Context, _ *v1pb.DeleteEnvironmentRequest) (*emptypb.Empty, error) {
	return nil, nil
}

func getEnvironmentID(name string) (string, error) {
	if !strings.HasPrefix(name, environmentNamePrefix) {
		return "", errors.Errorf("invalid environment name %q", name)
	}
	return strings.TrimPrefix(name, environmentNamePrefix), nil
}
