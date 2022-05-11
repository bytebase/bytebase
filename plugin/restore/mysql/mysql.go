package mysql

import (
	"bufio"
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/plugin/db/mysql"
)

// BinlogConfig is the binlog coordination for MySQL.
type BinlogConfig struct {
	Filename string
	Position int64
}

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
func (r *Restore) RestoreBinlog(ctx context.Context, config BinlogConfig) error {
	return fmt.Errorf("Unimplemented")
}

// RestorePITR is a wrapper for restore a full backup and a range of incremental backup
func (r *Restore) RestorePITR(ctx context.Context, fullBackup *bufio.Scanner, config BinlogConfig, database string, timestamp int64) error {
	pitrDatabaseName := getPITRDatabaseName(database, timestamp)
	query := fmt.Sprintf("CREATE DATABASE `%s`; USE `%s`; SET foreign_key_checks=OFF", pitrDatabaseName, pitrDatabaseName)
	if _, err := r.driver.DB.ExecContext(ctx, query); err != nil {
		return err
	}

	if err := r.driver.Restore(ctx, fullBackup); err != nil {
		return err
	}

	// TODO(dragonly): implement RestoreBinlog in mysql driver
	_ = r.RestoreBinlog(ctx, config)

	if _, err := r.driver.DB.ExecContext(ctx, fmt.Sprintf("SET foreign_key_checks=ON; USE `%s`", database)); err != nil {
		return err
	}

	return nil
}

// SwapPITRDatabase renames the pitr database to the target, and the original to the old database
func (r *Restore) SwapPITRDatabase(ctx context.Context, database string, timestamp int64) error {
	txn, err := r.driver.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer txn.Rollback()

	pitrOldDatabase := getPITRDatabaseOldName(database)
	pitrDatabaseName := getPITRDatabaseName(database, timestamp)

	if _, err := txn.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE `%s`", pitrOldDatabase)); err != nil {
		return err
	}

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

// getPITRDatabaseOldName composes a pitr database name for
func getPITRDatabaseOldName(database string) string {
	suffix := "old"
	return getSafeName(database, suffix)
}

func getSafeName(baseName, suffix string) string {
	name := fmt.Sprintf("%s_%s", baseName, suffix)
	if len(name) <= mysql.MaxDatabaseNameLength {
		return name
	}
	extraCharacters := len(name) - mysql.MaxDatabaseNameLength
	return fmt.Sprintf("%s_%s", baseName[0:len(baseName)-extraCharacters], suffix)
}
