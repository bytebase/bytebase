package cmd

import (
	"context"
	"os"

	"github.com/pkg/errors"
	"github.com/xo/dburl"

	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/resources/mysqlutil"
	"github.com/bytebase/bytebase/resources/postgres"

	// install mysql driver.
	_ "github.com/bytebase/bytebase/plugin/db/mysql"
	// register pg driver.
	_ "github.com/bytebase/bytebase/plugin/db/pg"
	// install pg driver.
)

func getDatabase(u *dburl.URL) string {
	if u.Path == "" {
		return ""
	}
	return u.Path[1:]
}

func open(ctx context.Context, u *dburl.URL) (db.Driver, error) {
	var dbType db.Type
	var dbBinDir string
	resourceDir := os.TempDir()
	switch u.Driver {
	case "mysql":
		dbType = db.MySQL
		dir, err := mysqlutil.Install(resourceDir)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot install mysqlutil in directory %q", resourceDir)
		}
		dbBinDir = dir
	case "postgres":
		dbType = db.Postgres
		dir, err := postgres.Install(resourceDir)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot install postgres in directory %q", resourceDir)
		}
		dbBinDir = dir
	default:
		return nil, errors.Errorf("database type %q not supported; supported types: mysql, pg", u.Driver)
	}
	passwd, _ := u.User.Password()
	driver, err := db.Open(
		ctx,
		dbType,
		db.DriverConfig{
			DbBinDir: dbBinDir,
		},
		db.ConnectionConfig{
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
		},
		db.ConnectionContext{},
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open database")
	}

	return driver, nil
}
