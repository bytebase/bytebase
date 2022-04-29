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
	if !useEmbeddedDB() {
		return nil, fmt.Errorf("slim build requires specifying an external PostgreSQL instance connection url by `--pg`")
	}
	return new(embeddedPgMgr), nil
}

func (m *embeddedPgMgr) newEmbeddedDB() (*store.DB, error) {
	return nil, fmt.Errorf("slim build doesn't support newEmbeddedDB")
}

func (m *embeddedPgMgr) stopEmbeddedDB() error {
	return nil
}
