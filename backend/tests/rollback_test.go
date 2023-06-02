package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/mysql"
	resourcemysql "github.com/bytebase/bytebase/backend/resources/mysql"
	"github.com/bytebase/bytebase/backend/resources/mysqlutil"
	"github.com/bytebase/bytebase/backend/tests/fake"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
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
	const binlogSizeLimit = 8 * 1024 * 1024
	rollbackSQL, err := mysqlDriver.GenerateRollbackSQL(ctx, binlogSizeLimit, binlogFileList, 0, math.MaxInt64, threadID, tableCatalog)
	a.NoError(err)
	_, err = db.ExecContext(ctx, rollbackSQL)
	a.NoError(err)

	// Check for rollback state.
	rows, err := db.QueryContext(ctx, "SELECT * FROM user;")
	a.NoError(err)
	defer rows.Close()
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
	a.NoError(rows.Err())
	want := []record{
		{1, "alice\nalice", 90},
		{2, "bob", 110},
	}
	a.Equal(want, records)
}

func TestCreateRollbackIssueMySQL(t *testing.T) {
	if testReleaseMode == common.ReleaseModeProd {
		t.Skip()
	}
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
	defer func() {
		_ = ctl.Close(ctx)
	}()

	// Create a MySQL instance.
	mysqlPort := getTestPort()
	stopInstance := resourcemysql.SetupTestInstance(t, mysqlPort, mysqlBinDir)
	defer stopInstance()

	// Create a project.
	project, err := ctl.createProject(ctx)
	a.NoError(err)
	projectUID, err := strconv.Atoi(project.Uid)
	a.NoError(err)

	prodEnvironment, _, err := ctl.getEnvironment(ctx, "prod")
	a.NoError(err)
	connCfg := getMySQLConnectionConfig(strconv.Itoa(mysqlPort), "")
	// Add MySQL instance to Bytebase.
	instance, err := ctl.instanceServiceClient.CreateInstance(ctx, &v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance", 10),
		Instance: &v1pb.Instance{
			Title:       "mysqlInstance",
			Engine:      v1pb.Engine_MYSQL,
			Environment: prodEnvironment.Name,
			DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: connCfg.Host, Port: connCfg.Port, Username: connCfg.Username}},
		},
	})
	a.NoError(err)
	t.Log("Instance added.")

	databaseName := t.Name()
	err = ctl.createDatabase(ctx, projectUID, instance, databaseName, "", nil)
	a.NoError(err)

	database, err := ctl.databaseServiceClient.GetDatabase(ctx, &v1pb.GetDatabaseRequest{
		Name: fmt.Sprintf("%s/databases/%s", instance.Name, databaseName),
	})
	a.NoError(err)
	databaseUID, err := strconv.Atoi(database.Uid)
	a.NoError(err)

	dbMySQL, err := connectTestMySQL(mysqlPort, "")
	a.NoError(err)
	_, err = dbMySQL.ExecContext(ctx, fmt.Sprintf(`
		USE %s;
		CREATE TABLE t (id INT PRIMARY KEY, name VARCHAR(20));
		INSERT INTO t VALUES (1, '1\n1'), (2, '2\n2'), (3, '3\n3')
	`, databaseName))
	a.NoError(err)
	t.Log("Schema initialized.")

	dmlSheet, err := ctl.sheetServiceClient.CreateSheet(ctx, &v1pb.CreateSheetRequest{
		Parent: project.Name,
		Sheet: &v1pb.Sheet{
			Title: "migration statement sheet",
			Content: []byte(`
			DELETE FROM t WHERE id = 1;
			UPDATE t SET name = 'unknown\nunknown';
		`),
			Visibility: v1pb.Sheet_VISIBILITY_PROJECT,
			Source:     v1pb.Sheet_SOURCE_BYTEBASE_ARTIFACT,
			Type:       v1pb.Sheet_TYPE_SQL,
		},
	})
	a.NoError(err)
	dmlSheetUID, err := strconv.Atoi(strings.TrimPrefix(dmlSheet.Name, fmt.Sprintf("%s/sheets/", project.Name)))
	a.NoError(err)

	// Run a DML issue.
	createContext, err := json.Marshal(&api.MigrationContext{
		DetailList: []*api.MigrationDetail{
			{
				MigrationType:   db.Data,
				DatabaseID:      databaseUID,
				SheetID:         dmlSheetUID,
				RollbackEnabled: true,
			},
		},
	})
	a.NoError(err)
	issue, err := ctl.createIssue(api.IssueCreate{
		ProjectID:     projectUID,
		Name:          "update data",
		Type:          api.IssueDatabaseDataUpdate,
		AssigneeID:    api.SystemBotID,
		CreateContext: string(createContext),
	})
	a.NoError(err)
	t.Logf("Issue %d created.", issue.ID)
	status, err := ctl.waitIssuePipeline(ctx, issue.ID)
	a.NoError(err)
	a.Equal(api.TaskDone, status)
	a.Len(issue.Pipeline.StageList, 1)
	a.Len(issue.Pipeline.StageList[0].TaskList, 1)

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
	a.NoError(rows1.Err())
	want1 := []record{{2, "unknown\nunknown"}, {3, "unknown\nunknown"}}
	a.Equal(want1, records1)

	// wait for rollback SQL generation
	for i := 0; i < 10; i++ {
		issue, err := ctl.getIssue(issue.ID)
		a.NoError(err)
		a.Len(issue.Pipeline.StageList, 1)
		a.Len(issue.Pipeline.StageList[0].TaskList, 1)
		task := issue.Pipeline.StageList[0].TaskList[0]
		var payload api.TaskDatabaseDataUpdatePayload
		err = json.Unmarshal([]byte(task.Payload), &payload)
		a.NoError(err)
		if payload.RollbackSQLStatus == api.RollbackSQLStatusDone {
			break
		}
		time.Sleep(3 * time.Second)
	}

	issue, err = ctl.getIssue(issue.ID)
	a.NoError(err)
	a.Len(issue.Pipeline.StageList, 1)
	a.Len(issue.Pipeline.StageList[0].TaskList, 1)
	task := issue.Pipeline.StageList[0].TaskList[0]
	var payload api.TaskDatabaseDataUpdatePayload
	err = json.Unmarshal([]byte(task.Payload), &payload)
	a.NoError(err)
	a.Equal(api.RollbackSQLStatusDone, payload.RollbackSQLStatus)

	// Run a rollback issue.
	var rollbackIssue *api.Issue
	rollbackCreateContext, err := json.Marshal(&api.MigrationContext{
		DetailList: []*api.MigrationDetail{
			{
				MigrationType: db.Data,
				DatabaseID:    databaseUID,
				SheetID:       payload.RollbackSheetID,
				RollbackDetail: &api.RollbackDetail{
					IssueID: issue.ID,
					TaskID:  task.ID,
				},
			},
		},
	})
	a.NoError(err)

	rollbackIssue, err = ctl.createIssue(api.IssueCreate{
		ProjectID:     projectUID,
		Name:          "rollback",
		Type:          api.IssueDatabaseDataUpdate,
		AssigneeID:    api.SystemBotID,
		CreateContext: string(rollbackCreateContext),
	})
	a.NoError(err)
	t.Logf("Rollback issue %d created.", rollbackIssue.ID)

	status, err = ctl.waitIssuePipeline(ctx, rollbackIssue.ID)
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
	a.NoError(rows2.Err())
	want2 := []record{{1, "1\n1"}, {2, "2\n2"}, {3, "3\n3"}}
	a.Equal(want2, records2)
}

func TestCreateRollbackIssueMySQLByPatch(t *testing.T) {
	if testReleaseMode == common.ReleaseModeProd {
		t.Skip()
	}
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
	defer func() {
		_ = ctl.Close(ctx)
	}()

	// Create a MySQL instance.
	mysqlPort := getTestPort()
	stopInstance := resourcemysql.SetupTestInstance(t, mysqlPort, mysqlBinDir)
	defer stopInstance()

	// Create a project.
	project, err := ctl.createProject(ctx)
	a.NoError(err)
	projectUID, err := strconv.Atoi(project.Uid)
	a.NoError(err)

	prodEnvironment, _, err := ctl.getEnvironment(ctx, "prod")
	a.NoError(err)
	connCfg := getMySQLConnectionConfig(strconv.Itoa(mysqlPort), "")
	// Add MySQL instance to Bytebase.
	instance, err := ctl.instanceServiceClient.CreateInstance(ctx, &v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance", 10),
		Instance: &v1pb.Instance{
			Title:       t.Name(),
			Engine:      v1pb.Engine_MYSQL,
			Environment: prodEnvironment.Name,
			DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: connCfg.Host, Port: connCfg.Port, Username: connCfg.Username}},
		},
	})
	a.NoError(err)
	t.Log("Instance added.")

	databaseName := t.Name()
	err = ctl.createDatabase(ctx, projectUID, instance, databaseName, "", nil)
	a.NoError(err)

	database, err := ctl.databaseServiceClient.GetDatabase(ctx, &v1pb.GetDatabaseRequest{
		Name: fmt.Sprintf("%s/databases/%s", instance.Name, databaseName),
	})
	a.NoError(err)
	databaseUID, err := strconv.Atoi(database.Uid)
	a.NoError(err)

	dbMySQL, err := connectTestMySQL(mysqlPort, "")
	a.NoError(err)
	_, err = dbMySQL.ExecContext(ctx, fmt.Sprintf(`
		USE %s;
		CREATE TABLE t (id INT PRIMARY KEY, name VARCHAR(20));
		INSERT INTO t VALUES (1, '1\n1'), (2, '2\n2'), (3, '3\n3')
	`, databaseName))
	a.NoError(err)
	t.Log("Schema initialized.")

	dmlSheet, err := ctl.sheetServiceClient.CreateSheet(ctx, &v1pb.CreateSheetRequest{
		Parent: project.Name,
		Sheet: &v1pb.Sheet{
			Title: "migration statement sheet",
			Content: []byte(`
			DELETE FROM t WHERE id = 1;
			UPDATE t SET name = 'unknown\nunknown';
		`),
			Visibility: v1pb.Sheet_VISIBILITY_PROJECT,
			Source:     v1pb.Sheet_SOURCE_BYTEBASE_ARTIFACT,
			Type:       v1pb.Sheet_TYPE_SQL,
		},
	})
	a.NoError(err)
	dmlSheetUID, err := strconv.Atoi(strings.TrimPrefix(dmlSheet.Name, fmt.Sprintf("%s/sheets/", project.Name)))
	a.NoError(err)

	// Run a DML issue with rollbackEnabled set to false.
	createContext, err := json.Marshal(&api.MigrationContext{
		DetailList: []*api.MigrationDetail{
			{
				MigrationType: db.Data,
				DatabaseID:    databaseUID,
				SheetID:       dmlSheetUID,
				// RollbackEnabled: true,
			},
		},
	})
	a.NoError(err)
	issue, err := ctl.createIssue(api.IssueCreate{
		ProjectID:     projectUID,
		Name:          "update data",
		Type:          api.IssueDatabaseDataUpdate,
		AssigneeID:    api.SystemBotID,
		CreateContext: string(createContext),
	})
	a.NoError(err)
	t.Logf("Issue %d created.", issue.ID)
	status, err := ctl.waitIssuePipeline(ctx, issue.ID)
	a.NoError(err)
	a.Equal(api.TaskDone, status)
	a.Len(issue.Pipeline.StageList, 1)
	a.Len(issue.Pipeline.StageList[0].TaskList, 1)
	task := issue.Pipeline.StageList[0].TaskList[0]

	// Patch rollbackEnabled to true.
	rollbackEnabled := true
	_, err = ctl.patchTask(api.TaskPatch{
		RollbackEnabled: &rollbackEnabled,
	}, issue.Pipeline.ID, task.ID)
	a.NoError(err)

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
	a.NoError(rows1.Err())
	want1 := []record{{2, "unknown\nunknown"}, {3, "unknown\nunknown"}}
	a.Equal(want1, records1)

	// wait for rollback SQL generation
	for i := 0; i < 10; i++ {
		issue, err := ctl.getIssue(issue.ID)
		a.NoError(err)
		a.Len(issue.Pipeline.StageList, 1)
		a.Len(issue.Pipeline.StageList[0].TaskList, 1)
		task := issue.Pipeline.StageList[0].TaskList[0]
		var payload api.TaskDatabaseDataUpdatePayload
		err = json.Unmarshal([]byte(task.Payload), &payload)
		a.NoError(err)
		if payload.RollbackSQLStatus == api.RollbackSQLStatusDone {
			break
		}
		time.Sleep(3 * time.Second)
	}

	issue, err = ctl.getIssue(issue.ID)
	a.NoError(err)
	a.Len(issue.Pipeline.StageList, 1)
	a.Len(issue.Pipeline.StageList[0].TaskList, 1)
	task = issue.Pipeline.StageList[0].TaskList[0]
	var payload api.TaskDatabaseDataUpdatePayload
	err = json.Unmarshal([]byte(task.Payload), &payload)
	a.NoError(err)
	a.Equal(api.RollbackSQLStatusDone, payload.RollbackSQLStatus)

	// Run a rollback issue.
	var rollbackIssue *api.Issue
	rollbackCreateContext, err := json.Marshal(&api.MigrationContext{
		DetailList: []*api.MigrationDetail{
			{
				MigrationType: db.Data,
				DatabaseID:    databaseUID,
				SheetID:       payload.RollbackSheetID,
				RollbackDetail: &api.RollbackDetail{
					IssueID: issue.ID,
					TaskID:  task.ID,
				},
			},
		},
	})
	a.NoError(err)

	rollbackIssue, err = ctl.createIssue(api.IssueCreate{
		ProjectID:     projectUID,
		Name:          "rollback",
		Type:          api.IssueDatabaseDataUpdate,
		AssigneeID:    api.SystemBotID,
		CreateContext: string(rollbackCreateContext),
	})
	a.NoError(err)
	t.Logf("Rollback issue %d created.", rollbackIssue.ID)

	status, err = ctl.waitIssuePipeline(ctx, rollbackIssue.ID)
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
	a.NoError(rows2.Err())
	want2 := []record{{1, "1\n1"}, {2, "2\n2"}, {3, "3\n3"}}
	a.Equal(want2, records2)
}

func TestRollbackCanceled(t *testing.T) {
	if testReleaseMode == common.ReleaseModeProd {
		t.Skip()
	}
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
	defer func() {
		_ = ctl.Close(ctx)
	}()

	// Create a MySQL instance.
	mysqlPort := getTestPort()
	stopInstance := resourcemysql.SetupTestInstance(t, mysqlPort, mysqlBinDir)
	defer stopInstance()

	// Create a project.
	project, err := ctl.createProject(ctx)
	a.NoError(err)
	projectUID, err := strconv.Atoi(project.Uid)
	a.NoError(err)

	prodEnvironment, _, err := ctl.getEnvironment(ctx, "prod")
	a.NoError(err)
	connCfg := getMySQLConnectionConfig(strconv.Itoa(mysqlPort), "")
	// Add MySQL instance to Bytebase.
	instance, err := ctl.instanceServiceClient.CreateInstance(ctx, &v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance", 10),
		Instance: &v1pb.Instance{
			Title:       t.Name(),
			Engine:      v1pb.Engine_MYSQL,
			Environment: prodEnvironment.Name,
			DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: connCfg.Host, Port: connCfg.Port, Username: connCfg.Username}},
		},
	})
	a.NoError(err)
	t.Log("Instance added.")

	databaseName := t.Name()
	err = ctl.createDatabase(ctx, projectUID, instance, databaseName, "", nil)
	a.NoError(err)

	database, err := ctl.databaseServiceClient.GetDatabase(ctx, &v1pb.GetDatabaseRequest{
		Name: fmt.Sprintf("%s/databases/%s", instance.Name, databaseName),
	})
	a.NoError(err)
	databaseUID, err := strconv.Atoi(database.Uid)
	a.NoError(err)

	dbMySQL, err := connectTestMySQL(mysqlPort, "")
	a.NoError(err)
	_, err = dbMySQL.ExecContext(ctx, fmt.Sprintf(`
		USE %s;
		CREATE TABLE t (id INT PRIMARY KEY, name VARCHAR(20));
		INSERT INTO t VALUES (1, '1\n1'), (2, '2\n2'), (3, '3\n3')
	`, databaseName))
	a.NoError(err)
	t.Log("Schema initialized.")

	sheet, err := ctl.sheetServiceClient.CreateSheet(ctx, &v1pb.CreateSheetRequest{
		Parent: project.Name,
		Sheet: &v1pb.Sheet{
			Title: "delete statement sheet",
			Content: []byte(`
			DELETE FROM t WHERE id = 1;
			UPDATE t SET name = 'unknown\nunknown';
		`),
			Visibility: v1pb.Sheet_VISIBILITY_PROJECT,
			Source:     v1pb.Sheet_SOURCE_BYTEBASE_ARTIFACT,
			Type:       v1pb.Sheet_TYPE_SQL,
		},
	})
	a.NoError(err)
	sheetUID, err := strconv.Atoi(strings.TrimPrefix(sheet.Name, fmt.Sprintf("%s/sheets/", project.Name)))
	a.NoError(err)

	// Run a DML issue.
	createContext, err := json.Marshal(&api.MigrationContext{
		DetailList: []*api.MigrationDetail{
			{
				MigrationType:   db.Data,
				DatabaseID:      databaseUID,
				SheetID:         sheetUID,
				RollbackEnabled: true,
			},
		},
	})
	a.NoError(err)
	issue, err := ctl.createIssue(api.IssueCreate{
		ProjectID:     projectUID,
		Name:          "update data",
		Type:          api.IssueDatabaseDataUpdate,
		AssigneeID:    api.SystemBotID,
		CreateContext: string(createContext),
	})
	a.NoError(err)
	t.Logf("Issue %d created.", issue.ID)
	status, err := ctl.waitIssuePipeline(ctx, issue.ID)
	a.NoError(err)
	a.Equal(api.TaskDone, status)
	a.Len(issue.Pipeline.StageList, 1)
	a.Len(issue.Pipeline.StageList[0].TaskList, 1)
	task := issue.Pipeline.StageList[0].TaskList[0]

	// Cancel rollback SQL generation.
	rollbackEnabled := false
	_, err = ctl.patchTask(api.TaskPatch{
		RollbackEnabled: &rollbackEnabled,
	}, task.PipelineID, task.ID)
	a.NoError(err)

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
	a.NoError(rows1.Err())
	want1 := []record{{2, "unknown\nunknown"}, {3, "unknown\nunknown"}}
	a.Equal(want1, records1)

	issue, err = ctl.getIssue(issue.ID)
	a.NoError(err)
	a.Len(issue.Pipeline.StageList, 1)
	a.Len(issue.Pipeline.StageList[0].TaskList, 1)
	task = issue.Pipeline.StageList[0].TaskList[0]
	var payload api.TaskDatabaseDataUpdatePayload
	err = json.Unmarshal([]byte(task.Payload), &payload)
	a.NoError(err)
	a.Equal(false, payload.RollbackEnabled)
}
