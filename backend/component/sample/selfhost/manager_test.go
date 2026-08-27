package selfhost

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/sample"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

func TestCheckAvailableRequiresEmbeddedPostgres(t *testing.T) {
	manager := NewManager(nil, &config.Profile{}, nil, sample.ManagerOptions{})
	require.NoError(t, manager.CheckAvailable(context.Background()))

	manager.profile.PgURL = "postgresql://metadata.example.com/bytebase"
	err := manager.CheckAvailable(context.Background())
	require.Error(t, err)
	require.Equal(t, sample.FailureFailedPrecondition, sample.FailureKindOf(err))
}

func TestNewPayloadCreatesOneProjectScopedInstance(t *testing.T) {
	environmentID := "test"
	payload, err := newPayload(bytes.NewReader([]byte("01234567")), "project-a", &environmentID)
	require.NoError(t, err)
	require.Len(t, payload.Instances, 1)
	require.Equal(t, "sample-3031323334353637", payload.Instances[0].InstanceId)
	require.NotNil(t, payload.Instances[0].ProjectId)
	require.Equal(t, "project-a", payload.Instances[0].GetProjectId())
	require.Equal(t, int32(0), payload.Instances[0].PortOffset)
	require.Equal(t, sampleDatabaseTest, payload.Instances[0].DatabaseName)
	require.Equal(t, &environmentID, payload.Instances[0].EnvironmentId)
}

func TestDecodeAcceptsOnlyCurrentPayload(t *testing.T) {
	completeEntry := func(id string) *storepb.SelfHostSampleInstanceSetupPayload_Instance {
		return managedEntry(id, "Sample Instance", 0, sampleDatabaseTest)
	}
	projectID := "project-a"
	currentEntry := completeEntry("sample-one")
	currentEntry.ProjectId = &projectID
	tests := []struct {
		name    string
		payload *storepb.SelfHostSampleInstanceSetupPayload
		wantErr bool
	}{
		{
			name: "current",
			payload: &storepb.SelfHostSampleInstanceSetupPayload{
				Instances: []*storepb.SelfHostSampleInstanceSetupPayload_Instance{currentEntry},
			},
		},
		{
			name: "single member without project owner",
			payload: &storepb.SelfHostSampleInstanceSetupPayload{
				Instances: []*storepb.SelfHostSampleInstanceSetupPayload_Instance{completeEntry("sample-one")},
			},
			wantErr: true,
		},
		{
			name: "multiple members",
			payload: &storepb.SelfHostSampleInstanceSetupPayload{
				Instances: []*storepb.SelfHostSampleInstanceSetupPayload_Instance{currentEntry, currentEntry},
			},
			wantErr: true,
		},
		{
			name:    "missing members",
			payload: &storepb.SelfHostSampleInstanceSetupPayload{},
			wantErr: true,
		},
		{
			name: "incomplete member",
			payload: &storepb.SelfHostSampleInstanceSetupPayload{
				Instances: []*storepb.SelfHostSampleInstanceSetupPayload_Instance{{InstanceId: "sample-one", ProjectId: &projectID}},
			},
			wantErr: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			encoded, err := protojson.Marshal(test.payload)
			require.NoError(t, err)
			_, err = decode(&store.SampleInstanceSetupMessage{Payload: encoded})
			if test.wantErr {
				require.Error(t, err)
				require.Equal(t, sample.FailureFailedPrecondition, sample.FailureKindOf(err))
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestSampleConfigUsesManagedDirectoryAndPortOffset(t *testing.T) {
	profile := &config.Profile{DataDir: t.TempDir(), Port: 8080}
	entry := managedEntry("sample-0123456789abcdef", "Test Sample Instance", 0, sampleDatabaseTest)
	config := sampleConfig(profile, entry)
	require.Equal(t, filepath.Join(profile.DataDir, "pgdata-sample-managed", entry.InstanceId), config.DataDir)
	require.Equal(t, 8083, config.Port)
	require.Equal(t, sampleDatabaseTest, config.DatabaseName)
}
