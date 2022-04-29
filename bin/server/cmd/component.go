//go:build !slim
// +build !slim

package cmd

import (
	"fmt"
	"os"
	"path"

	"github.com/bytebase/bytebase/common"
	dbdriver "github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/resources/postgres"
	"github.com/bytebase/bytebase/store"
	"go.uber.org/zap"
)

type embeddedPgMgr struct {
	profile    *Profile
	l          *zap.Logger
	pgInstance *postgres.Instance
	pgStarted  bool
}

func createEmbeddedPgMgr(activeProfile Profile, logger *zap.Logger) (*embeddedPgMgr, error) {
	embeddedPgMgr := new(embeddedPgMgr)
	embeddedPgMgr.profile = &activeProfile
	embeddedPgMgr.l = logger

	if !useEmbeddedDB() {
		return embeddedPgMgr, nil
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
	embeddedPgMgr.pgInstance, err = postgres.Install(resourceDir, pgDataDir, activeProfile.pgUser)
	if err != nil {
		return nil, err
	}

	return embeddedPgMgr, nil
}

func (m *embeddedPgMgr) newEmbeddedDB() (*store.DB, error) {
	if !useEmbeddedDB() {
		return nil, fmt.Errorf("cannot newEmbeddedDB if useEmbeddedDB is false")
	}

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

func (m *embeddedPgMgr) stopEmbeddedDB() error {
	if m.pgStarted {
		m.l.Info("Trying to shutdown postgresql server...")

		if err := m.pgInstance.Stop(os.Stdout, os.Stderr); err != nil {
			return err
		}
		m.pgStarted = false
	}
	return nil
}
