package validation

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/action/world"
)

func TestChooseSignIn(t *testing.T) {
	t.Parallel()

	const (
		serviceAccount   = "ci@service.bytebase.com"
		workloadIdentity = "ci@workload.bytebase.com"
	)
	tests := []struct {
		name                 string
		world                world.World
		workloadIdentityFlag bool
		want                 world.SignIn
		wantErr              string
	}{
		{
			name:  "access token beats everything, as before",
			world: world.World{AccessToken: "token", ServiceAccount: serviceAccount, ServiceAccountSecret: "secret", WorkloadIdentity: workloadIdentity},
			// Every workflow generated before this sign-in existed sets both.
			workloadIdentityFlag: true,
			want:                 world.SignInAccessToken,
		},
		{
			name:                 "the workload identity flag beats a service account",
			world:                world.World{ServiceAccount: serviceAccount, ServiceAccountSecret: "secret", WorkloadIdentity: workloadIdentity},
			workloadIdentityFlag: true,
			want:                 world.SignInWorkloadIdentity,
		},
		{
			name:  "a service account beats the workload identity variable",
			world: world.World{ServiceAccount: serviceAccount, ServiceAccountSecret: "secret", WorkloadIdentity: workloadIdentity},
			want:  world.SignInServiceAccount,
		},
		{
			name:  "the workload identity variable alone",
			world: world.World{WorkloadIdentity: workloadIdentity},
			want:  world.SignInWorkloadIdentity,
		},
		{
			name:  "a service account alone, as before",
			world: world.World{ServiceAccount: serviceAccount, ServiceAccountSecret: "secret"},
			want:  world.SignInServiceAccount,
		},
		{
			name:    "a service account without its secret does not fall through",
			world:   world.World{ServiceAccount: serviceAccount, WorkloadIdentity: workloadIdentity},
			wantErr: "service-account-secret is required",
		},
		{
			name:  "a stray secret does not block the workload identity variable",
			world: world.World{ServiceAccountSecret: "secret", WorkloadIdentity: workloadIdentity},
			want:  world.SignInWorkloadIdentity,
		},
		{
			name:    "nothing set names every method",
			wantErr: "--workload-identity, --service-account with --service-account-secret, or --access-token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := chooseSignIn(&tt.world, tt.workloadIdentityFlag)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

// Not parallel: ValidateFlags reads the process environment.
func TestValidateFlagsSignIn(t *testing.T) {
	const (
		serviceAccount   = "ci@service.bytebase.com"
		workloadIdentity = "ci@workload.bytebase.com"
	)
	tests := []struct {
		name         string
		env          map[string]string
		flag         string
		url          string
		want         world.SignIn
		wantAudience string
		wantLog      []string
		wantNoLog    []string
	}{
		{
			name: "a generated workflow from before: variable beside an access token",
			env:  map[string]string{"BYTEBASE_ACCESS_TOKEN": "token", "BYTEBASE_WORKLOAD_IDENTITY": workloadIdentity},
			url:  "https://bytebase.example.com",
			want: world.SignInAccessToken, wantAudience: "bytebase",
			wantNoLog: []string{"level=WARN"},
		},
		{
			name: "the variable loses to a service account in the environment, with a warning",
			env:  map[string]string{"BYTEBASE_SERVICE_ACCOUNT": serviceAccount, "BYTEBASE_SERVICE_ACCOUNT_SECRET": "secret", "BYTEBASE_WORKLOAD_IDENTITY": workloadIdentity},
			url:  "https://bytebase.example.com",
			want: world.SignInServiceAccount, wantAudience: "bytebase",
			wantLog:   []string{"signing in with a service account", "are both set"},
			wantNoLog: []string{"secret"},
		},
		{
			name: "the flag beats a service account in the environment, with a warning",
			env:  map[string]string{"BYTEBASE_SERVICE_ACCOUNT": serviceAccount, "BYTEBASE_SERVICE_ACCOUNT_SECRET": "secret", "BYTEBASE_WORKLOAD_IDENTITY_AUDIENCE": "custom"},
			flag: workloadIdentity,
			url:  "https://bytebase.example.com",
			want: world.SignInWorkloadIdentity, wantAudience: "custom",
			wantLog: []string{"signing in with a workload identity", "are both set"},
		},
		{
			name: "an ID token over http is warned about",
			env:  map[string]string{"BYTEBASE_WORKLOAD_IDENTITY": workloadIdentity},
			url:  "http://bytebase.example.com",
			want: world.SignInWorkloadIdentity, wantAudience: "bytebase",
			wantLog: []string{"over http"},
		},
		{
			name: "http to loopback is not",
			env:  map[string]string{"BYTEBASE_WORKLOAD_IDENTITY": workloadIdentity},
			url:  "http://127.0.0.1:8080",
			want: world.SignInWorkloadIdentity, wantAudience: "bytebase",
			wantNoLog: []string{"level=WARN"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, key := range []string{"BYTEBASE_ACCESS_TOKEN", "BYTEBASE_SERVICE_ACCOUNT", "BYTEBASE_SERVICE_ACCOUNT_SECRET", "BYTEBASE_WORKLOAD_IDENTITY", "BYTEBASE_WORKLOAD_IDENTITY_AUDIENCE"} {
				t.Setenv(key, tt.env[key])
			}
			var log bytes.Buffer
			w := &world.World{
				Logger:           slog.New(slog.NewTextHandler(&log, nil)),
				Platform:         world.LocalPlatform,
				URL:              tt.url,
				Project:          "projects/example",
				WorkloadIdentity: tt.flag,
			}

			require.NoError(t, ValidateFlags(w))
			require.Equal(t, tt.want, w.SignIn)
			require.Equal(t, tt.wantAudience, w.WorkloadIdentityAudience)
			for _, want := range tt.wantLog {
				require.Contains(t, log.String(), want)
			}
			for _, unwanted := range tt.wantNoLog {
				require.NotContains(t, log.String(), unwanted)
			}
		})
	}
}

func TestValidateTargets(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		targets []string
		wantErr string
	}{
		{
			name: "workspace database targets",
			targets: []string{
				"instances/test/databases/hr_test",
				"instances/prod/databases/hr_prod",
			},
		},
		{
			name: "project instance database targets",
			targets: []string{
				"projects/hr/instances/test/databases/hr_test",
				"projects/hr/instances/prod/databases/hr_prod",
			},
		},
		{
			name: "mixed workspace and project instance database targets",
			targets: []string{
				"instances/test/databases/hr_test",
				"projects/hr/instances/prod/databases/hr_prod",
			},
		},
		{
			name:    "database target and database group",
			targets: []string{"projects/hr/instances/test/databases/hr_test", "projects/hr/databaseGroups/all"},
			wantErr: "either database targets or a database group target",
		},
		{
			name:    "multiple database groups",
			targets: []string{"projects/hr/databaseGroups/one", "projects/hr/databaseGroups/two"},
			wantErr: "single database group target",
		},
		{
			name:    "malformed project instance target",
			targets: []string{"projects/hr/instances//databases/hr_test"},
			wantErr: "invalid target format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateTargets(tt.targets)
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}
