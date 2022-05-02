package metadb

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"path"

	"github.com/bytebase/bytebase/common"
	dbdriver "github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/resources/postgres"
	"github.com/bytebase/bytebase/server"
	"github.com/bytebase/bytebase/store"
	"go.uber.org/zap"
)

type MetadataDB struct {
	profile *server.Profile
	l       *zap.Logger

	embed bool
	// Only for external pg
	pgURL string
	// Only for embed postgres
	pgInstance *postgres.Instance
	pgStarted  bool
}

// NewMetadataDBWithEmbedPg install postgres in `datadir` returns an instance of MetadataDB
func NewMetadataDBWithEmbedPg(activeProfile *server.Profile, logger *zap.Logger) (*MetadataDB, error) {
	mgr := &MetadataDB{
		profile: activeProfile,
		l:       logger,
		embed:   true,
	}
	resourceDir := path.Join(activeProfile.DataDir, "resources")
	pgDataDir := common.GetPostgresDataDir(activeProfile.DataDir)
	fmt.Println("-----Embedded Postgres Config BEGIN-----")
	fmt.Printf("resourceDir=%s\n", resourceDir)
	fmt.Printf("pgdataDir=%s\n", pgDataDir)
	fmt.Println("-----Embedded Postgres Config END-----")

	logger.Info("Preparing embedded PostgreSQL instance...")
	// Installs the Postgres binary and creates the 'activeProfile.pgUser' user/database
	// to store Bytebase's own metadata.
	var err error
	mgr.pgInstance, err = postgres.Install(resourceDir, pgDataDir, activeProfile.PgUser)
	if err != nil {
		return nil, err
	}

	return mgr, nil
}

func NewMetadataDBWithExternalPg(activeProfile *server.Profile, logger *zap.Logger, pgURL string) (*MetadataDB, error) {
	return &MetadataDB{
		profile: activeProfile,
		l:       logger,
		embed:   false,
		pgURL:   pgURL,
	}, nil
}

func (m *MetadataDB) Connect(readonly bool, version string) (*store.DB, error) {
	if m.embed {
		return m.connectEmbed(readonly, version)
	}
	return m.connectExternal(readonly, version)

}

// connectEmbed starts the embed postgres server and returns an instance of store.DB
func (m *MetadataDB) connectEmbed(readonly bool, version string) (*store.DB, error) {
	if err := m.pgInstance.Start(m.profile.DatastorePort, os.Stderr, os.Stderr); err != nil {
		return nil, err
	}
	// mark pgStarted if start successfully, used in Close()
	m.pgStarted = true

	// Even when Postgres opens Unix domain socket only for connection, it still requires a port as ID to differentiate different Postgres instances.
	connCfg := dbdriver.ConnectionConfig{
		Username:    m.profile.PgUser,
		Password:    "",
		Host:        common.GetPostgresSocketDir(),
		Port:        fmt.Sprintf("%d", m.profile.DatastorePort),
		StrictUseDb: false,
	}
	db := store.NewDB(m.l, connCfg, m.profile.DemoDataDir, readonly, version, m.profile.Mode)
	return db, nil
}

// connectExternal returns an instance of store.DB
func (m *MetadataDB) connectExternal(readonly bool, version string) (*store.DB, error) {
	u, err := url.Parse(m.pgURL)
	if err != nil {
		return nil, err
	}

	m.l.Info("Establishing external PostgreSQL connection...", zap.String("pgURL", u.Redacted()))

	if u.Scheme != "postgresql" {
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

	db := store.NewDB(m.l, connCfg, m.profile.DemoDataDir, readonly, version, m.profile.Mode)
	return db, nil
}

// Close will stop postgres server if using embed postgres
func (m *MetadataDB) Close() error {
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
