//go:build !slim
// +build !slim

package cmd

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"path"

	"github.com/bytebase/bytebase/common"
	dbdriver "github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/resources/postgres"
	"github.com/bytebase/bytebase/store"
	"go.uber.org/zap"
)

type metadataDB struct {
	profile    *Profile
	l          *zap.Logger
	pgInstance *postgres.Instance
	pgStarted  bool
}

func createMetadataDB(activeProfile *Profile, logger *zap.Logger) (*metadataDB, error) {
	mgr := &metadataDB{
		profile: activeProfile,
		l:       logger,
	}

	if !useEmbedDB() {
		return mgr, nil
	}

	resourceDir := path.Join(activeProfile.dataDir, "resources")
	pgDataDir := common.GetPostgresDataDir(activeProfile.dataDir)
	fmt.Println("-----Embedded Postgres Config BEGIN-----")
	fmt.Printf("resourceDir=%s\n", resourceDir)
	fmt.Printf("pgdataDir=%s\n", pgDataDir)
	fmt.Println("-----Embedded Postgres Config END-----")

	logger.Info("Preparing embedded PostgreSQL instance...")
	// Installs the Postgres binary and creates the 'activeProfile.pgUser' user/database
	// to store Bytebase's own metadata.
	var err error
	mgr.pgInstance, err = postgres.Install(resourceDir, pgDataDir, activeProfile.pgUser)
	if err != nil {
		return nil, err
	}

	return mgr, nil
}

func (m *metadataDB) connect() (*store.DB, error) {
	if useEmbedDB() {
		if err := m.pgInstance.Start(m.profile.datastorePort, os.Stderr, os.Stderr); err != nil {
			return nil, err
		}
		m.pgStarted = true

		// Even when Postgres opens Unix domain socket only for connection, it still requires a port as ID to differentiate different Postgres instances.
		connCfg := dbdriver.ConnectionConfig{
			Username:    m.profile.pgUser,
			Password:    "",
			Host:        common.GetPostgresSocketDir(),
			Port:        fmt.Sprintf("%d", m.profile.datastorePort),
			StrictUseDb: false,
		}
		db := store.NewDB(m.l, connCfg, m.profile.demoDataDir, readonly, version, m.profile.mode)
		return db, nil
	}

	u, err := url.Parse(pgURL)
	if err != nil {
		return nil, err
	}

	m.l.Info("Establishing external PostgreSQL connection...", zap.String("pgURL", u.Redacted()))

	if u.Scheme != "postgres" && u.Scheme != "postgresql" {
		return nil, fmt.Errorf("invalid connection protocol: %s", u.Scheme)
	}

	connCfg := dbdriver.ConnectionConfig{
		StrictUseDb: true,
	}

	if u.User != nil {
		connCfg.Username = u.User.Username()
		connCfg.Password, _ = u.User.Password()
	}

	if connCfg.Username == "" {
		return nil, fmt.Errorf("missing user in the --pg connection string")
	}

	if host, port, err := net.SplitHostPort(u.Host); err != nil {
		connCfg.Host = u.Host
	} else {
		connCfg.Host = host
		connCfg.Port = port
	}

	// By default, follow the PG convention to use user name as the database name
	connCfg.Database = connCfg.Username

	if u.Path == "" {
		return nil, fmt.Errorf("missing database in the --pg connection string")
	}
	connCfg.Database = u.Path[1:]

	q := u.Query()
	connCfg.TLSConfig = dbdriver.TLSConfig{
		SslCA:   q.Get("sslrootcert"),
		SslKey:  q.Get("sslkey"),
		SslCert: q.Get("sslcert"),
	}

	db := store.NewDB(m.l, connCfg, m.profile.demoDataDir, readonly, version, m.profile.mode)
	return db, nil
}

func (m *metadataDB) close() error {
	if !m.pgStarted {
		return nil
	}

	m.l.Info("Trying to shutdown postgresql server...")
	if err := m.pgInstance.Stop(os.Stdout, os.Stderr); err != nil {
		return err
	}
	m.pgStarted = false
	return nil
}
