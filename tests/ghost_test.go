//go:build mysql
// +build mysql

package tests

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"testing"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/resources/mysql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
)

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

	port := getTestPort(t.Name()) + 3
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	dataDir := t.TempDir()
	err := ctl.StartServer(ctx, dataDir, getTestPort(t.Name()))
	a.NoError(err)
	defer ctl.Close(ctx)
	err = ctl.Login()
	a.NoError(err)
	err = ctl.setLicense()
	a.NoError(err)

	_, stopInstance := mysql.SetupTestInstance(t, port)
	defer stopInstance()

	mysqlDB, err := sql.Open("mysql", fmt.Sprintf("root@tcp(127.0.0.1:%d)/mysql", port))
	a.NoError(err)
	defer mysqlDB.Close()

	_, err = mysqlDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %v", databaseName))
	a.NoError(err)

	_, err = mysqlDB.Exec("DROP USER IF EXISTS bytebase")
	a.NoError(err)
	_, err = mysqlDB.Exec("CREATE USER 'bytebase' IDENTIFIED WITH mysql_native_password BY 'bytebase'")
	a.NoError(err)

	_, err = mysqlDB.Exec("GRANT ALTER, ALTER ROUTINE, CREATE, CREATE ROUTINE, CREATE VIEW, DELETE, DROP, EVENT, EXECUTE, INDEX, INSERT, PROCESS, REFERENCES, SELECT, SHOW DATABASES, SHOW VIEW, TRIGGER, UPDATE, USAGE, REPLICATION CLIENT, REPLICATION SLAVE, LOCK TABLES ON *.* to bytebase")
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
		Port:          strconv.Itoa(port),
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

	err = ctl.createDatabase(project, instance, databaseName, nil)
	a.NoError(err)

	databases, err = ctl.getDatabases(api.DatabaseFind{
		ProjectID: &project.ID,
	})

	a.NoError(err)
	a.Equal(1, len(databases))

	database := databases[0]
	a.Equal(database.Instance.ID, instance.ID)

	createContext, err := json.Marshal(&api.UpdateSchemaContext{
		MigrationType: db.Migrate,
		DetailList: []*api.UpdateSchemaDetail{
			{
				DatabaseID: database.ID,
				Statement:  mysqlMigrationStatement,
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

	createContext, err = json.Marshal(&api.UpdateSchemaGhostContext{
		DetailList: []*api.UpdateSchemaGhostDetail{
			{
				DatabaseID: database.ID,
				Statement:  mysqlGhostMigrationStatement,
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
	// Status will be FAILED because not implemented
	status, err = ctl.waitIssuePipeline(issue.ID)
	a.NoError(err)
	a.Equal(api.TaskDone, status)

	result, err = ctl.query(instance, databaseName, mysqlQueryBookTable)
	a.NoError(err)
	a.Equal(mysqlBookSchema2, result)
}
