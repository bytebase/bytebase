package v1

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	enterpriseAPI "github.com/bytebase/bytebase/enterprise/api"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
	"github.com/bytebase/bytebase/store"
)

const environmentNamePrefix = "environments/"

// EnvironmentService implements the environment service.
type EnvironmentService struct {
	v1pb.UnimplementedEnvironmentServiceServer
	store          *store.Store
	licenseService enterpriseAPI.LicenseService
}

// NewEnvironmentService creates a new EnvironmentService.
func NewEnvironmentService(store *store.Store, licenseService enterpriseAPI.LicenseService) *EnvironmentService {
	return &EnvironmentService{
		store:          store,
		licenseService: licenseService,
	}
}

// GetEnvironment gets an environment.
func (s *EnvironmentService) GetEnvironment(ctx context.Context, request *v1pb.GetEnvironmentRequest) (*v1pb.Environment, error) {
	environmentID, err := getEnvironmentID(request.Name)
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
	return convertEnvironment(environment), nil
}

// ListEnvironments lists all environments.
func (s *EnvironmentService) ListEnvironments(ctx context.Context, request *v1pb.ListEnvironmentsRequest) (*v1pb.ListEnvironmentsResponse, error) {
	environments, err := s.store.ListEnvironmentV2(ctx, request.ShowDeleted)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	response := &v1pb.ListEnvironmentsResponse{}
	for _, environment := range environments {
		response.Environments = append(response.Environments, convertEnvironment(environment))
	}
	return response, nil
}

// CreateEnvironment creates an environment.
func (s *EnvironmentService) CreateEnvironment(ctx context.Context, request *v1pb.CreateEnvironmentRequest) (*v1pb.Environment, error) {
	principalID := ctx.Value(common.PrincipalIDContextKey).(int)
	if request.Environment == nil {
		return nil, status.Errorf(codes.InvalidArgument, "environment must be set")
	}

	if err := api.IsValidEnvironmentName(request.Environment.Title); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid environment title, please visit https://www.bytebase.com/docs/vcs-integration/name-and-organize-schema-files#file-path-template?source=console to get more detail.")
	}
	if !isValidResourceID(request.EnvironmentId) {
		return nil, status.Errorf(codes.InvalidArgument, "invalid environment ID %v", request.EnvironmentId)
	}

	// Environment limit in the plan.
	environments, err := s.store.ListEnvironmentV2(ctx, false /* showDeleted */)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	maximumEnvironmentLimit := s.licenseService.GetPlanLimitValue(api.PlanLimitMaximumEnvironment)
	if int64(len(environments)) >= maximumEnvironmentLimit {
		return nil, status.Errorf(codes.ResourceExhausted, "current plan can create up to %d environments.", maximumEnvironmentLimit)
	}

	environment, err := s.store.CreateEnvironmentV2(ctx,
		&store.EnvironmentMessage{
			EnvironmentID: request.EnvironmentId,
			Title:         request.Environment.Title,
			Order:         request.Environment.Order,
		},
		principalID,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	return convertEnvironment(environment), nil
}

// UpdateEnvironment updates an environment.
func (s *EnvironmentService) UpdateEnvironment(ctx context.Context, request *v1pb.UpdateEnvironmentRequest) (*v1pb.Environment, error) {
	principalID := ctx.Value(common.PrincipalIDContextKey).(int)
	if request.Environment == nil {
		return nil, status.Errorf(codes.InvalidArgument, "environment must be set")
	}
	if request.UpdateMask == nil {
		return nil, status.Errorf(codes.InvalidArgument, "update_mask must be set")
	}
	environmentID, err := getEnvironmentID(request.Environment.Name)
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
	if environment.Deleted {
		return nil, status.Errorf(codes.InvalidArgument, "environment %q has been deleted", environmentID)
	}

	patch := &store.UpdateEnvironmentMessage{}
	for _, path := range request.UpdateMask.Paths {
		switch path {
		case "environment.title":
			patch.Name = &request.Environment.Title
		case "environment.order":
			patch.Order = &request.Environment.Order
		}
	}

	environment, err = s.store.UpdateEnvironmentV2(ctx,
		environmentID,
		patch,
		principalID,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	return convertEnvironment(environment), nil
}

// DeleteEnvironment deletes an environment.
func (s *EnvironmentService) DeleteEnvironment(ctx context.Context, request *v1pb.DeleteEnvironmentRequest) (*emptypb.Empty, error) {
	principalID := ctx.Value(common.PrincipalIDContextKey).(int)
	environmentID, err := getEnvironmentID(request.Name)
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
	if environment.Deleted {
		return nil, status.Errorf(codes.InvalidArgument, "environment %q has been deleted", environmentID)
	}

	// All instances in the environment must be deleted.
	instances, err := s.store.ListInstancesV2(ctx, &store.FindInstanceMessage{
		EnvironmentID: &environmentID,
		ShowDeleted:   false,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if len(instances) > 0 {
		return nil, status.Errorf(codes.FailedPrecondition, "all instances in the environment should be deleted")
	}

	if err := s.store.DeleteOrUndeleteEnvironmentV2(ctx, environmentID, true /* delete */, principalID); err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	return &emptypb.Empty{}, nil
}

// UndeleteEnvironment undeletes an environment.
func (s *EnvironmentService) UndeleteEnvironment(ctx context.Context, request *v1pb.UndeleteEnvironmentRequest) (*v1pb.Environment, error) {
	principalID := ctx.Value(common.PrincipalIDContextKey).(int)
	environmentID, err := getEnvironmentID(request.Name)
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
	if !environment.Deleted {
		return nil, status.Errorf(codes.InvalidArgument, "environment %q is active", environmentID)
	}

	if err := s.store.DeleteOrUndeleteEnvironmentV2(ctx, environmentID, false /* delete */, principalID); err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	resp := convertEnvironment(environment)
	resp.State = v1pb.State_STATE_ACTIVE
	return resp, nil
}

func getEnvironmentID(name string) (string, error) {
	if !strings.HasPrefix(name, environmentNamePrefix) {
		return "", errors.Errorf("invalid environment name %q", name)
	}
	environmentID := strings.TrimPrefix(name, environmentNamePrefix)
	if environmentID == "" {
		return "", errors.Errorf("environment cannot be empty")
	}
	return environmentID, nil
}

func convertEnvironment(environment *store.EnvironmentMessage) *v1pb.Environment {
	return &v1pb.Environment{
		Name:  fmt.Sprintf("%s%s", environmentNamePrefix, environment.EnvironmentID),
		Title: environment.Title,
		Order: environment.Order,
		State: convertDeletedToState(environment.Deleted),
	}
}
