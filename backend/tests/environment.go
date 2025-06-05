package tests

import (
	"context"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// getEnvironment gets the environments.
func (ctl *controller) getEnvironment(ctx context.Context, id string) (*v1pb.EnvironmentSetting_Environment, error) {
	setting, err := ctl.settingServiceClient.GetSetting(ctx,
		&v1pb.GetSettingRequest{
			Name: "settings/" + v1pb.Setting_ENVIRONMENT.String(),
		})
	if err != nil {
		return nil, err
	}
	for _, environment := range setting.Value.GetEnvironmentSetting().GetEnvironments() {
		if environment.Id == id {
			return environment, nil
		}
	}

	return nil, errors.Errorf("environment %q not found", id)
}

// createEnvironment creates a new environment.
// The created environment will be appended to the existing environments.
// It returns the created environment.
func (ctl *controller) createEnvironment(ctx context.Context, id, title string) (*v1pb.EnvironmentSetting_Environment, error) {
	setting, err := ctl.settingServiceClient.GetSetting(ctx,
		&v1pb.GetSettingRequest{
			Name: "settings/" + v1pb.Setting_ENVIRONMENT.String(),
		})
	if err != nil {
		return nil, err
	}
	environments := setting.Value.GetEnvironmentSetting().GetEnvironments()
	environments = append(environments, &v1pb.EnvironmentSetting_Environment{
		Id:    id,
		Title: title,
	})
	_, err = ctl.settingServiceClient.UpdateSetting(ctx,
		&v1pb.UpdateSettingRequest{
			Setting: &v1pb.Setting{
				Name: "settings/" + v1pb.Setting_ENVIRONMENT.String(),
				Value: &v1pb.Value{
					Value: &v1pb.Value_EnvironmentSetting{
						EnvironmentSetting: &v1pb.EnvironmentSetting{
							Environments: environments,
						},
					},
				},
			},
			UpdateMask: &fieldmaskpb.FieldMask{
				Paths: []string{"environment_setting"},
			},
		})
	if err != nil {
		return nil, err
	}
	return ctl.getEnvironment(ctx, id)
}
