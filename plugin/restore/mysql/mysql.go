package mysql

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/plugin/db/mysql"
)

// TODO(zp): refactor this when sure the mysqlbinlog path
var mysqlbinlogBinPath = "UNKNOWN"

const (
	// MaxDatabaseNameLength is the allowed max database name length in MySQL
	MaxDatabaseNameLength = 64
)

// Restore implements recovery functions for MySQL
type Restore struct {
	driver *mysql.Driver
}

// New creates a new instance of Restore
func New(driver *mysql.Driver) *Restore {
	return &Restore{
		driver: driver,
	}
}

// RestoreBinlog restores the database using incremental backup in time range of [config.Start, config.End).
func (r *Restore) RestoreBinlog(ctx context.Context, config mysql.BinlogInfo) error {
	return fmt.Errorf("Unimplemented")
}

// RestorePITR is a wrapper for restore a full backup and a range of incremental backup
func (r *Restore) RestorePITR(ctx context.Context, fullBackup *bufio.Scanner, binlog mysql.BinlogInfo, database string, suffixTs int64) error {
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

	// TODO(dragonly): implement RestoreBinlog in mysql driver
	_ = r.RestoreBinlog(ctx, binlog)

	return nil
}

// nolint
type backupComparator struct {
	backup      *api.Backup
	binlogInfo  *mysql.BinlogInfo
	binlogIndex int64
}

// backupSorter implements interface sort.Interface
type backupSorter []backupComparator // nolint

func (b backupSorter) Len() int      { return len(b) }           // nolint
func (b backupSorter) Swap(i, j int) { b[i], b[j] = b[j], b[i] } // nolint

// Newer binlog filename/position will be in front of older ones.
// nolint
func (b backupSorter) Less(i, j int) bool {
	if b[i].binlogIndex > b[j].binlogIndex {
		return true
	}
	return b[i].binlogInfo.Position > b[j].binlogInfo.Position
}

// parse the numeric extension part of the binlog file names, e.g., binlog.000001 -> 1
// nolint
func parseBinlogFileNameIndex(filename string) (int64, error) {
	parts := strings.Split(filename, ".")
	if len(parts) != 2 {
		return -1, fmt.Errorf("the filename %q is not a valid binlog file name format", filename)
	}
	index, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return -1, fmt.Errorf("failed to parse string[%s] to integer, error[%w]", parts[1], err)
	}
	return index, nil
}

// Parse the binlog at position, and return the timestamp in the binlog event.
// The current mechanism is by invoking mysqlbinlog with -v and parse the output string.
// Maybe we should parse the raw binlog header to get better documented structure?
// nolint
func (r *Restore) parseBinlogEventTimestamp(binlogInfo *mysql.BinlogInfo) (int64, error) {
	connConfig := r.driver.GetConnectionConfig()
	mysqlbinlogBinPath := "fake"
	args := []string{
		"--read-from-remote-server",
		fmt.Sprintf("--host %s", connConfig.Host),
		fmt.Sprintf("--user %s", connConfig.Username),
		fmt.Sprintf("--start-position %d", binlogInfo.Position),
	}
	cmd := exec.Command(mysqlbinlogBinPath, args...)
	cmd.Stderr = os.Stderr
	output, err := cmd.StdoutPipe()
	if err != nil {
		return -1, err
	}
	if err := cmd.Start(); err != nil {
		return -1, err
	}

	buf := make([]byte, 1024)
	var lineBuf bytes.Buffer
	var line string
	var timestamp int64
	// matches server time pattern like "#220421 14:49:26", which is basically "#YYMMDD hh:mm:ss"
	reServerTime := regexp.MustCompile(`^#(\d{2})(\d{2})(\d{2}) (\d{2}):(\d{2}):(\d{2})`)

	// Read lines from mysqlbinlog text output
	for {
		n, err := output.Read(buf)
		if n > 0 {
			bytesRead := buf[:n]
			index := bytes.Index(bytesRead, []byte("\n"))
			// no complete line found, append
			if index == -1 {
				lineBuf.Write(bytesRead)
			} else {
				// complete line found
				lineBuf.Write(bytesRead[:index])
				line = lineBuf.String()
				lineBuf.Reset()
				// write the rest of the next line into buffer
				if index < len(bytesRead)-1 {
					lineBuf.Write(bytesRead[index+1:])
				}
			}
		}
		if err != nil {
			if err == io.EOF {
				line = lineBuf.String()
				lineBuf.Reset()
			} else {
				return -1, err
			}
		}

		// parse timestamp from new complete line
		if len(line) != 0 {
			matches := reServerTime.FindStringSubmatch(line)
			if matches != nil {
				year, err := strconv.Atoi(matches[1])
				if err != nil {
					return -1, err
				}
				year += 2000
				month, err := strconv.Atoi(matches[2])
				if err != nil {
					return -1, err
				}
				day, err := strconv.Atoi(matches[3])
				if err != nil {
					return -1, err
				}
				hour, err := strconv.Atoi(matches[4])
				if err != nil {
					return -1, err
				}
				minute, err := strconv.Atoi(matches[5])
				if err != nil {
					return -1, err
				}
				second, err := strconv.Atoi(matches[6])
				if err != nil {
					return -1, err
				}
				timestamp = time.Date(year, time.Month(month), day, hour, minute, second, 0, time.UTC).Unix()
				break
			}
			// clear current line and enters the state of processing the next line
			line = ""
		}
	}

	// We eagerly exit the mysqlbinlog process as long as we get the timestamp to save resources.
	if err := cmd.Process.Kill(); err != nil {
		return -1, err
	}

	return timestamp, nil
}

// Find the latest logical backup and corresponding binlog info whose time is before `restoreTs`.
// The backupList should only contain DONE backups.
// TODO(dragonly)/TODO(zp): Use this when the apply binlog PR is ready, and remove the nolint comments
// nolint
func (r *Restore) findBackupAndBinlogInfo(ctx context.Context, backupList []*api.Backup, restoreTs int64) (*api.Backup, *mysql.BinlogInfo, error) {
	// Sort backups by their binlog filename and position, with the latest at the front of the list.
	backups, err := sortBackupInfo(backupList)
	if err != nil {
		return nil, nil, err
	}

	// TODO(dragonly): Download the latest (partial) binlog file when needed.
	backup, err := r.findNearestBinlogEventTimestamp(backups, restoreTs)
	if err != nil {
		return nil, nil, err
	}

	return backup.backup, backup.binlogInfo, nil
}

// nolint
func sortBackupInfo(backupList []*api.Backup) ([]backupComparator, error) {
	var backups []backupComparator
	for _, b := range backupList {
		payload := mysql.BackupPayload{}
		if err := json.Unmarshal([]byte(b.Payload), &payload); err != nil {
			return nil, fmt.Errorf("failed to unmarshal backup payload[%s], error[%w]", b.Payload, err)
		}
		binlogIndex, err := parseBinlogFileNameIndex(payload.BinlogInfo.FileName)
		if err != nil {
			return nil, err
		}
		backups = append(backups, backupComparator{
			backup:      b,
			binlogInfo:  &payload.BinlogInfo,
			binlogIndex: binlogIndex,
		})
	}

	sort.Sort(backupSorter(backups))

	return backups, nil
}

// Traverse the backups starting from the latest and find the first with timestamp less than the target restore timestamp.
// nolint
func (r *Restore) findNearestBinlogEventTimestamp(backups []backupComparator, restoreTs int64) (backupComparator, error) {
	var eventTs int64
	var err error
	for _, b := range backups {
		// Parse the binlog files and convert binlog positions into MySQL server timestamps.
		eventTs, err = r.parseBinlogEventTimestamp(b.binlogInfo)
		if err != nil {
			return backupComparator{}, err
		}
		if eventTs < restoreTs {
			return b, nil
		}
	}
	return backupComparator{}, fmt.Errorf("the target restore timestamp[%d] is earlier than the oldest backup time[%d]", restoreTs, eventTs)
}

// SwapPITRDatabase renames the pitr database to the target, and the original to the old database
func (r *Restore) SwapPITRDatabase(ctx context.Context, database string, timestamp int64) error {
	db, err := r.driver.GetDbConnection(ctx, "")
	if err != nil {
		return err
	}
	txn, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer txn.Rollback()

	pitrOldDatabase := getSafeName(database, "old")
	pitrDatabaseName := getPITRDatabaseName(database, timestamp)

	if _, err := txn.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE `%s`", pitrOldDatabase)); err != nil {
		return err
	}

	// TODO(dragonly): handle the case that the original database does not exist
	tables, err := mysql.GetTables(txn, database)
	if err != nil {
		return fmt.Errorf("failed to get tables of database %q, error[%w]", database, err)
	}
	tablesPITR, err := mysql.GetTables(txn, pitrDatabaseName)
	if err != nil {
		return fmt.Errorf("failed to get tables of database %q, error[%w]", pitrDatabaseName, err)
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
		return err
	}

	if err := txn.Commit(); err != nil {
		return err
	}

	return nil
}

// DeleteOldDatabase deletes the old database after the PITR swap task.
func (r *Restore) DeleteOldDatabase(ctx context.Context, database string, suffixTs int64) error {
	db, err := r.driver.GetDbConnection(ctx, "")
	if err != nil {
		return err
	}

	pitrOldDatabase := getPITROldDatabaseName(database, suffixTs)
	if _, err := db.ExecContext(ctx, fmt.Sprintf("DROP DATABASE `%s`;", pitrOldDatabase)); err != nil {
		return err
	}

	return nil
}

// getPITRDatabaseName composes a pitr database name
func getPITRDatabaseName(database string, suffixTs int64) string {
	suffix := fmt.Sprintf("pitr_%d", suffixTs)
	return getSafeName(database, suffix)
}

// getPITROldDatabaseName composes a pitr database name
func getPITROldDatabaseName(database string, suffixTs int64) string {
	suffix := fmt.Sprintf("pitr_%d_old", suffixTs)
	return getSafeName(database, suffix)
}

// SyncArchivedBinlogFiles syncs the binlogs between the instance and `saveDir`,
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
	cmd := exec.CommandContext(ctx, filepath.Join(mysqlbinlogBinPath, "bin", "mysqlbinlog"),
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

// getBinlogFilesMetaOnServer returns the metadata of binlogs
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
	//TODO(zp): refactor to reuse getBinlogInfo() in plugin/db/mysql.go
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
