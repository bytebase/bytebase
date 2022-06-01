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

	"github.com/blang/semver/v4"
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
	driver    *mysql.Driver
	mysqlutil *mysqlutil.Instance
	connCfg   db.ConnectionConfig
	binlogDir string
}

type binlogItem struct {
	name string
	seq  int64
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
		"--start-position", strconv.FormatInt(startBinlogInfo.Position, 10),
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
		mysqlArgs = append(mysqlArgs, "--password", r.connCfg.Password)
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

	var toBeReplayedBinlogNames []string

	for _, f := range binlogFiles {
		if f.IsDir() {
			continue
		}
		// for mysql binlog, after the serial number reaches 999999, the next serial number will not return to 000000, but 1000000,
		// so we cannot directly use string to compare lexicographical order.
		binlogSeq, err := getBinlogNameSeq(f.Name())
		if err != nil {
			return nil, fmt.Errorf("failed to parse the binlog file name[%s], error[%w]", f.Name(), err)
		}
		if binlogSeq >= startBinlogSeq {
			toBeReplayedBinlogNames = append(toBeReplayedBinlogNames, f.Name())
		}
	}

	toBeReplayedBinlogItems, err := parseAndSortBinlogFileNames(toBeReplayedBinlogNames)
	if err != nil {
		return nil, err
	}

	for i := 1; i < len(toBeReplayedBinlogItems); i++ {
		if toBeReplayedBinlogItems[i].seq != toBeReplayedBinlogItems[i-1].seq+1 {
			return nil, fmt.Errorf("discontinuous binlog file extensions detected in %s and %s", toBeReplayedBinlogItems[i-1].name, toBeReplayedBinlogItems[i].name)
		}
	}

	var needReplayBinlogFilePath []string
	for _, item := range toBeReplayedBinlogItems {
		needReplayBinlogFilePath = append(needReplayBinlogFilePath, filepath.Join(binlogDir, item.name))
	}

	return needReplayBinlogFilePath, nil
}

// parseAndSortBinlogFileNames will parse the binlog file name, and then sort them in ascending order by their numeric extension.
func parseAndSortBinlogFileNames(binlogFileNames []string) ([]binlogItem, error) {
	var items []binlogItem
	for _, name := range binlogFileNames {
		seq, err := getBinlogNameSeq(name)
		if err != nil {
			return nil, fmt.Errorf("failed to parse the binlog file name[%s], error[%w]", name, err)
		}
		items = append(items, binlogItem{
			name: name,
			seq:  seq,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].seq < items[j].seq
	})
	return items, nil
}

// Locate the binlog event at (filename, position), parse the event and return its timestamp.
// The current mechanism is by invoking mysqlbinlog and parse the output string.
// Maybe we should parse the raw binlog header to get better documented structure?
// nolint
func (r *Restore) parseBinlogEventTimestamp(ctx context.Context, binlogInfo api.BinlogInfo) (int64, error) {
	args := []string{
		path.Join(r.binlogDir, binlogInfo.FileName),
		"--start-position", strconv.FormatInt(binlogInfo.Position, 10),
		// This will trick mysqlbinlog to output the binlog event header followed by a warning message telling that
		// the --stop-position is in the middle of the binlog event.
		// It's OK, since we are only parsing for the timestamp in the binlog event header.
		"--stop-position", strconv.FormatInt(binlogInfo.Position+1, 10),
	}
	var buf bytes.Buffer
	cmd := exec.CommandContext(ctx, r.mysqlutil.GetPath(mysqlutil.MySQLBinlog), args...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = &buf

	if err := cmd.Run(); err != nil {
		log.Error("Command mysqlbinlog fails", zap.String("cmd", cmd.String()), zap.Error(err))
		return 0, fmt.Errorf("command %s fails: %w", cmd.Path, err)
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
		eventTs, err := r.parseBinlogEventTimestamp(ctx, b.Payload.BinlogInfo)
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
	if !dbExists {
		log.Debug("Database does not exist, creating...", zap.String("database", database))
		if _, err := db.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE `%s`", database)); err != nil {
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

	if _, err := db.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE `%s`", pitrOldDatabase)); err != nil {
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

	if _, err := db.ExecContext(ctx, renameStmt); err != nil {
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

// FetchArchivedBinlogFiles downloads the missing binlog files from the remote instance to `binlogDir`,
// but exclude latest binlog. We may  download the latest binlog only when doing PITR.
func (r *Restore) FetchArchivedBinlogFiles(ctx context.Context) error {
	binlogFilesLocal, err := ioutil.ReadDir(r.binlogDir)
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
			log.Warn("inconsistent binlog file size detected", zap.String("binlogFile", serverBinlog.Name), zap.Int64("server", serverBinlog.Size), zap.Int64("local", localBinlogSize))
			if err := os.Remove(filepath.Join(r.binlogDir, serverBinlog.Name)); err != nil {
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
		if err := r.downloadBinlogFile(ctx, serverFile); err != nil {
			return fmt.Errorf("cannot sync binlog %s, error: %w", serverFile.Name, err)
		}
	}

	return nil
}

// IncrementalFetchAllBinlogFiles fetches all the binlog files that on server but not on local.
// TODO(zp): It is not yet supported to synchronize some binlog files.
// The current practice is to download all binlog files, but in the future we hope to only synchronize to the PITR time point
func (r *Restore) IncrementalFetchAllBinlogFiles(ctx context.Context) error {
	if err := r.FetchArchivedBinlogFiles(ctx); err != nil {
		return err
	}
	latestBinlogFileOnServer, err := r.getLatestBinlogFileMeta(ctx)
	if err != nil {
		return err
	}
	return r.downloadBinlogFile(ctx, latestBinlogFileOnServer)
}

// downloadBinlogFile syncs the binlog specified by `meta` between the instance and local.
func (r *Restore) downloadBinlogFile(ctx context.Context, binlog mysql.BinlogFile) error {
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
		args = append(args, "--password", r.connCfg.Password)
	}

	cmd := exec.CommandContext(ctx, r.mysqlutil.GetPath(mysqlutil.MySQLBinlog), args...)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr

	log.Debug("downloading binary log files using mysqlbinlog", zap.String("cmd", cmd.String()))
	resultFilePath := filepath.Join(resultFileDir, binlog.Name)
	if err := cmd.Run(); err != nil {
		_ = os.Remove(resultFilePath)
		return fmt.Errorf("mysqlbinlog fails, error: %w", err)
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
func (r *Restore) getLatestBinlogFileMeta(ctx context.Context) (mysql.BinlogFile, error) {
	// TODO(zp): refactor to reuse getBinlogInfo() in plugin/db/mysql.go
	db, err := r.driver.GetDbConnection(ctx, "")
	if err != nil {
		return mysql.BinlogFile{}, err
	}

	rows, err := db.QueryContext(ctx, `SHOW MASTER STATUS;`)
	if err != nil {
		return mysql.BinlogFile{}, err
	}
	defer rows.Close()

	var binlogFile mysql.BinlogFile
	if rows.Next() {
		var unused interface{} /*Binlog_Do_DB, Binlog_Ignore_DB, Executed_Gtid_Set*/
		if err := rows.Scan(&binlogFile.Name, &binlogFile.Size, &unused, &unused, &unused); err != nil {
			return mysql.BinlogFile{}, err
		}
		return binlogFile, nil
	}
	return mysql.BinlogFile{}, fmt.Errorf("cannot find latest binlog on instance")
}

// getBinlogNameSeq returns the numeric extension to the binary log base name by using split the dot.
// For example: ("binlog.000001") => 1, ("binlog000001") => err
func getBinlogNameSeq(name string) (int64, error) {
	s := strings.Split(name, ".")
	if len(s) != 2 {
		return 0, fmt.Errorf("expecting two parts in the binlog file name, but get %d", len(s))
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

// getDateTime returns converts the targetTs to the local date-time.
func getDateTime(targetTs int64) string {
	t := time.Unix(targetTs, 0)
	return fmt.Sprintf("%d-%d-%d %d:%d:%d", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second())
}
