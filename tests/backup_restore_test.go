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
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common/log"
	"github.com/bytebase/bytebase/plugin/db"
	resourcemysql "github.com/bytebase/bytebase/resources/mysql"
	"github.com/bytebase/bytebase/tests/fake"

	"go.uber.org/zap"
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
// 5. validate.
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
	log.Debug("backup content", zap.String("content", buf.String()))

	// drop all tables and views
	_, err = db.Exec(fmt.Sprintf("DROP TABLE %s; DROP VIEW v_%s;", table, table))
	a.NoError(err)

	// restore
	err = driver.Restore(ctx, bufio.NewScanner(buf))
	a.NoError(err)

	// validate data
	tableAllRows, err := db.Query(fmt.Sprintf("SELECT * FROM %s ORDER BY id ASC", table))
	a.NoError(err)
	defer tableAllRows.Close()
	i := 0
	for tableAllRows.Next() {
		var col int
		a.NoError(tableAllRows.Scan(&col))
		a.Equal(i, col)
		i++
	}
	a.NoError(tableAllRows.Err())
	a.Equal(numRecords, i)
}

const (
	numRowsTime0 = 10
	numRowsTime1 = 20
	numRowsTime2 = 30
)

func TestPITR(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	log.SetLevel(zapcore.DebugLevel)

	ctx := context.Background()
	serverPort := getTestPort(t.Name())
	ctl := &controller{}
	dataDir := t.TempDir()
	err := ctl.StartServer(ctx, dataDir, fake.NewGitLab, serverPort)
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

	policy := api.BackupPlanPolicy{Schedule: api.BackupPlanPolicyScheduleUnset}
	buf, err := json.Marshal(policy)
	a.NoError(err)
	str := string(buf)
	err = ctl.upsertPolicy(api.PolicyUpsert{
		EnvironmentID: prodEnvironment.ID,
		Type:          api.PolicyTypeBackupPlan,
		Payload:       &str,
	})
	a.NoError(err)

	t.Run("Buggy Application", func(t *testing.T) {
		a := require.New(t)
		port := getTestPort(t.Name())
		mysqlDB, database, cleanFn := setUpForPITRTest(t, ctl, port, prodEnvironment.ID, project)
		defer cleanFn()

		insertRangeData(t, mysqlDB, numRowsTime0, numRowsTime1)

		ctxUpdateRow, cancelUpdateRow := context.WithCancel(ctx)
		targetTs := startUpdateRow(ctxUpdateRow, t, database.Name, port) + 1
		issue, err := createPITRIssue(ctl, project, database, targetTs)
		a.NoError(err)

		// Restore stage.
		status, err := ctl.waitIssueNextTaskWithTaskApproval(issue.ID)
		a.NoError(err)
		a.Equal(api.TaskDone, status)

		cancelUpdateRow()
		// We mimics the situation where the user waits for the target database idle before doing the cutover.
		time.Sleep(time.Second)

		// Cutover stage.
		status, err = ctl.waitIssueNextTaskWithTaskApproval(issue.ID)
		a.NoError(err)
		a.Equal(api.TaskDone, status)

		validateTbl0(t, mysqlDB, database.Name, numRowsTime1)
		validateTbl1(t, mysqlDB, database.Name, numRowsTime1)
		validateTableUpdateRow(t, mysqlDB, database.Name)
	})

	t.Run("Schema Migration Failure", func(t *testing.T) {
		a := require.New(t)
		port := getTestPort(t.Name())
		mysqlDB, database, cleanFn := setUpForPITRTest(t, ctl, port, prodEnvironment.ID, project)
		defer cleanFn()

		insertRangeData(t, mysqlDB, numRowsTime0, numRowsTime1)

		ctxUpdateRow, cancelUpdateRow := context.WithCancel(ctx)
		targetTs := startUpdateRow(ctxUpdateRow, t, database.Name, port) + 1

		dropColumnStmt := `ALTER TABLE tbl1 DROP COLUMN id;`
		log.Debug("mimics schema migration", zap.String("statement", dropColumnStmt))
		_, err = mysqlDB.ExecContext(ctx, dropColumnStmt)
		a.NoError(err)

		issue, err := createPITRIssue(ctl, project, database, targetTs)
		a.NoError(err)

		// Restore stage.
		status, err := ctl.waitIssueNextTaskWithTaskApproval(issue.ID)
		a.NoError(err)
		a.Equal(api.TaskDone, status)

		cancelUpdateRow()
		// We mimics the situation where the user waits for the target database idle before doing the cutover.
		time.Sleep(time.Second)

		// Cutover stage.
		status, err = ctl.waitIssueNextTaskWithTaskApproval(issue.ID)
		a.NoError(err)
		a.Equal(api.TaskDone, status)

		validateTbl0(t, mysqlDB, database.Name, numRowsTime1)
		validateTbl1(t, mysqlDB, database.Name, numRowsTime1)
		validateTableUpdateRow(t, mysqlDB, database.Name)
	})

	t.Run("Drop Database", func(t *testing.T) {
		a := require.New(t)
		port := getTestPort(t.Name())
		mysqlDB, database, cleanFn := setUpForPITRTest(t, ctl, port, prodEnvironment.ID, project)
		defer cleanFn()

		insertRangeData(t, mysqlDB, numRowsTime0, numRowsTime1)

		time.Sleep(1 * time.Second)
		targetTs := time.Now().Unix()

		dropStmt := fmt.Sprintf(`DROP DATABASE %s;`, database.Name)
		_, err = mysqlDB.ExecContext(ctx, dropStmt)
		a.NoError(err)

		dbRows, err := mysqlDB.Query(fmt.Sprintf(`SHOW DATABASES LIKE '%s';`, database.Name))
		a.NoError(err)
		defer dbRows.Close()
		for dbRows.Next() {
			var s string
			err := dbRows.Scan(&s)
			a.NoError(err)
			a.FailNow("Database still exists after dropped")
		}
		a.NoError(dbRows.Err())

		issue, err := createPITRIssue(ctl, project, database, targetTs)
		a.NoError(err)

		// Restore stage.
		status, err := ctl.waitIssueNextTaskWithTaskApproval(issue.ID)
		a.NoError(err)
		a.Equal(api.TaskDone, status)

		// We mimics the situation where the user waits for the target database idle before doing the cutover.
		time.Sleep(time.Second)

		// Cutover stage.
		status, err = ctl.waitIssueNextTaskWithTaskApproval(issue.ID)
		a.NoError(err)
		a.Equal(api.TaskDone, status)

		validateTbl0(t, mysqlDB, database.Name, numRowsTime1)
		validateTbl1(t, mysqlDB, database.Name, numRowsTime1)
		validateTableUpdateRow(t, mysqlDB, database.Name)
	})

	t.Run("Case Sensitive", func(t *testing.T) {
		a := require.New(t)
		port := getTestPort(t.Name())
		mysqlDB, database, cleanFn := setUpForPITRTest(t, ctl, port, prodEnvironment.ID, project)
		defer cleanFn()

		insertRangeData(t, mysqlDB, numRowsTime0, numRowsTime1)

		time.Sleep(1 * time.Second)
		targetTs := time.Now().Unix()

		dropStmt := fmt.Sprintf(`DROP DATABASE %s;`, database.Name)
		_, err = mysqlDB.ExecContext(ctx, dropStmt)
		a.NoError(err)

		dbRows, err := mysqlDB.Query(fmt.Sprintf(`SHOW DATABASES LIKE '%s';`, database.Name))
		a.NoError(err)
		defer dbRows.Close()
		for dbRows.Next() {
			var s string
			err := dbRows.Scan(&s)
			a.NoError(err)
			a.FailNow("Database still exists after dropped")
		}
		a.NoError(dbRows.Err())

		issue, err := createPITRIssue(ctl, project, database, targetTs)
		a.NoError(err)

		// Restore stage.
		status, err := ctl.waitIssueNextTaskWithTaskApproval(issue.ID)
		a.NoError(err)
		a.Equal(api.TaskDone, status)

		// We mimics the situation where the user waits for the target database idle before doing the cutover.
		time.Sleep(time.Second)

		// Cutover stage.
		status, err = ctl.waitIssueNextTaskWithTaskApproval(issue.ID)
		a.NoError(err)
		a.Equal(api.TaskDone, status)

		validateTbl0(t, mysqlDB, database.Name, numRowsTime1)
		validateTbl1(t, mysqlDB, database.Name, numRowsTime1)
		validateTableUpdateRow(t, mysqlDB, database.Name)
	})

	t.Run("PITR Twice", func(t *testing.T) {
		a := require.New(t)
		port := getTestPort(t.Name())
		mysqlDB, database, cleanFn := setUpForPITRTest(t, ctl, port, prodEnvironment.ID, project)
		defer cleanFn()

		log.Debug("Creating issue for the first PITR.")
		insertRangeData(t, mysqlDB, numRowsTime0, numRowsTime1)
		ctxUpdateRow, cancelUpdateRow := context.WithCancel(ctx)
		targetTs := startUpdateRow(ctxUpdateRow, t, database.Name, port) + 1
		issue, err := createPITRIssue(ctl, project, database, targetTs)
		a.NoError(err)

		// Restore stage.
		status, err := ctl.waitIssueNextTaskWithTaskApproval(issue.ID)
		a.NoError(err)
		a.Equal(api.TaskDone, status)

		cancelUpdateRow()
		// We mimics the situation where the user waits for the target database idle before doing the cutover.
		time.Sleep(time.Second)

		// Cutover stage.
		status, err = ctl.waitIssueNextTaskWithTaskApproval(issue.ID)
		a.NoError(err)
		a.Equal(api.TaskDone, status)

		validateTbl0(t, mysqlDB, database.Name, numRowsTime1)
		validateTbl1(t, mysqlDB, database.Name, numRowsTime1)
		validateTableUpdateRow(t, mysqlDB, database.Name)
		log.Debug("First PITR done.")

		log.Debug("Wait for the first PITR auto backup to finish.")
		backups, err := ctl.listBackups(database.ID)
		a.NoError(err)
		a.Equal(2, len(backups))
		sort.Slice(backups, func(i int, j int) bool {
			return backups[i].CreatedTs > backups[j].CreatedTs
		})
		err = ctl.waitBackup(database.ID, backups[0].ID)
		a.NoError(err)

		log.Debug("Creating issue for the second PITR.")
		ctxUpdateRow, cancelUpdateRow = context.WithCancel(ctx)
		targetTs = startUpdateRow(ctxUpdateRow, t, database.Name, port) + 1
		insertRangeData(t, mysqlDB, numRowsTime1, numRowsTime2)
		issue2, err := createPITRIssue(ctl, project, database, targetTs)
		a.NoError(err)

		// Restore stage.
		status, err = ctl.waitIssueNextTaskWithTaskApproval(issue2.ID)
		a.NoError(err)
		a.Equal(api.TaskDone, status)

		cancelUpdateRow()
		// We mimics the situation where the user waits for the target database idle before doing the cutover.
		time.Sleep(time.Second)

		// Cutover stage.
		status, err = ctl.waitIssueNextTaskWithTaskApproval(issue2.ID)
		a.NoError(err)
		a.Equal(api.TaskDone, status)

		// Second PITR
		validateTbl0(t, mysqlDB, database.Name, numRowsTime1)
		validateTbl1(t, mysqlDB, database.Name, numRowsTime1)
		validateTableUpdateRow(t, mysqlDB, database.Name)
		log.Debug("Second PITR done.")
	})

	t.Run("Invalid Time Point", func(t *testing.T) {
		a := require.New(t)
		port := getTestPort(t.Name())
		targetTs := time.Now().Unix()
		_, database, cleanFn := setUpForPITRTest(t, ctl, port, prodEnvironment.ID, project)
		defer cleanFn()

		issue, err := createPITRIssue(ctl, project, database, targetTs)
		a.NoError(err)

		status, err := ctl.waitIssueNextTaskWithTaskApproval(issue.ID)
		a.Error(err)
		a.Equal(api.TaskFailed, status)
	})
}

func createPITRIssue(ctl *controller, project *api.Project, database *api.Database, targetTs int64) (*api.Issue, error) {
	pitrIssueCtx, err := json.Marshal(&api.PITRContext{
		DatabaseID:    database.ID,
		PointInTimeTs: targetTs,
	})
	if err != nil {
		return nil, err
	}
	return ctl.createIssue(api.IssueCreate{
		ProjectID:     project.ID,
		Name:          fmt.Sprintf("Restore database %s to the time %d", database.Name, targetTs),
		Type:          api.IssueDatabasePITR,
		AssigneeID:    project.Creator.ID,
		CreateContext: string(pitrIssueCtx),
	})
}

func setUpForPITRTest(t *testing.T, ctl *controller, port, envID int, project *api.Project) (*sql.DB, *api.Database, func()) {
	a := require.New(t)

	baseName := strings.ReplaceAll(t.Name(), "/", "_")
	databaseName := baseName + "Database"

	_, stopInstance := resourcemysql.SetupTestInstance(t, port)
	connCfg := getMySQLConnectionConfig(strconv.Itoa(port), "")
	instance, err := ctl.addInstance(api.InstanceCreate{
		EnvironmentID: envID,
		Name:          baseName + "Instance",
		Engine:        db.MySQL,
		Host:          connCfg.Host,
		Port:          connCfg.Port,
		Username:      connCfg.Username,
	})
	a.NoError(err)

	err = ctl.createDatabase(project, instance, databaseName, "", nil)
	a.NoError(err)

	databases, err := ctl.getDatabases(api.DatabaseFind{
		InstanceID: &instance.ID,
	})
	a.NoError(err)
	a.Equal(1, len(databases))
	database := databases[0]

	err = ctl.disableAutomaticBackup(database.ID)
	a.NoError(err)

	mysqlDB := initPITRDB(t, databaseName, port)

	insertRangeData(t, mysqlDB, 0, numRowsTime0)

	log.Debug("Create a full backup")
	backup, err := ctl.createBackup(api.BackupCreate{
		DatabaseID:     database.ID,
		Name:           "first-backup",
		Type:           api.BackupTypeManual,
		StorageBackend: api.BackupStorageBackendLocal,
	})
	a.NoError(err)
	err = ctl.waitBackup(database.ID, backup.ID)
	a.NoError(err)

	return mysqlDB, database, func() {
		stopInstance()
		mysqlDB.Close()
	}
}

func initPITRDB(t *testing.T, database string, port int) *sql.DB {
	a := require.New(t)

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
	CREATE TABLE _update_row_ (id INT PRIMARY KEY);
	INSERT INTO _update_row_ VALUES (0);
	`, database)
	_, err = db.Exec(stmt)
	a.NoError(err)

	return db
}

func insertRangeData(t *testing.T, db *sql.DB, begin, end int) {
	log.Debug("Inserting range data", zap.Int("begin", begin), zap.Int("end", end))
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
	log.Debug("Validate table tbl0")
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
	log.Debug("Validate table tbl1")
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
	log.Debug("Validate table _update_row_")
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

func doBackup(ctx context.Context, driver db.Driver, database string) (*bytes.Buffer, *api.BackupPayload, error) {
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

	log.Debug("Start updating data concurrently")
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
				log.Debug("Stop updating data concurrently")
				return
			}
		}
	}()

	return initTimestamp
}
