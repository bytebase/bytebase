package cmd

import (
	"log/slog"
	"os"
	"time"

	"github.com/google/uuid"

	"github.com/bytebase/bytebase/backend/component/config"
)

func getBaseProfile(dataDir string) *config.Profile {
	egressIPs, invalidEgressIPs := config.ParseEgressIPs(os.Getenv("EGRESS_IPS"))
	if len(invalidEgressIPs) > 0 {
		slog.Warn("ignored EGRESS_IPS entries that are neither an IP address nor a CIDR range", slog.Any("entries", invalidEgressIPs))
	}
	config := &config.Profile{
		ExternalURL:                flags.externalURL,
		Port:                       flags.port,     // Using flags.port as our gRPC server port.
		DatastorePort:              flags.port + 2, // Using flags.port + 2 as our datastore port.
		HA:                         flags.ha,
		SaaS:                       flags.saas,
		MetricsRemoteAccess:        flags.metricsRemoteAccess,
		Debug:                      flags.debug,
		IsDocker:                   isDocker(),
		DataDir:                    dataDir,
		Version:                    version,
		GitCommit:                  gitcommit,
		PgURL:                      os.Getenv("PG_URL"),
		SampleProjectInstancePgURL: os.Getenv("SAMPLE_PROJECT_INSTANCE_PG_URL"),
		EgressIPs:                  egressIPs,
		ReplicaID:                  uuid.NewString(),
		StripeAPISecret:            os.Getenv("STRIPE_API_SECRET"),
		StripeWebhookSecret:        os.Getenv("STRIPE_WEBHOOK_SECRET"),
		LicensePrivateKey:          os.Getenv("LICENSE_PRIVATE_KEY"),
	}

	config.LastActiveTS.Store(time.Now().Unix())
	config.RuntimeMemoryProfileThreshold.Store(flags.memoryProfileThreshold)
	return config
}

func resolveProfile() (*config.Profile, error) {
	if err := checkDataDir(); err != nil {
		return nil, err
	}
	return activeProfile(flags.dataDir), nil
}

func isDocker() bool {
	if _, err := os.Stat("/etc/bb.env"); err == nil {
		return true
	}
	return false
}
