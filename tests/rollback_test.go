package tests

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/plugin/db/mysql"
	resourcemysql "github.com/bytebase/bytebase/resources/mysql"
	"github.com/bytebase/bytebase/resources/mysqlutil"
)

func TestRollback(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	port := getTestPort(t.Name())

	instance, stopFn := resourcemysql.SetupTestInstance(t, port)
	binlogDir := instance.DataDir()
	defer stopFn()

	resourceDir := t.TempDir()
	err := mysqlutil.Install(resourceDir)
	a.NoError(err)

	database := "funny\ndatabase"
	db, err := connectTestMySQL(port, "")
	a.NoError(err)
	defer db.Close()
	_, err = db.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE `%s`; USE `%s`;", database, database))
	a.NoError(err)
	_, err = db.ExecContext(ctx, "CREATE TABLE user (id INT PRIMARY KEY, name VARCHAR(64), balance INT);")
	a.NoError(err)
	_, err = db.ExecContext(ctx, "INSERT INTO user VALUES (1, 'alice\nalice', 100), (2, 'bob', 100), (3, 'cindy', 100);")
	a.NoError(err)

	txn, err := db.BeginTx(ctx, nil)
	a.NoError(err)
	_, err = txn.ExecContext(ctx, "UPDATE user SET balance=90 WHERE id=1;")
	a.NoError(err)
	_, err = txn.ExecContext(ctx, "UPDATE user SET balance=110 WHERE id=2;")
	a.NoError(err)
	_, err = txn.ExecContext(ctx, "DELETE FROM user WHERE id=3;")
	a.NoError(err)
	err = txn.Commit()
	a.NoError(err)

	// Rotate to binlog.000002 so that it's easy to rollback the following transactions and check that the state is the same as now.
	_, err = db.ExecContext(ctx, "FLUSH BINARY LOGS;")
	a.NoError(err)
	_, err = db.ExecContext(ctx, "UPDATE user SET balance=0;")
	a.NoError(err)
	_, err = db.ExecContext(ctx, "DELETE FROM user;")
	a.NoError(err)

	t.Log("Set up mysqlbinlog")
	args := []string{
		path.Join(binlogDir, "binlog.000002"),
		"--verify-binlog-checksum",
		"-v",
		"--base64-output=DECODE-ROWS",
	}
	cmd := exec.CommandContext(ctx, mysqlutil.GetPath(mysqlutil.MySQLBinlog, resourceDir), args...)
	cmd.Stderr = os.Stderr
	pr, err := cmd.StdoutPipe()
	a.NoError(err)
	err = cmd.Start()
	a.NoError(err)
	defer func() {
		err = pr.Close()
		a.NoError(err)
		err = cmd.Process.Kill()
		a.NoError(err)
	}()

	txList, err := mysql.ParseBinlogStream(pr)
	a.NoError(err)
	a.Equal(2, len(txList))
	var rollbackSQLList []string
	tableMap := map[string][]string{"user": {"id", "name", "balance"}}
	for _, tx := range txList {
		sql, err := tx.GetRollbackSQL(tableMap)
		a.NoError(err)
		rollbackSQLList = append(rollbackSQLList, sql)
	}

	// Execute the rollback SQL in reversed order.
	for i := len(rollbackSQLList) - 1; i >= 0; i-- {
		sql := rollbackSQLList[i]
		t.Log(sql)
		_, err = db.ExecContext(ctx, sql)
		a.NoError(err)
	}

	// Check for rollback state.
	rows, err := db.QueryContext(ctx, "SELECT * FROM user;")
	a.NoError(err)
	type record struct {
		id      int
		name    string
		balance int
	}
	var records []record
	for rows.Next() {
		var r record
		err = rows.Scan(&r.id, &r.name, &r.balance)
		a.NoError(err)
		records = append(records, r)
	}
	want := []record{
		{1, "alice\nalice", 90},
		{2, "bob", 110},
	}
	a.Equal(want, records)
}
