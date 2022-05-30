package store

import (
	"fmt"
	"net"
	"net/url"
	"os"

	"github.com/bytebase/bytebase/common"
	dbdriver "github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/resources/postgres"
	"go.uber.org/zap"
)

// MetadataDB abstracts the underlying Postgres instance
type MetadataDB struct {
	l           *zap.Logger
	mode        common.ReleaseMode
	demoDataDir string

	embed bool

	// Only for external pg
	pgURL string
	// Only for embed postgres
	pgUser     string
	pgInstance *postgres.Instance
	pgStarted  bool
}

// NewMetadataDBWithEmbedPg install postgres in `datadir` returns an instance of MetadataDB
func NewMetadataDBWithEmbedPg(logger *zap.Logger, pgInstance *postgres.Instance, pgUser, dataDir, demoDataDir string, mode common.ReleaseMode) *MetadataDB {
	return &MetadataDB{
		l:           logger,
		mode:        mode,
		demoDataDir: demoDataDir,
		embed:       true,
		pgUser:      pgUser,
		pgInstance:  pgInstance,
	}
}

// NewMetadataDBWithExternalPg constructs a new MetadataDB instance pointing to an external Postgres instance
func NewMetadataDBWithExternalPg(logger *zap.Logger, pgInstance *postgres.Instance, pgURL, demoDataDir string, mode common.ReleaseMode) *MetadataDB {
	return &MetadataDB{
		l:           logger,
		mode:        mode,
		demoDataDir: demoDataDir,
		embed:       false,
		pgURL:       pgURL,
		pgInstance:  pgInstance,
	}
}

// Connect connects to the underlying Postgres instance
func (m *MetadataDB) Connect(datastorePort int, readonly bool, version string) (*DB, error) {
	if m.embed {
		return m.connectEmbed(datastorePort, m.pgUser, readonly, m.demoDataDir, version, m.mode)
	}
	return m.connectExternal(readonly, version)

}

// connectEmbed starts the embed postgres server and returns an instance of store.DB
func (m *MetadataDB) connectEmbed(datastorePort int, pgUser string, readonly bool, demoDataDir, version string, mode common.ReleaseMode) (*DB, error) {
	if err := m.pgInstance.Start(datastorePort, os.Stderr, os.Stderr); err != nil {
		return nil, err
	}
	// mark pgStarted if start successfully, used in Close()
	m.pgStarted = true

	// Even when Postgres opens Unix domain socket only for connection, it still requires a port as ID to differentiate different Postgres instances.
	connCfg := dbdriver.ConnectionConfig{
		Username:    pgUser,
		Password:    "",
		Host:        common.GetPostgresSocketDir(),
		Port:        fmt.Sprintf("%d", datastorePort),
		StrictUseDb: false,
	}
	db := NewDB(m.l, connCfg, m.pgInstance.BaseDir, demoDataDir, readonly, version, mode)
	return db, nil
}

// connectExternal returns an instance of store.DB
func (m *MetadataDB) connectExternal(readonly bool, version string) (*DB, error) {
	u, err := url.Parse(m.pgURL)
	if err != nil {
		return nil, err
	}

	q := u.Query()

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
		// There is a hack. PostgreSQL document(https://www.postgresql.org/docs/14/libpq-connect.html)
		// specifies that a Unix-domain socket connection is chosen if the host part is either empty or **looks like an absolute path name**.
		// But url.Parse() does not meet this standard, for example:
		// url.Parse("postgresql://bbexternal@/tmp:3456/postgres"), it will consider `tmp:3456/postgres` as `path`,
		// and we use path as dbasename(same as PostgreSQL document) so that we get a wrong dbname.
		// So we put the socket path in the `host` key in the query,
		// note that in order to comply with the Postgresql document we are not using the `socket` key with obvious semantics.
		// To give a correct example: postgresql://bbexternal@:3456/postgres?host=/tmp
		hostInQuery := q.Get("host")
		if hostInQuery != "" && host != "" {
			// In this case, it is impossible to decide whether to use socket or tcp.
			return nil, fmt.Errorf("please only using socket or host instead of both")
		}
		connCfg.Host = host
		if hostInQuery != "" {
			connCfg.Host = hostInQuery
		}
		connCfg.Port = port
	}

	if u.Path == "" {
		return nil, fmt.Errorf("missing database in the --pg connection string")
	}
	connCfg.Database = u.Path[1:]

	connCfg.TLSConfig = dbdriver.TLSConfig{
		SslCA:   q.Get("sslrootcert"),
		SslKey:  q.Get("sslkey"),
		SslCert: q.Get("sslcert"),
	}

	db := NewDB(m.l, connCfg, m.pgInstance.BaseDir, m.demoDataDir, readonly, version, m.mode)
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
