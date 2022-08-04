package mysql

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/common/log"
	"github.com/bytebase/bytebase/plugin/db/util"
	"github.com/bytebase/bytebase/resources/mysqlutil"
	"github.com/pkg/errors"
)

// Dump and restore.
const (
	tableStmtFmt = "" +
		"--\n" +
		"-- Table structure for `%s`\n" +
		"--\n" +
		"%s;\n"
	viewStmtFmt = "" +
		"--\n" +
		"-- View structure for `%s`\n" +
		"--\n" +
		"%s;\n"
)

// Dump dumps the database.
func (driver *Driver) Dump(ctx context.Context, database string, out io.Writer, schemaOnly bool) (string, error) {
	// mysqldump -u root --databases dbName --no-data --routines --events --triggers --compact

	var err error
	// We must use the same MySQL connection to lock and unlock tables.
	conn, err := driver.db.Conn(ctx)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	mysqldumpArgs := []string{
		// Dump stored routines (procedures and functions) from dumped databases
		"--routines",
		// Dump events from dumped databases
		"--events",
		// Dump triggers for each dumped table
		"--triggers",
		// Produce more compact output
		"--compact",
		"--databases", database,
		"--host", driver.connCfg.Host,
		"--user", driver.connCfg.Username,
	}
	if driver.connCfg.Port != "" {
		mysqldumpArgs = append(mysqldumpArgs, "--port", driver.connCfg.Port)
	}
	if driver.connCfg.Password != "" {
		mysqldumpArgs = append(mysqldumpArgs, fmt.Sprintf("--password=%s", driver.connCfg.Password))
	}

	var payloadBytes []byte
	// Before we dump the real data, we should record the binlog position for PITR.
	// Please refer to https://github.com/bytebase/bytebase/blob/main/docs/design/pitr-mysql.md#full-backup for details.
	if !schemaOnly {
		log.Debug("flush tables in database with read locks",
			zap.String("database", database))
		if err := flushTablesWithReadLock(ctx, conn, database); err != nil {
			log.Error("flush tables failed", zap.Error(err))
			return "", err
		}
		defer func() {
			// Errors in unlocking can have serious consequences, we need detailed error information.
			if defererr := unlockTables(ctx, conn); defererr != nil {
				if err != nil {
					err = errors.Wrap(err, defererr.Error())
				} else {
					err = defererr
				}
			}
		}()

		binlog, err := GetBinlogInfo(ctx, conn)
		if err != nil {
			return "", err
		}
		log.Debug("binlog coordinate at dump time",
			zap.String("fileName", binlog.FileName),
			zap.Int64("position", binlog.Position))

		payload := api.BackupPayload{BinlogInfo: binlog}
		payloadBytes, err = json.Marshal(payload)
		if err != nil {
			return "", err
		}
	} else {
		mysqldumpArgs = append(mysqldumpArgs, "--no-data")
	}

	options := sql.TxOptions{}
	// TiDB does not support readonly, so we only set for MySQL.
	if driver.dbType == "MYSQL" {
		options.ReadOnly = true
	}
	// If `schemaOnly` is false, now we are still holding the tables' exclusive locks.
	// Beginning a transaction in the same session will implicitly release existing table locks.
	// ref: https://dev.mysql.com/doc/refman/8.0/en/lock-tables.html, section "Interaction of Table Locking and Transactions".
	mysqldumpCmd := exec.Command(mysqlutil.GetPath(mysqlutil.MySQLDump, driver.resourceDir), mysqldumpArgs...)
	mysqldumpCmd.Stdout = out
	var stderr bytes.Buffer
	mysqldumpCmd.Stderr = &stderr
	if err := mysqldumpCmd.Run(); err != nil {
		return "", fmt.Errorf("mysqldump command failed, error: %q", stderr.String())
	}

	return string(payloadBytes), err
}

// unlockTables runs `UNLOCK TABLES`.
func unlockTables(ctx context.Context, conn *sql.Conn) error {
	if _, err := conn.ExecContext(ctx, "UNLOCK TABLES;"); err != nil {
		return err
	}
	return nil
}

// flushTablesWithReadLock runs FLUSH TABLES table1, table2, ... WITH READ LOCK for all the tables in the database.
func flushTablesWithReadLock(ctx context.Context, conn *sql.Conn, database string) error {
	// The lock acquiring could take a long time if there are concurrent exclusive locks on the tables.
	// We ensures that the execution is canceled after 30 seconds, otherwise we may get dead lock and stuck forever.
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	txn, err := conn.BeginTx(ctxWithTimeout, nil)
	if err != nil {
		return err
	}
	defer txn.Rollback()

	tables, err := getTablesTx(txn, database)
	if err != nil {
		return err
	}

	var tableNames []string
	for _, table := range tables {
		if table.TableType != baseTableType {
			continue
		}
		tableNames = append(tableNames, fmt.Sprintf("`%s`", table.Name))
	}
	flushTableStmt := fmt.Sprintf("FLUSH TABLES %s WITH READ LOCK;", strings.Join(tableNames, ", "))

	if _, err := txn.ExecContext(ctxWithTimeout, flushTableStmt); err != nil {
		return err
	}

	if err := txn.Commit(); err != nil {
		return err
	}

	return nil
}

// GetBinlogInfo queries current binlog info from MySQL server.
func GetBinlogInfo(ctx context.Context, conn *sql.Conn) (api.BinlogInfo, error) {
	query := "SHOW MASTER STATUS"
	binlogInfo := api.BinlogInfo{}
	var unused interface{}
	if err := conn.QueryRowContext(ctx, query).Scan(
		&binlogInfo.FileName,
		&binlogInfo.Position,
		&unused,
		&unused,
		&unused,
	); err != nil {
		if err == sql.ErrNoRows {
			return api.BinlogInfo{}, common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return api.BinlogInfo{}, err
	}
	return binlogInfo, nil
}

// TableSchema describes the schema of a table or view.
type TableSchema struct {
	Name      string
	TableType string
	Statement string
}

// getTablesTx gets all tables of a database using the provided transaction.
func getTablesTx(txn *sql.Tx, dbName string) ([]*TableSchema, error) {
	return getTablesImpl(txn, dbName)
}

// getTables gets all tables of a database.
func getTables(ctx context.Context, conn *sql.Conn, dbName string) ([]*TableSchema, error) {
	txn, err := conn.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer txn.Rollback()
	return getTablesImpl(txn, dbName)
}

func getTablesImpl(txn *sql.Tx, dbName string) ([]*TableSchema, error) {
	var tables []*TableSchema
	query := fmt.Sprintf("SHOW FULL TABLES FROM `%s`;", dbName)
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var tbl TableSchema
		if err := rows.Scan(&tbl.Name, &tbl.TableType); err != nil {
			return nil, err
		}
		tables = append(tables, &tbl)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, tbl := range tables {
		stmt, err := getTableStmt(txn, dbName, tbl.Name, tbl.TableType)
		if err != nil {
			return nil, fmt.Errorf("getTableStmt(%q, %q, %q) got error: %s", dbName, tbl.Name, tbl.TableType, err)
		}
		tbl.Statement = stmt
	}
	return tables, nil
}

// getTableStmt gets the create statement of a table.
func getTableStmt(txn *sql.Tx, dbName, tblName, tblType string) (string, error) {
	switch tblType {
	case baseTableType:
		query := fmt.Sprintf("SHOW CREATE TABLE `%s`.`%s`;", dbName, tblName)
		var stmt, unused string
		if err := txn.QueryRow(query).Scan(&unused, &stmt); err != nil {
			if err == sql.ErrNoRows {
				return "", common.FormatDBErrorEmptyRowWithQuery(query)
			}
			return "", err
		}
		return fmt.Sprintf(tableStmtFmt, tblName, stmt), nil
	case viewTableType:
		// This differs from mysqldump as it includes.
		query := fmt.Sprintf("SHOW CREATE VIEW `%s`.`%s`;", dbName, tblName)
		var createStmt, unused string
		if err := txn.QueryRow(query).Scan(&unused, &createStmt, &unused, &unused); err != nil {
			if err == sql.ErrNoRows {
				return "", common.FormatDBErrorEmptyRowWithQuery(query)
			}
			return "", err
		}
		return fmt.Sprintf(viewStmtFmt, tblName, createStmt), nil
	default:
		return "", fmt.Errorf("unrecognized table type %q for database %q table %q", tblType, dbName, tblName)
	}
}

// Restore restores a database.
func (driver *Driver) Restore(ctx context.Context, sc *bufio.Scanner) (err error) {
	txn, err := driver.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer txn.Rollback()

	if err := restoreTx(ctx, txn, sc); err != nil {
		return err
	}

	if err := txn.Commit(); err != nil {
		return err
	}

	return nil
}

// RestoreTx restores a database in the given transaction.
func (*Driver) RestoreTx(ctx context.Context, tx *sql.Tx, sc *bufio.Scanner) error {
	return restoreTx(ctx, tx, sc)
}

func restoreTx(ctx context.Context, tx *sql.Tx, sc *bufio.Scanner) error {
	fnExecuteStmt := func(stmt string) error {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return err
		}
		return nil
	}

	if err := util.ApplyMultiStatements(sc, fnExecuteStmt); err != nil {
		return err
	}
	return nil
}
