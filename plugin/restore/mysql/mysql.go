package mysql

import (
	"bufio"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/bytebase/bytebase/api"
	mysql "github.com/bytebase/bytebase/plugin/db/mysql"
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

// SyncArchivedBinlogFiles syncs the binlogs between the instance and `saveDir`, but exclude latest binlog
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

	// compare file sizes and names to decide which files to download
	// downloadedIndex maintains the index of BinlogFile slice, for example:
	// `SHOW MASTER LOGS` return ["binlog.000001", "binlog.000002"],
	// if we had downloaded binlog.000001 and file size match, {0:struct{}{}} will be involved in downloadedIndex.
	downloadedIndex := make(map[int]struct{})
	for index, serverFile := range binlogFilesOnServer {
		// We don't download the latest binlog in SyncArchivedBinlogFiles()
		if serverFile.Name == latestBinlogFileOnServer.Name {
			continue
		}
		for _, localFile := range binlogFilesLocal {
			localFileName := localFile.Name()
			if localFileName == serverFile.Name {
				// binlog file exists on local
				if localFile.Size() != serverFile.Size {
					// File size not match, delete and then download it
					if err := os.Remove(filepath.Join(saveDir, localFileName)); err != nil {
						return fmt.Errorf("cannot remove %s, error: %w", localFileName, err)
					}
				} else {
					// file size match, record it in downloadedIndex
					downloadedIndex[index] = struct{}{}
				}
			}
		}
	}

	// download the binlog files not recorded in downloadedIndex
	for i, serverFile := range binlogFilesOnServer {
		if _, ok := downloadedIndex[i]; ok {
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
