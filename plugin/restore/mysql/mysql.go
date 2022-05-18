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
	"time"

	"github.com/bytebase/bytebase/api"
	mysql "github.com/bytebase/bytebase/plugin/db/mysql"
)

// TODO(zp): refactor this when sure the mysqlbinlog path
var mysqlbinlogPath = "UNKNOWN"

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

// getPITRDatabaseName composes a pitr database name
func getPITRDatabaseName(database string, timestamp int64) string {
	suffix := fmt.Sprintf("pitr_%d", timestamp)
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

	latestBinlogFileOnServer, err := r.showLatestBinlogFile(ctx)
	if err != nil {
		return err
	}

	// compare file sizes and name to decide which files to download
	downloadedIndex := make(map[int]struct{})
	for index, serverFile := range binlogFilesOnServer {
		// We don't download the latest binlog in SyncBinlogs()
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

	// download the binlog that not record in downloadedIndex
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
	latest, err := r.showLatestBinlogFile(ctx)
	if err != nil {
		return err
	}
	return r.downloadBinlogFile(ctx, instance, saveDir, *latest)
}

// downloadBinlogFile syncs the binlog specified by `meta` between the instance and local.
func (r *Restore) downloadBinlogFile(ctx context.Context, instance *api.Instance, saveDir string, meta mysql.BinlogFile) error {
	tmpFilePrefix := fmt.Sprintf("tmp_%d_", time.Now().UnixNano())
	// TODO(zp): support ssl?
	cmd := exec.CommandContext(ctx, filepath.Join(mysqlbinlogPath, "bin", "mysqlbinlog"),
		meta.Name,
		fmt.Sprintf("--read-from-remote-server --host=%s --port=%s --user=%s --password=%s", instance.Host, instance.Port, instance.Username, instance.Password),
		"--raw",
		// Note, recheck here when upgrade embedding mysqlbinlog binary
		// for mysqlbinlog with binlog.000001 --result-file=a_, it will save it as a_binlog.000001
		fmt.Sprintf("--result-file=%s", filepath.Join(saveDir, tmpFilePrefix)),
	)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr

	var runErr error
	defer func() {
		tmpFileName := fmt.Sprintf("%s%s", tmpFilePrefix, meta.Name)
		tmpFilePath := filepath.Join(saveDir, tmpFileName)
		if runErr != nil {
			// delete the download binlog if some error occur
			_ = os.Remove(tmpFilePath)
			return
		}

		fi, err := os.Stat(tmpFilePath)
		if err != nil {
			// delete the download binlog if cannot stat it
			_ = os.Remove(tmpFilePath)
			return
		}
		if fi.Size() != meta.Size {
			// delete the download binlog if filesize not match
			_ = os.Remove(tmpFilePath)
			return
		}

		// no error, rename it
		_ = os.Rename(tmpFilePath, filepath.Join(saveDir, meta.Name))
	}()

	if runErr = cmd.Run(); runErr != nil {
		return fmt.Errorf("cannot run mysqlbinlog, error: %w", runErr)
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
func (r *Restore) showLatestBinlogFile(ctx context.Context) (*mysql.BinlogFile, error) {
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
