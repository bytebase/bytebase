package cmd

import (
	"os"
	"time"

	"github.com/google/uuid"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/config"
)

func getBaseProfile(dataDir string) *config.Profile {
	sampleDatabasePort := 0
	if !flags.disableSample {
		// Using flags.port + 3 as our sample database port if not disabled.
		sampleDatabasePort = flags.port + 3
	}

	return &config.Profile{
		ExternalURL:          flags.externalURL,
		Port:                 flags.port,     // Using flags.port as our gRPC server port.
		DatastorePort:        flags.port + 2, // Using flags.port + 2 as our datastore port.
		SampleDatabasePort:   sampleDatabasePort,
		Readonly:             flags.readonly,
		SaaS:                 flags.saas,
		EnableJSONLogging:    flags.enableJSONLogging,
		Debug:                flags.debug,
		IsDocker:             isDocker(),
		DataDir:              dataDir,
		ResourceDir:          common.GetResourceDir(dataDir),
		DemoName:             flags.demoName,
		Version:              version,
		GitCommit:            gitcommit,
		PgURL:                flags.pgURL,
		DeployID:             uuid.NewString()[:8],
		LastActiveTs:         time.Now().Unix(),
		DevelopmentVersioned: flags.developmentVersioned,
	}
}

func isDocker() bool {
	if _, err := os.Stat("/etc/bb.env"); err == nil {
		return true
	}
	return false
}
