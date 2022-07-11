//go:build mysql
// +build mysql

package tests

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common/log"
	"github.com/bytebase/bytebase/plugin/db"
	dbplugin "github.com/bytebase/bytebase/plugin/db"
	resourcemysql "github.com/bytebase/bytebase/resources/mysql"
	"go.uber.org/zap/zapcore"

	"github.com/stretchr/testify/require"
)

// TestBackupRestoreBasic tests basic backup and restore behavior
// The test plan is:
// TODO(dragonly): add routine/event/trigger
// 1. create schema with index and constraint and populate data
// 2. create a full backup
// 3. clear data
// 4. restore data
// 5. validate
func TestBackupRestoreBasic(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()

	port := getTestPort(t.Name())
	database := "backup_restore"
	table := "backup_restore"

	_, stop := resourcemysql.SetupTestInstance(t, port)
	defer stop()

	db, err := connectTestMySQL(port, "")
	a.NoError(err)
	defer db.Close()

	_, err = db.Exec(fmt.Sprintf(`
	CREATE DATABASE %s;
	USE %s;
	CREATE TABLE %s (
		id INT,
		PRIMARY KEY (id),
		CHECK (id >= 0)
	);
	CREATE VIEW v_%s AS SELECT * FROM %s;
	`, database, database, table, table, table))
	a.NoError(err)

	const numRecords = 10
	tx, err := db.Begin()
	a.NoError(err)
	for i := 0; i < numRecords; i++ {
		_, err = tx.Exec(fmt.Sprintf("INSERT INTO %s VALUES (%d)", table, i))
		a.NoError(err)
	}
	err = tx.Commit()
	a.NoError(err)

	// make a full backup
	driver, err := getTestMySQLDriver(ctx, strconv.Itoa(port), database)
	a.NoError(err)
	defer driver.Close(ctx)

	buf, _, err := doBackup(ctx, driver, database)
	a.NoError(err)
	t.Logf("backup content:\n%s", buf.String())

	// drop all tables and views
	_, err = db.Exec(fmt.Sprintf("DROP TABLE %s; DROP VIEW v_%s;", table, table))
	a.NoError(err)

	// restore
	err = driver.Restore(ctx, bufio.NewScanner(buf))
	a.NoError(err)

	// validate data
	rows, err := db.Query(fmt.Sprintf("SELECT * FROM %s ORDER BY id ASC", table))
	a.NoError(err)
	defer rows.Close()
	i := 0
	for rows.Next() {
		var col int
		a.NoError(rows.Scan(&col))
		a.Equal(i, col)
		i++
	}
	a.NoError(rows.Err())
	a.Equal(numRecords, i)
}

func TestPITR(t *testing.T) {
	const (
		numRowsTime0 = 10
		numRowsTime1 = 20
	)
	t.Parallel()
	a := require.New(t)
	log.SetLevel(zapcore.DebugLevel)

	ctx := context.Background()
	port := getTestPort(t.Name())
	ctl := &controller{}
	datadir := t.TempDir()
	err := ctl.StartServer(ctx, datadir, port)
	a.NoError(err)
	defer ctl.Close(ctx)
	err = ctl.Login()
	a.NoError(err)
	err = ctl.setLicense()
	a.NoError(err)

	project, err := ctl.createProject(api.ProjectCreate{
		Name:       "PITRTest",
		Key:        "PTT",
		TenantMode: api.TenantModeDisabled,
	})
	a.NoError(err)

	environments, err := ctl.getEnvironments()
	a.NoError(err)
	prodEnvironment, err := findEnvironment(environments, "Prod")
	a.NoError(err)

	t.Run("Buggy Application", func(t *testing.T) {
		t.Log(t.Name())
		databaseName := "buggy_application"

		port := getTestPort(t.Name())
		_, stopFn := resourcemysql.SetupTestInstance(t, port)
		defer stopFn()

		connCfg := getMySQLConnectionConfig(strconv.Itoa(port), "")
		instance, err := ctl.addInstance(api.InstanceCreate{
			EnvironmentID: prodEnvironment.ID,
			Name:          "BuggyApplicationInstance",
			Engine:        db.MySQL,
			Host:          connCfg.Host,
			Port:          connCfg.Port,
			Username:      connCfg.Username,
		})
		a.NoError(err)

		err = ctl.createDatabase(project, instance, databaseName, nil)
		a.NoError(err)

		databases, err := ctl.getDatabases(api.DatabaseFind{
			InstanceID: &instance.ID,
		})
		a.NoError(err)
		a.Equal(1, len(databases))
		database := databases[0]

		mysqlDB, _ := initPITRDB(t, databaseName, port)
		defer mysqlDB.Close()

		t.Logf("Insert data range [%d, %d]\n", 0, numRowsTime0)
		insertRangeData(t, mysqlDB, 0, numRowsTime0)

		t.Log("Create a full backup")
		backup, err := ctl.createBackup(api.BackupCreate{
			DatabaseID:     database.ID,
			Name:           "first-backup",
			Type:           api.BackupTypeManual,
			StorageBackend: api.BackupStorageBackendLocal,
		})
		a.NoError(err)
		err = ctl.waitBackup(database.ID, backup.ID)
		a.NoError(err)

		t.Logf("Insert more data range [%d, %d]\n", numRowsTime0, numRowsTime1)
		insertRangeData(t, mysqlDB, numRowsTime0, numRowsTime1)

		ctxUpdateRow, cancelUpdateRow := context.WithCancel(ctx)
		targetTs := startUpdateRow(ctxUpdateRow, t, databaseName, port) + 1
		createCtx, err := json.Marshal(&api.PITRContext{
			DatabaseID:    database.ID,
			PointInTimeTs: targetTs,
		})
		a.NoError(err)
		issue, err := ctl.createIssue(api.IssueCreate{
			ProjectID:     project.ID,
			Name:          fmt.Sprintf("Restore database %s to the time %d", databaseName, targetTs),
			Type:          api.IssueDatabasePITR,
			AssigneeID:    project.Creator.ID,
			CreateContext: string(createCtx),
		})
		a.NoError(err)

		// Restore stage.
		status, err := ctl.waitIssueNextTaskWithTaskApproval(issue.ID)
		a.NoError(err)
		a.Equal(status, api.TaskDone)

		cancelUpdateRow()
		// We mimics the situation where the user waits for the target database idle before doing the cutover.
		time.Sleep(time.Second)

		// Cutover stage.
		status, err = ctl.waitIssueNextTaskWithTaskApproval(issue.ID)
		a.NoError(err)
		a.Equal(status, api.TaskDone)

		t.Log("Validate table tbl0")
		validateTbl0(t, mysqlDB, databaseName, numRowsTime1)
		t.Log("Validate table tbl0")
		validateTbl1(t, mysqlDB, databaseName, numRowsTime1)
		t.Log("validate table _update_row_")
		validateTableUpdateRow(t, mysqlDB, databaseName)
	})

	t.Run("Schema Migration Failure", func(t *testing.T) {
		t.Log(t.Name())
		databaseName := "schema_migration_failure"

		port := getTestPort(t.Name())
		_, stopFn := resourcemysql.SetupTestInstance(t, port)
		defer stopFn()

		connCfg := getMySQLConnectionConfig(strconv.Itoa(port), "")
		instance, err := ctl.addInstance(api.InstanceCreate{
			EnvironmentID: prodEnvironment.ID,
			Name:          "SchemaMigrationFailureInstance",
			Engine:        db.MySQL,
			Host:          connCfg.Host,
			Port:          connCfg.Port,
			Username:      connCfg.Username,
		})
		a.NoError(err)

		err = ctl.createDatabase(project, instance, databaseName, nil)
		a.NoError(err)

		databases, err := ctl.getDatabases(api.DatabaseFind{
			InstanceID: &instance.ID,
		})
		a.NoError(err)
		a.Equal(1, len(databases))
		database := databases[0]

		mysqlDB, _ := initPITRDB(t, databaseName, port)
		defer mysqlDB.Close()

		t.Logf("Insert data range [%d, %d]\n", 0, numRowsTime0)
		insertRangeData(t, mysqlDB, 0, numRowsTime0)

		t.Log("Create a full backup")
		backup, err := ctl.createBackup(api.BackupCreate{
			DatabaseID:     database.ID,
			Name:           "first-backup",
			Type:           api.BackupTypeManual,
			StorageBackend: api.BackupStorageBackendLocal,
		})
		a.NoError(err)
		err = ctl.waitBackup(database.ID, backup.ID)
		a.NoError(err)

		t.Logf("Insert more data range [%d, %d]\n", numRowsTime0, numRowsTime1)
		insertRangeData(t, mysqlDB, numRowsTime0, numRowsTime1)

		ctxUpdateRow, cancelUpdateRow := context.WithCancel(ctx)
		targetTs := startUpdateRow(ctxUpdateRow, t, databaseName, port) + 1

		dropColumnStmt := `ALTER TABLE tbl1 DROP COLUMN id;`
		t.Logf("mimics schema migration: %s\n", dropColumnStmt)
		_, err = mysqlDB.ExecContext(ctx, dropColumnStmt)
		a.NoError(err)

		createCtx, err := json.Marshal(&api.PITRContext{
			DatabaseID:    database.ID,
			PointInTimeTs: targetTs,
		})
		a.NoError(err)
		issue, err := ctl.createIssue(api.IssueCreate{
			ProjectID:     project.ID,
			Name:          fmt.Sprintf("Restore database %s to the time %d", databaseName, targetTs),
			Type:          api.IssueDatabasePITR,
			AssigneeID:    project.Creator.ID,
			CreateContext: string(createCtx),
		})
		a.NoError(err)

		// Restore stage.
		status, err := ctl.waitIssueNextTaskWithTaskApproval(issue.ID)
		a.NoError(err)
		a.Equal(status, api.TaskDone)

		cancelUpdateRow()
		// We mimics the situation where the user waits for the target database idle before doing the cutover.
		time.Sleep(time.Second)

		// Cutover stage.
		status, err = ctl.waitIssueNextTaskWithTaskApproval(issue.ID)
		a.NoError(err)
		a.Equal(status, api.TaskDone)

		t.Log("Validate table tbl0")
		validateTbl0(t, mysqlDB, databaseName, numRowsTime1)
		t.Log("Validate table tbl0")
		validateTbl1(t, mysqlDB, databaseName, numRowsTime1)
		t.Log("validate table _update_row_")
		validateTableUpdateRow(t, mysqlDB, databaseName)
	})

	t.Run("Drop Database", func(t *testing.T) {
		t.Log(t.Name())
		databaseName := "drop_database"

		port := getTestPort(t.Name())
		_, stopFn := resourcemysql.SetupTestInstance(t, port)
		defer stopFn()

		connCfg := getMySQLConnectionConfig(strconv.Itoa(port), "")
		instance, err := ctl.addInstance(api.InstanceCreate{
			EnvironmentID: prodEnvironment.ID,
			Name:          "DropDatabaseInstance",
			Engine:        db.MySQL,
			Host:          connCfg.Host,
			Port:          connCfg.Port,
			Username:      connCfg.Username,
		})
		a.NoError(err)

		err = ctl.createDatabase(project, instance, databaseName, nil)
		a.NoError(err)

		databases, err := ctl.getDatabases(api.DatabaseFind{
			InstanceID: &instance.ID,
		})
		a.NoError(err)
		a.Equal(1, len(databases))
		database := databases[0]

		mysqlDB, _ := initPITRDB(t, databaseName, port)
		defer mysqlDB.Close()

		t.Logf("Insert data range [%d, %d]\n", 0, numRowsTime0)
		insertRangeData(t, mysqlDB, 0, numRowsTime0)

		t.Log("Create a full backup")
		backup, err := ctl.createBackup(api.BackupCreate{
			DatabaseID:     database.ID,
			Name:           "first-backup",
			Type:           api.BackupTypeManual,
			StorageBackend: api.BackupStorageBackendLocal,
		})
		a.NoError(err)
		err = ctl.waitBackup(database.ID, backup.ID)
		a.NoError(err)

		t.Logf("Insert more data range [%d, %d]\n", numRowsTime0, numRowsTime1)
		insertRangeData(t, mysqlDB, numRowsTime0, numRowsTime1)

		time.Sleep(1 * time.Second)
		targetTs := time.Now().Unix()

		createCtx, err := json.Marshal(&api.PITRContext{
			DatabaseID:    database.ID,
			PointInTimeTs: targetTs,
		})
		a.NoError(err)
		issue, err := ctl.createIssue(api.IssueCreate{
			ProjectID:     project.ID,
			Name:          fmt.Sprintf("Restore database %s to the time %d", databaseName, targetTs),
			Type:          api.IssueDatabasePITR,
			AssigneeID:    project.Creator.ID,
			CreateContext: string(createCtx),
		})
		a.NoError(err)

		// Restore stage.
		status, err := ctl.waitIssueNextTaskWithTaskApproval(issue.ID)
		a.NoError(err)
		a.Equal(status, api.TaskDone)

		// We mimics the situation where the user waits for the target database idle before doing the cutover.
		time.Sleep(time.Second)

		// Cutover stage.
		status, err = ctl.waitIssueNextTaskWithTaskApproval(issue.ID)
		a.NoError(err)
		a.Equal(status, api.TaskDone)

		t.Log("Validate table tbl0")
		validateTbl0(t, mysqlDB, databaseName, numRowsTime1)
		t.Log("Validate table tbl0")
		validateTbl1(t, mysqlDB, databaseName, numRowsTime1)
	})

	t.Run("Case Sensitive", func(t *testing.T) {
		// TODO(zp): This test currently only passes correctly on the linux platform, and fails on non-case-sensitive platforms such as MacOS.
		// This if block will be removed after the fix is complete.
		if runtime.GOOS != "linux" {
			return
		}
		t.Log(t.Name())
		databaseName := "CASE_sensitive"

		port := getTestPort(t.Name())
		_, stopFn := resourcemysql.SetupTestInstance(t, port)
		defer stopFn()

		connCfg := getMySQLConnectionConfig(strconv.Itoa(port), "")
		instance, err := ctl.addInstance(api.InstanceCreate{
			EnvironmentID: prodEnvironment.ID,
			Name:          "DropSensitiveInstance",
			Engine:        db.MySQL,
			Host:          connCfg.Host,
			Port:          connCfg.Port,
			Username:      connCfg.Username,
		})
		a.NoError(err)

		err = ctl.createDatabase(project, instance, databaseName, nil)
		a.NoError(err)

		databases, err := ctl.getDatabases(api.DatabaseFind{
			InstanceID: &instance.ID,
		})
		a.NoError(err)
		a.Equal(1, len(databases))
		database := databases[0]

		mysqlDB, _ := initPITRDB(t, databaseName, port)
		defer mysqlDB.Close()

		t.Logf("Insert data range [%d, %d]\n", 0, numRowsTime0)
		insertRangeData(t, mysqlDB, 0, numRowsTime0)

		t.Log("Create a full backup")
		backup, err := ctl.createBackup(api.BackupCreate{
			DatabaseID:     database.ID,
			Name:           "first-backup",
			Type:           api.BackupTypeManual,
			StorageBackend: api.BackupStorageBackendLocal,
		})
		a.NoError(err)
		err = ctl.waitBackup(database.ID, backup.ID)
		a.NoError(err)

		t.Logf("Insert more data range [%d, %d]\n", numRowsTime0, numRowsTime1)
		insertRangeData(t, mysqlDB, numRowsTime0, numRowsTime1)

		time.Sleep(1 * time.Second)
		targetTs := time.Now().Unix()

		dropStmt := fmt.Sprintf(`DROP DATABASE %s;`, databaseName)
		_, err = mysqlDB.ExecContext(ctx, dropStmt)
		a.NoError(err)

		rows, err := mysqlDB.Query(fmt.Sprintf(`SHOW DATABASES LIKE '%s';`, databaseName))
		a.NoError(err)
		defer rows.Close()
		for rows.Next() {
			var s string
			err := rows.Scan(&s)
			a.NoError(err)
			a.FailNow("Database still exists after dropped")
		}
		a.NoError(rows.Err())

		createCtx, err := json.Marshal(&api.PITRContext{
			DatabaseID:    database.ID,
			PointInTimeTs: targetTs,
		})
		a.NoError(err)
		issue, err := ctl.createIssue(api.IssueCreate{
			ProjectID:     project.ID,
			Name:          fmt.Sprintf("Restore database %s to the time %d", databaseName, targetTs),
			Type:          api.IssueDatabasePITR,
			AssigneeID:    project.Creator.ID,
			CreateContext: string(createCtx),
		})
		a.NoError(err)

		// Restore stage.
		status, err := ctl.waitIssueNextTaskWithTaskApproval(issue.ID)
		a.NoError(err)
		a.Equal(status, api.TaskDone)

		// We mimics the situation where the user waits for the target database idle before doing the cutover.
		time.Sleep(time.Second)

		// Cutover stage.
		status, err = ctl.waitIssueNextTaskWithTaskApproval(issue.ID)
		a.NoError(err)
		a.Equal(status, api.TaskDone)

		t.Log("Validate table tbl0")
		validateTbl0(t, mysqlDB, databaseName, numRowsTime1)
		t.Log("Validate table tbl0")
		validateTbl1(t, mysqlDB, databaseName, numRowsTime1)
	})
}

func initPITRDB(t *testing.T, database string, port int) (*sql.DB, func()) {
	a := require.New(t)

	var stopFn func()
	var db *sql.DB
	db, err := connectTestMySQL(port, "")
	a.NoError(err)

	stmt := fmt.Sprintf(`
	USE %s;
	CREATE TABLE tbl0 (
		id INT,
		PRIMARY KEY (id),
		CHECK (id >= 0)
	);
	CREATE TABLE tbl1 (
		id INT,
		pid INT,
		PRIMARY KEY (id),
		UNIQUE INDEX (pid),
		CONSTRAINT FOREIGN KEY (pid) REFERENCES tbl0(id) ON DELETE NO ACTION
	);
	`, database)
	_, err = db.Exec(stmt)
	a.NoError(err)

	return db, stopFn
}

func insertRangeData(t *testing.T, db *sql.DB, begin, end int) {
	a := require.New(t)
	tx, err := db.Begin()
	a.NoError(err)

	for i := begin; i < end; i++ {
		_, err := tx.Exec(fmt.Sprintf("INSERT INTO tbl0 VALUES (%d);", i))
		a.NoError(err)
		_, err = tx.Exec(fmt.Sprintf("INSERT INTO tbl1 VALUES (%d, %d);", i, i))
		a.NoError(err)
	}

	err = tx.Commit()
	a.NoError(err)
}

func validateTbl0(t *testing.T, db *sql.DB, databaseName string, numRows int) {
	a := require.New(t)
	rows, err := db.Query(fmt.Sprintf("SELECT * FROM %s.tbl0;", databaseName))
	a.NoError(err)
	defer rows.Close()
	i := 0
	for rows.Next() {
		var col int
		a.NoError(rows.Scan(&col))
		a.Equal(i, col)
		i++
	}
	a.NoError(rows.Err())
	a.Equal(numRows, i)
}

func validateTbl1(t *testing.T, db *sql.DB, databaseName string, numRows int) {
	a := require.New(t)
	rows, err := db.Query(fmt.Sprintf("SELECT * FROM %s.tbl1;", databaseName))
	a.NoError(err)
	defer rows.Close()
	i := 0
	for rows.Next() {
		var col1, col2 int
		a.NoError(rows.Scan(&col1, &col2))
		a.Equal(i, col1)
		a.Equal(i, col2)
		i++
	}
	a.NoError(rows.Err())
	a.Equal(numRows, i)
}

func validateTableUpdateRow(t *testing.T, db *sql.DB, databaseName string) {
	a := require.New(t)
	rows, err := db.Query(fmt.Sprintf("SELECT * FROM %s._update_row_;", databaseName))
	a.NoError(err)
	defer rows.Close()

	a.Equal(true, rows.Next())
	var col int
	a.NoError(rows.Scan(&col))
	a.Equal(0, col)
	a.Equal(false, rows.Next())

	a.NoError(rows.Err())
}

func doBackup(ctx context.Context, driver dbplugin.Driver, database string) (*bytes.Buffer, *api.BackupPayload, error) {
	var buf bytes.Buffer
	var backupPayload api.BackupPayload
	backupPayloadString, err := driver.Dump(ctx, database, &buf, false)
	if err != nil {
		return nil, nil, err
	}
	if err := json.Unmarshal([]byte(backupPayloadString), &backupPayload); err != nil {
		return nil, nil, err
	}
	return &buf, &backupPayload, nil
}

// Concurrently update a single row to mimic the ongoing business workload.
// Returns the timestamp after inserting the initial value so we could check the PITR is done right.
func startUpdateRow(ctx context.Context, t *testing.T, database string, port int) int64 {
	a := require.New(t)
	db, err := connectTestMySQL(port, database)
	a.NoError(err)

	t.Log("Start updating data concurrently")
	_, err = db.Exec("CREATE TABLE _update_row_ (id INT PRIMARY KEY);")
	a.NoError(err)

	// initial value is (0)
	_, err = db.Exec("INSERT INTO _update_row_ VALUES (0);")
	a.NoError(err)
	initTimestamp := time.Now().Unix()

	// Sleep for one second so that the concurrent update will start no earlier than initTimestamp+1.
	// This will make a clear boundary for the binlog recovery --stop-datetime.
	// For example, the recovery command is mysqlbinlog --stop-datetime `initTimestamp+1`, and the concurrent updates
	// later will no be recovered. Then we can validate by checking the table _update_row_ has the initial value (0).
	time.Sleep(time.Second)

	go func() {
		defer db.Close()
		ticker := time.NewTicker(1 * time.Millisecond)
		defer ticker.Stop()
		i := 0
		for {
			select {
			case <-ticker.C:
				_, err = db.Exec(fmt.Sprintf("UPDATE _update_row_ SET id = %d;", i))
				a.NoError(err)
				i++
			case <-ctx.Done():
				t.Log("Stop updating data concurrently")
				return
			}
		}
	}()

	return initTimestamp
}
