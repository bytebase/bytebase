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
	resourcemysql "github.com/bytebase/bytebase/backend/resources/mysql"
	"github.com/bytebase/bytebase/backend/tests/fake"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"

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
	ctx, project, mysqlDB, instanceUID, _, databaseUID, databaseName, backup, _, cleanFn := setUpForPITRTest(ctx, t, ctl)
	defer cleanFn()

	metadata, err := ctl.getLatestSchemaMetadata(databaseUID)
	a.NoError(err)
	var latestSchemaMetadata storepb.DatabaseMetadata
	err = protojson.Unmarshal([]byte(metadata), &latestSchemaMetadata)
	a.NoError(err)

	backupUID, err := strconv.Atoi(backup.Uid)
	a.NoError(err)
	issue, err := createPITRIssue(ctl, project, api.PITRContext{
		DatabaseID: databaseUID,
		BackupID:   &backupUID,
		CreateDatabaseCtx: &api.CreateDatabaseContext{
			InstanceID:   instanceUID,
			DatabaseName: databaseName + "_new",
			CharacterSet: latestSchemaMetadata.CharacterSet,
			Collation:    latestSchemaMetadata.Collation,
			BackupID:     backupUID,
		},
	})
	a.NoError(err)

	// Restore stage.
	status, err := ctl.waitIssueNextTaskWithTaskApproval(ctx, issue.ID)
	a.NoError(err)
	a.Equal(api.TaskDone, status)

	// Cutover stage.
	status, err = ctl.waitIssueNextTaskWithTaskApproval(ctx, issue.ID)
	a.NoError(err)
	a.Equal(api.TaskDone, status)

	validateTbl0(t, mysqlDB, databaseName, numRowsTime0)
	validateTbl1(t, mysqlDB, databaseName, numRowsTime0)
	validateTableUpdateRow(t, mysqlDB, databaseName)
}

func TestRetentionPolicy(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, _, _, _, database, databaseUID, _, backup, _, cleanFn := setUpForPITRTest(ctx, t, ctl)
	defer cleanFn()

	metaDB, err := sql.Open("pgx", ctl.profile.PgURL)
	a.NoError(err)
	a.NoError(metaDB.Ping())

	// Check that the backup file exist
	backupResourceName := strings.TrimPrefix(backup.Name, fmt.Sprintf("%s/backups/", database.Name))
	backupFilePath := filepath.Join(ctl.profile.DataDir, "backup", "db", fmt.Sprintf("%d", databaseUID), fmt.Sprintf("%s.sql", backupResourceName))
	_, err = os.Stat(backupFilePath)
	a.NoError(err)

	// Change retention period to 1s, and the backup should be quickly removed.
	// TODO(d): clean-up the hack.
	_, err = metaDB.ExecContext(ctx, fmt.Sprintf("UPDATE backup_setting SET enabled=true, retention_period_ts=1 WHERE database_id=%d;", databaseUID))
	a.NoError(err)
	err = ctl.waitBackupArchived(ctx, database.Name, backup.Name)
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
	ctx, project, mysqlDB, _, _, databaseUID, databaseName, _, mysqlPort, cleanFn := setUpForPITRTest(ctx, t, ctl)
	defer cleanFn()

	insertRangeData(t, mysqlDB, numRowsTime0, numRowsTime1)

	ctxUpdateRow, cancelUpdateRow := context.WithCancel(ctx)
	targetTs := startUpdateRow(ctxUpdateRow, t, databaseName, mysqlPort) + 1

	dropColumnStmt := `ALTER TABLE tbl1 DROP COLUMN id;`
	log.Debug("mimics schema migration", zap.String("statement", dropColumnStmt))
	_, err := mysqlDB.ExecContext(ctx, dropColumnStmt)
	a.NoError(err)

	issue, err := createPITRIssue(ctl, project, api.PITRContext{
		DatabaseID:    databaseUID,
		PointInTimeTs: &targetTs,
	})
	a.NoError(err)

	// Restore stage.
	status, err := ctl.waitIssueNextTaskWithTaskApproval(ctx, issue.ID)
	a.NoError(err)
	a.Equal(api.TaskDone, status)

	cancelUpdateRow()
	// We mimics the situation where the user waits for the target database idle before doing the cutover.
	time.Sleep(time.Second)

	// Cutover stage.
	status, err = ctl.waitIssueNextTaskWithTaskApproval(ctx, issue.ID)
	a.NoError(err)
	a.Equal(api.TaskDone, status)

	validateTbl0(t, mysqlDB, databaseName, numRowsTime1)
	validateTbl1(t, mysqlDB, databaseName, numRowsTime1)
	validateTableUpdateRow(t, mysqlDB, databaseName)
}

func TestPITRDropDatabase(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, project, mysqlDB, _, _, databaseUID, databaseName, _, _, cleanFn := setUpForPITRTest(ctx, t, ctl)
	defer cleanFn()

	insertRangeData(t, mysqlDB, numRowsTime0, numRowsTime1)

	time.Sleep(1 * time.Second)
	targetTs := time.Now().Unix()

	dropStmt := fmt.Sprintf(`DROP DATABASE %s;`, databaseName)
	_, err := mysqlDB.ExecContext(ctx, dropStmt)
	a.NoError(err)

	dbRows, err := mysqlDB.Query(fmt.Sprintf(`SHOW DATABASES LIKE '%s';`, databaseName))
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
		DatabaseID:    databaseUID,
		PointInTimeTs: &targetTs,
	})
	a.NoError(err)

	// Restore stage.
	status, err := ctl.waitIssueNextTaskWithTaskApproval(ctx, issue.ID)
	a.NoError(err)
	a.Equal(api.TaskDone, status)

	// We mimics the situation where the user waits for the target database idle before doing the cutover.
	time.Sleep(time.Second)

	// Cutover stage.
	status, err = ctl.waitIssueNextTaskWithTaskApproval(ctx, issue.ID)
	a.NoError(err)
	a.Equal(api.TaskDone, status)

	validateTbl0(t, mysqlDB, databaseName, numRowsTime1)
	validateTbl1(t, mysqlDB, databaseName, numRowsTime1)
	validateTableUpdateRow(t, mysqlDB, databaseName)
}

func TestPITRTwice(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, project, mysqlDB, _, database, databaseUID, databaseName, _, mysqlPort, cleanFn := setUpForPITRTest(ctx, t, ctl)
	defer cleanFn()

	log.Debug("Creating issue for the first PITR.")
	insertRangeData(t, mysqlDB, numRowsTime0, numRowsTime1)
	ctxUpdateRow, cancelUpdateRow := context.WithCancel(ctx)
	targetTs := startUpdateRow(ctxUpdateRow, t, databaseName, mysqlPort) + 1

	issue, err := createPITRIssue(ctl, project, api.PITRContext{
		DatabaseID:    databaseUID,
		PointInTimeTs: &targetTs,
	})
	a.NoError(err)

	// Restore stage.
	status, err := ctl.waitIssueNextTaskWithTaskApproval(ctx, issue.ID)
	a.NoError(err)
	a.Equal(api.TaskDone, status)

	cancelUpdateRow()
	// We mimics the situation where the user waits for the target database idle before doing the cutover.
	time.Sleep(time.Second)

	// Cutover stage.
	status, err = ctl.waitIssueNextTaskWithTaskApproval(ctx, issue.ID)
	a.NoError(err)
	a.Equal(api.TaskDone, status)

	validateTbl0(t, mysqlDB, databaseName, numRowsTime1)
	validateTbl1(t, mysqlDB, databaseName, numRowsTime1)
	validateTableUpdateRow(t, mysqlDB, databaseName)
	log.Debug("First PITR done.")

	log.Debug("Wait for the first PITR auto backup to finish.")
	resp, err := ctl.databaseServiceClient.ListBackups(ctx, &v1pb.ListBackupsRequest{
		Parent: database.Name,
	})
	a.NoError(err)
	backups := resp.Backups
	a.Equal(2, len(backups))
	sort.Slice(backups, func(i int, j int) bool {
		return backups[i].CreateTime.AsTime().After(backups[j].CreateTime.AsTime())
	})
	err = ctl.waitBackup(ctx, database.Name, backups[0].Name)
	a.NoError(err)

	log.Debug("Creating issue for the second PITR.")
	ctxUpdateRow, cancelUpdateRow = context.WithCancel(ctx)
	targetTs = startUpdateRow(ctxUpdateRow, t, databaseName, mysqlPort) + 1
	insertRangeData(t, mysqlDB, numRowsTime1, numRowsTime2)

	issue2, err := createPITRIssue(ctl, project, api.PITRContext{
		DatabaseID:    databaseUID,
		PointInTimeTs: &targetTs,
	})
	a.NoError(err)

	// Restore stage.
	status, err = ctl.waitIssueNextTaskWithTaskApproval(ctx, issue2.ID)
	a.NoError(err)
	a.Equal(api.TaskDone, status)

	cancelUpdateRow()
	// We mimics the situation where the user waits for the target database idle before doing the cutover.
	time.Sleep(time.Second)

	// Cutover stage.
	status, err = ctl.waitIssueNextTaskWithTaskApproval(ctx, issue2.ID)
	a.NoError(err)
	a.Equal(api.TaskDone, status)

	// Second PITR
	validateTbl0(t, mysqlDB, databaseName, numRowsTime1)
	validateTbl1(t, mysqlDB, databaseName, numRowsTime1)
	validateTableUpdateRow(t, mysqlDB, databaseName)
	log.Debug("Second PITR done.")
}

func TestPITRToNewDatabaseInAnotherInstance(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, project, sourceMySQLDB, _, _, databaseUID, databaseName, _, mysqlPort, cleanFn := setUpForPITRTest(ctx, t, ctl)
	defer cleanFn()
	projectUID, err := strconv.Atoi(project.Uid)
	a.NoError(err)

	dstPort := getTestPort()
	dstStopFn := resourcemysql.SetupTestInstance(t, dstPort, mysqlBinDir)
	defer dstStopFn()
	dstConnCfg := getMySQLConnectionConfig(strconv.Itoa(dstPort), "")

	prodEnvironment, _, err := ctl.getEnvironment(ctx, "prod")
	a.NoError(err)
	dstInstance, err := ctl.instanceServiceClient.CreateInstance(ctx, &v1pb.CreateInstanceRequest{
		InstanceId: "destinationinstance",
		Instance: &v1pb.Instance{
			Title:       "DestinationInstance",
			Engine:      v1pb.Engine_MYSQL,
			Environment: prodEnvironment.Name,
			DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: dstConnCfg.Host, Port: dstConnCfg.Port, Username: dstConnCfg.Username}},
		},
	})
	a.NoError(err)
	dstInstanceUID, err := strconv.Atoi(dstInstance.Uid)
	a.NoError(err)
	prodEnvironmentResourceID := strings.TrimPrefix(prodEnvironment.Name, "environments/")
	a.NoError(err)

	insertRangeData(t, sourceMySQLDB, numRowsTime0, numRowsTime1)

	ctxUpdateRow, cancelUpdateRow := context.WithCancel(ctx)
	cancelUpdateRow()

	targetTs := startUpdateRow(ctxUpdateRow, t, databaseName, mysqlPort) + 1

	dropColumnStmt := `ALTER TABLE tbl1 DROP COLUMN id;`
	log.Debug("mimics schema migration", zap.String("statement", dropColumnStmt))
	_, err = sourceMySQLDB.ExecContext(ctx, dropColumnStmt)
	a.NoError(err)

	labels, err := marshalLabels(nil, prodEnvironmentResourceID)
	a.NoError(err)

	targetDatabaseName := "new_database"
	pitrIssueCtx, err := json.Marshal(&api.PITRContext{
		DatabaseID:    databaseUID,
		PointInTimeTs: &targetTs,
		CreateDatabaseCtx: &api.CreateDatabaseContext{
			InstanceID:   dstInstanceUID,
			DatabaseName: targetDatabaseName,
			Labels:       labels,
			CharacterSet: "utf8mb4",
			Collation:    "utf8mb4_general_ci",
		},
	})
	a.NoError(err)

	issue, err := ctl.createIssue(api.IssueCreate{
		ProjectID:     projectUID,
		Name:          fmt.Sprintf("Restore database %s to the time %d", databaseName, targetTs),
		Type:          api.IssueDatabaseRestorePITR,
		AssigneeID:    api.SystemBotID,
		CreateContext: string(pitrIssueCtx),
	})
	a.NoError(err)

	// Create database task
	status, err := ctl.waitIssueNextTaskWithTaskApproval(ctx, issue.ID)
	a.NoError(err)
	a.Equal(api.TaskDone, status)

	// Restore task
	status, err = ctl.waitIssueNextTaskWithTaskApproval(ctx, issue.ID)
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
	ctx, project, _, _, _, databaseUID, _, _, _, cleanFn := setUpForPITRTest(ctx, t, ctl)
	defer cleanFn()

	issue, err := createPITRIssue(ctl, project, api.PITRContext{
		DatabaseID:    databaseUID,
		PointInTimeTs: &targetTs,
	})
	a.NoError(err)

	status, err := ctl.waitIssueNextTaskWithTaskApproval(ctx, issue.ID)
	a.Error(err)
	a.Equal(api.TaskFailed, status)
}

func createPITRIssue(ctl *controller, project *v1pb.Project, pitrContext api.PITRContext) (*api.Issue, error) {
	projectUID, err := strconv.Atoi(project.Uid)
	if err != nil {
		return nil, err
	}
	pitrIssueCtx, err := json.Marshal(&pitrContext)
	if err != nil {
		return nil, err
	}
	return ctl.createIssue(api.IssueCreate{
		ProjectID:     projectUID,
		Name:          fmt.Sprintf("Restore database %d", pitrContext.DatabaseID),
		Type:          api.IssueDatabaseRestorePITR,
		AssigneeID:    api.SystemBotID,
		CreateContext: string(pitrIssueCtx),
	})
}

func setUpForPITRTest(ctx context.Context, t *testing.T, ctl *controller) (context.Context, *v1pb.Project, *sql.DB, int, *v1pb.Database, int, string, *v1pb.Backup, int, func()) {
	a := require.New(t)

	dataDir := t.TempDir()
	ctx, err := ctl.StartServerWithExternalPg(ctx, &config{
		dataDir:            dataDir,
		vcsProviderCreator: fake.NewGitLab,
	})
	a.NoError(err)
	err = ctl.setLicense()
	a.NoError(err)

	project, err := ctl.createProject(ctx)
	a.NoError(err)
	projectUID, err := strconv.Atoi(project.Uid)
	a.NoError(err)

	prodEnvironment, _, err := ctl.getEnvironment(ctx, "prod")
	a.NoError(err)

	_, err = ctl.orgPolicyServiceClient.CreatePolicy(ctx, &v1pb.CreatePolicyRequest{
		Parent: prodEnvironment.Name,
		Policy: &v1pb.Policy{
			Type: v1pb.PolicyType_BACKUP_PLAN,
			Policy: &v1pb.Policy_BackupPlanPolicy{
				BackupPlanPolicy: &v1pb.BackupPlanPolicy{
					Schedule: v1pb.BackupPlanSchedule_UNSET,
				},
			},
		},
	})
	a.NoError(err)

	baseName := strings.ReplaceAll(t.Name(), "/", "_")
	databaseName := baseName + "_Database"

	mysqlPort := getTestPort()
	stopInstance := resourcemysql.SetupTestInstance(t, mysqlPort, mysqlBinDir)
	connCfg := getMySQLConnectionConfig(strconv.Itoa(mysqlPort), "")
	a.NoError(err)
	instance, err := ctl.instanceServiceClient.CreateInstance(ctx, &v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance", 10),
		Instance: &v1pb.Instance{
			Title:       baseName + "_Instance",
			Engine:      v1pb.Engine_MYSQL,
			Environment: prodEnvironment.Name,
			DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: connCfg.Host, Port: connCfg.Port, Username: connCfg.Username}},
		},
	})
	a.NoError(err)
	instanceUID, err := strconv.Atoi(instance.Uid)
	a.NoError(err)

	err = ctl.createDatabase(ctx, projectUID, instance, databaseName, "", nil)
	a.NoError(err)

	database, err := ctl.databaseServiceClient.GetDatabase(ctx, &v1pb.GetDatabaseRequest{
		Name: fmt.Sprintf("%s/databases/%s", instance.Name, databaseName),
	})
	a.NoError(err)
	databaseUID, err := strconv.Atoi(database.Uid)
	a.NoError(err)

	err = ctl.disableAutomaticBackup(databaseUID)
	a.NoError(err)

	mysqlDB := initPITRDB(t, databaseName, mysqlPort)

	insertRangeData(t, mysqlDB, 0, numRowsTime0)

	log.Debug("Create a full backup")
	backup, err := ctl.databaseServiceClient.CreateBackup(ctx, &v1pb.CreateBackupRequest{
		Parent: database.Name,
		Backup: &v1pb.Backup{
			Name:       fmt.Sprintf("%s/backups/first-backup", database.Name),
			BackupType: v1pb.Backup_MANUAL,
		},
	})
	a.NoError(err)
	err = ctl.waitBackup(ctx, database.Name, backup.Name)
	a.NoError(err)

	return ctx, project, mysqlDB, instanceUID, database, databaseUID, databaseName, backup, mysqlPort, func() {
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
