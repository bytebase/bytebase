package mysql

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/blang/semver/v4"
	"github.com/bytebase/bytebase/api"
	mysql "github.com/bytebase/bytebase/plugin/db/mysql"
	"github.com/bytebase/bytebase/resources/mysqlbinlog"
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
	driver      *mysql.Driver
	mysqlbinlog *mysqlbinlog.Instance
}

// New creates a new instance of Restore
func New(driver *mysql.Driver, instance *mysqlbinlog.Instance) *Restore {
	return &Restore{
		driver:      driver,
		mysqlbinlog: instance,
	}
}

// ReplayBinlog replays the binlog about `originDatabase` from `startBinlogInfo.Position` to `targetTs`.
func (r *Restore) ReplayBinlog(ctx context.Context, originDatabase, pitrDatabase, binlogDir string, startBinlogInfo mysql.BinlogInfo, targetTs int64) error {
	db, err := r.driver.GetDbConnection(ctx, "")
	if err != nil {
		return err
	}

	binlogNamePrefix, err := r.getBinlogNamePrefix(ctx)
	if err != nil {
		return fmt.Errorf("cannot get the prefix of binlog name, error: %w", err)
	}

	if !strings.HasPrefix(startBinlogInfo.FileName, binlogNamePrefix) {
		return fmt.Errorf("the starting binlog file name must have the prefix %q, but get %q", binlogNamePrefix, startBinlogInfo.FileName)
	}

	startBinlogSeq, err := strconv.ParseInt(strings.TrimPrefix(startBinlogInfo.FileName, binlogNamePrefix), 10, 0)
	if err != nil {
		return fmt.Errorf("cannot parse the start binlog name [%s], error: %w", startBinlogInfo.FileName, err)
	}

	binlogFilesLocal, err := ioutil.ReadDir(binlogDir)
	if err != nil {
		return fmt.Errorf("cannot read directory %s, error %w", binlogDir, err)
	}

	var needReplay []string
	for _, f := range binlogFilesLocal {
		if f.IsDir() || !strings.HasPrefix(f.Name(), binlogNamePrefix) {
			continue
		}
		// for mysql binlog, after the serial number reaches 999999, the next serial number will not return to 000000, but 1000000,
		// so we cannot directly use string to compare lexicographical order.
		binlogSeq, err := strconv.ParseInt(strings.TrimPrefix(f.Name(), binlogNamePrefix), 10, 0)
		if err != nil {
			return fmt.Errorf("cannot parse the binlog name [%s], error: %w", f.Name(), err)
		}
		if binlogSeq >= startBinlogSeq {
			needReplay = append(needReplay, f.Name())
		}
	}
	sort.Strings(needReplay)

	stopTime := time.Unix(targetTs, 0)
	args := []string{
		fmt.Sprintf(`--rewrite-db="%s->%s"`, originDatabase, pitrDatabase),
		fmt.Sprintf("--database=%s", pitrDatabase),
		"--disable-log-bin",
		fmt.Sprintf("--start-position=%d", startBinlogInfo.Position),
		fmt.Sprintf(`--stop-datetime="%d-%d-%d %d:%d:%d"`, stopTime.Year(), stopTime.Month(), stopTime.Day(), stopTime.Hour(), stopTime.Minute(), stopTime.Second()),
	}

	for _, name := range needReplay {
		args = append(args, filepath.Join(binlogDir, name))
	}

	var stdout bytes.Buffer
	cmd := exec.Command(r.mysqlbinlog.GetPath(), args...)
	cmd.Stdout = &stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("cannot run %s, error: %w", cmd.String(), err)
	}

	stmt := stdout.String()
	if _, err := db.ExecContext(ctx, stmt); err != nil {
		return fmt.Errorf("cannot apply stmt %s to database %s, error: %w", stmt, pitrDatabase, err)
	}

	return nil
}

// RestorePITR is a wrapper for restore a full backup and a range of incremental backup.
// It performs the step 1 of the restore process.
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

	_ = r.ReplayBinlog(ctx, database, pitrDatabaseName, "", binlog, suffixTs)

	return nil
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

// getBinlogNamePrefix returns the prefix of binlog file name.
func (r *Restore) getBinlogNamePrefix(ctx context.Context) (string, error) {
	db, err := r.driver.GetDbConnection(ctx, "")
	if err != nil {
		return "", err
	}

	rows, err := db.QueryContext(ctx, `SHOW VARIABLES LIKE "log_bin_basename";`)
	if err != nil {
		return "", fmt.Errorf("cannot query log_bin_basename on mysql server, error: %w", err)
	}
	defer rows.Close()

	var basename string
	if rows.Next() {
		var unused interface{}
		if err := rows.Scan(&unused /*Variable_name*/, &basename); err != nil {
			return "", err
		}
		return filepath.Base(basename) + ".", nil
	}
	return "", fmt.Errorf("cannot find log_bin_basename on mysqlbin server, error: %w", err)
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
