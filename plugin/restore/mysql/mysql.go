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

// Restore implements recovery functions for MySQL.
// For example, the original database is `dbfoo`, and the suffixTs is 1653018005 (derived from the PITR issue's CreateTs),
// then Bytebase will do the following:
// 1. Create a database called `dbfoo_pitr_1653018005`, and do PITR restore to it.
// 2. Create a database called `dbfoo_pitr_1653018005_old`, and move tables
// 	  from `dbfoo` to `dbfoo_pitr_1653018005_old`, and table from `dbfoo_pitr_1653018005` to `dbfoo`.
// 3. Delete database `dbfoo_pitr_1653018005_old` and `dbfoo_pitr_1653018005`.
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

	// TODO(dragonly): implement RestoreBinlog in mysql driver
	_ = r.RestoreBinlog(ctx, binlog)

	return nil
}

// SwapPITRDatabase renames the pitr database to the target, and the original to the old database
// It returns the pitr and old database names after swap.
// It performs the step 2 of the restore process.
// TODO(dragonly): handle the case that the original database does not exist
func (r *Restore) SwapPITRDatabase(ctx context.Context, database string, suffixTs int64) (pitrDatabaseName string, pitrOldDatabase string, err error) {
	pitrDatabaseName = getPITRDatabaseName(database, suffixTs)
	pitrOldDatabase = getPITROldDatabaseName(database, suffixTs)

	db, err := r.driver.GetDbConnection(ctx, "")
	if err != nil {
		return
	}
	txn, err := db.BeginTx(ctx, nil)
	if err != nil {
		return
	}
	defer txn.Rollback()

	tables, err := mysql.GetTables(txn, database)
	if err != nil {
		err = fmt.Errorf("failed to get tables of database %q, error[%w]", database, err)
		return
	}
	tablesPITR, err := mysql.GetTables(txn, pitrDatabaseName)
	if err != nil {
		err = fmt.Errorf("failed to get tables of database %q, error[%w]", pitrDatabaseName, err)
		return
	}

	if _, err = txn.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE `%s`", pitrOldDatabase)); err != nil {
		return
	}

	var tableRenames []string
	for _, table := range tables {
		tableRenames = append(tableRenames, fmt.Sprintf("`%s`.`%s` TO `%s`.`%s`", database, table.Name, pitrOldDatabase, table.Name))
	}
	for _, table := range tablesPITR {
		tableRenames = append(tableRenames, fmt.Sprintf("`%s`.`%s` TO `%s`.`%s`", pitrDatabaseName, table.Name, database, table.Name))
	}
	renameStmt := fmt.Sprintf("RENAME TABLE %s;", strings.Join(tableRenames, ", "))

	if _, err = txn.ExecContext(ctx, renameStmt); err != nil {
		return
	}

	if err = txn.Commit(); err != nil {
		return
	}

	return pitrDatabaseName, pitrOldDatabase, nil
}

// DeletePITRDatabases deletes the temporary pitr databases after the PITR swap task.
// It performs the step 3 of the restore process.
func (r *Restore) DeletePITRDatabases(ctx context.Context, database string, suffixTs int64) error {
	db, err := r.driver.GetDbConnection(ctx, "")
	if err != nil {
		return err
	}

	// Since the old database contains user data, we do not automatically drop it.
	// We only drop the empty ephemeral pitr database.
	txn, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer txn.Rollback()
	tables, err := mysql.GetTables(txn, database)
	if err != nil {
		return fmt.Errorf("failed to get tables of database %q, error[%w]", database, err)
	}
	if len(tables) != 0 {
		return fmt.Errorf("the PITR database must be empty, but has tables[%v]", tables)
	}

	pitrDatabaseName := getPITRDatabaseName(database, suffixTs)
	if _, err := db.ExecContext(ctx, fmt.Sprintf("DROP DATABASE `%s`;", pitrDatabaseName)); err != nil {
		return err
	}

	return nil
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

func getSafeName(baseName, suffix string) string {
	name := fmt.Sprintf("%s_%s", baseName, suffix)
	if len(name) <= MaxDatabaseNameLength {
		return name
	}
	extraCharacters := len(name) - MaxDatabaseNameLength
	return fmt.Sprintf("%s_%s", baseName[0:len(baseName)-extraCharacters], suffix)
}
