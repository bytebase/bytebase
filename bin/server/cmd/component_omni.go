//go:build !slim
// +build !slim

package cmd

import (
	"fmt"
	"os"
	"path"

	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/resources/postgres"
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
