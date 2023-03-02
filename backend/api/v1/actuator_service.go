package v1

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bytebase/bytebase/backend/component/config"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// ActuatorService implements the actuator service.
type ActuatorService struct {
	v1pb.UnimplementedActuatorServiceServer
	store   *store.Store
	profile *config.Profile
}

// NewActuatorService creates a new ActuatorService.
func NewActuatorService(store *store.Store, profile *config.Profile) *ActuatorService {
	return &ActuatorService{
		store:   store,
		profile: profile,
	}
}

// GetActuatorInfo gets the actuator info.
func (s *ActuatorService) GetActuatorInfo(ctx context.Context, _ *v1pb.GetActuatorInfoRequest) (*v1pb.ActuatorInfo, error) {
	return s.getServerInfo(ctx)
}

func (s *ActuatorService) getServerInfo(ctx context.Context) (*v1pb.ActuatorInfo, error) {
	count, err := s.store.CountUsers(ctx, api.EndUser)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	setting, err := s.store.GetWorkspaceGeneralSetting(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find workspace setting: %v", err)
	}

	serverInfo := v1pb.ActuatorInfo{
		Version:        s.profile.Version,
		GitCommit:      s.profile.GitCommit,
		Readonly:       s.profile.Readonly,
		Saas:           s.profile.SaaS,
		DemoName:       s.profile.DemoName,
		NeedAdminSetup: count == 0,
		ExternalUrl:    setting.ExternalUrl,
		DisallowSignup: setting.DisallowSignup,
		LastActiveTs:   s.profile.LastActiveTs,
	}

	return &serverInfo, nil
}
