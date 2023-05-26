package tests

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/resources/mysql"
	"github.com/bytebase/bytebase/backend/tests/fake"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func TestSensitiveData(t *testing.T) {
	const (
		databaseName = "sensitive_data"
		tableName    = "tech_book"
		createTable  = `
			CREATE TABLE tech_book(
				id int primary key,
				name varchar(220),
				author varchar(220)
			);
		`
		insertData = `
			INSERT INTO tech_book VALUES
				(1, 'bytebase', 'bber'),
				(2, 'PostgreSQL 14 Internals', 'Egor Rogov'),
				(3, 'Designing Data-Intensive Applications', 'Martin Kleppmann');
		`
		queryTable = `SELECT * FROM tech_book`
		maskedData = "[[\"id\",\"name\",\"author\"],[\"INT\",\"VARCHAR\",\"VARCHAR\"],[[\"******\",\"bytebase\",\"******\"],[\"******\",\"PostgreSQL 14 Internals\",\"******\"],[\"******\",\"Designing Data-Intensive Applications\",\"******\"]],[true,false,true]]"
		originData = "[[\"id\",\"name\",\"author\"],[\"INT\",\"VARCHAR\",\"VARCHAR\"],[[1,\"bytebase\",\"bber\"],[2,\"PostgreSQL 14 Internals\",\"Egor Rogov\"],[3,\"Designing Data-Intensive Applications\",\"Martin Kleppmann\"]],[false,false,false]]"
	)
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

	// Create a MySQL instance.
	mysqlPort := getTestPort()
	stopInstance := mysql.SetupTestInstance(t, mysqlPort, mysqlBinDir)
	defer stopInstance()

	mysqlDB, err := sql.Open("mysql", fmt.Sprintf("root@tcp(127.0.0.1:%d)/mysql", mysqlPort))
	a.NoError(err)
	defer mysqlDB.Close()

	_, err = mysqlDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %v", databaseName))
	a.NoError(err)

	_, err = mysqlDB.Exec("DROP USER IF EXISTS bytebase")
	a.NoError(err)
	_, err = mysqlDB.Exec("CREATE USER 'bytebase' IDENTIFIED WITH mysql_native_password BY 'bytebase'")
	a.NoError(err)

	_, err = mysqlDB.Exec("GRANT ALTER, ALTER ROUTINE, CREATE, CREATE ROUTINE, CREATE VIEW, DELETE, DROP, EVENT, EXECUTE, INDEX, INSERT, PROCESS, REFERENCES, SELECT, SHOW DATABASES, SHOW VIEW, TRIGGER, UPDATE, USAGE, REPLICATION CLIENT, REPLICATION SLAVE, LOCK TABLES, RELOAD ON *.* to bytebase")
	a.NoError(err)

	// Create a project.
	project, err := ctl.createProject(ctx)
	a.NoError(err)
	projectUID, err := strconv.Atoi(project.Uid)
	a.NoError(err)

	prodEnvironment, _, err := ctl.getEnvironment(ctx, "prod")
	a.NoError(err)

	err = ctl.setLicense()
	a.NoError(err)

	instance, err := ctl.instanceServiceClient.CreateInstance(ctx, &v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance", 10),
		Instance: &v1pb.Instance{
			Title:       "mysqlInstance",
			Engine:      v1pb.Engine_MYSQL,
			Environment: prodEnvironment.Name,
			DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: "127.0.0.1", Port: strconv.Itoa(mysqlPort), Username: "bytebase", Password: "bytebase"}},
		},
	})
	a.NoError(err)
	instanceUID, err := strconv.Atoi(instance.Uid)
	a.NoError(err)

	databases, err := ctl.getDatabases(api.DatabaseFind{
		ProjectID: &projectUID,
	})
	a.NoError(err)
	a.Nil(databases)
	databases, err = ctl.getDatabases(api.DatabaseFind{
		InstanceID: &instanceUID,
	})
	a.NoError(err)
	a.Nil(databases)

	err = ctl.createDatabase(ctx, projectUID, instance, databaseName, "", nil)
	a.NoError(err)

	databases, err = ctl.getDatabases(api.DatabaseFind{
		ProjectID: &projectUID,
	})
	a.NoError(err)
	a.Equal(1, len(databases))

	database := databases[0]
	a.Equal(instanceUID, database.Instance.ID)

	sheet, err := ctl.createSheet(api.SheetCreate{
		ProjectID:  projectUID,
		Name:       "createTable",
		Statement:  createTable,
		Visibility: api.ProjectSheet,
		Source:     api.SheetFromBytebaseArtifact,
		Type:       api.SheetForSQL,
	})
	a.NoError(err)

	// Create an issue that updates database schema.
	createContext, err := json.Marshal(&api.MigrationContext{
		DetailList: []*api.MigrationDetail{
			{
				MigrationType: db.Migrate,
				DatabaseID:    database.ID,
				SheetID:       sheet.ID,
			},
		},
	})
	a.NoError(err)
	issue, err := ctl.createIssue(api.IssueCreate{
		ProjectID:     projectUID,
		Name:          fmt.Sprintf("Create table for database %q", databaseName),
		Type:          api.IssueDatabaseSchemaUpdate,
		Description:   fmt.Sprintf("Create table of database %q.", databaseName),
		AssigneeID:    api.SystemBotID,
		CreateContext: string(createContext),
	})
	a.NoError(err)
	status, err := ctl.waitIssuePipeline(ctx, issue.ID)
	a.NoError(err)
	a.Equal(api.TaskDone, status)

	// Create sensitive data policy.
	_, err = ctl.orgPolicyServiceClient.CreatePolicy(ctx, &v1pb.CreatePolicyRequest{
		Parent: fmt.Sprintf("%s/databases/%s", instance.Name, database.Name),
		Policy: &v1pb.Policy{
			Type: v1pb.PolicyType_SENSITIVE_DATA,
			Policy: &v1pb.Policy_SensitiveDataPolicy{
				SensitiveDataPolicy: &v1pb.SensitiveDataPolicy{
					SensitiveData: []*v1pb.SensitiveData{
						{
							Table:    tableName,
							Column:   "id",
							MaskType: v1pb.SensitiveDataMaskType_DEFAULT,
						},
						{
							Table:    tableName,
							Column:   "author",
							MaskType: v1pb.SensitiveDataMaskType_DEFAULT,
						},
					},
				},
			},
		},
	})
	a.NoError(err)

	insertDataSheet, err := ctl.createSheet(api.SheetCreate{
		ProjectID:  projectUID,
		Name:       "insertData",
		Statement:  insertData,
		Visibility: api.ProjectSheet,
		Source:     api.SheetFromBytebaseArtifact,
		Type:       api.SheetForSQL,
	})
	a.NoError(err)

	// Insert data into table tech_book.
	createContext, err = json.Marshal(&api.MigrationContext{
		DetailList: []*api.MigrationDetail{
			{
				MigrationType: db.Data,
				DatabaseID:    database.ID,
				SheetID:       insertDataSheet.ID,
			},
		},
	})
	a.NoError(err)
	issue, err = ctl.createIssue(api.IssueCreate{
		ProjectID:     projectUID,
		Name:          fmt.Sprintf("update data for database %q", databaseName),
		Type:          api.IssueDatabaseDataUpdate,
		Description:   fmt.Sprintf("This updates the data of database %q.", databaseName),
		AssigneeID:    api.SystemBotID,
		CreateContext: string(createContext),
	})
	a.NoError(err)
	status, err = ctl.waitIssuePipeline(ctx, issue.ID)
	a.NoError(err)
	a.Equal(api.TaskDone, status)

	// Query masked data.
	result, err := ctl.query(instance, databaseName, queryTable)
	a.NoError(err)
	a.Equal(maskedData, result)

	// Query origin data.
	singleSQLResults, err := ctl.adminQuery(instance, databaseName, queryTable)
	a.NoError(err)
	for _, singleSQLResult := range singleSQLResults {
		a.Equal("", singleSQLResult.Error)
		a.Equal(originData, singleSQLResult.Data)
	}
}
