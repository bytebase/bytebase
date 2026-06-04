package command

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/bytebase/bytebase/action/world"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
)

func TestCheckVersionCompatibility(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		cliVersion    string
		serverVersion string
		wantError     string
	}{
		{
			name:          "same release version",
			cliVersion:    "3.14.0",
			serverVersion: "3.14.0",
		},
		{
			name:          "release version within two minor versions",
			cliVersion:    "3.13.0",
			serverVersion: "3.15.0",
		},
		{
			name:          "release version outside two minor versions",
			cliVersion:    "3.12.0",
			serverVersion: "3.15.0",
			wantError:     "outside the compatibility window",
		},
		{
			name:          "newer release version is outside window",
			cliVersion:    "3.17.0",
			serverVersion: "3.15.0",
			wantError:     "outside the compatibility window",
		},
		{
			name:          "release version with different major is outside window",
			cliVersion:    "2.15.0",
			serverVersion: "3.15.0",
			wantError:     "outside the compatibility window",
		},
		{
			name:          "plain cloud action is not a dated cloud build",
			cliVersion:    "cloud-20260604",
			serverVersion: "cloud",
			wantError:     "unable to parse",
		},
		{
			name:          "plain cloud server is not a dated cloud build",
			cliVersion:    "cloud",
			serverVersion: "cloud-20260604",
			wantError:     "unable to parse",
		},
		{
			name:          "dated cloud builds match",
			cliVersion:    "cloud-20260604",
			serverVersion: "cloud-20260604",
		},
		{
			name:          "dated cloud builds within seven days",
			cliVersion:    "cloud-20260604",
			serverVersion: "cloud-20260611",
		},
		{
			name:          "newer dated cloud build is outside window",
			cliVersion:    "cloud-20260611",
			serverVersion: "cloud-20260604",
			wantError:     "outside the compatibility window",
		},
		{
			name:          "dated cloud builds outside seven days",
			cliVersion:    "cloud-20260603",
			serverVersion: "cloud-20260611",
			wantError:     "outside the compatibility window",
		},
		{
			name:          "cloud action is not compatible with release server",
			cliVersion:    "cloud-20260604",
			serverVersion: "3.14.0",
			wantError:     "unable to compare",
		},
		{
			name:          "invalid release version is parse error",
			cliVersion:    "3",
			serverVersion: "3.14.0",
			wantError:     "unable to parse",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client := newActuatorInfoTestClient(t, tt.serverVersion)
			w := &world.World{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}

			err := checkVersionCompatibility(w, client, tt.cliVersion)
			if tt.wantError == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.wantError)
		})
	}
}

func TestRecommendedActionTag(t *testing.T) {
	t.Parallel()

	releaseVersion, err := parseVersion("3.14.0")
	require.NoError(t, err)
	require.Equal(t, "3.14.0", recommendedActionTag(releaseVersion))

	cloudVersion, err := parseVersion("cloud-20260604")
	require.NoError(t, err)
	require.Equal(t, "cloud", recommendedActionTag(cloudVersion))
}

func TestCheckVersionCompatibilityReturnsErrorOutsideWindow(t *testing.T) {
	t.Parallel()

	client := newActuatorInfoTestClient(t, "3.15.0")
	w := &world.World{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}

	err := checkVersionCompatibility(w, client, "3.12.0")
	require.Error(t, err)
	require.Contains(t, err.Error(), "outside the compatibility window")
	require.Contains(t, err.Error(), "same major version")
	require.Contains(t, err.Error(), "within 2 minor versions")
}

func TestCheckVersionCompatibilityReturnsUsefulCloudError(t *testing.T) {
	t.Parallel()

	client := newActuatorInfoTestClient(t, "cloud-20260604")
	w := &world.World{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}

	err := checkVersionCompatibility(w, client, "cloud-20260527")
	require.Error(t, err)
	require.Contains(t, err.Error(), "cloud-YYYYMMDD")
	require.Contains(t, err.Error(), "within 7 days")
}

func TestCheckVersionCompatibilityReturnsParseError(t *testing.T) {
	t.Parallel()

	client := newActuatorInfoTestClient(t, "cloud")
	w := &world.World{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}

	err := checkVersionCompatibility(w, client, "cloud-20260604")
	require.Error(t, err)
	require.Contains(t, err.Error(), "unable to parse")
	require.NotContains(t, err.Error(), "outside the compatibility window")
}

func TestCheckVersionCompatibilityAllowsVersionsWithinWindow(t *testing.T) {
	t.Parallel()

	client := newActuatorInfoTestClient(t, "cloud-20260604")
	w := &world.World{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}

	require.NoError(t, checkVersionCompatibility(w, client, "cloud-20260528"))
}

func newActuatorInfoTestClient(t *testing.T, version string) *client {
	t.Helper()

	path, handler := v1connect.NewActuatorServiceHandler(&actuatorInfoTestService{version: version})
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	client, err := newClient(server.URL, "token", "", "", defaultClientOptions())
	require.NoError(t, err)
	t.Cleanup(client.close)
	return client
}

type actuatorInfoTestService struct {
	version string
}

func (s *actuatorInfoTestService) GetActuatorInfo(context.Context, *connect.Request[v1pb.GetActuatorInfoRequest]) (*connect.Response[v1pb.ActuatorInfo], error) {
	return connect.NewResponse(&v1pb.ActuatorInfo{Version: s.version}), nil
}

func (*actuatorInfoTestService) SetupSample(context.Context, *connect.Request[v1pb.SetupSampleRequest]) (*connect.Response[emptypb.Empty], error) {
	return connect.NewResponse(&emptypb.Empty{}), nil
}

func (*actuatorInfoTestService) DeleteCache(context.Context, *connect.Request[v1pb.DeleteCacheRequest]) (*connect.Response[emptypb.Empty], error) {
	return connect.NewResponse(&emptypb.Empty{}), nil
}
