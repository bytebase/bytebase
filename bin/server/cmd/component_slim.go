//go:build slim
// +build slim

package cmd

import (
	"fmt"

	"github.com/bytebase/bytebase/store"
	"go.uber.org/zap"
)

type metadataDB struct {
	profile *Profile
	l       *zap.Logger
}

func createMetadataDB(profile *Profile, logger *zap.Logger) (*metadataDB, error) {
	if useEmbedDB() {
		return nil, fmt.Errorf("slim build doesn't embed the PostgreSQL binary. Please use --pg to specify an external PostgreSQL instance.")
	}

	return &metadataDB{
		profile: profile,
		l:       logger,
	}, nil
}

// connect connects to the database that stores bytebase metadata.
func (m *metadataDB) connect() (*store.DB, error) {
	return m.connectExternalPostgres()
}

func (m *metadataDB) close() error {
	return nil
}
