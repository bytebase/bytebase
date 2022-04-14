package cmd

import (
	"context"
	"fmt"

	"github.com/bytebase/bytebase/plugin/db"
	"github.com/xo/dburl"
	"go.uber.org/zap"
)

func getDatabase(u *dburl.URL) string {
	if u.Path == "" {
		return ""
	}
	return u.Path[1:]
}

func open(ctx context.Context, logger *zap.Logger, u *dburl.URL) (db.Driver, error) {
	var dbType db.Type
	switch u.Driver {
	case "mysql":
		dbType = db.MySQL
	case "pg":
		dbType = db.Postgres
	default:
		return nil, fmt.Errorf("database type %q not supported; supported types: mysql, pg", u.Driver)
	}

	passwd, _ := u.User.Password()
	driver, err := db.Open(ctx, dbType, db.DriverConfig{Logger: logger}, db.ConnectionConfig{
		Host:     u.Hostname(),
		Port:     u.Port(),
		Username: u.User.Username(),
		Password: passwd,
		Database: getDatabase(u),
		TLSConfig: db.TLSConfig{
			SslCA:   u.Query().Get("ssl-ca"),
			SslCert: u.Query().Get("ssl-cert"),
			SslKey:  u.Query().Get("ssl-key"),
		},
	}, db.ConnectionContext{})
	if err != nil {
		return nil, fmt.Errorf("failed to open database, got error: %w", err)
	}

	return driver, nil
}
