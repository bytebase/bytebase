package tests

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/tests/fake"
)

func TestDataSource(t *testing.T) {
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	instanceRootDir := t.TempDir()
	instanceName := "testInstance1"
	instanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, instanceName)
	a.NoError(err)

	instanceResp, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance"),
		Instance: &v1pb.Instance{
			Title:       "test",
			Engine:      v1pb.Engine_SQLITE,
			Environment: stringPtr("environments/prod"),
			Activation:  true,
			DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Id: "admin-ds", Host: instanceDir}},
		},
	}))
	instance := instanceResp.Msg
	a.NoError(err)

	err = ctl.removeLicense(ctx)
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
	a.NoError(err)

	_, err = ctl.instanceServiceClient.RemoveDataSource(ctx, connect.NewRequest(&v1pb.RemoveDataSourceRequest{
		Name:       instance.Name,
		DataSource: &v1pb.DataSource{Id: "readonly"},
	}))
	a.NoError(err)
}

func TestExternalSecretManager(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	pgContainer, err := getPgContainer(ctx)
	defer func() {
		pgContainer.Close(ctx)
	}()
	a.NoError(err)

	smPort := getTestPort()
	sm := fake.NewSecretManager(smPort)
	go func() {
		if err := sm.Run(); err != http.ErrServerClosed {
			a.NoError(err)
		}
	}()
	defer sm.Close()

	pgDB := pgContainer.db
	err = pgDB.Ping()
	a.NoError(err)

	_, err = pgDB.Exec("CREATE USER bytebase WITH ENCRYPTED PASSWORD 'bytebase'")
	a.NoError(err)
	_, err = pgDB.Exec("ALTER USER bytebase WITH SUPERUSER")
	a.NoError(err)

	secretURL := fmt.Sprintf("{{http://localhost:%d/secrets/hello-secret-id:access}}", smPort)
	instanceResp, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance"),
		Instance: &v1pb.Instance{
			Title:       "pgInstance",
			Engine:      v1pb.Engine_POSTGRES,
			Environment: stringPtr("environments/prod"),
			Activation:  true,
			DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: pgContainer.host, Port: pgContainer.port, Username: "bytebase", Password: secretURL, Id: "admin"}},
		},
	}))
	a.NoError(err)
	instance := instanceResp.Msg

	err = ctl.createDatabaseV2(ctx, ctl.project, instance, nil /* environment */, databaseName, "bytebase")
	a.NoError(err)
}
