package tests

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

func TestDataSource(t *testing.T) {
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	pgContainer, err := provisionPgInstance(ctx, t)
	a.NoError(err)

	instanceResp, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance"),
		Instance: &v1pb.Instance{
			Title:       "test",
			Engine:      v1pb.Engine_POSTGRES,
			Environment: new("environments/prod"),
			Activation:  true,
			DataSources: []*v1pb.DataSource{pgContainer.dataSource(v1pb.DataSourceType_ADMIN, "admin-ds")},
		},
	}))
	instance := instanceResp.Msg
	a.NoError(err)

	err = ctl.removeLicense(ctx)
	a.NoError(err)
	_, err = ctl.instanceServiceClient.AddDataSource(ctx, connect.NewRequest(&v1pb.AddDataSourceRequest{
		Name:         instance.Name,
		DataSource:   pgContainer.dataSource(v1pb.DataSourceType_READ_ONLY, "readonly-validate-only"),
		ValidateOnly: true,
	}))
	a.ErrorContains(err, "TEAM feature, please upgrade to access it")

	_, err = ctl.instanceServiceClient.AddDataSource(ctx, connect.NewRequest(&v1pb.AddDataSourceRequest{
		Name: instance.Name,
		DataSource: &v1pb.DataSource{
			Id:       "readonly",
			Type:     v1pb.DataSourceType_READ_ONLY,
			Username: "ro_ds",
			Password: "",
			SslCa:    "",
			SslCert:  "",
			SslKey:   "",
		},
	}))
	a.ErrorContains(err, "TEAM feature, please upgrade to access it")

	err = ctl.setLicense(ctx)
	a.NoError(err)
	_, err = ctl.instanceServiceClient.AddDataSource(ctx, connect.NewRequest(&v1pb.AddDataSourceRequest{
		Name: instance.Name,
		DataSource: &v1pb.DataSource{
			Id:       "readonly",
			Type:     v1pb.DataSourceType_READ_ONLY,
			Username: "ro_ds",
			Password: "",
			SslCa:    "",
			SslCert:  "",
			SslKey:   "",
		},
	}))
	a.NoError(err)

	instanceResp, err = ctl.instanceServiceClient.GetInstance(ctx, connect.NewRequest(&v1pb.GetInstanceRequest{Name: instance.Name}))
	a.NoError(err)
	instance = instanceResp.Msg
	a.Equal(2, len(instance.DataSources))
	err = ctl.removeLicense(ctx)
	a.NoError(err)

	_, err = ctl.instanceServiceClient.UpdateDataSource(ctx, connect.NewRequest(&v1pb.UpdateDataSourceRequest{
		Name: instance.Name,
		DataSource: &v1pb.DataSource{
			Id:   "readonly",
			Host: "127.0.0.1",
			Port: "8000",
		},
		UpdateMask: &fieldmaskpb.FieldMask{
			Paths: []string{"host", "port"},
		},
	}))
	a.ErrorContains(err, "TEAM feature, please upgrade to access it")

	err = ctl.setLicense(ctx)
	a.NoError(err)
	_, err = ctl.instanceServiceClient.UpdateDataSource(ctx, connect.NewRequest(&v1pb.UpdateDataSourceRequest{
		Name: instance.Name,
		DataSource: &v1pb.DataSource{
			Id:   "readonly",
			Host: "127.0.0.1",
			Port: "8000",
		},
		UpdateMask: &fieldmaskpb.FieldMask{
			Paths: []string{"host", "port"},
		},
	}))
	a.NoError(err)

	_, err = ctl.instanceServiceClient.AddDataSource(ctx, connect.NewRequest(&v1pb.AddDataSourceRequest{
		Name: instance.Name,
		DataSource: &v1pb.DataSource{
			Id:       "second-read-only-datasource",
			Type:     v1pb.DataSourceType_READ_ONLY,
			Username: "ro_ds",
			Password: "",
			SslCa:    "",
			SslCert:  "",
			SslKey:   "",
		},
	}))
	a.ErrorContains(err, "require at most one read-only data source")

	_, err = ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance"),
		Instance: &v1pb.Instance{
			Title:       "invalid",
			Engine:      v1pb.Engine_POSTGRES,
			Environment: new("environments/prod"),
			Activation:  true,
			DataSources: []*v1pb.DataSource{
				pgContainer.adminDataSource(),
				pgContainer.dataSource(v1pb.DataSourceType_READ_ONLY, "readonly-1"),
				pgContainer.dataSource(v1pb.DataSourceType_READ_ONLY, "readonly-2"),
			},
		},
	}))
	a.ErrorContains(err, "require at most one read-only data source")

	_, err = ctl.instanceServiceClient.UpdateInstance(ctx, connect.NewRequest(&v1pb.UpdateInstanceRequest{
		Instance: &v1pb.Instance{
			Name: instance.Name,
			DataSources: []*v1pb.DataSource{
				pgContainer.dataSource(v1pb.DataSourceType_ADMIN, "admin-ds"),
				{Type: v1pb.DataSourceType_READ_ONLY, Id: "readonly", Username: "ro_ds", Host: "127.0.0.1", Port: "8000"},
				pgContainer.dataSource(v1pb.DataSourceType_READ_ONLY, "readonly-2"),
			},
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"data_sources"}},
	}))
	a.ErrorContains(err, "require at most one read-only data source")

	_, err = ctl.instanceServiceClient.RemoveDataSource(ctx, connect.NewRequest(&v1pb.RemoveDataSourceRequest{
		Name:       instance.Name,
		DataSource: &v1pb.DataSource{Id: "readonly"},
	}))
	a.NoError(err)
}

func TestDataSourceValidateOnly(t *testing.T) {
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	pgContainer, err := provisionPgInstance(ctx, t)
	a.NoError(err)

	instanceResp, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance"),
		Instance: &v1pb.Instance{
			Title:       "test",
			Engine:      v1pb.Engine_POSTGRES,
			Environment: new("environments/prod"),
			Activation:  true,
			DataSources: []*v1pb.DataSource{pgContainer.dataSource(v1pb.DataSourceType_ADMIN, "admin-ds")},
		},
	}))
	a.NoError(err)
	instance := instanceResp.Msg

	err = ctl.setLicense(ctx)
	a.NoError(err)

	addResp, err := ctl.instanceServiceClient.AddDataSource(ctx, connect.NewRequest(&v1pb.AddDataSourceRequest{
		Name:         instance.Name,
		DataSource:   pgContainer.dataSource(v1pb.DataSourceType_READ_ONLY, "readonly-validate-only"),
		ValidateOnly: true,
	}))
	a.NoError(err)
	a.Len(addResp.Msg.DataSources, 2)
	a.NotNil(findDataSource(addResp.Msg.DataSources, "readonly-validate-only"))

	instanceResp, err = ctl.instanceServiceClient.GetInstance(ctx, connect.NewRequest(&v1pb.GetInstanceRequest{Name: instance.Name}))
	a.NoError(err)
	a.Len(instanceResp.Msg.DataSources, 1)
	a.Nil(findDataSource(instanceResp.Msg.DataSources, "readonly-validate-only"))

	updateResp, err := ctl.instanceServiceClient.UpdateDataSource(ctx, connect.NewRequest(&v1pb.UpdateDataSourceRequest{
		Name:         instance.Name,
		DataSource:   pgContainer.dataSource(v1pb.DataSourceType_READ_ONLY, "readonly-allow-missing"),
		UpdateMask:   &fieldmaskpb.FieldMask{Paths: []string{"host"}},
		ValidateOnly: true,
		AllowMissing: true,
	}))
	a.NoError(err)
	a.Len(updateResp.Msg.DataSources, 2)
	a.NotNil(findDataSource(updateResp.Msg.DataSources, "readonly-allow-missing"))

	instanceResp, err = ctl.instanceServiceClient.GetInstance(ctx, connect.NewRequest(&v1pb.GetInstanceRequest{Name: instance.Name}))
	a.NoError(err)
	a.Len(instanceResp.Msg.DataSources, 1)
	a.Nil(findDataSource(instanceResp.Msg.DataSources, "readonly-allow-missing"))

	_, err = ctl.instanceServiceClient.AddDataSource(ctx, connect.NewRequest(&v1pb.AddDataSourceRequest{
		Name:       instance.Name,
		DataSource: pgContainer.dataSource(v1pb.DataSourceType_READ_ONLY, "readonly"),
	}))
	a.NoError(err)

	// Create the role so the validate-only connection test with the updated username succeeds.
	_, err = pgContainer.db.Exec(`CREATE ROLE "updated-user" LOGIN PASSWORD 'root-password'`)
	a.NoError(err)

	updateResp, err = ctl.instanceServiceClient.UpdateDataSource(ctx, connect.NewRequest(&v1pb.UpdateDataSourceRequest{
		Name: instance.Name,
		DataSource: &v1pb.DataSource{
			Id:       "readonly",
			Username: "updated-user",
		},
		UpdateMask:   &fieldmaskpb.FieldMask{Paths: []string{"username"}},
		ValidateOnly: true,
	}))
	a.NoError(err)
	updated := findDataSource(updateResp.Msg.DataSources, "readonly")
	a.NotNil(updated)
	a.Equal("updated-user", updated.Username)

	instanceResp, err = ctl.instanceServiceClient.GetInstance(ctx, connect.NewRequest(&v1pb.GetInstanceRequest{Name: instance.Name}))
	a.NoError(err)
	persisted := findDataSource(instanceResp.Msg.DataSources, "readonly")
	a.NotNil(persisted)
	a.Equal("postgres", persisted.Username)
}

func findDataSource(dataSources []*v1pb.DataSource, id string) *v1pb.DataSource {
	for _, dataSource := range dataSources {
		if dataSource.Id == id {
			return dataSource
		}
	}
	return nil
}
