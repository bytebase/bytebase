package tests

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/bytebase/bytebase/backend/common/log"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/tests/fake"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func TestDataSource(t *testing.T) {
	a := require.New(t)
	log.SetLevel(zapcore.DebugLevel)
	ctx := context.Background()
	ctl := &controller{}
	dataDir := t.TempDir()
	ctx, err := ctl.StartServerWithExternalPg(ctx, &config{
		dataDir:            dataDir,
		vcsProviderCreator: fake.NewGitLab,
	})
	a.NoError(err)
	defer ctl.Close(ctx)

	instanceRootDir := t.TempDir()
	instanceName := "testInstance1"
	instanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, instanceName)
	a.NoError(err)

	prodEnvironment, err := ctl.getEnvironment(ctx, "prod")
	a.NoError(err)

	instance, err := ctl.instanceServiceClient.CreateInstance(ctx, &v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance", 10),
		Instance: &v1pb.Instance{
			Title:       "test",
			Engine:      v1pb.Engine_SQLITE,
			Environment: prodEnvironment.Name,
			Activation:  true,
			DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Id: api.AdminDataSourceName, Host: instanceDir}},
		},
	})
	a.NoError(err)

	err = ctl.removeLicense()
	a.NoError(err)
	_, err = ctl.instanceServiceClient.AddDataSource(ctx, &v1pb.AddDataSourceRequest{
		Instance: instance.Name,
		DataSource: &v1pb.DataSource{
			Id:       api.ReadOnlyDataSourceName,
			Type:     v1pb.DataSourceType_READ_ONLY,
			Username: "ro_ds",
			Password: "",
			SslCa:    "",
			SslCert:  "",
			SslKey:   "",
		},
	})
	a.ErrorContains(err, "Read replica connection is a ENTERPRISE feature")
	err = ctl.setLicense()
	a.NoError(err)
	_, err = ctl.instanceServiceClient.AddDataSource(ctx, &v1pb.AddDataSourceRequest{
		Instance: instance.Name,
		DataSource: &v1pb.DataSource{
			Id:       api.ReadOnlyDataSourceName,
			Type:     v1pb.DataSourceType_READ_ONLY,
			Username: "ro_ds",
			Password: "",
			SslCa:    "",
			SslCert:  "",
			SslKey:   "",
		},
	})
	a.NoError(err)

	instance, err = ctl.instanceServiceClient.GetInstance(ctx, &v1pb.GetInstanceRequest{Name: instance.Name})
	a.NoError(err)
	a.Equal(2, len(instance.DataSources))
	err = ctl.removeLicense()
	a.NoError(err)

	_, err = ctl.instanceServiceClient.UpdateDataSource(ctx, &v1pb.UpdateDataSourceRequest{
		Instance: instance.Name,
		DataSource: &v1pb.DataSource{
			Id:   api.ReadOnlyDataSourceName,
			Host: "127.0.0.1",
			Port: "8000",
		},
		UpdateMask: &fieldmaskpb.FieldMask{
			Paths: []string{"host", "port"},
		},
	})
	a.ErrorContains(err, "Read replica connection is a ENTERPRISE feature")

	err = ctl.setLicense()
	a.NoError(err)
	_, err = ctl.instanceServiceClient.UpdateDataSource(ctx, &v1pb.UpdateDataSourceRequest{
		Instance: instance.Name,
		DataSource: &v1pb.DataSource{
			Id:   api.ReadOnlyDataSourceName,
			Host: "127.0.0.1",
			Port: "8000",
		},
		UpdateMask: &fieldmaskpb.FieldMask{
			Paths: []string{"host", "port"},
		},
	})
	a.NoError(err)

	_, err = ctl.instanceServiceClient.RemoveDataSource(ctx, &v1pb.RemoveDataSourceRequest{
		Instance:   instance.Name,
		DataSource: &v1pb.DataSource{Id: api.ReadOnlyDataSourceName},
	})
	a.NoError(err)
}
