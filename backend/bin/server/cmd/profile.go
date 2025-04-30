package cmd

import (
	"os"
	"time"

	"github.com/google/uuid"

	"github.com/bytebase/bytebase/backend/component/config"
)

func getBaseProfile(dataDir string) *config.Profile {
	sampleDatabasePort := 0
	if !flags.disableSample {
		// Using flags.port + 3 as our sample database port if not disabled.
		sampleDatabasePort = flags.port + 3
	}

	config := &config.Profile{
		ExternalURL:        flags.externalURL,
		Port:               flags.port,     // Using flags.port as our gRPC server port.
		DatastorePort:      flags.port + 2, // Using flags.port + 2 as our datastore port.
		SampleDatabasePort: sampleDatabasePort,
		HA:                 flags.ha,
		SaaS:               flags.saas,
		EnableJSONLogging:  flags.enableJSONLogging,
		IsDocker:           isDocker(),
		DataDir:            dataDir,
		Demo:               flags.demo,
		Version:            version,
		GitCommit:          gitcommit,
		PgURL:              flags.pgURL,
		DeployID:           uuid.NewString()[:8],
		LastActiveTS:       time.Now().Unix(),
	}

	config.RuntimeDebug.Store(flags.debug)
	config.RuntimeMemoryProfileThreshold.Store(flags.memoryProfileThreshold)
	return config
}

func isDocker() bool {
	if _, err := os.Stat("/etc/bb.env"); err == nil {
		return true
	}
	return false
}
