package tests

import (
	"context"
	_ "embed"
	"fmt"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"

	"connectrpc.com/connect"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

const (
	mysqlMigrationStatement = `
	CREATE TABLE book (
		id INT PRIMARY KEY AUTO_INCREMENT,
		name TEXT
	);
	`
	mysqlGhostMigrationStatement = `
	ALTER TABLE book ADD author VARCHAR(54)
	`
)

var (
	//go:embed test-data/ghost_test_schema1.result
	wantDBSchema1 string

	//go:embed test-data/ghost_test_schema2.result
	wantDBSchema2 string
)

func TestGhostSchemaUpdate(t *testing.T) {
	const databaseName = "testGhostSchemaUpdate"

	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	mysqlContainer, err := getMySQLContainer(ctx)
	defer func() {
		mysqlContainer.Close(ctx)
	}()
	a.NoError(err)

	mysqlDB := mysqlContainer.db

	_, err = mysqlDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %v", databaseName))
	a.NoError(err)

	_, err = mysqlDB.Exec("DROP USER IF EXISTS bytebase")
	a.NoError(err)
	_, err = mysqlDB.Exec("CREATE USER 'bytebase' IDENTIFIED WITH mysql_native_password BY 'bytebase'")
	a.NoError(err)

	_, err = mysqlDB.Exec("GRANT ALTER, ALTER ROUTINE, CREATE, CREATE ROUTINE, CREATE VIEW, DELETE, DROP, EVENT, EXECUTE, INDEX, INSERT, PROCESS, REFERENCES, SELECT, SHOW DATABASES, SHOW VIEW, TRIGGER, UPDATE, USAGE, REPLICATION CLIENT, REPLICATION SLAVE, LOCK TABLES, RELOAD ON *.* to bytebase")
	a.NoError(err)

	instanceResponse, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance"),
		Instance: &v1pb.Instance{
			Title:       "mysqlInstance",
			Engine:      v1pb.Engine_MYSQL,
			Environment: stringPtr("environments/prod"),
			Activation:  true,
			DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: mysqlContainer.host, Port: mysqlContainer.port, Username: "bytebase", Password: "bytebase", Id: "admin"}},
		},
	}))
	a.NoError(err)
	instance := instanceResponse.Msg

	// Create backup database for MySQL.
	backupDBName := common.BackupDatabaseNameOfEngine(storepb.Engine_MYSQL)
	err = ctl.createDatabase(ctx, ctl.project, instance, nil /* environment */, backupDBName, "")
	a.NoError(err)

	err = ctl.createDatabase(ctx, ctl.project, instance, nil /* environment */, databaseName, "")
	a.NoError(err)

	databaseResponse, err := ctl.databaseServiceClient.GetDatabase(ctx, connect.NewRequest(&v1pb.GetDatabaseRequest{
		Name: fmt.Sprintf("%s/databases/%s", instance.Name, databaseName),
	}))
	a.NoError(err)
	database := databaseResponse.Msg

	sheet1Response, err := ctl.sheetServiceClient.CreateSheet(ctx, connect.NewRequest(&v1pb.CreateSheetRequest{
		Parent: ctl.project.Name,
		Sheet: &v1pb.Sheet{
			Content: []byte(mysqlMigrationStatement),
		},
	}))
	a.NoError(err)
	sheet1 := sheet1Response.Msg

	err = ctl.changeDatabase(ctx, ctl.project, database, sheet1, false)
	a.NoError(err)

	dbMetadataResponse, err := ctl.databaseServiceClient.GetDatabaseSchema(ctx, connect.NewRequest(&v1pb.GetDatabaseSchemaRequest{Name: fmt.Sprintf("%s/schema", database.Name)}))
	a.NoError(err)
	dbMetadata := dbMetadataResponse.Msg
	a.Equal(wantDBSchema1, dbMetadata.Schema)

	sheet2Response, err := ctl.sheetServiceClient.CreateSheet(ctx, connect.NewRequest(&v1pb.CreateSheetRequest{
		Parent: ctl.project.Name,
		Sheet: &v1pb.Sheet{
			Content: []byte(mysqlGhostMigrationStatement),
		},
	}))
	a.NoError(err)
	sheet2 := sheet2Response.Msg

	err = ctl.changeDatabase(ctx, ctl.project, database, sheet2, true)
	a.NoError(err)
	dbMetadataResponse, err = ctl.databaseServiceClient.GetDatabaseSchema(ctx, connect.NewRequest(&v1pb.GetDatabaseSchemaRequest{Name: fmt.Sprintf("%s/schema", database.Name)}))
	a.NoError(err)
	dbMetadata = dbMetadataResponse.Msg

	a.Equal(wantDBSchema2, dbMetadata.Schema)
}
