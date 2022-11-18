package tests

import (
	"context"
	"fmt"
	"math"
	"strconv"
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
	database := "funny\ndatabase"

	_, stopFn := resourcemysql.SetupTestInstance(t, port)
	defer stopFn()

	db, err := connectTestMySQL(port, "")
	a.NoError(err)
	defer db.Close()
	_, err = db.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE `%s`; USE `%s`; CREATE TABLE user (id INT PRIMARY KEY, name VARCHAR(64), balance INT);", database, database))
	a.NoError(err)
	_, err = db.ExecContext(ctx, "INSERT INTO user VALUES (1, 'alice\nalice', 100), (2, 'bob', 100), (3, 'cindy', 100);")
	a.NoError(err)
	_, err = db.ExecContext(ctx, "UPDATE user SET balance=90 WHERE id=1; UPDATE user SET balance=110 WHERE id=2; DELETE FROM user WHERE id=3;")
	a.NoError(err)

	resourceDir := t.TempDir()
	err = mysqlutil.Install(resourceDir)
	a.NoError(err)
	driver, err := getTestMySQLDriver(ctx, t, strconv.Itoa(port), database, resourceDir)
	a.NoError(err)
	defer driver.Close(ctx)

	// Rotate to binlog.000002 so that it's easy to rollback the following transactions and check that the state is the same as now.
	_, err = db.ExecContext(ctx, "FLUSH BINARY LOGS;")
	a.NoError(err)
	err = driver.Execute(ctx, "UPDATE user SET balance=0;", false)
	a.NoError(err)
	err = driver.Execute(ctx, "DELETE FROM user;", false)
	a.NoError(err)

	// Restore data using generated rollback SQL.
	mysqlDriver, ok := driver.(*mysql.Driver)
	a.Equal(true, ok)
	binlogFileList := []string{"binlog.000002"}
	tableCatalog := map[string][]string{
		"user": {"id", "name", "balance"},
	}
	threadID, err := mysqlDriver.GetMigrationConnID(ctx)
	a.NoError(err)
	rollbackSQL, err := mysqlDriver.GenerateRollbackSQL(ctx, binlogFileList, 0, math.MaxInt64, threadID, tableCatalog)
	a.NoError(err)
	_, err = db.ExecContext(ctx, rollbackSQL)
	a.NoError(err)

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
