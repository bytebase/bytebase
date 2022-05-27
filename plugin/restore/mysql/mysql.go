package mysql

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/blang/semver/v4"
	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/plugin/db/mysql"
	"github.com/bytebase/bytebase/resources/mysqlbinlog"
	"go.uber.org/zap"
)

const (
	// MaxDatabaseNameLength is the allowed max database name length in MySQL
	MaxDatabaseNameLength = 64
)

// Restore implements recovery functions for MySQL.
// For example, the original database is `dbfoo`. The suffixTs, derived from the PITR issue's CreateTs, is 1653018005.
// Bytebase will do the following:
// 1. Create a database called `dbfoo_pitr_1653018005`, and do PITR restore to it.
// 2. Create a database called `dbfoo_pitr_1653018005_old`, and move tables
// 	  from `dbfoo` to `dbfoo_pitr_1653018005_old`, and tables from `dbfoo_pitr_1653018005` to `dbfoo`.
type Restore struct {
	l           *zap.Logger
	driver      *mysql.Driver
	mysqlbinlog *mysqlbinlog.Instance
}

// New creates a new instance of Restore
func New(l *zap.Logger, driver *mysql.Driver, instance *mysqlbinlog.Instance) *Restore {
	return &Restore{
		l:           l,
		driver:      driver,
		mysqlbinlog: instance,
	}
}

// replayBinlog restores the database using incremental backup in time range of [config.Start, config.End).
func replayBinlog(ctx context.Context, config api.BinlogInfo) error {
	return fmt.Errorf("Unimplemented")
}

// RestorePITR is a wrapper to perform PITR. It restores a full backup followed by replaying the binlog.
// It performs the step 1 of the restore process.
// TODO(dragonly): Refactor so that the first part is in driver.Restore, and remove this wrapper.
func (r *Restore) RestorePITR(ctx context.Context, fullBackup *bufio.Scanner, binlog api.BinlogInfo, database string, suffixTs int64) error {
	pitrDatabaseName := getPITRDatabaseName(database, suffixTs)
	query := fmt.Sprintf(""+
		// Create the pitr database.
		"CREATE DATABASE `%s`;"+
		// Change to the pitr database.
		"USE `%s`;"+
		// Set this to ignore foreign key constraints, otherwise the recovery of the full backup may encounter
		// wrong foreign key dependency order and fail.
		// We should turn it on after we the restore the full backup.
		"SET foreign_key_checks=OFF",
		pitrDatabaseName, pitrDatabaseName)

	db, err := r.driver.GetDbConnection(ctx, "")
	if err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, query); err != nil {
		return err
	}

	if err := r.driver.Restore(ctx, fullBackup); err != nil {
		return err
	}

	// The full backup is restored successfully, enable foreign key constraints as normal.
	if _, err := db.ExecContext(ctx, "SET foreign_key_checks=ON"); err != nil {
		return err
	}

	if err := replayBinlog(ctx, binlog); err != nil {
		// TODO(dragonly)/TODO(zp): Handle the error when implement replayBinlog.
		return nil
	}

	return nil
}

// Locate the binlog event at (filename, position), parse the event and return its timestamp.
// The current mechanism is by invoking mysqlbinlog and parse the output string.
// Maybe we should parse the raw binlog header to get better documented structure?
// nolint
func (r *Restore) parseBinlogEventTimestamp(ctx context.Context, binlogInfo api.BinlogInfo, binlogDir string) (int64, error) {
	args := []string{
		path.Join(binlogDir, binlogInfo.FileName),
		fmt.Sprintf("--start-position %d", binlogInfo.Position),
		// This will trick mysqlbinlog to output the binlog event header followed by a warning message telling that
		// the --stop-position is in the middle of the binlog event.
		// It's OK, since we are only parsing for the timestamp in the binlog event header.
		fmt.Sprintf("--stop-position %d", binlogInfo.Position+1),
	}
	var buf bytes.Buffer
	cmd := exec.CommandContext(ctx, r.mysqlbinlog.GetPath(), args...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = &buf

	if err := cmd.Run(); err != nil {
		return 0, err
	}

	timestamp, err := parseBinlogEventTimestampImpl(buf.String())
	if err != nil {
		return timestamp, fmt.Errorf("failed to parse binlog event timestamp, filename[%s], position[%d], error[%w]", binlogInfo.FileName, binlogInfo.Position, err)
	}

	return timestamp, nil
}

func parseBinlogEventTimestampImpl(output string) (int64, error) {
	lines := strings.Split(output, "\n")
	// The mysqlbinlog output will contains a line starting with "#220421 14:49:26 server id 1",
	// which has the timestamp we are looking for.
	// The first occurrence is the target.
	for _, line := range lines {
		if strings.Contains(line, "server id") {
			fields := strings.Fields(line)
			// fields should starts with ["#220421", "14:49:26", "server", "id"]
			if len(fields) < 4 ||
				(len(fields[0]) != 7 && len(fields[1]) != 8 && fields[2] != "server" && fields[3] != "id") {
				return 0, fmt.Errorf("invalid mysqlbinlog output line: %q", line)
			}
			date, err := time.ParseInLocation("060102 15:04:05", fmt.Sprintf("%s %s", fields[0][1:], fields[1]), time.Local)
			if err != nil {
				return 0, err
			}
			return date.Unix(), nil
		}
	}
	return 0, fmt.Errorf("no timestamp found in mysqlbinlog output")
}

// Find the latest logical backup and corresponding binlog info whose time is before or equal to `targetTs`.
// The backupList should only contain DONE backups.
// TODO(dragonly)/TODO(zp): Use this when the apply binlog PR is ready, and remove the nolint comments.
// nolint
func (r *Restore) getLatestBackupBeforeOrEqualTs(ctx context.Context, backupList []*api.Backup, targetTs int64, binlogDir string) (*api.Backup, error) {
	if len(backupList) == 0 {
		return nil, fmt.Errorf("no valid backup")
	}

	var eventTsList []int64
	for _, b := range backupList {
		eventTs, err := r.parseBinlogEventTimestamp(ctx, b.Payload.BinlogInfo, binlogDir)
		if err != nil {
			return nil, fmt.Errorf("failed to parse binlog event timestamp, error[%w]", err)
		}
		eventTsList = append(eventTsList, eventTs)
	}

	backup, err := getLatestBackupBeforeOrEqualTsImpl(backupList, eventTsList, targetTs)
	if err != nil {
		return nil, err
	}
	return backup, nil

}

// The backupList must 1 to 1 maps to the eventTsList, and the sorting order is not required.
func getLatestBackupBeforeOrEqualTsImpl(backupList []*api.Backup, eventTsList []int64, targetTs int64) (*api.Backup, error) {
	var maxEventTsLETargetTs int64
	var minEventTs int64 = math.MaxInt64
	var backup *api.Backup
	emptyBinlogInfo := api.BinlogInfo{}
	for i, b := range backupList {
		// Parse the binlog files and convert binlog positions into MySQL server timestamps.
		if b.Payload.BinlogInfo == emptyBinlogInfo {
			continue
		}
		eventTs := eventTsList[i]
		if eventTs <= targetTs && eventTs > maxEventTsLETargetTs {
			maxEventTsLETargetTs = eventTs
			backup = b
		}
		// This is only for composing the error message when no valid backup found.
		if eventTs < minEventTs {
			minEventTs = eventTs
		}
	}
	if maxEventTsLETargetTs == 0 {
		return nil, fmt.Errorf("the target restore timestamp[%d] is earlier than the oldest backup time[%d]", targetTs, minEventTs)
	}
	return backup, nil
}

// SwapPITRDatabase renames the pitr database to the target, and the original to the old database
// It returns the pitr and old database names after swap.
// It performs the step 2 of the restore process.
func (r *Restore) SwapPITRDatabase(ctx context.Context, database string, suffixTs int64) (string, string, error) {
	pitrDatabaseName := getPITRDatabaseName(database, suffixTs)
	pitrOldDatabase := getPITROldDatabaseName(database, suffixTs)

	db, err := r.driver.GetDbConnection(ctx, "")
	if err != nil {
		return pitrDatabaseName, pitrOldDatabase, err
	}

	// Handle the case that the original database does not exist, because user could drop a database and want to restore it.
	dbExists, err := r.databaseExists(ctx, database)
	if err != nil {
		return pitrDatabaseName, pitrOldDatabase, fmt.Errorf("failed to check whether database %q exists, error[%w]", database, err)
	}
	if !dbExists {
		if _, err := db.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE `%s`", database)); err != nil {
			return pitrDatabaseName, pitrOldDatabase, fmt.Errorf("failed to create non-exist database %q, error[%w]", database, err)
		}
	}

	// TODO(dragonly): Remove the transactions, because they do not have a clear semantic / are not necessary.
	txn, err := db.BeginTx(ctx, nil)
	if err != nil {
		return pitrDatabaseName, pitrOldDatabase, err
	}
	defer txn.Rollback()

	tables, err := mysql.GetTables(txn, database)
	if err != nil {
		return pitrDatabaseName, pitrOldDatabase, fmt.Errorf("failed to get tables of database %q, error[%w]", database, err)
	}
	tablesPITR, err := mysql.GetTables(txn, pitrDatabaseName)
	if err != nil {
		return pitrDatabaseName, pitrOldDatabase, fmt.Errorf("failed to get tables of database %q, error[%w]", pitrDatabaseName, err)
	}

	if _, err := txn.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE `%s`", pitrOldDatabase)); err != nil {
		return pitrDatabaseName, pitrOldDatabase, err
	}

	var tableRenames []string
	for _, table := range tables {
		tableRenames = append(tableRenames, fmt.Sprintf("`%s`.`%s` TO `%s`.`%s`", database, table.Name, pitrOldDatabase, table.Name))
	}
	for _, table := range tablesPITR {
		tableRenames = append(tableRenames, fmt.Sprintf("`%s`.`%s` TO `%s`.`%s`", pitrDatabaseName, table.Name, database, table.Name))
	}
	renameStmt := fmt.Sprintf("RENAME TABLE %s;", strings.Join(tableRenames, ", "))

	if _, err := txn.ExecContext(ctx, renameStmt); err != nil {
		return pitrDatabaseName, pitrOldDatabase, err
	}

	if err := txn.Commit(); err != nil {
		return pitrDatabaseName, pitrOldDatabase, err
	}

	return pitrDatabaseName, pitrOldDatabase, nil
}

func (r *Restore) databaseExists(ctx context.Context, database string) (bool, error) {
	db, err := r.driver.GetDbConnection(ctx, "")
	if err != nil {
		return false, err
	}
	stmt := fmt.Sprintf("SELECT SCHEMA_NAME FROM INFORMATION_SCHEMA.SCHEMATA WHERE SCHEMA_NAME='%s'", database)
	rows, err := db.QueryContext(ctx, stmt)
	if err != nil {
		return false, err
	}
	if exist := rows.Next(); exist {
		return true, nil
	}
	return false, nil
}

// Composes a pitr database name that we use as the target database for full backup recovery and binlog recovery.
// For example, getPITRDatabaseName("dbfoo", 1653018005) -> "dbfoo_pitr_1653018005"
func getPITRDatabaseName(database string, suffixTs int64) string {
	suffix := fmt.Sprintf("pitr_%d", suffixTs)
	return getSafeName(database, suffix)
}

// Composes a database name that we use as the target database for swapping out the original database.
// For example, getPITROldDatabaseName("dbfoo", 1653018005) -> "dbfoo_pitr_1653018005_old"
func getPITROldDatabaseName(database string, suffixTs int64) string {
	suffix := fmt.Sprintf("pitr_%d_old", suffixTs)
	return getSafeName(database, suffix)
}

// SyncArchivedBinlogFiles syncs the binlog files between the instance and `saveDir`,
// but exclude latest binlog. We will download the latest binlog only when doing PITR.
func (r *Restore) SyncArchivedBinlogFiles(ctx context.Context, instance *api.Instance, saveDir string) error {
	binlogFilesLocal, err := ioutil.ReadDir(saveDir)
	if err != nil {
		return err
	}

	binlogFilesOnServer, err := r.getBinlogFilesMetaOnServer(ctx)
	if err != nil {
		return err
	}

	latestBinlogFileOnServer, err := r.getLatestBinlogFileMeta(ctx)
	if err != nil {
		return err
	}

	// build a local file size map from file name to size
	localFileMap := make(map[string]int64)

	for _, localFile := range binlogFilesLocal {
		localFileMap[localFile.Name()] = localFile.Size()
	}

	todo := make(map[string]bool)

	for _, serverBinlog := range binlogFilesOnServer {
		// We don't download the latest binlog in SyncArchivedBinlogFiles()
		if serverBinlog.Name == latestBinlogFileOnServer.Name {
			continue
		}

		localBinlogSize, ok := localFileMap[serverBinlog.Name]
		if !ok {
			todo[serverBinlog.Name] = true
			continue
		}

		if localBinlogSize != serverBinlog.Size {
			// exist on local and file size not match, delete and then download it
			if err := os.Remove(filepath.Join(saveDir, serverBinlog.Name)); err != nil {
				return fmt.Errorf("cannot remove %s, error: %w", serverBinlog.Name, err)
			}
			todo[serverBinlog.Name] = true
		}
	}

	// download the binlog files not recorded in downloadedIndex
	for _, serverFile := range binlogFilesOnServer {
		if _, ok := todo[serverFile.Name]; !ok {
			continue
		}
		if err := r.downloadBinlogFile(ctx, instance, saveDir, serverFile); err != nil {
			return fmt.Errorf("cannot sync binlog %s, error: %w", serverFile.Name, err)
		}
	}

	return nil
}

// SyncLatestBinlog syncs the latest binlog between the instance and `saveDir`
func (r *Restore) SyncLatestBinlog(ctx context.Context, instance *api.Instance, saveDir string) error {
	latestBinlogFileOnServer, err := r.getLatestBinlogFileMeta(ctx)
	if err != nil {
		return err
	}
	return r.downloadBinlogFile(ctx, instance, saveDir, *latestBinlogFileOnServer)
}

// downloadBinlogFile syncs the binlog specified by `meta` between the instance and local.
func (r *Restore) downloadBinlogFile(ctx context.Context, instance *api.Instance, saveDir string, binlog mysql.BinlogFile) error {
	// for mysqlbinlog binary, --result-file must end with '/'
	resultFileDir := strings.TrimRight(saveDir, "/") + "/"
	// TODO(zp): support ssl?
	cmd := exec.CommandContext(ctx, r.mysqlbinlog.GetPath(),
		binlog.Name,
		fmt.Sprintf("--read-from-remote-server --host=%s --port=%s --user=%s --password=%s", instance.Host, instance.Port, instance.Username, instance.Password),
		"--raw",
		fmt.Sprintf("--result-file=%s", resultFileDir),
	)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr

	resultFilePath := filepath.Join(resultFileDir, binlog.Name)
	if err := cmd.Run(); err != nil {
		_ = os.Remove(resultFilePath)
		return fmt.Errorf("cannot run mysqlbinlog, error: %w", err)
	}

	fi, err := os.Stat(resultFilePath)
	if err != nil {
		_ = os.Remove(resultFilePath)
		return fmt.Errorf("cannot stat %s, error: %w", resultFilePath, err)
	}
	if fi.Size() != binlog.Size {
		_ = os.Remove(resultFilePath)
		return fmt.Errorf("download file %s size not match", resultFilePath)
	}

	return nil
}

// getBinlogFilesMetaOnServer returns the metadata of binlog files.
func (r *Restore) getBinlogFilesMetaOnServer(ctx context.Context) ([]mysql.BinlogFile, error) {
	db, err := r.driver.GetDbConnection(ctx, "")
	if err != nil {
		return nil, err
	}

	rows, err := db.QueryContext(ctx, `SHOW BINARY LOGS;`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var binlogFiles []mysql.BinlogFile
	for rows.Next() {
		var binlogFile mysql.BinlogFile
		var unused interface{}
		if err := rows.Scan(&binlogFile.Name, &binlogFile.Size, &unused /*Encrypted column*/); err != nil {
			return nil, err
		}
		binlogFiles = append(binlogFiles, binlogFile)
	}
	return binlogFiles, nil
}

// showLatestBinlogFile returns the metadata of latest binlog
func (r *Restore) getLatestBinlogFileMeta(ctx context.Context) (*mysql.BinlogFile, error) {
	// TODO(zp): refactor to reuse getBinlogInfo() in plugin/db/mysql.go
	db, err := r.driver.GetDbConnection(ctx, "")
	if err != nil {
		return nil, err
	}

	rows, err := db.QueryContext(ctx, `SHOW MASTER STATUS;`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var binlogFile mysql.BinlogFile
	if rows.Next() {
		var unused interface{} /*Binlog_Do_DB, Binlog_Ignore_DB, Executed_Gtid_Set*/
		if err := rows.Scan(&binlogFile.Name, &binlogFile.Size, &unused, &unused, &unused); err != nil {
			return nil, err
		}
		return &binlogFile, nil
	}
	return nil, fmt.Errorf("cannot find latest binlog on instance")
}

func getSafeName(baseName, suffix string) string {
	name := fmt.Sprintf("%s_%s", baseName, suffix)
	if len(name) <= MaxDatabaseNameLength {
		return name
	}
	extraCharacters := len(name) - MaxDatabaseNameLength
	return fmt.Sprintf("%s_%s", baseName[0:len(baseName)-extraCharacters], suffix)
}

// checks the MySQL version is >=5.7
func checkVersionForPITR(version string) error {
	v, err := semver.Parse(version)
	if err != nil {
		return err
	}
	v57 := semver.MustParse("5.7.0")
	if v.LT(v57) {
		return fmt.Errorf("version %s is not supported for PITR; the minimum supported version is 5.7", version)
	}
	return nil
}

// CheckEngineInnoDB checks that the tables in the database is all using InnoDB as the storage engine.
func (r *Restore) CheckEngineInnoDB(ctx context.Context, database string) error {
	db, err := r.driver.GetDbConnection(ctx, "")
	if err != nil {
		return err
	}

	// ref: https://dev.mysql.com/doc/refman/8.0/en/information-schema-tables-table.html
	stmt := fmt.Sprintf("SELECT table_name, engine FROM information_schema.tables WHERE table_schema='%s';", database)
	rows, err := db.QueryContext(ctx, stmt)
	if err != nil {
		return err
	}
	var tablesNotInnoDB []string
	for rows.Next() {
		var tableName, engine string
		if err := rows.Scan(&tableName, &engine); err != nil {
			return err
		}
		if strings.ToLower(engine) != "innodb" {
			tablesNotInnoDB = append(tablesNotInnoDB, tableName)
		}
	}
	if len(tablesNotInnoDB) != 0 {
		return fmt.Errorf("tables %v of database %s do not use the InnoDB engine, which is required for PITR", tablesNotInnoDB, database)
	}
	return nil
}
