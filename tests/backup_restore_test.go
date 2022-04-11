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

func TestBackupRestore(t *testing.T) {
	t.Parallel()
	a := require.New(t)

	localhost := "127.0.0.1"
	port := getTestPort(t.Name())
	username := "root"
	database := "backup_restore"
	table := "backup_restore"

	_, stop := mysql.SetupTestInstance(t, port)
	defer stop()

	db, err := sql.Open("mysql", fmt.Sprintf("root@tcp(localhost:%d)/mysql", port))
	a.NoError(err)
	defer db.Close()

	_, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s", database))
	a.NoError(err)

	_, err = db.Exec(fmt.Sprintf("USE %s", database))
	a.NoError(err)

	_, err = db.Exec(fmt.Sprintf(`
	CREATE Table %s (
		id INT,
		PRIMARY KEY (id)
	);
	`, table))
	a.NoError(err)

	tx, err := db.Begin()
	a.NoError(err)
	defer tx.Rollback()
	for i := 0; i < 100; i++ {
		_, err = tx.Exec(fmt.Sprintf("INSERT INTO %s VALUES (%d)", table, i))
		a.NoError(err)
	}
	err = tx.Commit()
	a.NoError(err)

	// make a full backup
	logger, err := zap.NewDevelopment()
	a.NoError(err)
	driver, err := dbPlugin.Open(
		context.TODO(),
		dbPlugin.MySQL,
		dbPlugin.DriverConfig{Logger: logger},
		dbPlugin.ConnectionConfig{
			Host:      localhost,
			Port:      fmt.Sprintf("%d", port),
			Username:  username,
			Password:  "",
			Database:  database,
			TLSConfig: dbPlugin.TLSConfig{},
		},
		dbPlugin.ConnectionContext{},
	)
	defer driver.Close(context.TODO())

	var buf bytes.Buffer
	err = driver.Dump(context.TODO(), database, &buf, false)
	a.NoError(err)

	// t.Logf("dump:\n%s", buf.String())

	// drop all tables
	_, err = db.Exec(fmt.Sprintf("DROP TABLE %s", table))
	a.NoError(err)

	// restore
	err = driver.Restore(context.TODO(), bufio.NewScanner(&buf), dbPlugin.RestoreConfig{})
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

	localhost := "127.0.0.1"
	port := getTestPort(t.Name())
	username := "root"
	database := "backup_restore"

	insertData := func(db *sql.DB, a *require.Assertions, begin, end int) {
		tx, err := db.Begin()
		a.NoError(err)
		defer tx.Rollback()

		for i := begin; i < end; i++ {
			_, err := tx.Exec(fmt.Sprintf("INSERT INTO t00 VALUES (%d)", i))
			a.NoError(err)
			_, err = tx.Exec(fmt.Sprintf("INSERT INTO t10 VALUES (%d, %d)", i, i))
			a.NoError(err)
			_, err = tx.Exec(fmt.Sprintf("INSERT INTO t11 VALUES (%d, %d)", i, i))
			a.NoError(err)
		}

		err = tx.Commit()
		a.NoError(err)
	}

	_, stop := mysql.SetupTestInstance(t, port)
	defer stop()

	db, err := sql.Open("mysql", fmt.Sprintf("root@tcp(localhost:%d)/mysql", port))
	a.NoError(err)
	defer db.Close()

	_, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s", database))
	a.NoError(err)

	_, err = db.Exec(fmt.Sprintf("USE %s", database))
	a.NoError(err)

	_, err = db.Exec(`
	CREATE Table t00 (
		id INT,
		PRIMARY KEY (id),
		CHECK (id > -1)
	);
	`)
	a.NoError(err)
	_, err = db.Exec(`
	CREATE Table t10 (
		id INT,
		pid INT,
		PRIMARY KEY (id),
		UNIQUE INDEX (pid),
		CONSTRAINT FOREIGN KEY (pid) REFERENCES t00(id) ON DELETE NO ACTION
	);
	`)
	a.NoError(err)
	_, err = db.Exec(`
	CREATE Table t11 (
		id INT,
		pid INT,
		PRIMARY KEY (id),
		UNIQUE INDEX (pid),
		CONSTRAINT FOREIGN KEY (pid) REFERENCES t00(id) ON DELETE NO ACTION
	);
	`)
	a.NoError(err)

	// insert data to make time point t0
	insertData(db, a, 0, 10)

	// make a full backup of t0
	logger, err := zap.NewDevelopment()
	a.NoError(err)
	driver, err := dbPlugin.Open(
		context.TODO(),
		dbPlugin.MySQL,
		dbPlugin.DriverConfig{Logger: logger},
		dbPlugin.ConnectionConfig{
			Host:      localhost,
			Port:      fmt.Sprintf("%d", port),
			Username:  username,
			Password:  "",
			Database:  database,
			TLSConfig: dbPlugin.TLSConfig{},
		},
		dbPlugin.ConnectionContext{},
	)
	defer driver.Close(context.TODO())

	var buf bytes.Buffer
	err = driver.Dump(context.TODO(), database, &buf, false)
	a.NoError(err)
	t.Log(buf.String())

	// insert more data to make time point t1
	insertData(db, a, 100, 200)

	// concurrently insert data
	stopChan := make(chan bool)
	go func() {
		i := 200

		db, err := sql.Open("mysql", fmt.Sprintf("root@tcp(localhost:%d)/mysql", port))
		a.NoError(err)
		defer db.Close()

		_, err = db.Exec(fmt.Sprintf("USE %s", database))
		a.NoError(err)

		t.Log("Start inserting data")
		for {
			select {
			case <-stopChan:
				t.Log("Stop inserting data")
				return
			default:
			}
			insertData(db, a, i, i+1)
			time.Sleep(10 * time.Millisecond)
		}
	}()
	defer func() { stopChan <- true }()

	// restore to ghost tables
	_, err = db.Exec("SET foreign_key_checks=OFF")
	a.NoError(err)

	err = driver.Restore(context.TODO(), bufio.NewScanner(&buf), dbPlugin.RestoreConfig{IsGhostTable: true})
	a.NoError(err)

	// TODO(dragonly): validate ghost table data and schema

	// TODO(dragonly): apply binlog from full backup to
	// need to use binlog package from gh-ost or go-mysql

	_, err = db.Exec("SET foreign_key_checks=ON")
	a.NoError(err)

	// TODO(dragonly): swap tables atomically
}
