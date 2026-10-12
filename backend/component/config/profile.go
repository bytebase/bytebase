// Package config includes all the server configurations in a component.
package config

import (
	"net/netip"
	"strings"
	"sync/atomic"

	"github.com/bytebase/bytebase/backend/common"
)

// Profile is the configuration to start main server.
// Profile must not be copied, its fields must not be modified unless mentioned otherwise.
type Profile struct {
	// Mode can be "prod" or "dev"
	Mode common.ReleaseMode
	// ExternalURL is the URL user visits Bytebase.
	ExternalURL string
	// DatastorePort is the binding port for database instance for storing Bytebase metadata.
	// Only applicable when using embedded PG (PgURL is empty).
	DatastorePort int
	// Port is the binding port for the server.
	Port int
	// When we are running in SaaS mode, some features are not allowed to edit by users.
	SaaS bool
	// MetricsRemoteAccess allows non-loopback clients to scrape /metrics.
	MetricsRemoteAccess bool
	// Stripe configuration for SaaS subscription purchase. Only used when SaaS is true.
	StripeAPISecret     string
	StripeWebhookSecret string
	// LicensePrivateKey is the PEM-encoded RSA private key for signing license JWTs (SaaS only).
	LicensePrivateKey string
	// DataDir is the directory stores the data including Bytebase's own database, backups, etc.
	DataDir string
	// HA replica mode.
	HA bool
	// Enable debug level. Only works when SaaS is true.
	Debug bool

	// Version is the bytebase's server version
	Version string
	// Git commit hash of the build
	GitCommit string
	// PgURL is the optional external PostgreSQL instance connection url
	PgURL string
	// SampleProjectInstancePgURL is the Cloud PostgreSQL target for managed
	// Sample Project Instances. It is read only from the environment and must
	// never be logged.
	SampleProjectInstancePgURL string
	// EgressIPs are the public addresses Bytebase connects to databases from,
	// shown in the instance form for users to allow in their firewall. Each
	// entry is an IP address or a CIDR range.
	EgressIPs []string

	// LastActiveTS is the service last active timestamp, any API calls will refresh this value.
	LastActiveTS atomic.Int64
	// Unique ID per Bytebase replica run.
	ReplicaID string
	// Whether the server is running in a docker container.
	IsDocker bool

	// can be set in runtime
	RuntimeDebug atomic.Bool
	// RuntimeMemoryProfileThreshold is the memory threshold in bytes for the server to trigger a pprof memory profile.
	// can be set in runtime
	// 0 means no threshold.
	RuntimeMemoryProfileThreshold atomic.Uint64
	// RuntimeEnableAuditLogStdout enables audit logging to stdout in structured JSON format.
	// can be set in runtime via workspace setting
	RuntimeEnableAuditLogStdout atomic.Bool
}

// UseEmbedDB returns whether to use embedDB.
func (prof *Profile) UseEmbedDB() bool {
	return len(prof.PgURL) == 0
}

var saasFeatureControlMap = map[string]bool{}

// IsFeatureUnavailable returns if the feature is unavailable in SaaS mode.
func (prof *Profile) IsFeatureUnavailable(feature string) bool {
	return prof.SaaS && saasFeatureControlMap[feature]
}

// ParseEgressIPs reads a comma-separated list of IP addresses and CIDR ranges.
// It returns the valid entries in order and the entries it dropped.
func ParseEgressIPs(raw string) (valid []string, invalid []string) {
	for entry := range strings.SplitSeq(raw, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if addr, err := netip.ParseAddr(entry); err == nil {
			valid = append(valid, addr.String())
			continue
		}
		if prefix, err := netip.ParsePrefix(entry); err == nil {
			valid = append(valid, prefix.String())
			continue
		}
		invalid = append(invalid, entry)
	}
	return valid, invalid
}
