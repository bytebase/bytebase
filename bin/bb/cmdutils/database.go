package cmdutils

import (
	"context"
	"fmt"

	"github.com/bytebase/bytebase/plugin/db"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// DatabaseOption is flags for database connecting.
type DatabaseOption struct {
	dbType   string
	host     string
	port     string
	username string
	password string
	Database string

	sslCA   string
	sslCert string
	sslKey  string
}

// NeedDatabaseDriver adds flags for database connecting to a command.
func NeedDatabaseDriver(cmd *cobra.Command) *DatabaseOption {
	var option DatabaseOption
	cmd.Flags().StringVar(&option.dbType, "type", "mysql", "Database type. (mysql, or pg).")
	cmd.Flags().StringVarP(&option.username, "username", "u", "", "Username to login database. (default mysql:root pg:postgres).")
	cmd.Flags().StringVar(&option.password, "password", "", "Password to login database.")
	cmd.Flags().StringVar(&option.host, "host", "", "Hostname of database.")
	cmd.Flags().StringVar(&option.port, "port", "", "Port of database. (default mysql:3306 pg:5432).")
	cmd.Flags().StringVar(&option.Database, "database", "", "Database to connect.")

	cmd.Flags().StringVar(&option.sslCA, "ssl-ca", "", "CA file in PEM format.")
	cmd.Flags().StringVar(&option.sslCert, "ssl-cert", "", "X509 cert in PEM format.")
	cmd.Flags().StringVar(&option.sslKey, "ssl-key", "", "X509 key in PEM format.")

	return &option
}

// Connect connects to a database.
func (opt *DatabaseOption) Connect(ctx context.Context, logger *zap.Logger, readOnly bool) (db.Driver, error) {
	var dbType db.Type
	switch opt.dbType {
	case "mysql":
		dbType = db.MySQL
		if opt.username == "" {
			opt.username = "root"
		}
		if opt.port == "" {
			opt.port = "3306"
		}
	case "pg":
		dbType = db.Postgres
		if opt.username == "" {
			opt.username = "postgres"
		}
		if opt.port == "" {
			opt.port = "5432"
		}
	default:
		return nil, fmt.Errorf("database type %q not supported; supported types: mysql, pg", opt.dbType)
	}

	return db.Open(ctx, dbType,
		db.DriverConfig{
			Logger: logger,
		},
		db.ConnectionConfig{
			Host:     opt.host,
			Port:     opt.port,
			Username: opt.username,
			Password: opt.password,
			Database: opt.Database,
			TLSConfig: db.TLSConfig{
				SslCA:   opt.sslCA,
				SslCert: opt.sslCert,
				SslKey:  opt.sslKey,
			},
			ReadOnly: readOnly,
		},
		db.ConnectionContext{},
	)
}
