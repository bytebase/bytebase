//go:build mysql
// +build mysql

package tests

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common/log"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/db"
	resourcemysql "github.com/bytebase/bytebase/backend/resources/mysql"
	"github.com/bytebase/bytebase/backend/tests/fake"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"

	"go.uber.org/zap"

	"github.com/stretchr/testify/require"
)

const (
	numRowsTime0 = 10
	numRowsTime1 = 20
	numRowsTime2 = 30
)

func TestRestoreToNewDatabase(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	project, mysqlDB, database, backup, _, cleanFn := setUpForPITRTest(ctx, t, ctl)
	defer cleanFn()

	metadata, err := ctl.getLatestSchemaMetadata(database.ID)
	a.NoError(err)
	var latestSchemaMetadata storepb.DatabaseMetadata
	err = protojson.Unmarshal([]byte(metadata), &latestSchemaMetadata)
	a.NoError(err)

	issue, err := createPITRIssue(ctl, project, api.PITRContext{
		DatabaseID: database.ID,
		BackupID:   &backup.ID,
		CreateDatabaseCtx: &api.CreateDatabaseContext{
			InstanceID:   database.InstanceID,
			DatabaseName: database.Name + "_new",
			CharacterSet: latestSchemaMetadata.CharacterSet,
			Collation:    latestSchemaMetadata.Collation,
			BackupID:     backup.ID,
		},
	})
	a.NoError(err)

	// Restore stage.
	status, err := ctl.waitIssueNextTaskWithTaskApproval(issue.ID)
	a.NoError(err)
	a.Equal(api.TaskDone, status)

	// Cutover stage.
	status, err = ctl.waitIssueNextTaskWithTaskApproval(issue.ID)
	a.NoError(err)
	a.Equal(api.TaskDone, status)

	validateTbl0(t, mysqlDB, database.Name, numRowsTime0)
	validateTbl1(t, mysqlDB, database.Name, numRowsTime0)
	validateTableUpdateRow(t, mysqlDB, database.Name)
}

func TestRetentionPolicy(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	_, _, database, backup, _, cleanFn := setUpForPITRTest(ctx, t, ctl)
	defer cleanFn()

	metaDB, err := sql.Open("pgx", ctl.profile.PgURL)
	a.NoError(err)
	a.NoError(metaDB.Ping())

	// Check that the backup file exist
	backupFilePath := filepath.Join(ctl.profile.DataDir, "backup", "db", fmt.Sprintf("%d", database.ID), fmt.Sprintf("%s.sql", backup.Name))
	_, err = os.Stat(backupFilePath)
	a.NoError(err)

	// Change retention period to 1s, and the backup should be quickly removed.
	// TODO(d): clean-up the hack.
	_, err = metaDB.ExecContext(ctx, fmt.Sprintf("UPDATE backup_setting SET enabled=true, retention_period_ts=1 WHERE database_id=%d;", database.ID))
	a.NoError(err)
	err = ctl.waitBackupArchived(database.ID, backup.ID)
	a.NoError(err)
	// Wait for 1s to delete the file.
	time.Sleep(1 * time.Second)
	_, err = os.Stat(backupFilePath)
	a.Equal(true, os.IsNotExist(err))
}

// TestPITRGeneral tests for the general PITR cases:
// 1. buggy application.
// 2. bad schema migration.
func TestPITRGeneral(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	project, mysqlDB, database, _, mysqlPort, cleanFn := setUpForPITRTest(ctx, t, ctl)
	defer cleanFn()

	insertRangeData(t, mysqlDB, numRowsTime0, numRowsTime1)

	ctxUpdateRow, cancelUpdateRow := context.WithCancel(ctx)
	targetTs := startUpdateRow(ctxUpdateRow, t, database.Name, mysqlPort) + 1

	dropColumnStmt := `ALTER TABLE tbl1 DROP COLUMN id;`
	log.Debug("mimics schema migration", zap.String("statement", dropColumnStmt))
	_, err := mysqlDB.ExecContext(ctx, dropColumnStmt)
	a.NoError(err)

	issue, err := createPITRIssue(ctl, project, api.PITRContext{
		DatabaseID:    database.ID,
		PointInTimeTs: &targetTs,
	})
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
}

func TestPITRDropDatabase(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	project, mysqlDB, database, _, _, cleanFn := setUpForPITRTest(ctx, t, ctl)
	defer cleanFn()

	insertRangeData(t, mysqlDB, numRowsTime0, numRowsTime1)

	time.Sleep(1 * time.Second)
	targetTs := time.Now().Unix()

	dropStmt := fmt.Sprintf(`DROP DATABASE %s;`, database.Name)
	_, err := mysqlDB.ExecContext(ctx, dropStmt)
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

	issue, err := createPITRIssue(ctl, project, api.PITRContext{
		DatabaseID:    database.ID,
		PointInTimeTs: &targetTs,
	})
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
}

func TestPITRTwice(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	project, mysqlDB, database, _, mysqlPort, cleanFn := setUpForPITRTest(ctx, t, ctl)
	defer cleanFn()

	log.Debug("Creating issue for the first PITR.")
	insertRangeData(t, mysqlDB, numRowsTime0, numRowsTime1)
	ctxUpdateRow, cancelUpdateRow := context.WithCancel(ctx)
	targetTs := startUpdateRow(ctxUpdateRow, t, database.Name, mysqlPort) + 1

	issue, err := createPITRIssue(ctl, project, api.PITRContext{
		DatabaseID:    database.ID,
		PointInTimeTs: &targetTs,
	})
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
	targetTs = startUpdateRow(ctxUpdateRow, t, database.Name, mysqlPort) + 1
	insertRangeData(t, mysqlDB, numRowsTime1, numRowsTime2)

	issue2, err := createPITRIssue(ctl, project, api.PITRContext{
		DatabaseID:    database.ID,
		PointInTimeTs: &targetTs,
	})
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
}

func TestPITRToNewDatabaseInAnotherInstance(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	project, sourceMySQLDB, database, _, mysqlPort, cleanFn := setUpForPITRTest(ctx, t, ctl)
	defer cleanFn()

	dstPort := getTestPort()
	dstStopFn := resourcemysql.SetupTestInstance(t, dstPort, mysqlBinDir)
	defer dstStopFn()
	dstConnCfg := getMySQLConnectionConfig(strconv.Itoa(dstPort), "")

	environments, err := ctl.getEnvironments()
	a.NoError(err)
	prodEnvironment, err := findEnvironment(environments, "Prod")
	a.NoError(err)
	dstInstance, err := ctl.addInstance(api.InstanceCreate{
		ResourceID: generateRandomString("instance", 10),
		// The target instance must be within the same environment as the instance of the original database now.
		EnvironmentID: prodEnvironment.ID,
		Name:          "DestinationInstance",
		Engine:        db.MySQL,
		Host:          dstConnCfg.Host,
		Port:          dstConnCfg.Port,
		Username:      dstConnCfg.Username,
	})
	a.NoError(err)

	insertRangeData(t, sourceMySQLDB, numRowsTime0, numRowsTime1)

	ctxUpdateRow, cancelUpdateRow := context.WithCancel(ctx)
	cancelUpdateRow()

	targetTs := startUpdateRow(ctxUpdateRow, t, database.Name, mysqlPort) + 1

	dropColumnStmt := `ALTER TABLE tbl1 DROP COLUMN id;`
	log.Debug("mimics schema migration", zap.String("statement", dropColumnStmt))
	_, err = sourceMySQLDB.ExecContext(ctx, dropColumnStmt)
	a.NoError(err)

	labels, err := marshalLabels(nil, dstInstance.Environment.ResourceID)
	a.NoError(err)

	targetDatabaseName := "new_database"
	pitrIssueCtx, err := json.Marshal(&api.PITRContext{
		DatabaseID:    database.ID,
		PointInTimeTs: &targetTs,
		CreateDatabaseCtx: &api.CreateDatabaseContext{
			InstanceID:   dstInstance.ID,
			DatabaseName: targetDatabaseName,
			Labels:       labels,
			CharacterSet: "utf8mb4",
			Collation:    "utf8mb4_general_ci",
		},
	})
	a.NoError(err)

	issue, err := ctl.createIssue(api.IssueCreate{
		ProjectID:     project.ID,
		Name:          fmt.Sprintf("Restore database %s to the time %d", database.Name, targetTs),
		Type:          api.IssueDatabaseRestorePITR,
		AssigneeID:    api.SystemBotID,
		CreateContext: string(pitrIssueCtx),
	})
	a.NoError(err)

	// Create database task
	status, err := ctl.waitIssueNextTaskWithTaskApproval(issue.ID)
	a.NoError(err)
	a.Equal(api.TaskDone, status)

	// Restore task
	status, err = ctl.waitIssueNextTaskWithTaskApproval(issue.ID)
	a.NoError(err)
	a.Equal(api.TaskDone, status)

	cancelUpdateRow()

	targetDB, err := connectTestMySQL(dstPort, targetDatabaseName)
	a.NoError(err)

	validateTbl0(t, targetDB, targetDatabaseName, numRowsTime1)
	validateTbl1(t, targetDB, targetDatabaseName, numRowsTime1)
	validateTableUpdateRow(t, targetDB, targetDatabaseName)
}

func TestPITRInvalidTimePoint(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	targetTs := time.Now().Unix()
	project, _, database, _, _, cleanFn := setUpForPITRTest(ctx, t, ctl)
	defer cleanFn()

	issue, err := createPITRIssue(ctl, project, api.PITRContext{
		DatabaseID:    database.ID,
		PointInTimeTs: &targetTs,
	})
	a.NoError(err)

	status, err := ctl.waitIssueNextTaskWithTaskApproval(issue.ID)
	a.Error(err)
	a.Equal(api.TaskFailed, status)
}

func createPITRIssue(ctl *controller, project *api.Project, pitrContext api.PITRContext) (*api.Issue, error) {
	pitrIssueCtx, err := json.Marshal(&pitrContext)
	if err != nil {
		return nil, err
	}
	return ctl.createIssue(api.IssueCreate{
		ProjectID:     project.ID,
		Name:          fmt.Sprintf("Restore database %d", pitrContext.DatabaseID),
		Type:          api.IssueDatabaseRestorePITR,
		AssigneeID:    api.SystemBotID,
		CreateContext: string(pitrIssueCtx),
	})
}

func setUpForPITRTest(ctx context.Context, t *testing.T, ctl *controller) (*api.Project, *sql.DB, *api.Database, *api.Backup, int, func()) {
	a := require.New(t)

	dataDir := t.TempDir()
	err := ctl.StartServerWithExternalPg(ctx, &config{
		dataDir:            dataDir,
		vcsProviderCreator: fake.NewGitLab,
	})
	a.NoError(err)
	err = ctl.setLicense()
	a.NoError(err)

	project, err := ctl.createProject(api.ProjectCreate{
		ResourceID: generateRandomString("project", 10),
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
	_, err = ctl.upsertPolicy(api.PolicyResourceTypeEnvironment, prodEnvironment.ID, api.PolicyTypeBackupPlan, api.PolicyUpsert{
		Payload: &str,
	})
	a.NoError(err)

	baseName := strings.ReplaceAll(t.Name(), "/", "_")
	databaseName := baseName + "_Database"

	mysqlPort := getTestPort()
	stopInstance := resourcemysql.SetupTestInstance(t, mysqlPort, mysqlBinDir)
	connCfg := getMySQLConnectionConfig(strconv.Itoa(mysqlPort), "")
	instance, err := ctl.addInstance(api.InstanceCreate{
		ResourceID:    generateRandomString("instance", 10),
		EnvironmentID: prodEnvironment.ID,
		Name:          baseName + "_Instance",
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

	mysqlDB := initPITRDB(t, databaseName, mysqlPort)

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

	return project, mysqlDB, database, backup, mysqlPort, func() {
		a.NoError(ctl.Close(ctx))
		stopInstance()
		a.NoError(mysqlDB.Close())
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
