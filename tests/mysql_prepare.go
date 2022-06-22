//go:build mysql
// +build mysql

package tests

import (
	"bytes"
	"database/sql"
	"fmt"
	"math/rand"

	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
)

func prepare(port int, tableSize int) error {
	const createTableSql = `create table sbtest1 (
id int not null ,
k int not null default 0,
c char(120) not null default '',
pad char(60) not null default '',
primary key (id),
key k_1 (k)
)`
	db, err := connectTestMySQL(port, "")
	if err != nil {
		return nil
	}
	defer db.Close()

	_, err = db.Exec("create database sbtest; use sbtest;")
	if err != nil {
		return err
	}
	_, err = db.Exec(createTableSql)
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
	const insertSql = `insert into sbtest1 (id, k, c, pad) values `
	var buf bytes.Buffer
	for i := 0; i < insertCount; i += batchCount {
		buf.Reset()
		buf.WriteString(insertSql)
		for j := 0; j < batchCount; j++ {
			id := i + j
			if id >= insertCount {
				break
			}
			delimiter := ""
			if j > 0 {
				delimiter = ", "
			}
			fmt.Fprintf(&buf, "%s(%d, %d, '%s', '%s')", delimiter, id, rand.Intn(insertCount), uuid.New().String(), uuid.New().String())
		}
		_, err := db.Exec(buf.String())
		if err != nil {
			return err
		}
	}
	return nil
}
