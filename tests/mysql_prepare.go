//go:build mysql
// +build mysql

package tests

import (
	"bytes"
	"database/sql"
	"fmt"
	"math/rand"

	"github.com/google/uuid"
)

func prepare(port int, tableSize int) error {
	const createTableSQL = `CREATE TABLE sbtest1 (
id INT NOT NULL,
k INT NOT NULL DEFAULT 0,
c CHAR(120) NOT NULL DEFAULT '',
pad CHAR(60) NOT NULL DEFAULT'',
PRIMARY KEY (id),
KEY k_1 (k)
)`
	db, err := connectTestMySQL(port, "")
	if err != nil {
		return nil
	}
	defer db.Close()

	_, err = db.Exec("CREATE DATABASE sbtest; USE sbtest;")
	if err != nil {
		return err
	}
	_, err = db.Exec(createTableSQL)
	if err != nil {
		return err
	}

	err = insert(db, tableSize, 100)
	if err != nil {
		return err
	}

	return nil
}

func insert(db *sql.DB, insertCount int, batchCount int) error {
	const insertSQL = `INSERT INTO sbtest1 (id, k, c, pad) VALUES `
	var buf bytes.Buffer
	for i := 0; i < insertCount; i += batchCount {
		buf.Reset()
		// We use bytes.Buffer here to concat the strings because strings are too slow.
		buf.WriteString(insertSQL)
		for j := 0; j < batchCount; j++ {
			id := i + j
			if id >= insertCount {
				break
			}
			delimiter := ""
			if j > 0 {
				delimiter = ", "
			}
			// append the values after "VALUES ".
			fmt.Fprintf(&buf, "%s(%d, %d, '%s', '%s')", delimiter, id, rand.Intn(insertCount), uuid.New().String(), uuid.New().String())
		}
		_, err := db.Exec(buf.String())
		if err != nil {
			return err
		}
	}
	return nil
}
