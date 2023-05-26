//go:build mysql
// +build mysql

package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"testing"

	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/resources/mysql"
	"github.com/bytebase/bytebase/backend/tests/fake"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"

	ghostsql "github.com/github/gh-ost/go/sql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
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
	mysqlQueryBookTable = `
	SELECT * FROM INFORMATION_SCHEMA.COLUMNS
	WHERE table_name = 'book'
	ORDER BY ORDINAL_POSITION
	`
	mysqlBookSchema1 = `[["TABLE_CATALOG","TABLE_SCHEMA","TABLE_NAME","COLUMN_NAME","ORDINAL_POSITION","COLUMN_DEFAULT","IS_NULLABLE","DATA_TYPE","CHARACTER_MAXIMUM_LENGTH","CHARACTER_OCTET_LENGTH","NUMERIC_PRECISION","NUMERIC_SCALE","DATETIME_PRECISION","CHARACTER_SET_NAME","COLLATION_NAME","COLUMN_TYPE","COLUMN_KEY","EXTRA","PRIVILEGES","COLUMN_COMMENT","GENERATION_EXPRESSION","SRS_ID"],["VARCHAR","VARCHAR","VARCHAR","VARCHAR","UNSIGNED INT","TEXT","VARCHAR","TEXT","BIGINT","BIGINT","UNSIGNED BIGINT","UNSIGNED BIGINT","UNSIGNED INT","VARCHAR","VARCHAR","TEXT","CHAR","VARCHAR","VARCHAR","TEXT","TEXT","UNSIGNED INT"],[["def","testGhostSchemaUpdate","book","id","1",null,"NO","int",null,null,"10","0",null,null,null,"int","PRI","auto_increment","select,insert,update,references","","",null],["def","testGhostSchemaUpdate","book","name","2",null,"YES","text","65535","65535",null,null,null,"utf8mb4","utf8mb4_general_ci","text","","","select,insert,update,references","","",null]],[false,false,false,false,false,false,false,false,false,false,false,false,false,false,false,false,false,false,false,false,false,false]]`
	mysqlBookSchema2 = `[["TABLE_CATALOG","TABLE_SCHEMA","TABLE_NAME","COLUMN_NAME","ORDINAL_POSITION","COLUMN_DEFAULT","IS_NULLABLE","DATA_TYPE","CHARACTER_MAXIMUM_LENGTH","CHARACTER_OCTET_LENGTH","NUMERIC_PRECISION","NUMERIC_SCALE","DATETIME_PRECISION","CHARACTER_SET_NAME","COLLATION_NAME","COLUMN_TYPE","COLUMN_KEY","EXTRA","PRIVILEGES","COLUMN_COMMENT","GENERATION_EXPRESSION","SRS_ID"],["VARCHAR","VARCHAR","VARCHAR","VARCHAR","UNSIGNED INT","TEXT","VARCHAR","TEXT","BIGINT","BIGINT","UNSIGNED BIGINT","UNSIGNED BIGINT","UNSIGNED INT","VARCHAR","VARCHAR","TEXT","CHAR","VARCHAR","VARCHAR","TEXT","TEXT","UNSIGNED INT"],[["def","testGhostSchemaUpdate","book","id","1",null,"NO","int",null,null,"10","0",null,null,null,"int","PRI","auto_increment","select,insert,update,references","","",null],["def","testGhostSchemaUpdate","book","name","2",null,"YES","text","65535","65535",null,null,null,"utf8mb4","utf8mb4_general_ci","text","","","select,insert,update,references","","",null],["def","testGhostSchemaUpdate","book","author","3",null,"YES","varchar","54","216",null,null,null,"utf8mb4","utf8mb4_general_ci","varchar(54)","","","select,insert,update,references","","",null]],[false,false,false,false,false,false,false,false,false,false,false,false,false,false,false,false,false,false,false,false,false,false]]`
)

func TestGhostParser(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	const statement = `
	ALTER TABLE
  		test
	ADD
		COLUMN ghost_play_2 int;
	`
	t.Run("fail to parse", func(t *testing.T) {
		t.Parallel()
		parser := ghostsql.NewParserFromAlterStatement(statement)
		a.Equal(false, parser.HasExplicitTable())
	})
	t.Run("succeed to parse", func(t *testing.T) {
		t.Parallel()
		s := strings.Join(strings.Fields(statement), " ")
		parser := ghostsql.NewParserFromAlterStatement(s)
		a.Equal(true, parser.HasExplicitTable())
		a.Equal("test", parser.GetExplicitTable())
	})
}

func TestGhostSchemaUpdate(t *testing.T) {
	const databaseName = "testGhostSchemaUpdate"

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

	mysqlPort := getTestPort()
	stopInstance := mysql.SetupTestInstance(t, mysqlPort, mysqlBinDir)
	defer stopInstance()

	mysqlDB, err := connectTestMySQL(mysqlPort, "")
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

	project, err := ctl.createProject(ctx)
	a.NoError(err)
	projectUID, err := strconv.Atoi(project.Uid)
	a.NoError(err)

	prodEnvironment, _, err := ctl.getEnvironment(ctx, "prod")
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
	a.Zero(len(databases))
	databases, err = ctl.getDatabases(api.DatabaseFind{
		InstanceID: &instanceUID,
	})
	a.NoError(err)
	a.Zero(len(databases))

	err = ctl.createDatabase(ctx, projectUID, instance, databaseName, "", nil)
	a.NoError(err)

	databases, err = ctl.getDatabases(api.DatabaseFind{
		ProjectID: &projectUID,
	})

	a.NoError(err)
	a.Equal(1, len(databases))

	database := databases[0]
	a.Equal(instanceUID, database.Instance.ID)

	sheet1, err := ctl.createSheet(api.SheetCreate{
		ProjectID:  projectUID,
		Name:       "mysql migration statement sheet 1",
		Statement:  mysqlMigrationStatement,
		Visibility: api.ProjectSheet,
		Source:     api.SheetFromBytebaseArtifact,
		Type:       api.SheetForSQL,
	})
	a.NoError(err)

	createContext, err := json.Marshal(&api.MigrationContext{
		DetailList: []*api.MigrationDetail{
			{
				MigrationType: db.Migrate,
				DatabaseID:    database.ID,
				SheetID:       sheet1.ID,
			},
		},
	})
	a.NoError(err)
	// Create an issue updating database schema.
	issue, err := ctl.createIssue(api.IssueCreate{
		ProjectID:     projectUID,
		Name:          fmt.Sprintf("update schema for database %q", databaseName),
		Type:          api.IssueDatabaseSchemaUpdate,
		Description:   fmt.Sprintf("This updates the schema of database %q.", databaseName),
		AssigneeID:    api.SystemBotID,
		CreateContext: string(createContext),
	})
	a.NoError(err)
	status, err := ctl.waitIssuePipeline(ctx, issue.ID)
	a.NoError(err)
	a.Equal(api.TaskDone, status)

	result, err := ctl.query(instance, databaseName, mysqlQueryBookTable)
	a.NoError(err)
	a.Equal(mysqlBookSchema1, result)

	sheet2, err := ctl.createSheet(api.SheetCreate{
		ProjectID:  projectUID,
		Name:       "migration statement sheet 2",
		Statement:  mysqlGhostMigrationStatement,
		Visibility: api.ProjectSheet,
		Source:     api.SheetFromBytebaseArtifact,
		Type:       api.SheetForSQL,
	})
	a.NoError(err)

	createContext, err = json.Marshal(&api.MigrationContext{
		DetailList: []*api.MigrationDetail{
			{
				MigrationType: db.Migrate,
				DatabaseID:    database.ID,
				SheetID:       sheet2.ID,
			},
		},
	})
	a.NoError(err)
	issue, err = ctl.createIssue(api.IssueCreate{
		ProjectID:     projectUID,
		Name:          fmt.Sprintf("update schema for database %q", databaseName),
		Type:          api.IssueDatabaseSchemaUpdateGhost,
		Description:   fmt.Sprintf("This updates the schema of database %q using gh-ost", databaseName),
		AssigneeID:    api.SystemBotID,
		CreateContext: string(createContext),
	})
	a.NoError(err)

	status, err = ctl.waitIssuePipeline(ctx, issue.ID)
	a.NoError(err)
	a.Equal(api.TaskDone, status)

	result, err = ctl.query(instance, databaseName, mysqlQueryBookTable)
	a.NoError(err)
	a.Equal(mysqlBookSchema2, result)
}

func TestGhostTenant(t *testing.T) {
	const databaseName = "testGhostSchemaUpdate"

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
	err = ctl.setLicense()
	a.NoError(err)

	// Create a project.
	project, err := ctl.createProject(ctx)
	a.NoError(err)
	projectUID, err := strconv.Atoi(project.Uid)
	a.NoError(err)

	testEnvironment, _, err := ctl.getEnvironment(ctx, "test")
	a.NoError(err)
	prodEnvironment, _, err := ctl.getEnvironment(ctx, "prod")
	a.NoError(err)

	// Provision instances.
	var testInstances []*v1pb.Instance
	var prodInstances []*v1pb.Instance
	for i := 0; i < testTenantNumber; i++ {
		port, err := getMySQLInstanceForGhostTest(t)
		a.NoError(err)
		// Add the provisioned instances.
		instance, err := ctl.instanceServiceClient.CreateInstance(ctx, &v1pb.CreateInstanceRequest{
			InstanceId: generateRandomString("instance", 10),
			Instance: &v1pb.Instance{
				Title:       fmt.Sprintf("%s-%d", testInstanceName, i),
				Engine:      v1pb.Engine_MYSQL,
				Environment: testEnvironment.Name,
				DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: "127.0.0.1", Port: strconv.Itoa(port), Username: "bytebase", Password: "bytebase"}},
			},
		})
		a.NoError(err)
		testInstances = append(testInstances, instance)
	}
	for i := 0; i < prodTenantNumber; i++ {
		port, err := getMySQLInstanceForGhostTest(t)
		a.NoError(err)
		// Add the provisioned instances.
		instance, err := ctl.instanceServiceClient.CreateInstance(ctx, &v1pb.CreateInstanceRequest{
			InstanceId: generateRandomString("instance", 10),
			Instance: &v1pb.Instance{
				Title:       fmt.Sprintf("%s-%d", testInstanceName, i),
				Engine:      v1pb.Engine_MYSQL,
				Environment: prodEnvironment.Name,
				DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: "127.0.0.1", Port: strconv.Itoa(port), Username: "bytebase", Password: "bytebase"}},
			},
		})
		a.NoError(err)
		prodInstances = append(prodInstances, instance)
	}

	// Create deployment configuration.
	_, err = ctl.upsertDeploymentConfig(
		api.DeploymentConfigUpsert{
			ProjectID: projectUID,
		},
		deploymentSchedule,
	)
	a.NoError(err)

	// Create issues that create databases.
	for i, testInstance := range testInstances {
		err := ctl.createDatabase(ctx, projectUID, testInstance, databaseName, "", map[string]string{api.TenantLabelKey: fmt.Sprintf("tenant%d", i)})
		a.NoError(err)
	}
	for i, prodInstance := range prodInstances {
		err := ctl.createDatabase(ctx, projectUID, prodInstance, databaseName, "", map[string]string{api.TenantLabelKey: fmt.Sprintf("tenant%d", i)})
		a.NoError(err)
	}

	// Getting databases for each environment.
	databases, err := ctl.getDatabases(api.DatabaseFind{
		ProjectID: &projectUID,
	})
	a.NoError(err)

	var testDatabases []*api.Database
	var prodDatabases []*api.Database
	for _, testInstance := range testInstances {
		testInstanceUID, err := strconv.Atoi(testInstance.Uid)
		a.NoError(err)
		for _, database := range databases {
			if database.Instance.ID == testInstanceUID {
				testDatabases = append(testDatabases, database)
				break
			}
		}
	}
	for _, prodInstance := range prodInstances {
		prodInstanceUID, err := strconv.Atoi(prodInstance.Uid)
		a.NoError(err)
		for _, database := range databases {
			if database.Instance.ID == prodInstanceUID {
				prodDatabases = append(prodDatabases, database)
				break
			}
		}
	}
	a.Equal(testTenantNumber, len(testDatabases))
	a.Equal(prodTenantNumber, len(prodDatabases))

	sheet1, err := ctl.createSheet(api.SheetCreate{
		ProjectID:  projectUID,
		Name:       "migration statement sheet 1",
		Statement:  mysqlMigrationStatement,
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
				SheetID:       sheet1.ID,
			},
		},
	})
	a.NoError(err)
	issue, err := ctl.createIssue(api.IssueCreate{
		ProjectID:     projectUID,
		Name:          fmt.Sprintf("update schema for database %q", databaseName),
		Type:          api.IssueDatabaseSchemaUpdate,
		Description:   fmt.Sprintf("This updates the schema of database %q.", databaseName),
		AssigneeID:    api.SystemBotID,
		CreateContext: string(createContext),
	})
	a.NoError(err)
	status, err := ctl.waitIssuePipeline(ctx, issue.ID)
	a.NoError(err)
	a.Equal(api.TaskDone, status)

	// Query schema.
	for _, testInstance := range testInstances {
		result, err := ctl.query(testInstance, databaseName, mysqlQueryBookTable)
		a.NoError(err)
		a.Equal(mysqlBookSchema1, result)
	}
	for _, prodInstance := range prodInstances {
		result, err := ctl.query(prodInstance, databaseName, mysqlQueryBookTable)
		a.NoError(err)
		a.Equal(mysqlBookSchema1, result)
	}

	sheet2, err := ctl.createSheet(api.SheetCreate{
		ProjectID:  projectUID,
		Name:       "migration statement sheet 2",
		Statement:  mysqlGhostMigrationStatement,
		Visibility: api.ProjectSheet,
		Source:     api.SheetFromBytebaseArtifact,
		Type:       api.SheetForSQL,
	})
	a.NoError(err)

	// Create an issue that updates database schema using gh-ost.
	createContext, err = json.Marshal(&api.MigrationContext{
		DetailList: []*api.MigrationDetail{
			{
				MigrationType: db.Migrate,
				DatabaseID:    0,
				SheetID:       sheet2.ID,
			},
		},
	})
	a.NoError(err)
	issue, err = ctl.createIssue(api.IssueCreate{
		ProjectID:     projectUID,
		Name:          fmt.Sprintf("update schema for database %q", databaseName),
		Type:          api.IssueDatabaseSchemaUpdateGhost,
		Description:   fmt.Sprintf("This updates the schema of database %q.", databaseName),
		AssigneeID:    api.SystemBotID,
		CreateContext: string(createContext),
	})
	a.NoError(err)
	status, err = ctl.waitIssuePipeline(ctx, issue.ID)
	a.NoError(err)
	a.Equal(api.TaskDone, status)

	// Query schema.
	for _, testInstance := range testInstances {
		result, err := ctl.query(testInstance, databaseName, mysqlQueryBookTable)
		a.NoError(err)
		a.Equal(mysqlBookSchema2, result)
	}
	for _, prodInstance := range prodInstances {
		result, err := ctl.query(prodInstance, databaseName, mysqlQueryBookTable)
		a.NoError(err)
		a.Equal(mysqlBookSchema2, result)
	}
}

func getMySQLInstanceForGhostTest(t *testing.T) (int, error) {
	mysqlPort := getTestPort()
	stopInstance := mysql.SetupTestInstance(t, mysqlPort, mysqlBinDir)
	t.Cleanup(stopInstance)

	mysqlDB, err := connectTestMySQL(mysqlPort, "")
	if err != nil {
		return 0, err
	}
	defer mysqlDB.Close()

	if _, err := mysqlDB.Exec("DROP USER IF EXISTS bytebase"); err != nil {
		return 0, err
	}

	if _, err = mysqlDB.Exec("CREATE USER 'bytebase' IDENTIFIED WITH mysql_native_password BY 'bytebase'"); err != nil {
		return 0, err
	}

	if _, err = mysqlDB.Exec("GRANT ALTER, ALTER ROUTINE, CREATE, CREATE ROUTINE, CREATE VIEW, DELETE, DROP, EVENT, EXECUTE, INDEX, INSERT, PROCESS, REFERENCES, SELECT, SHOW DATABASES, SHOW VIEW, TRIGGER, UPDATE, USAGE, REPLICATION CLIENT, REPLICATION SLAVE, LOCK TABLES, RELOAD ON *.* to bytebase"); err != nil {
		return 0, err
	}
	return mysqlPort, nil
}
