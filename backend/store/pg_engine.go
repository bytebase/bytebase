package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/pkg/errors"

	dbdriver "github.com/bytebase/bytebase/backend/plugin/db"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"

	// Register postgres driver.

	"github.com/bytebase/bytebase/backend/common"
)

// DB represents the metdata database connection.
type DB struct {
	metadataDriver dbdriver.Driver
	db             *sql.DB

	// db.ConnCfg is the connection configuration to a Postgres database.
	// The user has superuser privilege to the database.
	ConnCfg dbdriver.ConnectionConfig

	// Demo name, empty string means do not load demo data.
	demoName string

	// Dir for postgres and its utility binaries
	binDir string

	// If true, database will be opened in readonly mode
	readonly bool

	// Bytebase server release version
	serverVersion string

	// mode is the mode of the release such as prod or dev.
	mode common.ReleaseMode

	// Returns the current time. Defaults to time.Now().
	// Can be mocked for tests.
	Now func() time.Time
}

// NewDB returns a new instance of DB associated with the given datasource name.
func NewDB(connCfg dbdriver.ConnectionConfig, binDir, demoName string, readonly bool, serverVersion string, mode common.ReleaseMode) *DB {
	db := &DB{
		ConnCfg:       connCfg,
		demoName:      demoName,
		binDir:        binDir,
		readonly:      readonly,
		Now:           time.Now,
		serverVersion: serverVersion,
		mode:          mode,
	}
	return db
}

// Open opens the database connection.
func (db *DB) Open(ctx context.Context, createDB bool) error {
	if createDB {
		createCfg := db.ConnCfg
		// connect to the "postgres" as the target database has not been created yet.
		createCfg.Database = "postgres"
		// Create the metadata database.
		defaultDriver, err := dbdriver.Open(
			ctx,
			storepb.Engine_POSTGRES,
			dbdriver.DriverConfig{DbBinDir: db.binDir},
			createCfg,
			dbdriver.ConnectionContext{},
		)
		if err != nil {
			return err
		}
		defer defaultDriver.Close(ctx)
		// Underlying driver handles the case where database already exists.
		if _, err := defaultDriver.Execute(ctx, fmt.Sprintf("CREATE DATABASE %s", db.ConnCfg.Database), true, dbdriver.ExecuteOptions{}); err != nil {
			return err
		}
	}

	metadataDriver, err := dbdriver.Open(
		ctx,
		storepb.Engine_POSTGRES,
		dbdriver.DriverConfig{DbBinDir: db.binDir},
		db.ConnCfg,
		dbdriver.ConnectionContext{},
	)
	if err != nil {
		return err
	}
	// Don't close metadataDriver.
	db.metadataDriver = metadataDriver
	db.db = metadataDriver.GetDB()
	// Set the max open connections so that we won't exceed the connection limit of metaDB.
	// The limit is the max connections minus connections reserved for superuser.
	var maxConns, reservedConns int
	if err := db.db.QueryRowContext(ctx, `SHOW max_connections`).Scan(&maxConns); err != nil {
		return errors.Wrap(err, "failed to get max_connections from metaDB")
	}
	if err := db.db.QueryRowContext(ctx, `SHOW superuser_reserved_connections`).Scan(&reservedConns); err != nil {
		return errors.Wrap(err, "failed to get superuser_reserved_connections from metaDB")
	}
	maxOpenConns := maxConns - reservedConns
	if maxOpenConns > 50 {
		// capped to 50
		maxOpenConns = 50
	}
	db.db.SetMaxOpenConns(maxOpenConns)
	return nil
}

// Close closes the database connection.
func (db *DB) Close(ctx context.Context) error {
	return db.metadataDriver.Close(ctx)
}

// BeginTx starts a transaction and returns a wrapper Tx type. This type
// provides a reference to the database and a fixed timestamp at the start of
// the transaction. The timestamp allows us to mock time during tests as well.
func (db *DB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*Tx, error) {
	ptx, err := db.db.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}

	// Return wrapper Tx that includes the transaction start time.
	return &Tx{
		Tx:  ptx,
		db:  db,
		now: db.Now().UTC().Truncate(time.Second),
	}, nil
}

// Tx wraps the SQL Tx object to provide a timestamp at the start of the transaction.
type Tx struct {
	*sql.Tx
	db  *DB
	now time.Time
}
