package tests

import (
	"context"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

func TestDatabaseEnvironment(t *testing.T) {
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

	prodEnvironment, err := ctl.getEnvironment(ctx, "prod")
	a.NoError(err)
	testEnvironment, err := ctl.getEnvironment(ctx, "test")
	a.NoError(err)

	instanceResp, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance"),
		Instance: &v1pb.Instance{
			Title:       "test",
			Engine:      v1pb.Engine_SQLITE,
			Environment: stringPtr(prodEnvironment.Name),
			Activation:  true,
			DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Id: "admin-ds", Host: instanceDir}},
		},
	}))
	a.NoError(err)
	instance := instanceResp.Msg

	db0Name := "db0"
	err = ctl.createDatabase(ctx, ctl.project, instance, testEnvironment /* environment */, db0Name, "")
	a.NoError(err)
	db0Resp, err := ctl.databaseServiceClient.GetDatabase(ctx, connect.NewRequest(&v1pb.GetDatabaseRequest{
		Name: fmt.Sprintf("%s/databases/%s", instance.Name, db0Name),
	}))
	a.NoError(err)
	db0 := db0Resp.Msg
	a.NotNil(db0.Environment)
	a.Equal(testEnvironment.Name, *db0.Environment)
	a.NotNil(db0.EffectiveEnvironment)
	a.Equal(testEnvironment.Name, *db0.EffectiveEnvironment)

	db1Name := "db1"
	err = ctl.createDatabase(ctx, ctl.project, instance, nil /* environment */, db1Name, "")
	a.NoError(err)
	db1Resp, err := ctl.databaseServiceClient.GetDatabase(ctx, connect.NewRequest(&v1pb.GetDatabaseRequest{
		Name: fmt.Sprintf("%s/databases/%s", instance.Name, db1Name),
	}))
	a.NoError(err)
	db1 := db1Resp.Msg
	a.Nil(db1.Environment)
	a.NotNil(db1.EffectiveEnvironment)
	a.Equal(prodEnvironment.Name, *db1.EffectiveEnvironment)

	db2Name := "db2"
	err = ctl.createDatabase(ctx, ctl.project, instance, nil /* environment */, db2Name, "")
	a.NoError(err)
	db2Resp, err := ctl.databaseServiceClient.GetDatabase(ctx, connect.NewRequest(&v1pb.GetDatabaseRequest{
		Name: fmt.Sprintf("%s/databases/%s", instance.Name, db2Name),
	}))
	a.NoError(err)
	db2 := db2Resp.Msg

	// Update database environment for db2.
	db2Resp, err = ctl.databaseServiceClient.UpdateDatabase(ctx, connect.NewRequest(&v1pb.UpdateDatabaseRequest{
		Database: &v1pb.Database{
			Name:        db2.Name,
			Environment: stringPtr(testEnvironment.Name),
		},
		UpdateMask: &fieldmaskpb.FieldMask{
			Paths: []string{"environment"},
		},
	}))
	a.NoError(err)
	db2 = db2Resp.Msg
	a.NotNil(db2.Environment)
	a.Equal(testEnvironment.Name, *db2.Environment)
	a.NotNil(db2.EffectiveEnvironment)
	a.Equal(testEnvironment.Name, *db2.EffectiveEnvironment)

	// Unset database environment for db2.
	db2Resp, err = ctl.databaseServiceClient.UpdateDatabase(ctx, connect.NewRequest(&v1pb.UpdateDatabaseRequest{
		Database: &v1pb.Database{
			Name:        db2.Name,
			Environment: stringPtr(""),
		},
		UpdateMask: &fieldmaskpb.FieldMask{
			Paths: []string{"environment"},
		},
	}))
	a.NoError(err)
	db2 = db2Resp.Msg
	a.Nil(db2.Environment)
	a.NotNil(db2.EffectiveEnvironment)
	a.Equal(prodEnvironment.Name, *db2.EffectiveEnvironment)
}
