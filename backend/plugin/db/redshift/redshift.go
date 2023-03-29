package redshift

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

var (
	excludedDatabaseList = map[string]bool{}

	// driverName is the driver name that our driver dependence register, now is "pgx".
	driverName = "pgx"

	_ db.Driver = (*Driver)(nil)
)

func init() {
	db.Register(db.Redshift, newDriver)
}

// Driver is the Postgres driver.
type Driver struct {
	connectionCtx db.ConnectionContext
	config        db.ConnectionConfig

	db *sql.DB
	// connectionString is the connection string registered by pgx.
	// Unregister connectionString if we don't need it.
	connectionString string
	baseDSN          string
	databaseName     string
}

func newDriver(config db.DriverConfig) db.Driver {
	return &Driver{}
}

// General execution
// A driver might support multiple engines (e.g. MySQL driver can support both MySQL and TiDB),
// So we pass the dbType to tell the exact engine.
func (driver *Driver) Open(ctx context.Context, dbType db.Type, config db.ConnectionConfig, connCtx db.ConnectionContext) (db.Driver, error) {
	// Require username for Postgres, as the guessDSN 1st guess is to use the username as the connecting database
	// if database name is not explicitly specified.
	if config.Username == "" {
		return nil, errors.Errorf("user must be set")
	}

	if (config.TLSConfig.SslCert == "" && config.TLSConfig.SslKey != "") ||
		(config.TLSConfig.SslCert != "" && config.TLSConfig.SslKey == "") {
		return nil, errors.Errorf("ssl-cert and ssl-key must be both set or unset")
	}

	databaseName, dsn, err := guessDSN(
		config.Username,
		config.Password,
		config.Host,
		config.Port,
		config.Database,
		config.TLSConfig.SslCA,
		config.TLSConfig.SslCert,
		config.TLSConfig.SslKey,
	)
	if err != nil {
		return nil, err
	}
	if config.ReadOnly {
		dsn = fmt.Sprintf("%s default_transaction_read_only=true", dsn)
	}
	driver.databaseName = databaseName
	driver.baseDSN = dsn
	driver.connectionCtx = connCtx
	driver.config = config

	connectionString, err := registerConnectionConfig(dsn, driver.config.TLSConfig)
	if err != nil {
		return nil, err
	}
	driver.connectionString = connectionString

	db, err := sql.Open(driverName, driver.connectionString)
	if err != nil {
		return nil, err
	}
	driver.db = db
	return driver, nil
}

func registerConnectionConfig(dsn string, tlsConfig db.TLSConfig) (string, error) {
	connConfig, err := pgx.ParseConfig(dsn)
	if err != nil {
		return "", err
	}

	if tlsConfig.SslCA != "" {
		sslConfig, err := tlsConfig.GetSslConfig()
		if err != nil {
			return "", err
		}
		connConfig.TLSConfig = sslConfig
	}

	return stdlib.RegisterConnConfig(connConfig), nil
}

func unregisterConnectionConfig(connectionString string) {
	stdlib.UnregisterConnConfig(connectionString)
}

// guessDSN will guess a valid DB connection and its database name.
func guessDSN(username, password, hostname, port, database, sslCA, sslCert, sslKey string) (string, string, error) {
	// dbname is guessed if not specified.
	m := map[string]string{
		"host":     hostname,
		"port":     port,
		"user":     username,
		"password": password,
	}
	if database != "" {
		m["dbname"] = database
	}

	tlsConfig := db.TLSConfig{
		SslCA:   sslCA,
		SslCert: sslCert,
		SslKey:  sslKey,
	}

	var tokens []string
	for k, v := range m {
		if v != "" {
			tokens = append(tokens, fmt.Sprintf("%s=%s", k, v))
		}
	}
	dsn := strings.Join(tokens, " ")

	if database != "" {
		return database, dsn, nil
	}

	// The dafault database name is "dev" for Redshift.
	guesses := []string{"dev"}
	//  dsn+" dbname=bytebase"
	for _, guessDatabase := range guesses {
		guessDSN := fmt.Sprintf("%s dbname=%s", dsn, guessDatabase)
		if err := func() error {
			connectionString, err := registerConnectionConfig(guessDSN, tlsConfig)
			if err != nil {
				return err
			}
			defer unregisterConnectionConfig(connectionString)
			db, err := sql.Open(driverName, connectionString)
			if err != nil {
				return err
			}
			defer db.Close()
			return db.Ping()
		}(); err != nil {
			log.Debug("guessDSN attemp	t failed", zap.Error(err))
			continue
		}
		return guessDatabase, guessDSN, nil
	}
	return "", "", errors.Errorf("cannot connect to the instance, make sure the connection info is correct")
}

// Close closes the database and prevents new queries from starting.
// Close then waits for all queries that have started processing on the server to finish.
func (d *Driver) Close(_ context.Context) error {
	return d.db.Close()
}

// Ping verifies a connection to the database is still alive, establishing a connection if necessary.
func (d *Driver) Ping(ctx context.Context) error {
	return d.db.PingContext(ctx)
}

// GetType returns the database type.
func (d *Driver) GetType() db.Type {
	return db.Redshift
}

// GetDBConnection returns the database connection.
func (d *Driver) GetDBConnection(ctx context.Context, database string) (*sql.DB, error) {
	return nil, errors.Errorf("get db connection for Redshift is not implemented yet")
}

// Execute will execute the statement. For CREATE DATABASE statement, some types of databases such as Postgres
// will not use transactions to execute the statement but will still use transactions to execute the rest of statements.
func (d *Driver) Execute(ctx context.Context, statement string, createDatabase bool) (int64, error) {
	return 0, errors.Errorf("execute for Redshift is not implemented yet")
}

// Used for execute readonly SELECT statement
func (d *Driver) QueryConn(ctx context.Context, conn *sql.Conn, statement string, queryContext *db.QueryContext) ([]interface{}, error) {
	return nil, errors.Errorf("query conn for Redshift is not implemented yet")
}
