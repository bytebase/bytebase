package v1

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/bytebase/bytebase/backend/common"
	enterpriseAPI "github.com/bytebase/bytebase/backend/enterprise/api"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

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
	environment, err := s.getEnvironmentMessage(ctx, request.Name)
	if err != nil {
		return nil, err
	}
	return convertToEnvironment(environment), nil
}

// ListEnvironments lists all environments.
func (s *EnvironmentService) ListEnvironments(ctx context.Context, request *v1pb.ListEnvironmentsRequest) (*v1pb.ListEnvironmentsResponse, error) {
	environments, err := s.store.ListEnvironmentV2(ctx, &store.FindEnvironmentMessage{ShowDeleted: request.ShowDeleted})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	response := &v1pb.ListEnvironmentsResponse{}
	for _, environment := range environments {
		response.Environments = append(response.Environments, convertToEnvironment(environment))
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
	environments, err := s.store.ListEnvironmentV2(ctx, &store.FindEnvironmentMessage{ShowDeleted: false})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	maximumEnvironmentLimit := s.licenseService.GetPlanLimitValue(api.PlanLimitMaximumEnvironment)
	if int64(len(environments)) >= maximumEnvironmentLimit {
		return nil, status.Errorf(codes.ResourceExhausted, "current plan can create up to %d environments.", maximumEnvironmentLimit)
	}

	environment, err := s.store.CreateEnvironmentV2(ctx,
		&store.EnvironmentMessage{
			ResourceID: request.EnvironmentId,
			Title:      request.Environment.Title,
			Order:      request.Environment.Order,
			Protected:  request.Environment.Tier == v1pb.EnvironmentTier_PROTECTED,
		},
		principalID,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	return convertToEnvironment(environment), nil
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

	environment, err := s.getEnvironmentMessage(ctx, request.Environment.Name)
	if err != nil {
		return nil, err
	}
	if environment.Deleted {
		return nil, status.Errorf(codes.InvalidArgument, "environment %q has been deleted", request.Environment.Name)
	}

	patch := &store.UpdateEnvironmentMessage{}
	for _, path := range request.UpdateMask.Paths {
		switch path {
		case "environment.title":
			patch.Name = &request.Environment.Title
		case "environment.order":
			patch.Order = &request.Environment.Order
		case "environment.tier":
			protected := request.Environment.Tier == v1pb.EnvironmentTier_PROTECTED
			patch.Protected = &protected
		}
	}

	environment, err = s.store.UpdateEnvironmentV2(ctx,
		environment.ResourceID,
		patch,
		principalID,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	return convertToEnvironment(environment), nil
}

// DeleteEnvironment deletes an environment.
func (s *EnvironmentService) DeleteEnvironment(ctx context.Context, request *v1pb.DeleteEnvironmentRequest) (*emptypb.Empty, error) {
	principalID := ctx.Value(common.PrincipalIDContextKey).(int)

	environment, err := s.getEnvironmentMessage(ctx, request.Name)
	if err != nil {
		return nil, err
	}
	if environment.Deleted {
		return nil, status.Errorf(codes.InvalidArgument, "environment %q has been deleted", request.Name)
	}

	// All instances in the environment must be deleted.
	count, err := s.store.CountInstance(ctx, &store.CountInstanceMessage{EnvironmentID: &environment.ResourceID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if count > 0 {
		return nil, status.Errorf(codes.FailedPrecondition, "all instances in the environment should be deleted")
	}

	if _, err := s.store.UpdateEnvironmentV2(ctx, environment.ResourceID, &store.UpdateEnvironmentMessage{Delete: &deletePatch}, principalID); err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	return &emptypb.Empty{}, nil
}

// UndeleteEnvironment undeletes an environment.
func (s *EnvironmentService) UndeleteEnvironment(ctx context.Context, request *v1pb.UndeleteEnvironmentRequest) (*v1pb.Environment, error) {
	principalID := ctx.Value(common.PrincipalIDContextKey).(int)

	environment, err := s.getEnvironmentMessage(ctx, request.Name)
	if err != nil {
		return nil, err
	}
	if !environment.Deleted {
		return nil, status.Errorf(codes.InvalidArgument, "environment %q is active", request.Name)
	}

	environment, err = s.store.UpdateEnvironmentV2(ctx, environment.ResourceID, &store.UpdateEnvironmentMessage{Delete: &undeletePatch}, principalID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return convertToEnvironment(environment), nil
}

func (s *EnvironmentService) getEnvironmentMessage(ctx context.Context, name string) (*store.EnvironmentMessage, error) {
	environmentID, err := getEnvironmentID(name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	environment, err := s.store.GetEnvironmentV2(ctx, &store.FindEnvironmentMessage{
		ResourceID: &environmentID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if environment == nil {
		return nil, status.Errorf(codes.NotFound, "environment %q not found", name)
	}

	return environment, nil
}

func convertToEnvironment(environment *store.EnvironmentMessage) *v1pb.Environment {
	tier := v1pb.EnvironmentTier_UNPROTECTED
	if environment.Protected {
		tier = v1pb.EnvironmentTier_PROTECTED
	}
	return &v1pb.Environment{
		Name:  fmt.Sprintf("%s%s", environmentNamePrefix, environment.ResourceID),
		Uid:   fmt.Sprintf("%d", environment.UID),
		State: convertDeletedToState(environment.Deleted),
		Title: environment.Title,
		Order: environment.Order,
		Tier:  tier,
	}
}
