//go:build slim
// +build slim

package cmd

import (
	"fmt"

	"github.com/bytebase/bytebase/store"
	"go.uber.org/zap"
)

type embeddedPgMgr struct{}

func createEmbeddedPgMgr(_ Profile, _ *zap.Logger) (*embeddedPgMgr, error) {
	if useEmbeddedDB() {
		return nil, fmt.Errorf("slim build requires specifying an external PostgreSQL instance connection url by `--pg`")
	}
	return new(embeddedPgMgr), nil
}

func (m *embeddedPgMgr) newEmbeddedDB() (*store.DB, error) {
	return nil, fmt.Errorf("slim build doesn't embed the PostgreSQL binary. Please either use --pg to specify an external PostgreSQL instance or use the full build embedding the PostgreSQL binary.")
}

func (m *embeddedPgMgr) stopEmbeddedDB() error {
	return nil
}
