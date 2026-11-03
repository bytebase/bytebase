//nolint:revive
package util

import (
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

func TestQuotePGValueKeepsTheValueWhole(t *testing.T) {
	for _, value := range []string{
		"plain",
		"with space",
		"it's",
		`back\slash`,
		"a=b c=d",
		"x' host='other.example.com",
	} {
		config, err := pgconn.ParseConfig("application_name=" + QuotePGValue(value))
		require.NoError(t, err, value)
		require.Equal(t, value, config.RuntimeParams["application_name"], value)
	}
}

func TestQuotePGValueTrimsSurroundingWhitespace(t *testing.T) {
	config, err := pgconn.ParseConfig("application_name=" + QuotePGValue("  bytebase \t"))
	require.NoError(t, err)
	require.Equal(t, "bytebase", config.RuntimeParams["application_name"])
}

func TestPGHostPortKeepsHostListsAddressesAndSockets(t *testing.T) {
	hostPort, err := PGHostPort("a.example.com,b.example.com", "5432,5433")
	require.NoError(t, err)
	config, err := pgconn.ParseConfig(hostPort + " sslmode=disable")
	require.NoError(t, err)
	require.Equal(t, "a.example.com", config.Host)
	require.Equal(t, uint16(5432), config.Port)
	require.Len(t, config.Fallbacks, 1)
	require.Equal(t, "b.example.com", config.Fallbacks[0].Host)
	require.Equal(t, uint16(5433), config.Fallbacks[0].Port)

	for _, host := range []string{"::1", "/var/run/postgresql"} {
		hostPort, err := PGHostPort(host, "5432")
		require.NoError(t, err, host)
		config, err := pgconn.ParseConfig(hostPort)
		require.NoError(t, err, host)
		require.Equal(t, host, config.Host)
	}
}

func TestPGHostPortRefusesBlankHostOrPort(t *testing.T) {
	for _, blank := range []string{"", "  "} {
		_, err := PGHostPort(blank, "5432")
		require.ErrorContains(t, err, "host must be set")
		_, err = PGHostPort("db.example.com", blank)
		require.ErrorContains(t, err, "port must be set")
	}
}
