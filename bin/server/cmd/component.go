package cmd

import (
	"fmt"
	"net"
	"net/url"

	dbdriver "github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/store"
	"go.uber.org/zap"
)

func useEmbedDB() bool {
	return len(pgURL) == 0
}

// connectExternalPostgres connects to the external postgres database specified by user(i.e.: use --pg option).
func (m *metadataDB) connectExternalPostgres() (*store.DB, error) {
	u, err := url.Parse(pgURL)
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

	db := store.NewDB(m.l, connCfg, m.profile.demoDataDir, readonly, version, m.profile.mode)
	return db, nil
}
