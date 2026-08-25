package selfhost

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/component/config"
)

func TestNewPayloadCreatesTwoShortRandomInstances(t *testing.T) {
	payload, err := newPayload(bytes.NewReader([]byte("0123456789abcdef")), "project-a", nil, nil)
	require.NoError(t, err)
	require.Equal(t, "project-a", payload.DatabaseProjectId)
	require.Len(t, payload.Instances, 2)
	require.Equal(t, "sample-3031323334353637", payload.Instances[0].InstanceId)
	require.Equal(t, "sample-3839616263646566", payload.Instances[1].InstanceId)
	require.Equal(t, int32(0), payload.Instances[0].PortOffset)
	require.Equal(t, int32(1), payload.Instances[1].PortOffset)
	require.Equal(t, sampleDatabaseTest, payload.Instances[0].DatabaseName)
	require.Equal(t, sampleDatabaseProd, payload.Instances[1].DatabaseName)
}

func TestSampleConfigUsesManagedDirectoryAndPortOffset(t *testing.T) {
	profile := &config.Profile{DataDir: t.TempDir(), Port: 8080}
	entry := managedEntry("sample-0123456789abcdef", "Test Sample Instance", 0, sampleDatabaseTest)
	config := sampleConfig(profile, entry)
	require.Equal(t, filepath.Join(profile.DataDir, "pgdata-sample-managed", entry.InstanceId), config.DataDir)
	require.Equal(t, 8083, config.Port)
	require.Equal(t, sampleDatabaseTest, config.DatabaseName)
}
