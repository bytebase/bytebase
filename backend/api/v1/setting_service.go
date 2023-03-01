package v1

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// SettingService implements the setting service.
type SettingService struct {
	v1pb.UnimplementedSettingServiceServer
	store *store.Store
}

// NewSettingService creates a new setting service.
func NewSettingService(store *store.Store) *SettingService {
	return &SettingService{
		store: store,
	}
}

// Some settings contain secret info so we only return settings that are needed by the client.
var whitelistSettings = []api.SettingName{
	api.SettingBrandingLogo,
	api.SettingAppIM,
	api.SettingWatermark,
}

// GetSetting gets the setting by name.
func (s *SettingService) GetSetting(ctx context.Context, request *v1pb.GetSettingRequest) (*v1pb.Setting, error) {
	settingName, err := getSettingName(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "setting name is invalid: %v", err)
	}
	if settingName == "" {
		return nil, status.Errorf(codes.InvalidArgument, "setting name is empty")
	}
	apiSettingName := api.SettingName(settingName)
	setting, err := s.store.GetSettingV2(ctx, &store.FindSettingMessage{
		Name: &apiSettingName,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get setting: %v", err)
	}
	if setting == nil {
		return nil, status.Errorf(codes.NotFound, "setting %s not found", settingName)
	}
	// Only return whitelisted setting.
	for _, whitelist := range whitelistSettings {
		if setting.Name == whitelist {
			return convertToSettingMessage(setting), nil
		}
	}

	return nil, status.Errorf(codes.InvalidArgument, "setting %s is not whitelisted", settingName)
}

// SetSetting set the setting by name.
func (s *SettingService) SetSetting(ctx context.Context, request *v1pb.SetSettingRequest) (*v1pb.Setting, error) {
	settingName, err := getSettingName(request.Setting.Name)
	if err != nil {
		return nil, err
	}
	if settingName == "" {
		return nil, status.Errorf(codes.InvalidArgument, "setting name is empty")
	}
	apiSettingName := api.SettingName(settingName)
	setting, err := s.store.UpsertSettingV2(ctx, &store.SetSettingMessage{
		Name:  apiSettingName,
		Value: request.Setting.Value.GetStringValue(),
	}, ctx.Value(common.PrincipalIDContextKey).(int))
	if err != nil {
		return nil, err
	}

	return convertToSettingMessage(setting), nil
}

func convertToSettingMessage(setting *store.SettingMessage) *v1pb.Setting {
	return &v1pb.Setting{
		Name: settingNamePrefix + string(setting.Name),
		Value: &v1pb.Value{
			Value: &v1pb.Value_StringValue{
				StringValue: setting.Value,
			},
		},
	}
}
