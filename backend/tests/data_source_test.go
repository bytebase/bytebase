package tests

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/bytebase/bytebase/backend/resources/postgres"
	"github.com/bytebase/bytebase/backend/tests/fake"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func TestDataSource(t *testing.T) {
	a := require.New(t)
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

	instance, err := ctl.instanceServiceClient.CreateInstance(ctx, &v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance", 10),
		Instance: &v1pb.Instance{
			Title:       "test",
			Engine:      v1pb.Engine_SQLITE,
			Environment: "environments/prod",
			Activation:  true,
			DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Id: "admin-ds", Host: instanceDir}},
		},
	})
	a.NoError(err)

	err = ctl.removeLicense(ctx)
	a.NoError(err)
	_, err = ctl.instanceServiceClient.AddDataSource(ctx, &v1pb.AddDataSourceRequest{
		Instance: instance.Name,
		DataSource: &v1pb.DataSource{
			Id:       "readonly",
			Type:     v1pb.DataSourceType_READ_ONLY,
			Username: "ro_ds",
			Password: "",
			SslCa:    "",
			SslCert:  "",
			SslKey:   "",
		},
	})
	a.ErrorContains(err, "Read replica connection is a ENTERPRISE feature")

	err = ctl.setLicense(ctx)
	a.NoError(err)
	_, err = ctl.instanceServiceClient.AddDataSource(ctx, &v1pb.AddDataSourceRequest{
		Instance: instance.Name,
		DataSource: &v1pb.DataSource{
			Id:       "readonly",
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
	err = ctl.removeLicense(ctx)
	a.NoError(err)

	_, err = ctl.instanceServiceClient.UpdateDataSource(ctx, &v1pb.UpdateDataSourceRequest{
		Instance: instance.Name,
		DataSource: &v1pb.DataSource{
			Id:   "readonly",
			Host: "127.0.0.1",
			Port: "8000",
		},
		UpdateMask: &fieldmaskpb.FieldMask{
			Paths: []string{"host", "port"},
		},
	})
	a.ErrorContains(err, "Read replica connection is a ENTERPRISE feature")

	err = ctl.setLicense(ctx)
	a.NoError(err)
	_, err = ctl.instanceServiceClient.UpdateDataSource(ctx, &v1pb.UpdateDataSourceRequest{
		Instance: instance.Name,
		DataSource: &v1pb.DataSource{
			Id:   "readonly",
			Host: "127.0.0.1",
			Port: "8000",
		},
		UpdateMask: &fieldmaskpb.FieldMask{
			Paths: []string{"host", "port"},
		},
	})
	a.NoError(err)

	_, err = ctl.instanceServiceClient.AddDataSource(ctx, &v1pb.AddDataSourceRequest{
		Instance: instance.Name,
		DataSource: &v1pb.DataSource{
			Id:       "second-read-only-datasource",
			Type:     v1pb.DataSourceType_READ_ONLY,
			Username: "ro_ds",
			Password: "",
			SslCa:    "",
			SslCert:  "",
			SslKey:   "",
		},
	})
	a.NoError(err)

	_, err = ctl.instanceServiceClient.RemoveDataSource(ctx, &v1pb.RemoveDataSourceRequest{
		Instance:   instance.Name,
		DataSource: &v1pb.DataSource{Id: "readonly"},
	})
	a.NoError(err)
}

func TestExternalSecretManager(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	dataDir := t.TempDir()
	ctx, err := ctl.StartServerWithExternalPg(ctx, &config{
		dataDir:            dataDir,
		vcsProviderCreator: fake.NewGitLab,
	})
	a.NoError(err)
	defer ctl.Close(ctx)

	// Create a PostgreSQL instance.
	pgPort := getTestPort()
	stopInstance := postgres.SetupTestInstance(pgBinDir, t.TempDir(), pgPort)
	defer stopInstance()

	smPort := getTestPort()
	sm := fake.NewSecretManager(smPort)
	go func() {
		if err := sm.Run(); err != http.ErrServerClosed {
			a.NoError(err)
		}
	}()
	defer sm.Close()

	pgDB, err := sql.Open("pgx", fmt.Sprintf("host=/tmp port=%d user=root database=postgres", pgPort))
	a.NoError(err)
	defer pgDB.Close()
	err = pgDB.Ping()
	a.NoError(err)

	_, err = pgDB.Exec("CREATE USER bytebase WITH ENCRYPTED PASSWORD 'bytebase'")
	a.NoError(err)
	_, err = pgDB.Exec("ALTER USER bytebase WITH SUPERUSER")
	a.NoError(err)

	secretURL := fmt.Sprintf("{{http://localhost:%d/secrets/hello-secret-id:access}}", smPort)
	instance, err := ctl.instanceServiceClient.CreateInstance(ctx, &v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance", 10),
		Instance: &v1pb.Instance{
			Title:       "pgInstance",
			Engine:      v1pb.Engine_POSTGRES,
			Environment: "environments/prod",
			Activation:  true,
			DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: "/tmp", Port: strconv.Itoa(pgPort), Username: "bytebase", Password: secretURL}},
		},
	})
	a.NoError(err)

	err = ctl.createDatabaseV2(ctx, ctl.project, instance, nil /* environment */, databaseName, "bytebase", nil)
	a.NoError(err)
}
