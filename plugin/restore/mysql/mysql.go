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
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common/log"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/plugin/db/mysql"
	"github.com/bytebase/bytebase/resources/mysqlutil"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/blang/semver/v4"
)

const (
	// MaxDatabaseNameLength is the allowed max database name length in MySQL
	MaxDatabaseNameLength = 64
)

// BinlogFile is the metadata of the MySQL binlog file
type BinlogFile struct {
	Name string
	Size int64

	// Seq is parsed from Name and is for the sorting purpose.
	Seq int64
}

func newBinlogFile(name string, size int64) (BinlogFile, error) {
	seq, err := getBinlogNameSeq(name)
	if err != nil {
		return BinlogFile{}, err
	}
	return BinlogFile{Name: name, Size: size, Seq: seq}, nil
}

// ZapBinlogFiles is a helper to format zap.Array
type ZapBinlogFiles []BinlogFile

// MarshalLogArray implements the zapcore.ArrayMarshaler interface
func (files ZapBinlogFiles) MarshalLogArray(arr zapcore.ArrayEncoder) error {
	for _, file := range files {
		arr.AppendString(fmt.Sprintf("%s[%d]", file.Name, file.Size))
	}
	return nil
}

// Restore implements recovery functions for MySQL.
// For example, the original database is `dbfoo`. The suffixTs, derived from the PITR issue's CreateTs, is 1653018005.
// Bytebase will do the following:
// 1. Create a database called `dbfoo_pitr_1653018005`, and do PITR restore to it.
// 2. Create a database called `dbfoo_pitr_1653018005_old`, and move tables
// 	  from `dbfoo` to `dbfoo_pitr_1653018005_old`, and tables from `dbfoo_pitr_1653018005` to `dbfoo`.
type Restore struct {
	driver    *mysql.Driver
	mysqlutil *mysqlutil.Instance
	connCfg   db.ConnectionConfig
	binlogDir string
}

// New creates a new instance of Restore
func New(driver *mysql.Driver, instance *mysqlutil.Instance, connCfg db.ConnectionConfig, binlogDir string) *Restore {
	return &Restore{
		driver:    driver,
		mysqlutil: instance,
		connCfg:   connCfg,
		binlogDir: binlogDir,
	}
}

// ReplayBinlog replays the binlog for `originDatabase` from `startBinlogInfo.Position` to `targetTs`.
func (r *Restore) replayBinlog(ctx context.Context, originalDatabase, pitrDatabase string, startBinlogInfo api.BinlogInfo, targetTs int64) error {
	replayBinlogPaths, err := getBinlogReplayList(startBinlogInfo, r.binlogDir)
	if err != nil {
		return err
	}

	// Extract the SQL statements from the binlog and replay them to the pitrDatabase via the mysql client by pipe.
	mysqlbinlogArgs := []string{
		// Disable binary logging.
		"--disable-log-bin",
		// Create rewrite rules for databases when playing back from logs written in row-based format, so that we can apply the binlog to PITR database instead of the original database.
		"--rewrite-db", fmt.Sprintf("%s->%s", originalDatabase, pitrDatabase),
		// List entries for just this database. It's applied after the --rewrite-db option, so we should provide the rewritten database, i.e., pitrDatabase.
		"--database", pitrDatabase,
		// Start decoding the binary log at the log position, this option applies to the first log file named on the command line.
		"--start-position", fmt.Sprintf("%d", startBinlogInfo.Position),
		// Stop reading the binary log at the first event having a timestamp equal to or later than the datetime argument.
		"--stop-datetime", formatDateTime(targetTs),
	}

	mysqlbinlogArgs = append(mysqlbinlogArgs, replayBinlogPaths...)

	mysqlArgs := []string{
		"--host", r.connCfg.Host,
		"--user", r.connCfg.Username,
	}
	if r.connCfg.Port != "" {
		mysqlArgs = append(mysqlArgs, "--port", r.connCfg.Port)
	}
	if r.connCfg.Password != "" {
		// The --password parameter of mysql/mysqlbinlog does not support the "--password PASSWORD" format (split by space).
		// If provided like that, the program will hang.
		mysqlArgs = append(mysqlArgs, fmt.Sprintf("--password=%s", r.connCfg.Password))
	}

	mysqlbinlogCmd := exec.CommandContext(ctx, r.mysqlutil.GetPath(mysqlutil.MySQLBinlog), mysqlbinlogArgs...)
	mysqlCmd := exec.CommandContext(ctx, r.mysqlutil.GetPath(mysqlutil.MySQL), mysqlArgs...)
	log.Debug("Start replay binlog commands",
		zap.String("mysqlbinlog", mysqlbinlogCmd.String()),
		zap.String("mysql", mysqlCmd.String()))

	mysqlRead, err := mysqlbinlogCmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("cannot get mysqlbinlog stdout pipe, error: %w", err)
	}
	defer mysqlRead.Close()

	mysqlbinlogCmd.Stderr = os.Stderr

	mysqlCmd.Stderr = os.Stderr
	mysqlCmd.Stdout = os.Stderr
	mysqlCmd.Stdin = mysqlRead

	if err := mysqlbinlogCmd.Start(); err != nil {
		return fmt.Errorf("cannot start mysqlbinlog command, error [%w]", err)
	}
	if err := mysqlCmd.Run(); err != nil {
		return fmt.Errorf("mysql command fails, error [%w]", err)
	}
	if err := mysqlbinlogCmd.Wait(); err != nil {
		return fmt.Errorf("error occurred while waiting for mysqlbinlog to exit: %w", err)
	}
	return nil
}

// RestorePITR is a wrapper to perform PITR. It restores a full backup followed by replaying the binlog.
// It performs the step 1 of the restore process.
// TODO(dragonly): Refactor so that the first part is in driver.Restore, and remove this wrapper.
func (r *Restore) RestorePITR(ctx context.Context, fullBackup *bufio.Scanner, startBinlogInfo api.BinlogInfo, database string, suffixTs, targetTs int64) error {
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

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, query); err != nil {
		return err
	}

	if err := r.driver.RestoreTx(ctx, tx, fullBackup); err != nil {
		return err
	}

	// The full backup is restored successfully, enable foreign key constraints as normal.
	if _, err := tx.ExecContext(ctx, "SET foreign_key_checks=ON"); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	if err := r.replayBinlog(ctx, database, pitrDatabaseName, startBinlogInfo, targetTs); err != nil {
		return fmt.Errorf("failed to replay binlog, error[%w]", err)
	}

	return nil
}

// getBinlogReplayList returns the path list of the binlog that need be replayed.
func getBinlogReplayList(startBinlogInfo api.BinlogInfo, binlogDir string) ([]string, error) {
	startBinlogSeq, err := getBinlogNameSeq(startBinlogInfo.FileName)
	if err != nil {
		return nil, fmt.Errorf("cannot parse the start binlog file name[%s], error[%w]", startBinlogInfo.FileName, err)
	}

	binlogFiles, err := ioutil.ReadDir(binlogDir)
	if err != nil {
		return nil, fmt.Errorf("cannot read binlog directory %s, error %w", binlogDir, err)
	}

	var binlogFilesToReplay []BinlogFile
	for _, f := range binlogFiles {
		if f.IsDir() {
			continue
		}
		binlogFile, err := newBinlogFile(f.Name(), f.Size())
		if err != nil {
			return nil, err
		}
		if binlogFile.Seq >= startBinlogSeq {
			binlogFilesToReplay = append(binlogFilesToReplay, binlogFile)
		}
	}
	if len(binlogFilesToReplay) == 0 {
		log.Error("No binlog files found locally after given start binlog info", zap.Any("startBinlogInfo", startBinlogInfo))
		return nil, fmt.Errorf("no binlog files found locally after given start binlog info: %v", startBinlogInfo)
	}

	binlogFilesToReplaySorted := sortBinlogFiles(binlogFilesToReplay)

	if binlogFilesToReplaySorted[0].Seq != startBinlogSeq {
		log.Error("The starting binlog file does not exist locally", zap.String("filename", startBinlogInfo.FileName))
		return nil, fmt.Errorf("the starting binlog file[%s] does not exist locally", startBinlogInfo.FileName)
	}

	if !binlogFilesAreContinuous(binlogFilesToReplaySorted) {
		return nil, fmt.Errorf("discontinuous binlog file extensions detected, skip ")
	}

	var binlogReplayList []string
	for _, binlogFile := range binlogFilesToReplaySorted {
		binlogReplayList = append(binlogReplayList, filepath.Join(binlogDir, binlogFile.Name))
	}

	return binlogReplayList, nil
}

// sortBinlogFiles will sort binlog files in ascending order by their numeric extension.
// For mysql binlog, after the serial number reaches 999999, the next serial number will not return to 000000, but 1000000,
// so we cannot directly use string to compare lexicographical order.
func sortBinlogFiles(binlogFiles []BinlogFile) []BinlogFile {
	var sorted []BinlogFile
	sorted = append(sorted, binlogFiles...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Seq < sorted[j].Seq
	})
	return sorted
}

// Locate the binlog event at (filename, position), parse the event and return its timestamp.
// The current mechanism is by invoking mysqlbinlog and parse the output string.
// Maybe we should parse the raw binlog header to get better documented structure?
func (r *Restore) parseLocalBinlogEventTimestamp(ctx context.Context, binlogInfo api.BinlogInfo) (int64, error) {
	args := []string{
		path.Join(r.binlogDir, binlogInfo.FileName),
		"--start-position", fmt.Sprintf("%d", binlogInfo.Position),
		// This will trick mysqlbinlog to output the binlog event header followed by a warning message telling that
		// the --stop-position is in the middle of the binlog event.
		// It's OK, since we are only parsing for the timestamp in the binlog event header.
		"--stop-position", fmt.Sprintf("%d", binlogInfo.Position+1),
	}
	var buf bytes.Buffer
	cmd := exec.CommandContext(ctx, r.mysqlutil.GetPath(mysqlutil.MySQLBinlog), args...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = &buf

	if err := cmd.Run(); err != nil {
		log.Error("mysqlbinlog command fails", zap.String("cmd", cmd.String()), zap.Error(err))
		return 0, fmt.Errorf("mysqlbinlog command[%s] fails, error[%w]", cmd.String(), err)
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
			if strings.Contains(line, "end_log_pos 0") {
				// https://github.com/mysql/mysql-server/blob/8.0/client/mysqlbinlog.cc#L1209-L1212
				// Fake events with end_log_pos=0 could be generated and we need to ignore them.
				continue
			}
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

// GetLatestBackupBeforeOrEqualTs finds the latest logical backup and corresponding binlog info whose time is before or equal to `targetTs`.
// The backupList should only contain DONE backups.
func (r *Restore) GetLatestBackupBeforeOrEqualTs(ctx context.Context, backupList []*api.Backup, targetTs int64) (*api.Backup, error) {
	if len(backupList) == 0 {
		return nil, fmt.Errorf("no valid backup")
	}

	var eventTsList []int64
	var validBackupList []*api.Backup
	for _, b := range backupList {
		if b.Payload.BinlogInfo.IsEmpty() {
			log.Debug("Skip parsing binlog event timestamp of the backup where BinlogInfo is empty", zap.Int("backupId", b.ID), zap.String("backupName", b.Name))
			continue
		}
		validBackupList = append(validBackupList, b)
		eventTs, err := r.parseLocalBinlogEventTimestamp(ctx, b.Payload.BinlogInfo)
		if err != nil {
			return nil, fmt.Errorf("failed to parse binlog event timestamp, error: %w", err)
		}
		eventTsList = append(eventTsList, eventTs)
	}
	log.Debug("Binlog event ts list of backups", zap.Int64s("eventTsList", eventTsList))

	backup, err := getLatestBackupBeforeOrEqualTsImpl(validBackupList, eventTsList, targetTs)
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
	for i, b := range backupList {
		// Parse the binlog files and convert binlog positions into MySQL server timestamps.
		if b.Payload.BinlogInfo.IsEmpty() {
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
		targetDateTime := time.Unix(targetTs, 0).Format(time.RFC822)
		minEventDateTime := time.Unix(minEventTs, 0).Format(time.RFC822)
		log.Debug("the target restore time is earlier than the oldest backup time",
			zap.String("targetDatetime", targetDateTime),
			zap.Int64("targetTimestamp", targetTs),
			zap.String("minEventDateTime", minEventDateTime),
			zap.Int64("minEventTimestamp", minEventTs))

		return nil, fmt.Errorf("the target restore time %s is earlier than the oldest backup time %s", targetDateTime, minEventDateTime)
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
	log.Debug("Check database exists", zap.String("database", database))
	dbExists, err := r.databaseExists(ctx, database)
	if err != nil {
		return pitrDatabaseName, pitrOldDatabase, fmt.Errorf("failed to check whether database %q exists, error[%w]", database, err)
	}

	// We use a connection to ensure that the following database write operations are in the same MySQL session.
	conn, err := db.Conn(ctx)
	if err != nil {
		return pitrDatabaseName, pitrDatabaseName, err
	}
	defer conn.Close()

	// Set OFF the session variable sql_log_bin so that the writes in the following SQL statements will not be recorded in the binlog.
	if _, err := conn.ExecContext(ctx, "SET sql_log_bin=OFF"); err != nil {
		return pitrDatabaseName, pitrOldDatabase, err
	}

	if !dbExists {
		log.Debug("Database does not exist, creating...", zap.String("database", database))
		if _, err := conn.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE `%s`", database)); err != nil {
			return pitrDatabaseName, pitrOldDatabase, fmt.Errorf("failed to create non-exist database %q, error[%w]", database, err)
		}
	}

	tables, err := mysql.GetTables(ctx, db, database)
	if err != nil {
		return pitrDatabaseName, pitrOldDatabase, fmt.Errorf("failed to get tables of database %q, error[%w]", database, err)
	}
	tablesPITR, err := mysql.GetTables(ctx, db, pitrDatabaseName)
	if err != nil {
		return pitrDatabaseName, pitrOldDatabase, fmt.Errorf("failed to get tables of database %q, error[%w]", pitrDatabaseName, err)
	}

	if len(tables) == 0 && len(tablesPITR) == 0 {
		log.Warn("Both databases are empty, skip renaming tables",
			zap.String("originalDatabase", database),
			zap.String("pitrDatabase", pitrDatabaseName))
		return pitrDatabaseName, pitrOldDatabase, nil
	}

	if _, err := conn.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE `%s`", pitrOldDatabase)); err != nil {
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
	log.Debug("generated RENAME TABLE statement", zap.String("stmt", renameStmt))

	if _, err := conn.ExecContext(ctx, renameStmt); err != nil {
		return pitrDatabaseName, pitrOldDatabase, err
	}

	if _, err := conn.ExecContext(ctx, "SET sql_log_bin=ON"); err != nil {
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

// GetSortedLocalBinlogFiles returns a sorted BinlogFile list in the given binlog dir.
func GetSortedLocalBinlogFiles(binlogDir string) ([]BinlogFile, error) {
	binlogFilesInfoLocal, err := ioutil.ReadDir(binlogDir)
	if err != nil {
		return nil, err
	}
	var binlogFilesLocal []BinlogFile
	for _, fileInfo := range binlogFilesInfoLocal {
		binlogFile, err := newBinlogFile(fileInfo.Name(), fileInfo.Size())
		if err != nil {
			return nil, err
		}
		binlogFilesLocal = append(binlogFilesLocal, binlogFile)
	}
	return sortBinlogFiles(binlogFilesLocal), nil
}

func binlogFilesAreContinuous(files []BinlogFile) bool {
	for i := 0; i < len(files)-1; i++ {
		if files[i].Seq+1 != files[i+1].Seq {
			return false
		}
	}
	return true
}

// Download binlog files on server.
func (r *Restore) downloadBinlogFilesOnServer(ctx context.Context, binlogFilesLocal, binlogFilesOnServerSorted []BinlogFile) error {
	if len(binlogFilesOnServerSorted) == 0 {
		log.Debug("No binlog file found on server to download")
		return nil
	}
	latestBinlogFileOnServer := binlogFilesOnServerSorted[len(binlogFilesOnServerSorted)-1]
	binlogFilesLocalMap := make(map[string]BinlogFile)
	for _, file := range binlogFilesLocal {
		binlogFilesLocalMap[file.Name] = file
	}
	log.Debug("Downloading binlog files", zap.Array("fileList", ZapBinlogFiles(binlogFilesOnServerSorted)))
	for _, fileOnServer := range binlogFilesOnServerSorted {
		fileLocal, existLocal := binlogFilesLocalMap[fileOnServer.Name]
		path := filepath.Join(r.binlogDir, fileOnServer.Name)
		if !existLocal {
			if err := r.downloadBinlogFile(ctx, fileOnServer, fileOnServer.Name == latestBinlogFileOnServer.Name); err != nil {
				log.Error("Failed to download binlog file", zap.String("path", path), zap.Error(err))
				return fmt.Errorf("failed to download binlog file[%s], error[%w]", path, err)
			}
		} else if fileLocal.Size != fileOnServer.Size {
			log.Debug("Deleting inconsistent local binlog file",
				zap.String("path", path),
				zap.Int64("sizeLocal", fileOnServer.Size),
				zap.Int64("sizeOnServer", fileOnServer.Size))
			if err := os.Remove(path); err != nil {
				log.Error("Failed to remove inconsistent local binlog file", zap.String("path", path), zap.Error(err))
				return fmt.Errorf("failed to remove inconsistent local binlog file[%s], error[%w]", path, err)
			}
			if err := r.downloadBinlogFile(ctx, fileOnServer, fileOnServer.Name == latestBinlogFileOnServer.Name); err != nil {
				log.Error("Failed to re-download inconsistent local binlog file", zap.String("path", path), zap.Error(err))
				return fmt.Errorf("failed to re-download inconsistent local binlog file[%s], error[%w]", path, err)
			}
		}
	}
	return nil
}

// FetchAllBinlogFiles downloads all binlog files on server to `binlogDir`.
func (r *Restore) FetchAllBinlogFiles(ctx context.Context) error {
	// Read binlog files list on server.
	binlogFilesOnServerSorted, err := r.GetSortedBinlogFilesMetaOnServer(ctx)
	if err != nil {
		return err
	}
	if len(binlogFilesOnServerSorted) == 0 {
		log.Debug("No binlog file found on server to download")
		return nil
	}
	log.Debug("Got sorted binlog file list on server", zap.Array("list", ZapBinlogFiles(binlogFilesOnServerSorted)))

	// Read the local binlog files.
	binlogFilesLocalSorted, err := GetSortedLocalBinlogFiles(r.binlogDir)
	if err != nil {
		return fmt.Errorf("failed to read local binlog files, error[%w]", err)
	}

	return r.downloadBinlogFilesOnServer(ctx, binlogFilesLocalSorted, binlogFilesOnServerSorted)
}

// Syncs the binlog specified by `meta` between the instance and local.
// If isLast is true, it means that this is the last binlog file containing the targetTs event.
// It may keep growing as there are ongoing writes to the database. So we just need to check that
// the file size is larger or equal to the binlog file size we queried from the MySQL server earlier.
func (r *Restore) downloadBinlogFile(ctx context.Context, binlogFileToDownload BinlogFile, isLast bool) error {
	// for mysqlbinlog binary, --result-file must end with '/'
	resultFileDir := strings.TrimRight(r.binlogDir, "/") + "/"
	// TODO(zp): support ssl?
	args := []string{
		binlogFileToDownload.Name,
		"--read-from-remote-server",
		"--raw",
		"--host", r.connCfg.Host,
		"--user", r.connCfg.Username,
		"--result-file", resultFileDir,
	}
	if r.connCfg.Port != "" {
		args = append(args, "--port", r.connCfg.Port)
	}
	if r.connCfg.Password != "" {
		args = append(args, fmt.Sprintf("--password=%s", r.connCfg.Password))
	}

	cmd := exec.CommandContext(ctx, r.mysqlutil.GetPath(mysqlutil.MySQLBinlog), args...)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr

	log.Debug("Downloading binlog files using mysqlbinlog", zap.String("cmd", cmd.String()))
	resultFilePath := filepath.Join(resultFileDir, binlogFileToDownload.Name)
	if err := cmd.Run(); err != nil {
		_ = os.Remove(resultFilePath)
		return fmt.Errorf("executing mysqlbinlog fails, error: %w", err)
	}

	log.Debug("Checking downloaded binlog file stat", zap.String("path", resultFilePath))
	fileInfo, err := os.Stat(resultFilePath)
	if err != nil {
		_ = os.Remove(resultFilePath)
		return fmt.Errorf("cannot get file[%s] stat, error[%w]", resultFilePath, err)
	}
	if isLast {
		// Case 1: It's the last binlog file we need (contains the targetTs).
		// If it's the last (incomplete) binlog file on the MySQL server, it will grow as new writes hit the database server.
		// We just need to check that the downloaded file size >= queried size, so it contains the targetTs event.
		if fileInfo.Size() < binlogFileToDownload.Size {
			log.Error("Downloaded latest binlog file size is smaller than size queried on the MySQL server",
				zap.String("binlog", binlogFileToDownload.Name),
				zap.Int64("sizeInfo", binlogFileToDownload.Size),
				zap.Int64("downloadedSize", fileInfo.Size()),
			)
			_ = os.Remove(resultFilePath)
			return fmt.Errorf("downloaded latest binlog file[%s] size[%d] is smaller than size[%d] queried on MySQL server earlier", resultFilePath, fileInfo.Size(), binlogFileToDownload.Size)
		}
	} else {
		// Case 2: It's an archived binlog file, and we must ensure the file size equals what we queried from the MySQL server earlier.
		if fileInfo.Size() != binlogFileToDownload.Size {
			log.Error("Downloaded binlog file size is not equal to size queried on the MySQL server",
				zap.String("binlog", binlogFileToDownload.Name),
				zap.Int64("sizeInfo", binlogFileToDownload.Size),
				zap.Int64("downloadedSize", fileInfo.Size()),
			)
			_ = os.Remove(resultFilePath)
			return fmt.Errorf("downloaded binlog file[%s] size[%d] is not equal to size[%d] queried on MySQL server earlier", resultFilePath, fileInfo.Size(), binlogFileToDownload.Size)
		}
	}

	return nil
}

// GetSortedBinlogFilesMetaOnServer returns the metadata of binlog files in ascending order by their numeric extension.
func (r *Restore) GetSortedBinlogFilesMetaOnServer(ctx context.Context) ([]BinlogFile, error) {
	db, err := r.driver.GetDbConnection(ctx, "")
	if err != nil {
		return nil, err
	}

	rows, err := db.QueryContext(ctx, `SHOW BINARY LOGS;`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var binlogFiles []BinlogFile
	for rows.Next() {
		var name string
		var size int64
		var unused interface{}
		if err := rows.Scan(&name, &size, &unused /*Encrypted column*/); err != nil {
			return nil, err
		}
		binlogFile, err := newBinlogFile(name, size)
		if err != nil {
			return nil, err
		}
		binlogFiles = append(binlogFiles, binlogFile)
	}

	return sortBinlogFiles(binlogFiles), nil
}

// getBinlogNameSeq returns the numeric extension to the binary log base name by using split the dot.
// For example: ("binlog.000001") => 1, ("binlog000001") => err
func getBinlogNameSeq(name string) (int64, error) {
	s := strings.Split(name, ".")
	if len(s) != 2 {
		return 0, fmt.Errorf("failed to parse binlog extension, expecting two parts in the binlog file name[%s] but get %d", name, len(s))
	}
	return strconv.ParseInt(s[1], 10, 0)
}

func getSafeName(baseName, suffix string) string {
	name := fmt.Sprintf("%s_%s", baseName, suffix)
	if len(name) <= MaxDatabaseNameLength {
		return name
	}
	extraCharacters := len(name) - MaxDatabaseNameLength
	return fmt.Sprintf("%s_%s", baseName[0:len(baseName)-extraCharacters], suffix)
}

// checks the MySQL version is >=8.0
func checkVersionForPITR(version string) error {
	v, err := semver.Parse(version)
	if err != nil {
		return err
	}
	v8 := semver.MustParse("8.0.0")
	if v.LT(v8) {
		return fmt.Errorf("version %s is not supported for PITR; the minimum supported version is 8.0", version)
	}
	return nil
}

// CheckServerVersionForPITR checks that the MySQL server version meets the requirements of PITR.
func (r *Restore) CheckServerVersionForPITR(ctx context.Context) error {
	value, err := r.getServerVariable(ctx, "version")
	if err != nil {
		return err
	}
	if err := checkVersionForPITR(value); err != nil {
		return err
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

func (r *Restore) getServerVariable(ctx context.Context, varName string) (string, error) {
	db, err := r.driver.GetDbConnection(ctx, "")
	if err != nil {
		return "", err
	}

	rows, err := db.QueryContext(ctx, fmt.Sprintf("SHOW VARIABLES LIKE '%s';", varName))
	if err != nil {
		return "", err
	}
	if ok := rows.Next(); !ok {
		return "", fmt.Errorf("SHOW VARIABLES LIKE '%s' returns empty set", varName)
	}
	var varNameFound, value string
	if err := rows.Scan(&varNameFound, &value); err != nil {
		return "", err
	}
	if varName != varNameFound {
		return "", fmt.Errorf("expecting variable %s, but got %s", varName, varNameFound)
	}
	return value, nil
}

// CheckBinlogEnabled checks whether binlog is enabled for the current instance.
func (r *Restore) CheckBinlogEnabled(ctx context.Context) error {
	value, err := r.getServerVariable(ctx, "log_bin")
	if err != nil {
		return err
	}
	if strings.ToUpper(value) != "ON" {
		return fmt.Errorf("binlog is not enabled")
	}
	return nil
}

// CheckBinlogRowFormat checks whether the binlog format is ROW.
func (r *Restore) CheckBinlogRowFormat(ctx context.Context) error {
	value, err := r.getServerVariable(ctx, "binlog_format")
	if err != nil {
		return err
	}
	if strings.ToUpper(value) != "ROW" {
		return fmt.Errorf("binlog format is not ROW but %s", value)
	}
	return nil
}

// formatDateTime formats the timestamp to the local time string.
func formatDateTime(ts int64) string {
	t := time.Unix(ts, 0)
	return fmt.Sprintf("%d-%d-%d %d:%d:%d", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second())
}
