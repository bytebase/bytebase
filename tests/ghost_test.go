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

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/resources/mysql"
	"github.com/bytebase/bytebase/tests/fake"

	ghostsql "github.com/github/gh-ost/go/sql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
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
	const (
		databaseName            = "testGhostSchemaUpdate"
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
		mysqlBookSchema1 = `[["TABLE_CATALOG","TABLE_SCHEMA","TABLE_NAME","COLUMN_NAME","ORDINAL_POSITION","COLUMN_DEFAULT","IS_NULLABLE","DATA_TYPE","CHARACTER_MAXIMUM_LENGTH","CHARACTER_OCTET_LENGTH","NUMERIC_PRECISION","NUMERIC_SCALE","DATETIME_PRECISION","CHARACTER_SET_NAME","COLLATION_NAME","COLUMN_TYPE","COLUMN_KEY","EXTRA","PRIVILEGES","COLUMN_COMMENT","GENERATION_EXPRESSION","SRS_ID"],["VARCHAR","VARCHAR","VARCHAR","VARCHAR","INT","TEXT","VARCHAR","TEXT","BIGINT","BIGINT","BIGINT","BIGINT","INT","VARCHAR","VARCHAR","TEXT","CHAR","VARCHAR","VARCHAR","TEXT","TEXT","INT"],[["def","testGhostSchemaUpdate","book","id",1,null,"NO","int",null,null,"10","0",null,null,null,"int","PRI","auto_increment","select,insert,update,references","","",null],["def","testGhostSchemaUpdate","book","name",2,null,"YES","text","65535","65535",null,null,null,"utf8mb4","utf8mb4_general_ci","text","","","select,insert,update,references","","",null]]]`
		mysqlBookSchema2 = `[["TABLE_CATALOG","TABLE_SCHEMA","TABLE_NAME","COLUMN_NAME","ORDINAL_POSITION","COLUMN_DEFAULT","IS_NULLABLE","DATA_TYPE","CHARACTER_MAXIMUM_LENGTH","CHARACTER_OCTET_LENGTH","NUMERIC_PRECISION","NUMERIC_SCALE","DATETIME_PRECISION","CHARACTER_SET_NAME","COLLATION_NAME","COLUMN_TYPE","COLUMN_KEY","EXTRA","PRIVILEGES","COLUMN_COMMENT","GENERATION_EXPRESSION","SRS_ID"],["VARCHAR","VARCHAR","VARCHAR","VARCHAR","INT","TEXT","VARCHAR","TEXT","BIGINT","BIGINT","BIGINT","BIGINT","INT","VARCHAR","VARCHAR","TEXT","CHAR","VARCHAR","VARCHAR","TEXT","TEXT","INT"],[["def","testGhostSchemaUpdate","book","id",1,null,"NO","int",null,null,"10","0",null,null,null,"int","PRI","auto_increment","select,insert,update,references","","",null],["def","testGhostSchemaUpdate","book","name",2,null,"YES","text","65535","65535",null,null,null,"utf8mb4","utf8mb4_general_ci","text","","","select,insert,update,references","","",null],["def","testGhostSchemaUpdate","book","author",3,null,"YES","varchar","54","216",null,null,null,"utf8mb4","utf8mb4_general_ci","varchar(54)","","","select,insert,update,references","","",null]]]`
	)

	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	dataDir := t.TempDir()
	err := ctl.StartServerWithExternalPg(ctx, &config{
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

	project, err := ctl.createProject(api.ProjectCreate{
		Name: "Test Ghost Project",
		Key:  "TestGhostSchemaUpdate",
	})
	a.NoError(err)

	environments, err := ctl.getEnvironments()
	a.NoError(err)
	prodEnvironment, err := findEnvironment(environments, "Prod")
	a.NoError(err)

	instance, err := ctl.addInstance(api.InstanceCreate{
		EnvironmentID: prodEnvironment.ID,
		Name:          "mysqlInstance",
		Engine:        db.MySQL,
		Host:          "127.0.0.1",
		Port:          strconv.Itoa(mysqlPort),
		Username:      "bytebase",
		Password:      "bytebase",
	})
	a.NoError(err)

	databases, err := ctl.getDatabases(api.DatabaseFind{
		ProjectID: &project.ID,
	})
	a.NoError(err)
	a.Zero(len(databases))
	databases, err = ctl.getDatabases(api.DatabaseFind{
		InstanceID: &instance.ID,
	})
	a.NoError(err)
	a.Zero(len(databases))

	err = ctl.createDatabase(project, instance, databaseName, "", nil)
	a.NoError(err)

	databases, err = ctl.getDatabases(api.DatabaseFind{
		ProjectID: &project.ID,
	})

	a.NoError(err)
	a.Equal(1, len(databases))

	database := databases[0]
	a.Equal(instance.ID, database.Instance.ID)

	createContext, err := json.Marshal(&api.MigrationContext{
		DetailList: []*api.MigrationDetail{
			{
				MigrationType: db.Migrate,
				DatabaseID:    database.ID,
				Statement:     mysqlMigrationStatement,
			},
		},
	})
	a.NoError(err)
	// Create an issue updating database schema.
	issue, err := ctl.createIssue(api.IssueCreate{
		ProjectID:   project.ID,
		Name:        fmt.Sprintf("update schema for database %q", databaseName),
		Type:        api.IssueDatabaseSchemaUpdate,
		Description: fmt.Sprintf("This updates the schema of database %q.", databaseName),
		// Assign to self.
		AssigneeID:    project.Creator.ID,
		CreateContext: string(createContext),
	})
	a.NoError(err)
	status, err := ctl.waitIssuePipeline(issue.ID)
	a.NoError(err)
	a.Equal(api.TaskDone, status)

	result, err := ctl.query(instance, databaseName, mysqlQueryBookTable)
	a.NoError(err)
	a.Equal(mysqlBookSchema1, result)

	createContext, err = json.Marshal(&api.MigrationContext{
		DetailList: []*api.MigrationDetail{
			{
				MigrationType: db.Migrate,
				DatabaseID:    database.ID,
				Statement:     mysqlGhostMigrationStatement,
			},
		},
	})
	a.NoError(err)
	issue, err = ctl.createIssue(api.IssueCreate{
		ProjectID:     project.ID,
		Name:          fmt.Sprintf("update schema for database %q", databaseName),
		Type:          api.IssueDatabaseSchemaUpdateGhost,
		Description:   fmt.Sprintf("This updates the schema of database %q using gh-ost", databaseName),
		AssigneeID:    project.Creator.ID,
		CreateContext: string(createContext),
	})
	a.NoError(err)

	status, err = ctl.waitIssuePipeline(issue.ID)
	a.NoError(err)
	a.Equal(api.TaskDone, status)

	result, err = ctl.query(instance, databaseName, mysqlQueryBookTable)
	a.NoError(err)
	a.Equal(mysqlBookSchema2, result)
}

func TestGhostTenant(t *testing.T) {
	var (
		databaseName            = "testGhostSchemaUpdate"
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
		mysqlBookSchema1 = `[["TABLE_CATALOG","TABLE_SCHEMA","TABLE_NAME","COLUMN_NAME","ORDINAL_POSITION","COLUMN_DEFAULT","IS_NULLABLE","DATA_TYPE","CHARACTER_MAXIMUM_LENGTH","CHARACTER_OCTET_LENGTH","NUMERIC_PRECISION","NUMERIC_SCALE","DATETIME_PRECISION","CHARACTER_SET_NAME","COLLATION_NAME","COLUMN_TYPE","COLUMN_KEY","EXTRA","PRIVILEGES","COLUMN_COMMENT","GENERATION_EXPRESSION","SRS_ID"],["VARCHAR","VARCHAR","VARCHAR","VARCHAR","INT","TEXT","VARCHAR","TEXT","BIGINT","BIGINT","BIGINT","BIGINT","INT","VARCHAR","VARCHAR","TEXT","CHAR","VARCHAR","VARCHAR","TEXT","TEXT","INT"],[["def","testGhostSchemaUpdate","book","id",1,null,"NO","int",null,null,"10","0",null,null,null,"int","PRI","auto_increment","select,insert,update,references","","",null],["def","testGhostSchemaUpdate","book","name",2,null,"YES","text","65535","65535",null,null,null,"utf8mb4","utf8mb4_general_ci","text","","","select,insert,update,references","","",null]]]`
		mysqlBookSchema2 = `[["TABLE_CATALOG","TABLE_SCHEMA","TABLE_NAME","COLUMN_NAME","ORDINAL_POSITION","COLUMN_DEFAULT","IS_NULLABLE","DATA_TYPE","CHARACTER_MAXIMUM_LENGTH","CHARACTER_OCTET_LENGTH","NUMERIC_PRECISION","NUMERIC_SCALE","DATETIME_PRECISION","CHARACTER_SET_NAME","COLLATION_NAME","COLUMN_TYPE","COLUMN_KEY","EXTRA","PRIVILEGES","COLUMN_COMMENT","GENERATION_EXPRESSION","SRS_ID"],["VARCHAR","VARCHAR","VARCHAR","VARCHAR","INT","TEXT","VARCHAR","TEXT","BIGINT","BIGINT","BIGINT","BIGINT","INT","VARCHAR","VARCHAR","TEXT","CHAR","VARCHAR","VARCHAR","TEXT","TEXT","INT"],[["def","testGhostSchemaUpdate","book","id",1,null,"NO","int",null,null,"10","0",null,null,null,"int","PRI","auto_increment","select,insert,update,references","","",null],["def","testGhostSchemaUpdate","book","name",2,null,"YES","text","65535","65535",null,null,null,"utf8mb4","utf8mb4_general_ci","text","","","select,insert,update,references","","",null],["def","testGhostSchemaUpdate","book","author",3,null,"YES","varchar","54","216",null,null,null,"utf8mb4","utf8mb4_general_ci","varchar(54)","","","select,insert,update,references","","",null]]]`
	)
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	dataDir := t.TempDir()
	err := ctl.StartServerWithExternalPg(ctx, &config{
		dataDir:            dataDir,
		vcsProviderCreator: fake.NewGitLab,
	})
	a.NoError(err)
	defer ctl.Close(ctx)
	err = ctl.setLicense()
	a.NoError(err)

	// Create a project.
	project, err := ctl.createProject(api.ProjectCreate{
		Name:       "Test Project",
		Key:        "TestTenantGhost",
		TenantMode: api.TenantModeTenant,
	})
	a.NoError(err)

	environments, err := ctl.getEnvironments()
	a.NoError(err)
	testEnvironment, err := findEnvironment(environments, "Test")
	a.NoError(err)
	prodEnvironment, err := findEnvironment(environments, "Prod")
	a.NoError(err)

	// Provision instances.
	var testInstances []*api.Instance
	var prodInstances []*api.Instance
	for i := 0; i < testTenantNumber; i++ {
		port, err := getMySQLInstanceForGhostTest(t)
		a.NoError(err)
		// Add the provisioned instances.
		instance, err := ctl.addInstance((api.InstanceCreate{
			EnvironmentID: testEnvironment.ID,
			Name:          fmt.Sprintf("%s-%d", testInstanceName, i),
			Engine:        db.MySQL,
			Host:          "127.0.0.1",
			Port:          strconv.Itoa(port),
			Username:      "bytebase",
			Password:      "bytebase",
		}))
		a.NoError(err)
		testInstances = append(testInstances, instance)
	}
	for i := 0; i < prodTenantNumber; i++ {
		port, err := getMySQLInstanceForGhostTest(t)
		a.NoError(err)
		// Add the provisioned instances.
		instance, err := ctl.addInstance((api.InstanceCreate{
			EnvironmentID: prodEnvironment.ID,
			Name:          fmt.Sprintf("%s-%d", prodInstanceName, i),
			Engine:        db.MySQL,
			Host:          "127.0.0.1",
			Port:          strconv.Itoa(port),
			Username:      "bytebase",
			Password:      "bytebase",
		}))
		a.NoError(err)
		prodInstances = append(prodInstances, instance)
	}

	// Set up label values for tenants.
	// Prod and test are using the same tenant values. Use prodInstancesNumber because it's larger than testInstancesNumber.
	var tenants []string
	for i := 0; i < prodTenantNumber; i++ {
		tenants = append(tenants, fmt.Sprintf("tenant%d", i))
	}
	err = ctl.addLabelValues(api.TenantLabelKey, tenants)
	a.NoError(err)

	// Create deployment configuration.
	_, err = ctl.upsertDeploymentConfig(
		api.DeploymentConfigUpsert{
			ProjectID: project.ID,
		},
		deploymentSchedule,
	)
	a.NoError(err)

	// Create issues that create databases.
	for i, testInstance := range testInstances {
		err := ctl.createDatabase(project, testInstance, databaseName, "", map[string]string{api.TenantLabelKey: fmt.Sprintf("tenant%d", i)})
		a.NoError(err)
	}
	for i, prodInstance := range prodInstances {
		err := ctl.createDatabase(project, prodInstance, databaseName, "", map[string]string{api.TenantLabelKey: fmt.Sprintf("tenant%d", i)})
		a.NoError(err)
	}

	// Getting databases for each environment.
	databases, err := ctl.getDatabases(api.DatabaseFind{
		ProjectID: &project.ID,
	})
	a.NoError(err)

	var testDatabases []*api.Database
	var prodDatabases []*api.Database
	for _, testInstance := range testInstances {
		for _, database := range databases {
			if database.Instance.ID == testInstance.ID {
				testDatabases = append(testDatabases, database)
				break
			}
		}
	}
	for _, prodInstance := range prodInstances {
		for _, database := range databases {
			if database.Instance.ID == prodInstance.ID {
				prodDatabases = append(prodDatabases, database)
				break
			}
		}
	}
	a.Equal(testTenantNumber, len(testDatabases))
	a.Equal(prodTenantNumber, len(prodDatabases))

	// Create an issue that updates database schema.
	createContext, err := json.Marshal(&api.MigrationContext{
		DetailList: []*api.MigrationDetail{
			{
				MigrationType: db.Migrate,
				Statement:     mysqlMigrationStatement,
			},
		},
	})
	a.NoError(err)
	issue, err := ctl.createIssue(api.IssueCreate{
		ProjectID:   project.ID,
		Name:        fmt.Sprintf("update schema for database %q", databaseName),
		Type:        api.IssueDatabaseSchemaUpdate,
		Description: fmt.Sprintf("This updates the schema of database %q.", databaseName),
		// Assign to self.
		AssigneeID:    project.Creator.ID,
		CreateContext: string(createContext),
	})
	a.NoError(err)
	status, err := ctl.waitIssuePipeline(issue.ID)
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

	// Create an issue that updates database schema using gh-ost.
	createContext, err = json.Marshal(&api.MigrationContext{
		DetailList: []*api.MigrationDetail{
			{
				MigrationType: db.Migrate,
				DatabaseID:    0,
				Statement:     mysqlGhostMigrationStatement,
			},
		},
	})
	a.NoError(err)
	issue, err = ctl.createIssue(api.IssueCreate{
		ProjectID:   project.ID,
		Name:        fmt.Sprintf("update schema for database %q", databaseName),
		Type:        api.IssueDatabaseSchemaUpdateGhost,
		Description: fmt.Sprintf("This updates the schema of database %q.", databaseName),
		// Assign to self.
		AssigneeID:    project.Creator.ID,
		CreateContext: string(createContext),
	})
	a.NoError(err)
	status, err = ctl.waitIssuePipeline(issue.ID)
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
