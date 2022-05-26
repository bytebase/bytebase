package tests

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	pluginmysql "github.com/bytebase/bytebase/plugin/db/mysql"
	restoremysql "github.com/bytebase/bytebase/plugin/restore/mysql"
	resourcemysql "github.com/bytebase/bytebase/resources/mysql"
	"github.com/bytebase/bytebase/resources/mysqlbinlog"
	"github.com/stretchr/testify/require"
)

func TestCheckEngineInnoDB(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()

	port := getTestPort(t.Name())

	t.Log("install mysqlbinlog binary")
	tmpDir := t.TempDir()
	mysqlbinlogIns, err := mysqlbinlog.Install(tmpDir)
	a.NoError(err)

	t.Run("success", func(t *testing.T) {
		port := port
		t.Parallel()
		_, stopFn := resourcemysql.SetupTestInstance(t, port)
		defer stopFn()
		t.Log("create test database")
		database := "test_success"
		db, err := connectTestMySQL(port, "")
		a.NoError(err)
		defer db.Close()
		_, err = db.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE `%s`; USE `%s`;", database, database))
		a.NoError(err)

		t.Log("create table with InnoDB engine")
		_, err = db.ExecContext(ctx, "CREATE TABLE t_innodb (id INT PRIMARY KEY) ENGINE=InnoDB;")
		a.NoError(err)

		t.Log("check InnoDB engine")
		driver, err := getTestMySQLDriver(ctx, strconv.Itoa(port), database)
		a.NoError(err)
		defer driver.Close(ctx)
		mysqlDriver, ok := driver.(*pluginmysql.Driver)
		a.Equal(true, ok)
		mysqlRestore := restoremysql.New(mysqlDriver, mysqlbinlogIns)
		err = mysqlRestore.CheckEngineInnoDB(ctx, database)
		a.NoError(err)
	})

	t.Run("fail", func(t *testing.T) {
		port := port + 1
		t.Parallel()
		_, stopFn := resourcemysql.SetupTestInstance(t, port)
		defer stopFn()
		t.Log("create test database")
		database := "test_fail"
		db, err := connectTestMySQL(port, "")
		a.NoError(err)
		defer db.Close()
		_, err = db.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE `%s`; USE `%s`;", database, database))
		a.NoError(err)

		t.Log("create table with InnoDB engine")
		_, err = db.ExecContext(ctx, "CREATE TABLE t_innodb (id INT PRIMARY KEY) ENGINE=InnoDB;")
		a.NoError(err)
		t.Log("create table with MyISAM engine")
		_, err = db.ExecContext(ctx, "CREATE TABLE t_myisam (id INT PRIMARY KEY) ENGINE=MyISAM;")
		a.NoError(err)

		t.Log("check InnoDB engine")
		driver, err := getTestMySQLDriver(ctx, strconv.Itoa(port), database)
		a.NoError(err)
		defer driver.Close(ctx)
		mysqlDriver, ok := driver.(*pluginmysql.Driver)
		a.Equal(true, ok)
		mysqlRestore := restoremysql.New(mysqlDriver, mysqlbinlogIns)
		err = mysqlRestore.CheckEngineInnoDB(ctx, database)
		a.Error(err)
	})
}
