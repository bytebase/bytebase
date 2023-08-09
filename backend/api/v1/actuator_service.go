package v1

import (
	"context"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// ActuatorService implements the actuator service.
type ActuatorService struct {
	v1pb.UnimplementedActuatorServiceServer
	store           *store.Store
	profile         *config.Profile
	errorRecordRing *api.ErrorRecordRing
}

// NewActuatorService creates a new ActuatorService.
func NewActuatorService(store *store.Store, profile *config.Profile, errorRecordRing *api.ErrorRecordRing) *ActuatorService {
	return &ActuatorService{
		store:           store,
		profile:         profile,
		errorRecordRing: errorRecordRing,
	}
}

// GetActuatorInfo gets the actuator info.
func (s *ActuatorService) GetActuatorInfo(ctx context.Context, _ *v1pb.GetActuatorInfoRequest) (*v1pb.ActuatorInfo, error) {
	return s.getServerInfo(ctx)
}

// UpdateActuatorInfo updates the actuator info.
func (s *ActuatorService) UpdateActuatorInfo(ctx context.Context, request *v1pb.UpdateActuatorInfoRequest) (*v1pb.ActuatorInfo, error) {
	for _, path := range request.UpdateMask.Paths {
		if path == "debug" {
			lvl := zap.InfoLevel
			if request.Actuator.Debug {
				lvl = zap.DebugLevel
			}
			log.SetLevel(lvl)
		}
	}

	return s.getServerInfo(ctx)
}

// ListDebugLog lists the debug log.
func (s *ActuatorService) ListDebugLog(_ context.Context, _ *v1pb.ListDebugLogRequest) (*v1pb.ListDebugLogResponse, error) {
	resp := &v1pb.ListDebugLogResponse{}

	s.errorRecordRing.RWMutex.RLock()
	defer s.errorRecordRing.RWMutex.RUnlock()

	s.errorRecordRing.Ring.Do(func(p any) {
		if p == nil {
			return
		}
		errRecord, ok := p.(*v1pb.DebugLog)
		if !ok {
			return
		}
		resp.Logs = append(resp.Logs, errRecord)
	})

	return resp, nil
}

// DeleteCache deletes the cache.
func (s *SettingService) DeleteCache(_ context.Context, _ *v1pb.DeleteCacheRequest) (*emptypb.Empty, error) {
	s.store.DeleteCache()
	return &emptypb.Empty{}, nil
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

	workspaceID, err := s.store.GetWorkspaceID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	serverInfo := v1pb.ActuatorInfo{
		Version:                   s.profile.Version,
		GitCommit:                 s.profile.GitCommit,
		Readonly:                  s.profile.Readonly,
		Saas:                      s.profile.SaaS,
		DemoName:                  s.profile.DemoName,
		NeedAdminSetup:            count == 0,
		ExternalUrl:               setting.ExternalUrl,
		DisallowSignup:            setting.DisallowSignup,
		Require_2Fa:               setting.Require_2Fa,
		LastActiveTime:            timestamppb.New(time.Unix(s.profile.LastActiveTs, 0)),
		WorkspaceId:               workspaceID,
		GitopsWebhookUrl:          setting.GitopsWebhookUrl,
		Debug:                     log.EnabledLevel(zap.DebugLevel),
		DevelopmentUseV2Scheduler: s.profile.DevelopmentUseV2Scheduler,
	}

	return &serverInfo, nil
}
