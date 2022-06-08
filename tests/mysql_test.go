package tests

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	pluginmysql "github.com/bytebase/bytebase/plugin/db/mysql"
	restoremysql "github.com/bytebase/bytebase/plugin/restore/mysql"
	resourcemysql "github.com/bytebase/bytebase/resources/mysql"
	"github.com/stretchr/testify/require"
)

func TestCheckEngineInnoDB(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()

	port := getTestPort(t.Name())

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

		connCfg := getMySQLConnectionConfig(strconv.Itoa(port), database)

		mysqlDriver, ok := driver.(*pluginmysql.Driver)
		a.Equal(true, ok)
		mysqlRestore := restoremysql.New(mysqlDriver, nil, connCfg, "" /*binlog directory*/)
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

		connCfg := getMySQLConnectionConfig(strconv.Itoa(port), database)

		mysqlDriver, ok := driver.(*pluginmysql.Driver)
		a.Equal(true, ok)
		mysqlRestore := restoremysql.New(mysqlDriver, nil, connCfg, "" /*binlog directory*/)

		err = mysqlRestore.CheckEngineInnoDB(ctx, database)
		a.Error(err)
	})
}

func TestCheckServerVersionAndBinlogForPITR(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()

	port := getTestPort(t.Name())
	_, stopFn := resourcemysql.SetupTestInstance(t, port)
	defer stopFn()

	db, err := connectTestMySQL(port, "")
	a.NoError(err)
	defer db.Close()

	driver, err := getTestMySQLDriver(ctx, strconv.Itoa(port), "")
	a.NoError(err)
	defer driver.Close(ctx)

	connCfg := getMySQLConnectionConfig(strconv.Itoa(port), "")
	mysqlDriver, ok := driver.(*pluginmysql.Driver)
	a.Equal(true, ok)
	mysqlRestore := restoremysql.New(mysqlDriver, nil, connCfg, "" /*binlog directory*/)

	// the test MySQL instance is 8.0.
	err = mysqlRestore.CheckServerVersionForPITR(ctx)
	a.NoError(err)

	// binlog is default to ON in MySQL 8.0.
	err = mysqlRestore.CheckBinlogEnabled(ctx)
	a.NoError(err)

	// binlog format is default to ROW in MySQL 8.0.
	err = mysqlRestore.CheckBinlogRowFormat(ctx)
	a.NoError(err)
}
