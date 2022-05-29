//go:build mysql
// +build mysql

package tests

import (
	"context"
	"database/sql"
	"fmt"

	dbplugin "github.com/bytebase/bytebase/plugin/db"
)

// connectTestMySQL connects to the test mysql instance.
func connectTestMySQL(port int, database string) (*sql.DB, error) {
	// If we connect using "localhost" on Unix, MySQL will use a socket file.
	// We don't want to be bothered by the socket file conflicts here.
	// ref: https://dev.mysql.com/doc/refman/8.0/en/connecting.html
	return sql.Open("mysql", fmt.Sprintf("root@tcp(127.0.0.1:%d)/%s?multiStatements=true", port, database))
}

func getTestMySQLDriver(ctx context.Context, port, database string) (dbplugin.Driver, error) {
	return dbplugin.Open(
		ctx,
		dbplugin.MySQL,
		dbplugin.DriverConfig{},
		dbplugin.ConnectionConfig{
			Host:      "localhost",
			Port:      port,
			Username:  "root",
			Password:  "",
			Database:  database,
			TLSConfig: dbplugin.TLSConfig{},
		},
		dbplugin.ConnectionContext{},
	)
}
