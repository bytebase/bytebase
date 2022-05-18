package mysql

import (
	"bufio"
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/plugin/db/mysql"
)

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
func (r *Restore) RestorePITR(ctx context.Context, fullBackup *bufio.Scanner, binlog mysql.BinlogInfo, database string, timestamp int64) error {
	pitrDatabaseName := getPITRDatabaseName(database, timestamp)
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

func getSafeName(baseName, suffix string) string {
	name := fmt.Sprintf("%s_%s", baseName, suffix)
	if len(name) <= MaxDatabaseNameLength {
		return name
	}
	extraCharacters := len(name) - MaxDatabaseNameLength
	return fmt.Sprintf("%s_%s", baseName[0:len(baseName)-extraCharacters], suffix)
}
