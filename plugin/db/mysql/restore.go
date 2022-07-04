package mysql

// This file implements recovery functions for MySQL.
// For example, the original database is `dbfoo`. The suffixTs, derived from the PITR issue's CreateTs, is 1653018005.
// Bytebase will do the following:
// 1. Create a database called `dbfoo_pitr_1653018005`, and do PITR restore to it.
// 2. Create a database called `dbfoo_pitr_1653018005_old`, and move tables
// 	  from `dbfoo` to `dbfoo_pitr_1653018005_old`, and tables from `dbfoo_pitr_1653018005` to `dbfoo`.

import (
	"bufio"
	"context"
	"database/sql"
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
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/common/log"
	"github.com/bytebase/bytebase/plugin/db/util"
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

type binlogCoordinate struct {
	Seq int64
	Pos int64
}

func newBinlogCoordinate(binlogFileName string, pos int64) (binlogCoordinate, error) {
	seq, err := getBinlogNameSeq(binlogFileName)
	if err != nil {
		return binlogCoordinate{}, err
	}
	return binlogCoordinate{Seq: seq, Pos: pos}, nil
}

// SetUpForPITR sets necessary fields for MySQL PITR recovery.
func (driver *Driver) SetUpForPITR(mysqlutilInstance mysqlutil.Instance, binlogDir string) {
	driver.mysqlutil = mysqlutilInstance
	driver.binlogDir = binlogDir
}

// ReplayBinlog replays the binlog for `originDatabase` from `startBinlogInfo.Position` to `targetTs`.
func (driver *Driver) replayBinlog(ctx context.Context, originalDatabase, pitrDatabase string, startBinlogInfo api.BinlogInfo, targetTs int64) error {
	replayBinlogPaths, err := getBinlogReplayList(startBinlogInfo, driver.binlogDir)
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
		"--host", driver.connCfg.Host,
		"--user", driver.connCfg.Username,
	}
	if driver.connCfg.Port != "" {
		mysqlArgs = append(mysqlArgs, "--port", driver.connCfg.Port)
	}
	if driver.connCfg.Password != "" {
		// The --password parameter of mysql/mysqlbinlog does not support the "--password PASSWORD" format (split by space).
		// If provided like that, the program will hang.
		mysqlArgs = append(mysqlArgs, fmt.Sprintf("--password=%s", driver.connCfg.Password))
	}

	mysqlbinlogCmd := exec.CommandContext(ctx, driver.mysqlutil.GetPath(mysqlutil.MySQLBinlog), mysqlbinlogArgs...)
	mysqlCmd := exec.CommandContext(ctx, driver.mysqlutil.GetPath(mysqlutil.MySQL), mysqlArgs...)
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
		return fmt.Errorf("cannot start mysqlbinlog command, error: %w", err)
	}
	if err := mysqlCmd.Run(); err != nil {
		return fmt.Errorf("mysql command fails, error: %w", err)
	}
	if err := mysqlbinlogCmd.Wait(); err != nil {
		return fmt.Errorf("error occurred while waiting for mysqlbinlog to exit: %w", err)
	}
	return nil
}

// RestorePITR is a wrapper to perform PITR. It restores a full backup followed by replaying the binlog.
// It performs the step 1 of the restore process.
// TODO(dragonly): Refactor so that the first part is in driver.Restore, and remove this wrapper.
func (driver *Driver) RestorePITR(ctx context.Context, fullBackup *bufio.Scanner, startBinlogInfo api.BinlogInfo, database string, suffixTs, targetTs int64) error {
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

	db, err := driver.GetDbConnection(ctx, "")
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

	if err := driver.RestoreTx(ctx, tx, fullBackup); err != nil {
		return err
	}

	// The full backup is restored successfully, enable foreign key constraints as normal.
	if _, err := tx.ExecContext(ctx, "SET foreign_key_checks=ON"); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	if err := driver.replayBinlog(ctx, database, pitrDatabaseName, startBinlogInfo, targetTs); err != nil {
		return fmt.Errorf("failed to replay binlog, error: %w", err)
	}

	return nil
}

// getBinlogReplayList returns the path list of the binlog that need be replayed.
func getBinlogReplayList(startBinlogInfo api.BinlogInfo, binlogDir string) ([]string, error) {
	startBinlogSeq, err := getBinlogNameSeq(startBinlogInfo.FileName)
	if err != nil {
		return nil, fmt.Errorf("cannot parse the start binlog file name %q, error: %w", startBinlogInfo.FileName, err)
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
		return nil, fmt.Errorf("the starting binlog file %q does not exist locally", startBinlogInfo.FileName)
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

// GetLatestBackupBeforeOrEqualTs finds the latest logical backup and corresponding binlog info whose time is before or equal to `targetTs`.
// The backupList should only contain DONE backups.
func (driver *Driver) GetLatestBackupBeforeOrEqualTs(ctx context.Context, backupList []*api.Backup, targetTs int64) (*api.Backup, error) {
	if len(backupList) == 0 {
		return nil, fmt.Errorf("no valid backup")
	}

	targetBinlogCoordinate, err := driver.getBinlogCoordinateByTs(ctx, targetTs)
	if err != nil {
		return nil, fmt.Errorf("failed to get binlog coordinate by targetTs %d, error: %w", targetTs, err)
	}

	var validBackupList []*api.Backup
	for _, b := range backupList {
		if b.Payload.BinlogInfo.IsEmpty() {
			log.Debug("Skip parsing binlog event timestamp of the backup where BinlogInfo is empty", zap.Int("backupId", b.ID), zap.String("backupName", b.Name))
			continue
		}
		validBackupList = append(validBackupList, b)
	}

	return getLatestBackupBeforeOrEqualBinlogCoord(validBackupList, *targetBinlogCoordinate)
}

func getLatestBackupBeforeOrEqualBinlogCoord(backupList []*api.Backup, targetBinlogCoordinate binlogCoordinate) (*api.Backup, error) {
	type backupBinlogCoordinate struct {
		binlogCoordinate
		backup *api.Backup
	}
	var backupCoordinateListSorted []backupBinlogCoordinate
	for _, b := range backupList {
		c, err := newBinlogCoordinate(b.Payload.BinlogInfo.FileName, b.Payload.BinlogInfo.Position)
		if err != nil {
			return nil, err
		}
		backupCoordinateListSorted = append(backupCoordinateListSorted, backupBinlogCoordinate{binlogCoordinate: c, backup: b})
	}

	// Sort in order that latest binlog coordinate comes first.
	sort.Slice(backupCoordinateListSorted, func(i, j int) bool {
		return backupCoordinateListSorted[i].Seq > backupCoordinateListSorted[j].Seq ||
			(backupCoordinateListSorted[i].Seq == backupCoordinateListSorted[j].Seq && backupCoordinateListSorted[i].Pos > backupCoordinateListSorted[j].Pos)
	})

	var backup *api.Backup
	for _, bc := range backupCoordinateListSorted {
		if bc.Seq < targetBinlogCoordinate.Seq || (bc.Seq == targetBinlogCoordinate.Seq && bc.Pos <= targetBinlogCoordinate.Pos) {
			backup = bc.backup
			break
		}
	}

	if backup == nil {
		oldestBackupBinlogCoordinate := backupCoordinateListSorted[len(backupCoordinateListSorted)-1]
		log.Error("The target binlog coordinate is earlier than the oldest backup's binlog coordinate",
			zap.Any("targetBinlogCoordinate", targetBinlogCoordinate),
			zap.Any("oldestBackupBinlogCoordinate", oldestBackupBinlogCoordinate))
		return nil, fmt.Errorf("the target binlog coordinate %v is earlier than the oldest backup's binlog coordinate %v", targetBinlogCoordinate, oldestBackupBinlogCoordinate)
	}

	return backup, nil
}

// SwapPITRDatabase renames the pitr database to the target, and the original to the old database
// It returns the pitr and old database names after swap.
// It performs the step 2 of the restore process.
func SwapPITRDatabase(ctx context.Context, conn *sql.Conn, database string, suffixTs int64) (string, string, error) {
	pitrDatabaseName := getPITRDatabaseName(database, suffixTs)
	pitrOldDatabase := getPITROldDatabaseName(database, suffixTs)

	// Handle the case that the original database does not exist, because user could drop a database and want to restore it.
	log.Debug("Checking database exists.", zap.String("database", database))
	dbExists, err := databaseExists(ctx, conn, database)
	if err != nil {
		return pitrDatabaseName, pitrOldDatabase, fmt.Errorf("failed to check whether database %q exists, error: %w", database, err)
	}

	log.Debug("Turning binlog OFF.")
	// Set OFF the session variable sql_log_bin so that the writes in the following SQL statements will not be recorded in the binlog.
	if _, err := conn.ExecContext(ctx, "SET sql_log_bin=OFF"); err != nil {
		return pitrDatabaseName, pitrOldDatabase, err
	}

	if !dbExists {
		log.Debug("Database does not exist, creating...", zap.String("database", database))
		if _, err := conn.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE `%s`", database)); err != nil {
			return pitrDatabaseName, pitrOldDatabase, fmt.Errorf("failed to create non-exist database %q, error: %w", database, err)
		}
	}

	log.Debug("Getting tables in the original and PITR databases.")
	tables, err := getTables(ctx, conn, database)
	if err != nil {
		return pitrDatabaseName, pitrOldDatabase, fmt.Errorf("failed to get tables of database %q, error: %w", database, err)
	}
	tablesPITR, err := getTables(ctx, conn, pitrDatabaseName)
	if err != nil {
		return pitrDatabaseName, pitrOldDatabase, fmt.Errorf("failed to get tables of database %q, error: %w", pitrDatabaseName, err)
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

func databaseExists(ctx context.Context, conn *sql.Conn, database string) (bool, error) {
	query := fmt.Sprintf("SELECT SCHEMA_NAME FROM INFORMATION_SCHEMA.SCHEMATA WHERE SCHEMA_NAME='%s'", database)
	row, err := conn.QueryContext(ctx, query)
	if err != nil {
		return false, err
	}
	defer row.Close()
	if row.Next() {
		return true, nil
	}
	if err := row.Err(); err != nil {
		return false, util.FormatErrorWithQuery(err, query)
	}
	// The query returns empty row, which means there's no such database.
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
func (driver *Driver) downloadBinlogFilesOnServer(ctx context.Context, binlogFilesLocal, binlogFilesOnServerSorted []BinlogFile) error {
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
		path := filepath.Join(driver.binlogDir, fileOnServer.Name)
		if !existLocal {
			if err := driver.downloadBinlogFile(ctx, fileOnServer, fileOnServer.Name == latestBinlogFileOnServer.Name); err != nil {
				log.Error("Failed to download binlog file", zap.String("path", path), zap.Error(err))
				return fmt.Errorf("failed to download binlog file %q, error: %w", path, err)
			}
		} else if fileLocal.Size != fileOnServer.Size {
			log.Debug("Deleting inconsistent local binlog file",
				zap.String("path", path),
				zap.Int64("sizeLocal", fileOnServer.Size),
				zap.Int64("sizeOnServer", fileOnServer.Size))
			if err := os.Remove(path); err != nil {
				log.Error("Failed to remove inconsistent local binlog file", zap.String("path", path), zap.Error(err))
				return fmt.Errorf("failed to remove inconsistent local binlog file %q, error: %w", path, err)
			}
			if err := driver.downloadBinlogFile(ctx, fileOnServer, fileOnServer.Name == latestBinlogFileOnServer.Name); err != nil {
				log.Error("Failed to re-download inconsistent local binlog file", zap.String("path", path), zap.Error(err))
				return fmt.Errorf("failed to re-download inconsistent local binlog file %q, error: %w", path, err)
			}
		}
	}
	return nil
}

// FetchAllBinlogFiles downloads all binlog files on server to `binlogDir`.
func (driver *Driver) FetchAllBinlogFiles(ctx context.Context) error {
	// Read binlog files list on server.
	binlogFilesOnServerSorted, err := driver.GetSortedBinlogFilesMetaOnServer(ctx)
	if err != nil {
		return err
	}
	if len(binlogFilesOnServerSorted) == 0 {
		log.Debug("No binlog file found on server to download")
		return nil
	}
	log.Debug("Got sorted binlog file list on server", zap.Array("list", ZapBinlogFiles(binlogFilesOnServerSorted)))

	// Read the local binlog files.
	binlogFilesLocalSorted, err := GetSortedLocalBinlogFiles(driver.binlogDir)
	if err != nil {
		return fmt.Errorf("failed to read local binlog files, error: %w", err)
	}

	return driver.downloadBinlogFilesOnServer(ctx, binlogFilesLocalSorted, binlogFilesOnServerSorted)
}

// Syncs the binlog specified by `meta` between the instance and local.
// If isLast is true, it means that this is the last binlog file containing the targetTs event.
// It may keep growing as there are ongoing writes to the database. So we just need to check that
// the file size is larger or equal to the binlog file size we queried from the MySQL server earlier.
func (driver *Driver) downloadBinlogFile(ctx context.Context, binlogFileToDownload BinlogFile, isLast bool) error {
	// for mysqlbinlog binary, --result-file must end with '/'
	resultFileDir := strings.TrimRight(driver.binlogDir, "/") + "/"
	// TODO(zp): support ssl?
	args := []string{
		binlogFileToDownload.Name,
		"--read-from-remote-server",
		"--raw",
		"--host", driver.connCfg.Host,
		"--user", driver.connCfg.Username,
		"--result-file", resultFileDir,
	}
	if driver.connCfg.Port != "" {
		args = append(args, "--port", driver.connCfg.Port)
	}
	if driver.connCfg.Password != "" {
		args = append(args, fmt.Sprintf("--password=%s", driver.connCfg.Password))
	}

	cmd := exec.CommandContext(ctx, driver.mysqlutil.GetPath(mysqlutil.MySQLBinlog), args...)
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
		return fmt.Errorf("cannot get file %q stat, error: %w", resultFilePath, err)
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
			return fmt.Errorf("downloaded latest binlog file %q size[%d] is smaller than size[%d] queried on MySQL server earlier", resultFilePath, fileInfo.Size(), binlogFileToDownload.Size)
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
			return fmt.Errorf("downloaded binlog file %q size[%d] is not equal to size[%d] queried on MySQL server earlier", resultFilePath, fileInfo.Size(), binlogFileToDownload.Size)
		}
	}

	return nil
}

// GetSortedBinlogFilesMetaOnServer returns the metadata of binlog files in ascending order by their numeric extension.
func (driver *Driver) GetSortedBinlogFilesMetaOnServer(ctx context.Context) ([]BinlogFile, error) {
	db, err := driver.GetDbConnection(ctx, "")
	if err != nil {
		return nil, err
	}

	query := "SHOW BINARY LOGS"
	rows, err := db.QueryContext(ctx, query)
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
	if err := rows.Err(); err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}

	return sortBinlogFiles(binlogFiles), nil
}

// getBinlogCoordinateByTs converts a timestamp to binlog coordinate using local binlog files.
func (driver *Driver) getBinlogCoordinateByTs(ctx context.Context, targetTs int64) (*binlogCoordinate, error) {
	binlogFilesLocalSorted, err := GetSortedLocalBinlogFiles(driver.binlogDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read sorted local binlog files, error: %w", err)
	}
	if len(binlogFilesLocalSorted) == 0 {
		return nil, fmt.Errorf("no local binlog files found")
	}
	if !binlogFilesAreContinuous(binlogFilesLocalSorted) {
		return nil, fmt.Errorf("local binlog files are not continuous")
	}

	var binlogFileTarget *BinlogFile
	for i, file := range binlogFilesLocalSorted {
		eventTs, err := driver.parseLocalBinlogFirstEventTs(ctx, file.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to parse the local binlog file %q's first binlog event ts, error: %w", file.Name, err)
		}
		if eventTs >= targetTs {
			if i == 0 {
				return nil, fmt.Errorf("the targetTs %d is before the first event ts %d of the oldest binlog file %q", targetTs, eventTs, file.Name)
			}
			// The previous local binlog file contains targetTs.
			binlogFileTarget = &binlogFilesLocalSorted[i-1]
			break
		}
	}
	// All of the local binlog files' first event start ts <= targetTs, so we choose the last binlog file as probably "containing" targetTs.
	// This may not be true, because possibly targetTs > last eventTs of the last binlog file.
	// In this case, we should return an error.
	var isLastBinlogFile bool
	if binlogFileTarget == nil {
		isLastBinlogFile = true
		binlogFileTarget = &binlogFilesLocalSorted[len(binlogFilesLocalSorted)-1]
	}
	targetSeq, err := getBinlogNameSeq(binlogFileTarget.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to parse seq from binlog file name %q", binlogFileTarget.Name)
	}

	eventPos, err := driver.getBinlogEventPositionAtOrAfterTs(ctx, *binlogFileTarget, targetTs)
	if err != nil {
		if common.ErrorCode(err) == common.NotFound {
			// All the binlog events in this binlog file have ts < targetTs.
			// If this is the last binlog file, the user wants to recover to a time in the future and we should return an error.
			// Otherwise, we should return the end position of the current binlog file.
			if isLastBinlogFile {
				return nil, fmt.Errorf("the targetTs %d is after the last event ts of the latest binlog file %q", targetTs, binlogFileTarget.Name)
			}
			return &binlogCoordinate{Seq: targetSeq, Pos: math.MaxInt64}, nil
		}
		return nil, fmt.Errorf("failed to find the binlog event after targetTs %d, error: %w", targetTs, err)
	}
	return &binlogCoordinate{Seq: targetSeq, Pos: eventPos}, nil
}

func parseBinlogEventTsInLine(line string) (eventTs int64, found bool, err error) {
	// The target line starts with string like "#220421 14:49:26 server id 1"
	if !strings.Contains(line, "server id") {
		return 0, false, nil
	}
	if strings.Contains(line, "end_log_pos 0") {
		// https://github.com/mysql/mysql-server/blob/8.0/client/mysqlbinlog.cc#L1209-L1212
		// Fake events with end_log_pos=0 could be generated and we need to ignore them.
		return 0, false, nil
	}
	fields := strings.Fields(line)
	// fields should starts with ["#220421", "14:49:26", "server", "id", "1", "end_log_pos", "34794"]
	if len(fields) < 7 ||
		(len(fields[0]) != 7 || fields[2] != "server" || fields[3] != "id" || fields[5] != "end_log_pos") {
		return 0, false, fmt.Errorf("found unexpected mysqlbinlog output line %q when parsing binlog event timestamp", line)
	}
	datetime, err := time.ParseInLocation("060102 15:04:05", fmt.Sprintf("%s %s", fields[0][1:], fields[1]), time.Local)
	if err != nil {
		return 0, false, err
	}
	return datetime.Unix(), true, nil
}

func parseBinlogEventPosInLine(line string) (pos int64, found bool, err error) {
	// The mysqlbinlog output will contains a line starting with "# at 35065", which is the binlog event's start position.
	if !strings.HasPrefix(line, "# at ") {
		return 0, false, nil
	}
	// This is the line containing the start position of the binlog event.
	fields := strings.Fields(line)
	if len(fields) != 3 {
		return 0, false, fmt.Errorf("unexpected mysqlbinlog output line %q when parsing binlog event start position", line)
	}
	pos, err = strconv.ParseInt(fields[2], 10, 0)
	if err != nil {
		return 0, false, err
	}
	return pos, true, nil
}

// Parse the first binlog eventTs of a local binlog file.
func (driver *Driver) parseLocalBinlogFirstEventTs(ctx context.Context, fileName string) (int64, error) {
	args := []string{
		// Local binlog file path.
		path.Join(driver.binlogDir, fileName),
		// Tell mysqlbinlog to suppress the BINLOG statements for row events, which reduces the unneeded output.
		"--base64-output=DECODE-ROWS",
	}
	cmd := exec.CommandContext(ctx, driver.mysqlutil.GetPath(mysqlutil.MySQLBinlog), args...)
	cmd.Stderr = os.Stderr
	pr, err := cmd.StdoutPipe()
	if err != nil {
		return 0, err
	}
	s := bufio.NewScanner(pr)
	if err := cmd.Start(); err != nil {
		return 0, err
	}
	defer func() {
		_ = pr.Close()
		_ = cmd.Process.Kill()
	}()

	var eventTs int64
	for s.Scan() {
		line := s.Text()
		eventTsParsed, found, err := parseBinlogEventTsInLine(line)
		if err != nil {
			return 0, fmt.Errorf("failed to parse binlog eventTs from mysqlbinlog output, error: %w", err)
		}
		if !found {
			continue
		}
		eventTs = eventTsParsed
		break
	}

	return eventTs, nil
}

// Use command like mysqlbinlog --start-datetime=targetTs binlog.000001 to parse the first binlog event position with timestamp equal or after targetTs.
// TODO(dragonly): Add integration test.
func (driver *Driver) getBinlogEventPositionAtOrAfterTs(ctx context.Context, binlogFile BinlogFile, targetTs int64) (int64, error) {
	args := []string{
		// Local binlog file path.
		path.Join(driver.binlogDir, binlogFile.Name),
		// Tell mysqlbinlog to suppress the BINLOG statements for row events, which reduces the unneeded output.
		"--base64-output=DECODE-ROWS",
		// Instruct mysqlbinlog to start output only after encountering the first binlog event with timestamp equal or after targetTs.
		"--start-datetime", formatDateTime(targetTs),
	}
	cmd := exec.CommandContext(ctx, driver.mysqlutil.GetPath(mysqlutil.MySQLBinlog), args...)
	cmd.Stderr = os.Stderr
	pr, err := cmd.StdoutPipe()
	if err != nil {
		return 0, err
	}
	s := bufio.NewScanner(pr)
	if err := cmd.Start(); err != nil {
		return 0, err
	}
	defer func() {
		_ = pr.Close()
		_ = cmd.Process.Kill()
	}()

	var pos int64
	for s.Scan() {
		line := s.Text()
		posParsed, found, err := parseBinlogEventPosInLine(line)
		if err != nil {
			return 0, fmt.Errorf("failed to parse binlog event start position from mysqlbinlog output, error: %w", err)
		}
		if !found {
			continue
		}
		if posParsed == 4 {
			// When invoking mysqlbinlog with --start-datetime, the first valid event will always be FORMAT_DESCRIPTION_EVENT which should be skipped.
			continue
		}
		pos = posParsed
		break
	}

	if pos == 0 {
		return 0, common.Errorf(common.NotFound, fmt.Errorf("failed to find event position at or after targetTs %d", targetTs))
	}

	return pos, nil
}

// getBinlogNameSeq returns the numeric extension to the binary log base name by using split the dot.
// For example: ("binlog.000001") => 1, ("binlog000001") => err
func getBinlogNameSeq(name string) (int64, error) {
	s := strings.Split(name, ".")
	if len(s) != 2 {
		return 0, fmt.Errorf("failed to parse binlog extension, expecting two parts in the binlog file name %q but got %d", name, len(s))
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
func (driver *Driver) CheckServerVersionForPITR(ctx context.Context) error {
	value, err := driver.getServerVariable(ctx, "version")
	if err != nil {
		return err
	}
	if err := checkVersionForPITR(value); err != nil {
		return err
	}
	return nil
}

// CheckEngineInnoDB checks that the tables in the database is all using InnoDB as the storage engine.
func (driver *Driver) CheckEngineInnoDB(ctx context.Context, database string) error {
	db, err := driver.GetDbConnection(ctx, "")
	if err != nil {
		return err
	}

	// ref: https://dev.mysql.com/doc/refman/8.0/en/information-schema-tables-table.html
	query := fmt.Sprintf("SELECT table_name, engine FROM information_schema.tables WHERE table_schema='%s';", database)
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()
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
	if err := rows.Err(); err != nil {
		return util.FormatErrorWithQuery(err, query)
	}
	if len(tablesNotInnoDB) != 0 {
		return fmt.Errorf("tables %v of database %s do not use the InnoDB engine, which is required for PITR", tablesNotInnoDB, database)
	}
	return nil
}

func (driver *Driver) getServerVariable(ctx context.Context, varName string) (string, error) {
	db, err := driver.GetDbConnection(ctx, "")
	if err != nil {
		return "", err
	}

	query := fmt.Sprintf("SHOW VARIABLES LIKE '%s'", varName)
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	var found bool
	if rows.Next() {
		found = true
	}
	if err := rows.Err(); err != nil {
		return "", util.FormatErrorWithQuery(err, query)
	}
	if !found {
		return "", common.FormatDBErrorEmptyRowWithQuery(query)
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
func (driver *Driver) CheckBinlogEnabled(ctx context.Context) error {
	value, err := driver.getServerVariable(ctx, "log_bin")
	if err != nil {
		return err
	}
	if strings.ToUpper(value) != "ON" {
		return fmt.Errorf("binlog is not enabled")
	}
	return nil
}

// CheckBinlogRowFormat checks whether the binlog format is ROW.
func (driver *Driver) CheckBinlogRowFormat(ctx context.Context) error {
	value, err := driver.getServerVariable(ctx, "binlog_format")
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
