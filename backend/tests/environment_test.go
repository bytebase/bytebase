package tests

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/bytebase/bytebase/backend/tests/fake"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func TestDatabaseEnvironment(t *testing.T) {
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

	prodEnvironment, err := ctl.getEnvironment(ctx, "prod")
	a.NoError(err)
	testEnvironment, err := ctl.getEnvironment(ctx, "test")
	a.NoError(err)

	instance, err := ctl.instanceServiceClient.CreateInstance(ctx, &v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance", 10),
		Instance: &v1pb.Instance{
			Title:       "test",
			Engine:      v1pb.Engine_SQLITE,
			Environment: prodEnvironment.Name,
			Activation:  true,
			DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Id: "admin-ds", Host: instanceDir}},
		},
	})
	a.NoError(err)

	db0Name := "db0"
	err = ctl.createDatabaseV2(ctx, ctl.project, instance, testEnvironment /* environment */, db0Name, "", nil /* labelMap */)
	a.NoError(err)
	db0, err := ctl.databaseServiceClient.GetDatabase(ctx, &v1pb.GetDatabaseRequest{
		Name: fmt.Sprintf("%s/databases/%s", instance.Name, db0Name),
	})
	a.NoError(err)
	a.Equal(testEnvironment.Name, db0.Environment)
	a.Equal(testEnvironment.Name, db0.EffectiveEnvironment)
	db1Name := "db1"
	err = ctl.createDatabaseV2(ctx, ctl.project, instance, nil /* environment */, db1Name, "", nil /* labelMap */)
	a.NoError(err)
	db1, err := ctl.databaseServiceClient.GetDatabase(ctx, &v1pb.GetDatabaseRequest{
		Name: fmt.Sprintf("%s/databases/%s", instance.Name, db1Name),
	})
	a.NoError(err)
	a.Equal("", db1.Environment)
	a.Equal(prodEnvironment.Name, db1.EffectiveEnvironment)

	db2Name := "db2"
	err = ctl.createDatabaseV2(ctx, ctl.project, instance, nil /* environment */, db2Name, "", nil /* labelMap */)
	a.NoError(err)
	db2, err := ctl.databaseServiceClient.GetDatabase(ctx, &v1pb.GetDatabaseRequest{
		Name: fmt.Sprintf("%s/databases/%s", instance.Name, db2Name),
	})
	a.NoError(err)

	// Update database environment for db2.
	db2, err = ctl.databaseServiceClient.UpdateDatabase(ctx, &v1pb.UpdateDatabaseRequest{
		Database: &v1pb.Database{
			Name:        db2.Name,
			Environment: testEnvironment.Name,
		},
		UpdateMask: &fieldmaskpb.FieldMask{
			Paths: []string{"environment"},
		},
	})
	a.NoError(err)
	a.Equal(testEnvironment.Name, db2.Environment)
	a.Equal(testEnvironment.Name, db2.EffectiveEnvironment)

	// Unset database environment for db2.
	db2, err = ctl.databaseServiceClient.UpdateDatabase(ctx, &v1pb.UpdateDatabaseRequest{
		Database: &v1pb.Database{
			Name:        db2.Name,
			Environment: "",
		},
		UpdateMask: &fieldmaskpb.FieldMask{
			Paths: []string{"environment"},
		},
	})
	a.NoError(err)
	a.Equal("", db2.Environment)
	a.Equal(prodEnvironment.Name, db2.EffectiveEnvironment)
}
