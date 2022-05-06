//go:build mysql
// +build mysql

package tests

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	dbPlugin "github.com/bytebase/bytebase/plugin/db"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/resources/mysql"
	"github.com/stretchr/testify/require"
)

// TestBackupRestoreBasic tests basic backup and restore behavior
// The test plan is:
// 1. create schema with index and constraint and populate data (TODO(dragonly): add routine/event/trigger)
// 2. create a full backup
// 3. clear data
// 4. restore data
// 5. validate
func TestBackupRestoreBasic(t *testing.T) {
	t.Parallel()
	a := require.New(t)

	localhost := "127.0.0.1"
	port := getTestPort(t.Name())
	username := "root"
	database := "backup_restore"
	table := "backup_restore"

	_, stop := mysql.SetupTestInstance(t, port)
	defer stop()

	db, err := sql.Open("mysql", fmt.Sprintf("%s@tcp(%s:%d)/mysql", username, localhost, port))
	a.NoError(err)
	defer db.Close()

	_, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s", database))
	a.NoError(err)

	_, err = db.Exec(fmt.Sprintf("USE %s", database))
	a.NoError(err)

	_, err = db.Exec(fmt.Sprintf(`
	CREATE TABLE %s (
		id INT,
		PRIMARY KEY (id),
		CHECK (id >= 0)
	);
	`, table))
	a.NoError(err)

	const numRecords = 100
	tx, err := db.Begin()
	a.NoError(err)
	defer tx.Rollback()
	for i := 0; i < numRecords; i++ {
		_, err = tx.Exec(fmt.Sprintf("INSERT INTO %s VALUES (%d)", table, i))
		a.NoError(err)
	}
	err = tx.Commit()
	a.NoError(err)

	// make a full backup
	driver := getDbDriver(t, localhost, fmt.Sprintf("%d", port), username, database)
	defer func() {
		err := driver.Close(context.TODO())
		a.NoError(err)
	}()

	buf := doBackup(t, driver, database)
	// t.Logf("dump:\n%s", buf.String())

	// drop all tables
	_, err = db.Exec(fmt.Sprintf("DROP TABLE %s", table))
	a.NoError(err)

	// restore
	err = driver.Restore(context.TODO(), bufio.NewScanner(buf))
	a.NoError(err)

	// validate data
	rows, err := db.Query(fmt.Sprintf("SELECT * FROM %s ORDER BY id ASC", table))
	a.NoError(err)
	i := 0
	for rows.Next() {
		var col int
		err := rows.Scan(&col)
		a.NoError(err)
		a.Equal(col, i)
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
// 4.2. restore full backup at t0 to ghost tables
// 4.3. apply binlog from t0 to t1 to ghost tables
// 4.4. foreign_key_checks=ON
// 5. lock tables and atomically swap original and ghost tables
func TestPITR(t *testing.T) {
	t.Parallel()
	a := require.New(t)

	// common configs
	localhost := "127.0.0.1"
	port := getTestPort(t.Name())
	username := "root"
	database := "backup_restore"

	// common PITR routines
	initDB := func(t *testing.T, database, username, localhost string, port int) (*sql.DB, func()) {
		a := require.New(t)

		_, stopFn := mysql.SetupTestInstance(t, port)

		db, err := sql.Open("mysql", fmt.Sprintf("%s@tcp(%s:%d)/mysql", username, localhost, port))
		a.NoError(err)

		_, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s", database))
		a.NoError(err)

		_, err = db.Exec(fmt.Sprintf("USE %s", database))
		a.NoError(err)

		_, err = db.Exec(`
		CREATE TABLE t0 (
			id INT,
			PRIMARY KEY (id),
			CHECK (id >= 0)
		);
		`)
		a.NoError(err)
		_, err = db.Exec(`
		CREATE TABLE t1 (
			id INT,
			pid INT,
			PRIMARY KEY (id),
			UNIQUE INDEX (pid),
			CONSTRAINT FOREIGN KEY (pid) REFERENCES t0(id) ON DELETE NO ACTION
		);
		`)
		a.NoError(err)

		return db, stopFn
	}

	insertRangeData := func(t *testing.T, db *sql.DB, begin, end int) {
		a := require.New(t)

		tx, err := db.Begin()
		a.NoError(err)
		defer tx.Rollback()

		for i := begin; i < end; i++ {
			_, err := tx.Exec(fmt.Sprintf("INSERT INTO t0 VALUES (%d)", i))
			a.NoError(err)
			_, err = tx.Exec(fmt.Sprintf("INSERT INTO t1 VALUES (%d, %d)", i, i))
			a.NoError(err)
		}

		err = tx.Commit()
		a.NoError(err)
	}

	// test cases
	t.Run("Buggy Application", func(t *testing.T) {
		db, stopFn := initDB(t, database, username, localhost, port)
		defer db.Close()
		defer stopFn()

		// insert data to make time point t0
		insertRangeData(t, db, 0, 10)

		// make a full backup of t0
		driverBackup := getDbDriver(t, localhost, fmt.Sprintf("%d", port), username, database)
		defer func() {
			err := driverBackup.Close(context.TODO())
			a.NoError(err)
		}()

		buf := doBackup(t, driverBackup, database)
		t.Log(buf.String())

		// insert more data to make time point t1
		insertRangeData(t, db, 10, 20)

		// concurrently update data
		stopChan := make(chan bool)
		go updateRow(t, username, localhost, database, port, stopChan)
		defer func() { stopChan <- true }()

		// restore to ghost database
		_, err := db.Exec("SET foreign_key_checks=OFF")
		a.NoError(err)

		_, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s_ghost", database))
		a.NoError(err)
		_, err = db.Exec(fmt.Sprintf("USE %s_ghost", database))
		a.NoError(err)

		driverRestore := getDbDriver(t, localhost, fmt.Sprintf("%d", port), username, fmt.Sprintf("%s_ghost", database))
		defer func() {
			err := driverRestore.Close(context.TODO())
			a.NoError(err)
		}()
		err = driverRestore.Restore(context.TODO(), bufio.NewScanner(buf))
		a.NoError(err)

		// TODO(dragonly): validate ghost table data and schema

		// TODO(dragonly): apply binlog from full backup to
		// need to use binlog package from gh-ost or go-mysql

		_, err = db.Exec("SET foreign_key_checks=ON")
		a.NoError(err)

		// the user can inspect the ghost table for now, and decide whether to execute the cut over
		// cut over stage, swap tables
		_, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s_old", database))
		a.NoError(err)
		queryRename := fmt.Sprintf("RENAME TABLE"+
			" `%s`.`t0` TO `%s_old`.`t0`, `%s`.`t1` TO `%s_old`.`t1`,"+
			" `%s_ghost`.`t0` TO `%s`.`t0`, `%s_ghost`.`t1` TO `%s`.`t1`",
			database, database, database, database, database, database, database, database)
		t.Log(queryRename)
		_, err = db.Exec(queryRename)
		a.NoError(err)
	})
}

func getDbDriver(t *testing.T, host, port, username, database string) dbPlugin.Driver {
	a := require.New(t)

	logger, err := zap.NewDevelopment()
	a.NoError(err)
	driver, err := dbPlugin.Open(
		context.TODO(),
		dbPlugin.MySQL,
		dbPlugin.DriverConfig{Logger: logger},
		dbPlugin.ConnectionConfig{
			Host:      host,
			Port:      port,
			Username:  username,
			Password:  "",
			Database:  database,
			TLSConfig: dbPlugin.TLSConfig{},
		},
		dbPlugin.ConnectionContext{},
	)
	a.NoError(err)
	return driver
}

func doBackup(t *testing.T, driver dbPlugin.Driver, database string) *bytes.Buffer {
	a := require.New(t)

	var buf bytes.Buffer
	err := driver.Dump(context.TODO(), database, &buf, false)
	a.NoError(err)

	return &buf
}

func updateRow(t *testing.T, username, localhost, database string, port int, stopChan chan bool) {
	a := require.New(t)
	db, err := sql.Open("mysql", fmt.Sprintf("%s@tcp(%s:%d)/mysql", username, localhost, port))
	a.NoError(err)
	defer db.Close()

	_, err = db.Exec(fmt.Sprintf("USE %s", database))
	a.NoError(err)

	t.Log("Start updating data")
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS _update_row_ (id INT PRIMARY KEY)")
	a.NoError(err)
	_, err = db.Exec("REPLACE INTO _update_row_ VALUES (0)")
	a.NoError(err)
	i := 0
	for {
		select {
		case <-stopChan:
			t.Log("Stop updating data")
			return
		default:
		}
		_, err = db.Exec(fmt.Sprintf("UPDATE _update_row_ SET id = %d", i))
		a.NoError(err)
		i++
		time.Sleep(10 * time.Millisecond)
	}
}
