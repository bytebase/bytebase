package v1

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
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

// UpdateActuatorInfo updates the actuator info.
func (s *ActuatorService) UpdateActuatorInfo(ctx context.Context, request *v1pb.UpdateActuatorInfoRequest) (*v1pb.ActuatorInfo, error) {
	role := ctx.Value(common.RoleContextKey).(api.Role)

	for _, path := range request.UpdateMask.Paths {
		switch path {
		case "actuator.disallow_signup":
			if role != api.Owner {
				return nil, status.Errorf(codes.PermissionDenied, "only workspace owner can update the disallow_signup field")
			}
			s.profile.DisallowSignup = request.Actuator.DisallowSignup
		case "actuator.debug":
			lvl := zap.InfoLevel
			if request.Actuator.Debug {
				lvl = zap.DebugLevel
			}
			log.SetLevel(lvl)
		}
	}

	return s.getServerInfo(ctx)
}

func (s *ActuatorService) getServerInfo(ctx context.Context) (*v1pb.ActuatorInfo, error) {
	owner := api.Owner
	principalType := api.EndUser
	count, err := s.store.CountUsers(ctx, &store.FindUserMessage{
		Role: &owner,
		Type: &principalType,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	serverInfo := v1pb.ActuatorInfo{
		Version:        s.profile.Version,
		GitCommit:      s.profile.GitCommit,
		Readonly:       s.profile.Readonly,
		DemoName:       s.profile.DemoName,
		ExternalUrl:    s.profile.ExternalURL,
		NeedAdminSetup: count == 0,
		DisallowSignup: s.profile.DisallowSignup,
		Debug:          log.EnabledLevel(zap.DebugLevel),
	}

	return &serverInfo, nil
}
