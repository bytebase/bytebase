//go:build mysql
// +build mysql

package tests

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/bytebase/bytebase/api"
	dbplugin "github.com/bytebase/bytebase/plugin/db"
	pluginmysql "github.com/bytebase/bytebase/plugin/db/mysql"
	restoremysql "github.com/bytebase/bytebase/plugin/restore/mysql"
	resourcemysql "github.com/bytebase/bytebase/resources/mysql"
	"github.com/bytebase/bytebase/resources/mysqlutil"

	"github.com/stretchr/testify/require"
)

// TestBackupRestoreBasic tests basic backup and restore behavior
// The test plan is:
// TODO(dragonly): add routine/event/trigger
// 1. create schema with index and constraint and populate data
// 2. create a full backup
// 3. clear data
// 4. restore data
// 5. validate
func TestBackupRestoreBasic(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()

	port := getTestPort(t.Name())
	database := "backup_restore"
	table := "backup_restore"

	_, stop := resourcemysql.SetupTestInstance(t, port)
	defer stop()

	db, err := connectTestMySQL(port, "")
	a.NoError(err)
	defer db.Close()

	_, err = db.Exec(fmt.Sprintf(`
	CREATE DATABASE %s;
	USE %s;
	CREATE TABLE %s (
		id INT,
		PRIMARY KEY (id),
		CHECK (id >= 0)
	);
	`, database, database, table))
	a.NoError(err)

	const numRecords = 10
	tx, err := db.Begin()
	a.NoError(err)
	for i := 0; i < numRecords; i++ {
		_, err = tx.Exec(fmt.Sprintf("INSERT INTO %s VALUES (%d)", table, i))
		a.NoError(err)
	}
	err = tx.Commit()
	a.NoError(err)

	// make a full backup
	driver, err := getTestMySQLDriver(ctx, strconv.Itoa(port), database)
	a.NoError(err)
	defer driver.Close(ctx)

	buf, err := doBackup(ctx, driver, database)
	a.NoError(err)
	t.Logf("backup content:\n%s", buf.String())

	// drop all tables
	_, err = db.Exec(fmt.Sprintf("DROP TABLE %s", table))
	a.NoError(err)

	// restore
	err = driver.Restore(ctx, bufio.NewScanner(buf))
	a.NoError(err)

	// validate data
	rows, err := db.Query(fmt.Sprintf("SELECT * FROM %s ORDER BY id ASC", table))
	a.NoError(err)
	i := 0
	for rows.Next() {
		var col int
		a.NoError(rows.Scan(&col))
		a.Equal(i, col)
		i++
	}
	a.NoError(rows.Err())
	a.Equal(numRecords, i)
}

// TestPITR tests the PITR behavior
// The test plan is:
// 0. prepare tables with foreign key constraints dependencies
// 1. insert data and make a full backup (denoted as t0), which defines the checkpoint
// 2. insert more data, and this is the point-in-time (denoted as t1) that we want to recover
// 3. keep inserting data into the original tables
// 4.1. set foreign_key_checks=OFF
// 4.2. restore full backup at t0 to pitr tables
// 4.3. apply binlog from t0 to t1 to pitr tables
// 4.4. foreign_key_checks=ON
// 5. lock tables and atomically swap original and pitr tables
func TestPITR(t *testing.T) {
	t.Parallel()
	a := require.New(t)

	// common configs
	const (
		database     = "backup_restore"
		numRowsTime0 = 10
		numRowsTime1 = 20
	)
	port := getTestPort(t.Name())

	t.Log("install mysqlbinlog binary")
	tmpDir := t.TempDir()
	mysqlutilInstance, err := mysqlutil.Install(tmpDir)
	a.NoError(err)

	// test cases
	t.Run("Buggy Application", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()

		t.Log("initialize database")
		// For parallel sub-tests, we use different port for MySQL
		mysqlPort := port
		db, stopFn := initPITRDB(t, database, mysqlPort)
		defer stopFn()
		defer db.Close()

		t.Log("insert data")
		insertRangeData(t, db, 0, numRowsTime0)

		t.Log("make a full backup")
		driver, err := getTestMySQLDriver(ctx, strconv.Itoa(mysqlPort), database)
		a.NoError(err)
		defer driver.Close(ctx)

		connCfg := getMySQLConnectionConfig(strconv.Itoa(mysqlPort), database)

		buf, err := doBackup(ctx, driver, database)
		a.NoError(err)
		t.Logf("backup content:\n%s", buf.String())

		t.Log("insert more data")
		insertRangeData(t, db, numRowsTime0, numRowsTime1)

		ctxUpdateRow, cancelUpdateRow := context.WithCancel(ctx)
		t1 := startUpdateRow(ctxUpdateRow, t, database, mysqlPort)
		t.Logf("start to concurrently update data at t1: %v", t1)

		t.Log("restore to pitr database")
		createPITRIssueTimestamp := time.Now().Unix()
		mysqlDriver, ok := driver.(*pluginmysql.Driver)
		a.Equal(true, ok)
		mysqlRestore := restoremysql.New(mysqlDriver, mysqlutilInstance, connCfg)
		binlogInfo := api.BinlogInfo{}
		binlogSaveDir := t.TempDir()
		err = mysqlRestore.RestorePITR(ctx, bufio.NewScanner(buf), binlogInfo, database, createPITRIssueTimestamp, binlogSaveDir)
		a.NoError(err)

		t.Log("cutover stage")
		cancelUpdateRow()
		// We mimics the situation where the user waits for the target database idle before doing the cutover.
		time.Sleep(time.Second)

		_, _, err = mysqlRestore.SwapPITRDatabase(ctx, database, createPITRIssueTimestamp)
		a.NoError(err)

		t.Log("validate table tbl0")
		// TODO(dragonly): change to numRowsTime1 when RestoreIncremental is implemented
		validateTbl0(t, db, numRowsTime0)
		t.Log("validate table tbl1")
		validateTbl1(t, db, numRowsTime0)
		// TODO(dragonly): validate table _update_row_ when RestoreIncremental is implemented
		t.Log("validate table _update_row_")
	})

	t.Run("Drop Database", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()

		t.Logf("test %s initialize database %s", t.Name(), database)

		// 1. create database for PITR test
		// For parallel sub-tests, we use different port for MySQL
		mysqlPort := port + 1
		db, stopFn := initPITRDB(t, database, mysqlPort)
		defer stopFn()
		defer db.Close()

		// 2. insert data for full backup
		t.Log("insert data")
		insertRangeData(t, db, 0, numRowsTime0)

		t.Log("make a full backup")
		driver, err := getTestMySQLDriver(ctx, strconv.Itoa(mysqlPort), database)
		a.NoError(err)
		defer driver.Close(ctx)

		connCfg := getMySQLConnectionConfig(strconv.Itoa(mysqlPort), database)

		buf, err := doBackup(ctx, driver, database)
		a.NoError(err)
		t.Logf("backup content:\n%s", buf.String())

		// 3. insert more data for incremental restore
		t.Log("insert more data")
		insertRangeData(t, db, numRowsTime0, numRowsTime1)

		// 4. drop database
		dropStmt := fmt.Sprintf(`DROP DATABASE %s;`, database)
		_, err = db.ExecContext(ctx, dropStmt)
		a.NoError(err)

		// 5. check that query from the database that had dropped will fail
		rows, err := db.Query(fmt.Sprintf(`SHOW DATABASES LIKE '%s';`, database))
		a.NoError(err)
		defer rows.Close()
		for rows.Next() {
			var s string
			err := rows.Scan(&s)
			a.NoError(err)
			a.FailNow("Database still exists after dropped")
		}

		// 6. restore
		t.Log("restore to pitr database")
		createPITRIssueTimestamp := time.Now().Unix()
		mysqlDriver, ok := driver.(*pluginmysql.Driver)
		a.Equal(true, ok)
		mysqlRestore := restoremysql.New(mysqlDriver, mysqlutilInstance, connCfg)
		binlogInfo := api.BinlogInfo{}
		binlogSaveDir := t.TempDir()
		err = mysqlRestore.RestorePITR(ctx, bufio.NewScanner(buf), binlogInfo, database, createPITRIssueTimestamp, binlogSaveDir)
		a.NoError(err)

		t.Log("cutover stage")
		_, _, err = mysqlRestore.SwapPITRDatabase(ctx, database, createPITRIssueTimestamp)
		a.NoError(err)
	})

	t.Run("Schema Migration Failure", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()

		t.Logf("test %s initialize database %s", t.Name(), database)

		mysqlPort := port + 2
		db, stopFn := initPITRDB(t, database, mysqlPort)
		defer stopFn()
		defer db.Close()

		t.Log("insert data")
		insertRangeData(t, db, 0, numRowsTime0)

		t.Log("make a full backup")
		driver, err := getTestMySQLDriver(ctx, strconv.Itoa(mysqlPort), database)
		a.NoError(err)
		defer driver.Close(ctx)

		connCfg := getMySQLConnectionConfig(strconv.Itoa(mysqlPort), database)

		buf, err := doBackup(ctx, driver, database)
		a.NoError(err)
		t.Logf("backup content:\n%s\n", buf.String())

		t.Log("insert more data")
		insertRangeData(t, db, numRowsTime0, numRowsTime1)

		ctxUpdateRow, cancelUpdateRow := context.WithCancel(ctx)
		t1 := startUpdateRow(ctxUpdateRow, t, database, mysqlPort)
		t.Logf("start to concurrently update data at t1: %v", t1)

		createPITRIssueTimestamp := time.Now().Unix()

		t.Log("mimics schema migration")
		dropColumnStmt := `ALTER TABLE tbl1 DROP COLUMN id;`
		_, err = db.ExecContext(ctx, dropColumnStmt)
		a.NoError(err)

		t.Log("restore to pitr database")
		mysqlDriver, ok := driver.(*pluginmysql.Driver)
		a.Equal(true, ok)
		mysqlRestore := restoremysql.New(mysqlDriver, mysqlutilInstance, connCfg)
		binlogInfo := api.BinlogInfo{}
		binlogSaveDir := t.TempDir()
		err = mysqlRestore.RestorePITR(ctx, bufio.NewScanner(buf), binlogInfo, database, createPITRIssueTimestamp, binlogSaveDir)
		a.NoError(err)

		t.Log("cutover stage")
		cancelUpdateRow()
		// We mimics the situation where the user waits for the target database idle before doing the cutover.
		time.Sleep(time.Second)

		_, _, err = mysqlRestore.SwapPITRDatabase(ctx, database, createPITRIssueTimestamp)
		a.NoError(err)

		t.Log("validate table tbl0")
		// TODO(zp): change to numRowsTime1 when RestoreIncremental is implemented
		validateTbl0(t, db, numRowsTime0)
		t.Log("validate table tbl1")
		validateTbl1(t, db, numRowsTime0)
		// TODO(zp): validate table _update_row_ when RestoreIncremental is implemented
		t.Log("validate table _update_row_")
	})
}

func initPITRDB(t *testing.T, database string, port int) (*sql.DB, func()) {
	a := require.New(t)

	_, stopFn := resourcemysql.SetupTestInstance(t, port)

	db, err := connectTestMySQL(port, "")
	a.NoError(err)

	_, err = db.Exec(fmt.Sprintf(`
	CREATE DATABASE %s;
	USE %s;
	CREATE TABLE tbl0 (
		id INT,
		PRIMARY KEY (id),
		CHECK (id >= 0)
	);
	CREATE TABLE tbl1 (
		id INT,
		pid INT,
		PRIMARY KEY (id),
		UNIQUE INDEX (pid),
		CONSTRAINT FOREIGN KEY (pid) REFERENCES tbl0(id) ON DELETE NO ACTION
	);
	`, database, database))
	a.NoError(err)

	return db, stopFn
}

func insertRangeData(t *testing.T, db *sql.DB, begin, end int) {
	a := require.New(t)

	tx, err := db.Begin()
	a.NoError(err)

	for i := begin; i < end; i++ {
		_, err := tx.Exec(fmt.Sprintf("INSERT INTO tbl0 VALUES (%d);", i))
		a.NoError(err)
		_, err = tx.Exec(fmt.Sprintf("INSERT INTO tbl1 VALUES (%d, %d);", i, i))
		a.NoError(err)
	}

	err = tx.Commit()
	a.NoError(err)
}

func validateTbl0(t *testing.T, db *sql.DB, numRows int) {
	a := require.New(t)
	rows, err := db.Query("SELECT * FROM tbl0;")
	a.NoError(err)
	i := 0
	for rows.Next() {
		var col int
		a.NoError(rows.Scan(&col))
		a.Equal(i, col)
		i++
	}
	a.NoError(rows.Err())
	a.Equal(numRows, i)
}

func validateTbl1(t *testing.T, db *sql.DB, numRows int) {
	a := require.New(t)
	rows, err := db.Query("SELECT * FROM tbl1;")
	a.NoError(err)
	i := 0
	for rows.Next() {
		var col1, col2 int
		a.NoError(rows.Scan(&col1, &col2))
		a.Equal(i, col1)
		a.Equal(i, col2)
		i++
	}
	a.NoError(rows.Err())
	// TODO(dragonly): change to numRowsTime1 when RestoreIncremental is implemented
	a.Equal(numRows, i)
}

func doBackup(ctx context.Context, driver dbplugin.Driver, database string) (*bytes.Buffer, error) {
	var buf bytes.Buffer
	_, err := driver.Dump(ctx, database, &buf, false)
	return &buf, err
}

func startUpdateRow(ctx context.Context, t *testing.T, database string, port int) int64 {
	a := require.New(t)
	db, err := connectTestMySQL(port, database)
	a.NoError(err)

	t.Log("Start updating data")
	_, err = db.Exec("CREATE TABLE _update_row_ (id INT PRIMARY KEY);")
	a.NoError(err)

	// init value is (0)
	_, err = db.Exec("INSERT INTO _update_row_ VALUES (0);")
	a.NoError(err)
	initTimestamp := time.Now().Unix()

	go func() {
		defer db.Close()
		ticker := time.NewTicker(10 * time.Millisecond)
		i := 0
		for {
			select {
			case <-ticker.C:
				_, err = db.Exec(fmt.Sprintf("UPDATE _update_row_ SET id = %d;", i))
				a.NoError(err)
				i++
			case <-ctx.Done():
				t.Log("Stop updating data")
				return
			}
		}
	}()

	return initTimestamp
}
