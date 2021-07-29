// connect is a library for establishing connection to databases provided by bytebase.com.
package connect

import (
	"crypto/tls"
	"database/sql"
	"fmt"

	"github.com/go-sql-driver/mysql"
)

// MysqlConnect is a class for MySQL database connections.
type MysqlConnect struct {
	DB *sql.DB
}

// NewMysql creates a new MySQL connection.
func NewMysql(username, password, hostname, port, database string, tlsCfg *tls.Config) (*MysqlConnect, error) {
	portPart := ""
	if port != "" {
		portPart = fmt.Sprintf(":%s", port)
	}
	dns := fmt.Sprintf("%s:%s@tcp(%s%s)/%s", username, password, hostname, portPart, database)
	if tlsCfg != nil {
		mysql.RegisterTLSConfig("custom", tlsCfg)
		dns += "?tls=custom"
	}
	db, err := sql.Open("mysql", dns)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %s", err)
	}
	return &MysqlConnect{
		DB: db,
	}, nil
}

// Close closes the connection.
func (c *MysqlConnect) Close() error {
	return c.DB.Close()
}
