package db

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase"

	_ "github.com/go-sql-driver/mysql"
)

func init() {
	register(Mysql, newDriver)
}

type MySQLDriver struct {
	l *bytebase.Logger
}

func newDriver(config DriverConfig) Driver {
	return &MySQLDriver{
		l: config.Logger,
	}
}

func (driver *MySQLDriver) Open(config ConnectionConfig) (*sql.DB, error) {
	protocol := "tcp"
	if strings.HasPrefix(config.Host, "/") {
		protocol = "unix"
	}

	dsn := fmt.Sprintf("%s:%s@%s(%s:%s)/", config.Username, config.Password, protocol, config.Host, config.Port)
	driver.l.Debugf("DSN: %s", dsn)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		panic(err)
	}

	return db, nil
}
