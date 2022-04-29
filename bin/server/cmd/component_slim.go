//go:build slim
// +build slim

package cmd

import (
	"fmt"

	"github.com/bytebase/bytebase/store"
	"go.uber.org/zap"
)

type MetadataDBManager struct {
	profile *Profile
	l       *zap.Logger
}

func createMetadataDBManager(profile *Profile, logger *zap.Logger) (*MetadataDBManager, error) {
	if useEmbeddedDB() {
		return nil, fmt.Errorf("slim build doesn't embed the PostgreSQL binary. Please either use --pg to specify an external PostgreSQL instance or use the full build embedding the PostgreSQL binary.")
	}
	return &MetadataDBManager{
		profile: profile,
		l:       logger,
	}, nil
}

func (m *MetadataDBManager) newDB() (*store.DB, error) {
	if useEmbeddedDB() {
		return nil, fmt.Errorf("slim build doesn't embed the PostgreSQL binary. Please either use --pg to specify an external PostgreSQL instance or use the full build embedding the PostgreSQL binary.")
	}
	return newExternalDB(m.profile, m.l)
}

func (m *MetadataDBManager) stopIfEmbeddedDB() error {
	return nil
}
