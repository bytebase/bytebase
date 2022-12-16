package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/plugin/db/mysql"
	resourcemysql "github.com/bytebase/bytebase/resources/mysql"
	"github.com/bytebase/bytebase/resources/mysqlutil"
	"github.com/bytebase/bytebase/tests/fake"
)

func TestRollback(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	database := "funny\ndatabase"

	mysqlPort := getTestPort()
	stopFn := resourcemysql.SetupTestInstance(t, mysqlPort, mysqlBinDir)
	defer stopFn()

	db, err := connectTestMySQL(mysqlPort, "")
	a.NoError(err)
	defer db.Close()
	_, err = db.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE `%s`; USE `%s`; CREATE TABLE user (id INT PRIMARY KEY, name VARCHAR(64), balance INT);", database, database))
	a.NoError(err)
	_, err = db.ExecContext(ctx, "INSERT INTO user VALUES (1, 'alice\nalice', 100), (2, 'bob', 100), (3, 'cindy', 100);")
	a.NoError(err)
	_, err = db.ExecContext(ctx, "UPDATE user SET balance=90 WHERE id=1; UPDATE user SET balance=110 WHERE id=2; DELETE FROM user WHERE id=3;")
	a.NoError(err)

	resourceDir := t.TempDir()
	binDir, err := mysqlutil.Install(resourceDir)
	a.NoError(err)
	driver, err := getTestMySQLDriver(ctx, t, strconv.Itoa(mysqlPort), database, binDir)
	a.NoError(err)
	defer driver.Close(ctx)

	// Rotate to binlog.000002 so that it's easy to rollback the following transactions and check that the state is the same as now.
	_, err = db.ExecContext(ctx, "FLUSH BINARY LOGS;")
	a.NoError(err)
	_, err = driver.Execute(ctx, "UPDATE user SET balance=0;", false)
	a.NoError(err)
	_, err = driver.Execute(ctx, "DELETE FROM user;", false)
	a.NoError(err)

	// Restore data using generated rollback SQL.
	mysqlDriver, ok := driver.(*mysql.Driver)
	a.Equal(true, ok)
	binlogFileList := []string{"binlog.000002"}
	tableCatalog := map[string][]string{
		"user": {"id", "name", "balance"},
	}
	threadID, err := mysqlDriver.GetMigrationConnID(ctx)
	a.NoError(err)
	rollbackSQL, err := mysqlDriver.GenerateRollbackSQL(ctx, binlogFileList, 0, math.MaxInt64, threadID, tableCatalog)
	a.NoError(err)
	_, err = db.ExecContext(ctx, rollbackSQL)
	a.NoError(err)

	// Check for rollback state.
	rows, err := db.QueryContext(ctx, "SELECT * FROM user;")
	a.NoError(err)
	type record struct {
		id      int
		name    string
		balance int
	}
	var records []record
	for rows.Next() {
		var r record
		err = rows.Scan(&r.id, &r.name, &r.balance)
		a.NoError(err)
		records = append(records, r)
	}
	want := []record{
		{1, "alice\nalice", 90},
		{2, "bob", 110},
	}
	a.Equal(want, records)
}

func TestCreateRollbackIssueMySQL(t *testing.T) {
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
	defer func() {
		_ = ctl.Close(ctx)
	}()

	err = ctl.setLicense()
	a.NoError(err)

	// Create a MySQL instance.
	mysqlPort := getTestPort()
	stopInstance := resourcemysql.SetupTestInstance(t, mysqlPort, mysqlBinDir)
	defer stopInstance()

	// Create a project.
	project, err := ctl.createProject(
		api.ProjectCreate{
			Name: fmt.Sprintf("Project %s", t.Name()),
			Key:  "ROLLBACK",
		},
	)
	a.NoError(err)

	environments, err := ctl.getEnvironments()
	a.NoError(err)
	prodEnvironment, err := findEnvironment(environments, "Prod")
	a.NoError(err)
	connCfg := getMySQLConnectionConfig(strconv.Itoa(mysqlPort), "")
	// Add MySQL instance to Bytebase.
	instance, err := ctl.addInstance(api.InstanceCreate{
		EnvironmentID: prodEnvironment.ID,
		Name:          t.Name(),
		Engine:        db.MySQL,
		Host:          connCfg.Host,
		Port:          connCfg.Port,
		Username:      connCfg.Username,
	})
	a.NoError(err)
	t.Log("Instance added.")

	databaseName := t.Name()
	err = ctl.createDatabase(project, instance, databaseName, "", nil)
	a.NoError(err)
	databases, err := ctl.getDatabases(api.DatabaseFind{
		InstanceID: &instance.ID,
	})
	a.NoError(err)
	a.Equal(1, len(databases))
	database := databases[0]

	dbMySQL, err := connectTestMySQL(mysqlPort, "")
	a.NoError(err)
	_, err = dbMySQL.ExecContext(ctx, fmt.Sprintf(`
		USE %s;
		CREATE TABLE t (id INT PRIMARY KEY, name VARCHAR(20));
		INSERT INTO t VALUES (1, '1\n1'), (2, '2\n2'), (3, '3\n3')
	`, databaseName))
	a.NoError(err)
	t.Log("Schema initialized.")

	// Run a DML issue.
	createContext, err := json.Marshal(&api.MigrationContext{
		DetailList: []*api.MigrationDetail{
			{
				MigrationType: db.Data,
				DatabaseID:    database.ID,
				Statement: `
					DELETE FROM t WHERE id = 1;
					UPDATE t SET name = 'unknown\nunknown';
				`,
			},
		},
	})
	a.NoError(err)
	issue, err := ctl.createIssue(api.IssueCreate{
		ProjectID:     project.ID,
		Name:          "update data",
		Type:          api.IssueDatabaseDataUpdate,
		AssigneeID:    project.Creator.ID,
		CreateContext: string(createContext),
	})
	a.NoError(err)
	t.Logf("Issue %d created.", issue.ID)
	status, err := ctl.waitIssuePipeline(issue.ID)
	a.NoError(err)
	a.Equal(api.TaskDone, status)
	a.Len(issue.Pipeline.StageList, 1)
	a.Len(issue.Pipeline.StageList[0].TaskList, 1)
	task := issue.Pipeline.StageList[0].TaskList[0]

	// Check that the data is changed.
	type record struct {
		id   int
		name string
	}
	rows1, err := dbMySQL.QueryContext(ctx, "SELECT * FROM t;")
	a.NoError(err)
	defer rows1.Close()
	var records1 []record
	for rows1.Next() {
		var r record
		err = rows1.Scan(&r.id, &r.name)
		a.NoError(err)
		records1 = append(records1, r)
	}
	want1 := []record{{2, "unknown\nunknown"}, {3, "unknown\nunknown"}}
	a.Equal(want1, records1)

	// Run a rollback issue.
	var rollbackIssue *api.Issue
	rollbackCreateContext, err := json.Marshal(&api.RollbackContext{
		IssueID:    issue.ID,
		TaskIDList: []int{task.ID},
	})
	a.NoError(err)
	for i := 0; i < 10; i++ {
		// rollbackIssue, err = ctl.createRollbackIssue(issue.ID, []int{task.ID})
		rollbackIssue, err = ctl.createIssue(api.IssueCreate{
			ProjectID:     project.ID,
			Name:          "rollback",
			Type:          api.IssueDatabaseRollback,
			AssigneeID:    project.Creator.ID,
			CreateContext: string(rollbackCreateContext),
		})
		if err == nil {
			break
		}
		// Wait for the rollback SQL generation and retry.
		time.Sleep(3 * time.Second)
	}
	t.Logf("Rollback issue %d created.", rollbackIssue.ID)
	status, err = ctl.waitIssuePipeline(rollbackIssue.ID)
	a.NoError(err)
	a.Equal(api.TaskDone, status)
	// Re-query the issue to get the updated task, which has the RollbackFromIssueID and RollbackFromTaskID fields.
	rollbackIssue, err = ctl.getIssue(rollbackIssue.ID)
	a.NoError(err)
	a.Len(rollbackIssue.Pipeline.StageList, 1)
	a.Len(rollbackIssue.Pipeline.StageList[0].TaskList, 1)
	rollbackTask := rollbackIssue.Pipeline.StageList[0].TaskList[0]
	rollbackTaskPayload := &api.TaskDatabaseDataUpdatePayload{}
	err = json.Unmarshal([]byte(rollbackTask.Payload), rollbackTaskPayload)
	a.NoError(err)
	a.Equal(issue.ID, rollbackTaskPayload.RollbackFromIssueID)
	a.Equal(task.ID, rollbackTaskPayload.RollbackFromTaskID)

	// Check that the data is restored.
	rows2, err := dbMySQL.QueryContext(ctx, "SELECT * FROM t;")
	a.NoError(err)
	defer rows2.Close()
	var records2 []record
	for rows2.Next() {
		var r record
		err = rows2.Scan(&r.id, &r.name)
		a.NoError(err)
		records2 = append(records2, r)
	}
	want2 := []record{{1, "1\n1"}, {2, "2\n2"}, {3, "3\n3"}}
	a.Equal(want2, records2)
}
