package cmd

import (
	"os"
	"time"

	"github.com/google/uuid"

	"github.com/bytebase/bytebase/backend/args"
	"github.com/bytebase/bytebase/backend/component/config"
)

func getBaseProfile(dataDir string) *config.Profile {
	config := &config.Profile{
		ExternalURL:       flags.externalURL,
		Port:              flags.port,     // Using flags.port as our gRPC server port.
		DatastorePort:     flags.port + 2, // Using flags.port + 2 as our datastore port.
		HA:                flags.ha,
		SaaS:              flags.saas,
		EnableJSONLogging: flags.enableJSONLogging,
		IsDocker:          isDocker(),
		DataDir:           dataDir,
		Demo:              flags.demo,
		Version:           args.Version,
		GitCommit:         args.GitCommit,
		PgURL:             os.Getenv("PG_URL"),
		DeployID:          uuid.NewString()[:8],
	}

	config.LastActiveTS.Store(time.Now().Unix())
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
