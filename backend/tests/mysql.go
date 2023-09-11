//go:build mysql

package tests

import (
	"database/sql"
	"fmt"

	"github.com/bytebase/bytebase/backend/plugin/db"
)

func getMySQLConnectionConfig(port string, database string) db.ConnectionConfig {
	return db.ConnectionConfig{
		Host:      "127.0.0.1",
		Port:      port,
		Username:  "root",
		Password:  "",
		Database:  database,
		TLSConfig: db.TLSConfig{},
	}
}

// connectTestMySQL connects to the test mysql instance.
func connectTestMySQL(port int, database string) (*sql.DB, error) {
	// If we connect using "localhost" on Unix, MySQL will use a socket file.
	// We don't want to be bothered by the socket file conflicts here.
	// ref: https://dev.mysql.com/doc/refman/8.0/en/connecting.html
	return sql.Open("mysql", fmt.Sprintf("root@tcp(127.0.0.1:%d)/%s?multiStatements=true", port, database))
}
