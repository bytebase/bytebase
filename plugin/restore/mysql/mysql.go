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
		"--stop-datetime", getDateTime(targetTs),
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

	binlogFilesToReplay, err = sortBinlogFiles(binlogFilesToReplay)
	if err != nil {
		return nil, err
	}

	for i := 1; i < len(binlogFilesToReplay); i++ {
		if binlogFilesToReplay[i].Seq != binlogFilesToReplay[i-1].Seq+1 {
			return nil, fmt.Errorf("discontinuous binlog file extensions detected in %s and %s", binlogFilesToReplay[i-1].Name, binlogFilesToReplay[i].Name)
		}
	}

	var binlogReplayList []string
	for _, binlogFile := range binlogFilesToReplay {
		binlogReplayList = append(binlogReplayList, filepath.Join(binlogDir, binlogFile.Name))
	}

	return binlogReplayList, nil
}

// sortBinlogFiles will parse the binlog file name, and then sort them in ascending order by their numeric extension.
// For mysql binlog, after the serial number reaches 999999, the next serial number will not return to 000000, but 1000000,
// so we cannot directly use string to compare lexicographical order.
func sortBinlogFiles(binlogFiles []BinlogFile) ([]BinlogFile, error) {
	var ret []BinlogFile
	for _, binlogFile := range binlogFiles {
		seq, err := getBinlogNameSeq(binlogFile.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to parse the binlog file name[%s], error[%w]", binlogFile.Name, err)
		}
		binlogFile.Seq = seq
		ret = append(ret, binlogFile)
	}
	sort.Slice(ret, func(i, j int) bool {
		return ret[i].Seq < ret[j].Seq
	})
	return ret, nil
}

// Locate the binlog event at (filename, position), parse the event and return its timestamp.
// The current mechanism is by invoking mysqlbinlog and parse the output string.
// Maybe we should parse the raw binlog header to get better documented structure?
// nolint
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
	for _, b := range backupList {
		eventTs, err := r.parseLocalBinlogEventTimestamp(ctx, b.Payload.BinlogInfo)
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

// Parse the first eventTs of a local binlog file.
func (r *Restore) parseFirstLocalBinlogEventTimestamp(ctx context.Context, fileName string) (int64, error) {
	// https://dev.mysql.com/doc/internals/en/binlog-file-header.html
	// > A binlog file starts with a Binlog File Header \0xfe'bin'
	// The starting point of the first binlog event is at position 4 of a binlog file.
	firstBinlogEvent := api.BinlogInfo{FileName: fileName, Position: 4}
	return r.parseLocalBinlogEventTimestamp(ctx, firstBinlogEvent)
}

// FetchBinlogFilesToTargetTs downloads the locally missing binlog files from the remote instance to `binlogDir`.
// After downloading a binlog file, we check it's first event timestamp. If the timestamp is larger than
// targetTs, we stop the following downloads and return eagerly, because now the local binlog files contain
// all the binlog events we need for this recovery task.
func (r *Restore) FetchBinlogFilesToTargetTs(ctx context.Context, targetTs int64) error {
	// Read the local binlog files.
	binlogFilesInfoLocal, err := ioutil.ReadDir(r.binlogDir)
	if err != nil {
		return err
	}
	var binlogFilesLocal []BinlogFile
	for _, fileInfo := range binlogFilesInfoLocal {
		binlogFile, err := newBinlogFile(fileInfo.Name(), fileInfo.Size())
		if err != nil {
			return err
		}
		binlogFilesLocal = append(binlogFilesLocal, binlogFile)
	}

	// Read binlog files list on server.
	// We compare it with the local binlog files list to generate a download list.
	// Also compare file sizes to validate local binlog files, and re-download invalid ones.
	binlogFilesOnServerSorted, err := r.GetSortedBinlogFilesMetaOnServer(ctx)
	if err != nil {
		return err
	}
	if len(binlogFilesOnServerSorted) == 0 {
		// No binlog files on server, so there's nothing to download.
		return nil
	}
	latestBinlogFileOnServer := binlogFilesOnServerSorted[len(binlogFilesOnServerSorted)-1]
	log.Debug("Got sorted binlog file list on server", zap.Array("list", ZapBinlogFiles(binlogFilesOnServerSorted)))

	binlogFilesLocalMap := make(map[string]BinlogFile)
	for _, file := range binlogFilesLocal {
		binlogFilesLocalMap[file.Name] = file
	}
	// Record invalid local binlog files to re-download.
	binlogFilesLocalInvalidMap := make(map[string]bool)

	// Check for:
	// 1. binlog files on server but not locally exist
	// 2. local binlog files with invalid file size
	var downloadFileList []BinlogFile
	for _, serverBinlog := range binlogFilesOnServerSorted {
		localBinlogFile, ok := binlogFilesLocalMap[serverBinlog.Name]
		if !ok {
			downloadFileList = append(downloadFileList, serverBinlog)
			continue
		}

		if localBinlogFile.Size != serverBinlog.Size {
			// Existing local binlog file size does not match size queried from MySQL server, delete and re-download.
			log.Warn("Deleting partial local binlog file",
				zap.String("fileName", serverBinlog.Name),
				zap.Int64("fileSize", localBinlogFile.Size),
				zap.Int64("serverFileSize", serverBinlog.Size))
			if err := os.Remove(filepath.Join(r.binlogDir, serverBinlog.Name)); err != nil {
				log.Error("Failed to remove the partial local binlog file")
				return fmt.Errorf("failed to remove the partial local binlog file[%s], error[%w]", serverBinlog.Name, err)
			}
			downloadFileList = append(downloadFileList, serverBinlog)
			binlogFilesLocalInvalidMap[serverBinlog.Name] = true
		}
	}

	var binlogFilesLocalValid []BinlogFile
	for _, file := range binlogFilesLocalMap {
		invalid := binlogFilesLocalInvalidMap[file.Name]
		if !invalid {
			binlogFilesLocalValid = append(binlogFilesLocalValid, file)
		}
	}
	binlogFilesLocalValidSorted, err := sortBinlogFiles(binlogFilesLocalValid)
	if err != nil {
		return err
	}

	// Find if some local binlog file's first binlog event ts is already larger than targetTs.
	// If so, we do not need to download the following binlog files on server.
	var foundLocalFileLTTargetTs bool
	var localFileExceedsTargetTs BinlogFile
	for _, file := range binlogFilesLocalValidSorted {
		eventTs, err := r.parseFirstLocalBinlogEventTimestamp(ctx, file.Name)
		path := filepath.Join(r.binlogDir, file.Name)
		if err != nil {
			log.Error("Failed to parse the first binlog event timestamp for local binlog file",
				zap.String("path", path),
				zap.Error(err))
			return fmt.Errorf("failed to parse the first binlog event timestamp for local binlog file[%s], error[%w]", path, err)
		}
		if eventTs > targetTs {
			log.Info("Found a local binlog file whose first event's timestamp is larger than targetTs",
				zap.Any("binlogFile", file),
				zap.Int64("targetTs", targetTs),
				zap.Int64("eventTs", eventTs))
			foundLocalFileLTTargetTs = true
			localFileExceedsTargetTs = file
			break
		}
	}

	// Now it's time to download binlog files up to targetTs.
	log.Debug("Download binlog file list", zap.Array("list", ZapBinlogFiles(downloadFileList)))
	for _, file := range downloadFileList {
		log.Debug("Downloading file", zap.Any("binlogFile", file))
		// The about-to-download binlog file has eventTs > targetTs, so we should stop downloading the following binlog files.
		if foundLocalFileLTTargetTs && localFileExceedsTargetTs.Seq <= file.Seq {
			log.Debug("Skip downloading file, because there's already a local binlog file whose first eventTs > targetTs", zap.String("binlogFile", file.Name))
			break
		}

		isLast := file == latestBinlogFileOnServer
		if err := r.downloadBinlogFile(ctx, file, isLast); err != nil {
			return fmt.Errorf("failed to download binlog file[%s], error[%w]", file.Name, err)
		}
		// We parsed the first binlog event's timestamp of the newly downloaded binlog file.
		// If the timestamp > targetTs, we skip downloading the following binlog files, because we have already
		// downloaded all the binlog files needed before targetTs.
		eventTs, err := r.parseFirstLocalBinlogEventTimestamp(ctx, file.Name)
		path := filepath.Join(r.binlogDir, file.Name)
		if err != nil {
			log.Error("Failed to parse the first binlog event timestamp for the newly downloaded binlog file",
				zap.String("path", path),
				zap.Error(err))
			return fmt.Errorf("failed to parse the first binlog event timestamp for the newly downloaded binlog file[%s], error[%w]", path, err)
		}
		log.Debug("Checking first event ts with targetTs", zap.Int64("eventTs", eventTs), zap.Int64("targetTs", targetTs))
		if eventTs > targetTs {
			log.Info("Downloaded a binlog file whose first event's timestamp is larger than targetTs",
				zap.String("path", path),
				zap.Int64("targetTs", targetTs),
				zap.Int64("eventTs", eventTs))
			break
		}
	}

	return nil
}

// Syncs the binlog specified by `meta` between the instance and local.
// If isLast is true, it means that this is the last binlog file containing the targetTs event.
// It may keep growing as there are ongoing writes to the database. So we just need to check that
// the file size is larger or equal to the binlog file size we queried from the MySQL server earlier.
func (r *Restore) downloadBinlogFile(ctx context.Context, binlog BinlogFile, isLast bool) (err error) {
	var resultFilePath string
	defer func() {
		if err != nil {
			os.Remove(resultFilePath)
		}
	}()
	// for mysqlbinlog binary, --result-file must end with '/'
	resultFileDir := strings.TrimRight(r.binlogDir, "/") + "/"
	// TODO(zp): support ssl?
	args := []string{
		binlog.Name,
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
	resultFilePath = filepath.Join(resultFileDir, binlog.Name)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("executing mysqlbinlog fails, error: %w", err)
	}

	log.Debug("Checking binlog file stat", zap.String("path", resultFilePath))
	fileInfo, err := os.Stat(resultFilePath)
	if err != nil {
		return fmt.Errorf("cannot get file[%s] stat, error[%w]", resultFilePath, err)
	}
	if !isLast {
		// Case 1: It's an archived binlog file, and we must ensure the file size equals what we queried from the MySQL server earlier.
		if fileInfo.Size() != binlog.Size {
			log.Error("Downloaded binlog file size does not match size queried on the MySQL server",
				zap.String("binlog", binlog.Name),
				zap.Int64("sizeInfo", binlog.Size),
				zap.Int64("downloadedSize", fileInfo.Size()),
			)
			return fmt.Errorf("downloaded binlog file[%s] size[%d] does not equal to size[%d] queried on MySQL server earlier", resultFilePath, fileInfo.Size(), binlog.Size)
		}
	} else {
		// Case 2: It's the last binlog file we need (contains the targetTs).
		// If it's the last (incomplete) binlog file on the MySQL server, it will grow as new writes hit the database server.
		// We just need to check that the downloaded file size >= queried size, so it contains the targetTs event.
		if fileInfo.Size() < binlog.Size {
			log.Error("Downloaded latest binlog file size is smaller than size queried on the MySQL server",
				zap.String("binlog", binlog.Name),
				zap.Int64("sizeInfo", binlog.Size),
				zap.Int64("downloadedSize", fileInfo.Size()),
			)
			return fmt.Errorf("downloaded latest binlog file[%s] size[%d] is smaller than size[%d] queried on MySQL server earlier", resultFilePath, fileInfo.Size(), binlog.Size)
		}
	}

	return nil
}

// GetSortedBinlogFilesMetaOnServer returns the metadata of binlog files.
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
		var binlogFile BinlogFile
		var unused interface{}
		if err := rows.Scan(&binlogFile.Name, &binlogFile.Size, &unused /*Encrypted column*/); err != nil {
			return nil, err
		}
		binlogFiles = append(binlogFiles, binlogFile)
	}

	binlogFiles, err = sortBinlogFiles(binlogFiles)
	if err != nil {
		return nil, err
	}
	return binlogFiles, nil
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

// getDateTime returns converts the targetTs to the local date-time.
func getDateTime(targetTs int64) string {
	t := time.Unix(targetTs, 0)
	return fmt.Sprintf("%d-%d-%d %d:%d:%d", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second())
}
