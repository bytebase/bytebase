package v1

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	colorpb "google.golang.org/genproto/googleapis/type/color"
	"google.golang.org/genproto/googleapis/type/expr"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/sample"
	"github.com/bytebase/bytebase/backend/component/sample/selfhost"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
)

func TestValidateMembers(t *testing.T) {
	t.Parallel()
	tests := []struct {
		member  string
		wantErr bool
	}{
		{
			member:  "",
			wantErr: true,
		},
		{
			member:  "foo",
			wantErr: true,
		},
		{
			member:  "user",
			wantErr: true,
		},
		{
			member:  "user:foo",
			wantErr: false,
		},
		{
			member:  "group:team@example.com",
			wantErr: false,
		},
		{
			member:  "serviceAccount:sa@service.bytebase.com",
			wantErr: false,
		},
		{
			member:  "serviceAccount:sa@project.service.bytebase.com",
			wantErr: false,
		},
		{
			member:  "workloadIdentity:wi@workload.bytebase.com",
			wantErr: false,
		},
		{
			member:  "workloadIdentity:wi@project.workload.bytebase.com",
			wantErr: false,
		},
		{
			member:  "serviceAccount:",
			wantErr: true,
		},
		{
			member:  "workloadIdentity:",
			wantErr: true,
		},
	}

	a := require.New(t)
	for _, tt := range tests {
		err := validateMember(tt.member)
		if tt.wantErr {
			a.Error(err)
		} else {
			a.NoError(err)
		}
	}
}

func TestValidateBindings(t *testing.T) {
	t.Parallel()
	tests := []struct {
		bindings []*v1pb.Binding
		roles    []*store.RoleMessage
		wantErr  bool
	}{
		// Empty binding list.
		{
			bindings: []*v1pb.Binding{},
			wantErr:  false,
		},
		// Invalid project role.
		{
			bindings: []*v1pb.Binding{
				{
					Role: "roles/haha",
				},
			},
			roles: []*store.RoleMessage{
				{
					ResourceID: "projectOwner",
				},
			},
			wantErr: true,
		},
		// Binding members can be empty.
		{
			bindings: []*v1pb.Binding{
				{
					Role:    "roles/projectOwner",
					Members: []string{"user:bytebase"},
				},
				{
					Role:    "roles/projectDeveloper",
					Members: []string{},
				},
			},
			roles: []*store.RoleMessage{
				{
					ResourceID: "projectOwner",
				},
				{
					ResourceID: "projectDeveloper",
				},
			},
			wantErr: false,
		},
		{
			bindings: []*v1pb.Binding{
				{
					Role:    "roles/projectOwner",
					Members: []string{},
				},
			},
			roles: []*store.RoleMessage{
				{
					ResourceID: "projectOwner",
				},
				{
					ResourceID: "projectDeveloper",
				},
			},
			wantErr: false,
		},
		// Invalid condition
		{
			bindings: []*v1pb.Binding{
				{
					Role:    "roles/projectOwner",
					Members: []string{"user:bytebase"},
					Condition: &expr.Expr{
						Expression: `database == "employee" && environment_name == "test"`,
					},
				},
			},
			roles: []*store.RoleMessage{
				{
					ResourceID: "projectOwner",
				},
			},
			wantErr: true,
		},
		// Must contain one owner binding.
		{
			bindings: []*v1pb.Binding{
				{
					Role:    "roles/projectDeveloper",
					Members: []string{"user:bytebase"},
				},
			},
			roles: []*store.RoleMessage{
				{
					ResourceID: "projectOwner",
				},
				{
					ResourceID: "projectDeveloper",
				},
			},
			wantErr: false,
		},
		{
			bindings: []*v1pb.Binding{
				{

					Role:    "roles/projectOwner",
					Members: []string{"user:bytebase"},
				},
				{
					Role:    "roles/projectOwner",
					Members: []string{"user:foo"},
				},
			},
			roles: []*store.RoleMessage{
				{
					ResourceID: "projectOwner",
				},
				{
					ResourceID: "projectDeveloper",
				},
			},
			wantErr: false,
		},
		// Valid case.
		{
			bindings: []*v1pb.Binding{
				{
					Role:    "roles/projectOwner",
					Members: []string{"user:bytebase"},
				},
			},
			roles: []*store.RoleMessage{
				{
					ResourceID: "projectOwner",
				},
				{
					ResourceID: "projectDeveloper",
				},
			},
			wantErr: false,
		},
		{
			bindings: []*v1pb.Binding{
				{
					Role:    "roles/projectOwner",
					Members: []string{"user:bytebase"},
				},
				{
					Role:    "roles/projectDeveloper",
					Members: []string{"user:foo"},
				},
			},
			roles: []*store.RoleMessage{
				{
					ResourceID: "projectOwner",
				},
				{
					ResourceID: "projectDeveloper",
				},
			},
			wantErr: false,
		},
	}

	a := require.New(t)
	for _, tt := range tests {
		err := validateBindings("projects/sample", tt.bindings, tt.roles, nil)
		if tt.wantErr {
			a.Error(err)
		} else {
			a.NoError(err)
		}
	}
}

func TestValidateIAMPolicyExpression(t *testing.T) {
	t.Parallel()
	timeNow := time.Now()
	thirtyDays := &durationpb.Duration{Seconds: 60 * 60 * 24 * 30}
	withinCap := fmt.Sprintf("request.time < timestamp(\"%s\")", timeNow.AddDate(0, 0, 15).Format(time.RFC3339))
	overCap := fmt.Sprintf("request.time < timestamp(\"%s\")", timeNow.AddDate(0, 0, 60).Format(time.RFC3339))
	tests := []struct {
		name              string
		expr              string
		maximumExpiration *durationpb.Duration
		wantErr           bool
	}{
		{
			name:              "within cap",
			expr:              withinCap,
			maximumExpiration: thirtyDays,
		},
		{
			name:              "exceeds cap",
			expr:              overCap,
			maximumExpiration: thirtyDays,
			wantErr:           true,
		},
		{
			name:              "no cap configured",
			expr:              overCap,
			maximumExpiration: nil,
		},
		{
			name:              "bound with database scoping",
			expr:              fmt.Sprintf(`%s && (resource.database == "a" || resource.database == "b")`, withinCap),
			maximumExpiration: thirtyDays,
		},
		{
			name:              "missing request.time",
			expr:              `resource.database == "a"`,
			maximumExpiration: thirtyDays,
			wantErr:           true,
		},
	}

	for _, tt := range tests {
		err := validateExpirationInExpression(tt.expr, tt.maximumExpiration)
		if tt.wantErr {
			require.Error(t, err, tt.name)
		} else {
			require.NoError(t, err, tt.name)
		}
	}
}

func TestFindIamPolicyDeltas(t *testing.T) {
	t.Parallel()
	tests := []struct {
		oldPolicy    *storepb.IamPolicy
		newIamPolicy *storepb.IamPolicy
		want         []*v1pb.BindingDelta
	}{
		// test with redundant roles.
		{
			oldPolicy: &storepb.IamPolicy{
				Bindings: []*storepb.Binding{
					{
						Role: "roles/sqlEditorUser",
						Members: []string{
							"usr103",
						},
						Condition: &expr.Expr{},
					},
					{
						Role: "roles/sqlEditorUser",
						Members: []string{
							"usr103",
						},
						Condition: &expr.Expr{},
					},
				},
			},
			newIamPolicy: &storepb.IamPolicy{
				Bindings: []*storepb.Binding{
					{
						Role: "roles/sqlEditorUser",
						Members: []string{
							"usr103",
						},
						Condition: &expr.Expr{},
					},
				},
			},
			want: nil,
		},
		// simply test remove and add.
		{
			oldPolicy: &storepb.IamPolicy{
				Bindings: []*storepb.Binding{
					{
						Role: "roles/sqlEditorUser",
						Members: []string{
							"usr103",
						},
						Condition: &expr.Expr{
							Expression: "time > 500",
						},
					},
				},
			},
			newIamPolicy: &storepb.IamPolicy{
				Bindings: []*storepb.Binding{
					{
						Role: "roles/sqlEditorUser",
						Members: []string{
							"usr103",
						},
						Condition: &expr.Expr{
							Expression: "time > 1000",
						},
					},
					{
						Role: "roles/projectOwner",
						Members: []string{
							"usr101",
							"usr102",
						},
						Condition: &expr.Expr{},
					},
				},
			},
			want: []*v1pb.BindingDelta{
				{
					Action: v1pb.BindingDelta_ADD,
					Member: "usr103",
					Role:   "roles/sqlEditorUser",
					Condition: &expr.Expr{
						Expression: "time > 1000",
					},
				},
				{
					Action:    v1pb.BindingDelta_ADD,
					Member:    "usr101",
					Role:      "roles/projectOwner",
					Condition: &expr.Expr{},
				},
				{
					Action:    v1pb.BindingDelta_ADD,
					Member:    "usr102",
					Role:      "roles/projectOwner",
					Condition: &expr.Expr{},
				},
				{
					Action: v1pb.BindingDelta_REMOVE,
					Member: "usr103",
					Role:   "roles/sqlEditorUser",
					Condition: &expr.Expr{
						Expression: "time > 500",
					},
				},
			},
		},
	}

	for i, test := range tests {
		deltas := findIamPolicyDeltas(test.oldPolicy, test.newIamPolicy)
		if !cmp.Equal(test.want, deltas, protocmp.Transform()) {
			t.Fatalf("index %d\n%s", i, cmp.Diff(test.want, deltas, protocmp.Transform()))
		}
	}
}

func TestValidateLabels(t *testing.T) {
	tests := []struct {
		name    string
		labels  map[string]string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "Valid labels",
			labels:  map[string]string{"environment": "production", "team": "backend"},
			wantErr: false,
		},
		{
			name:    "Empty labels",
			labels:  map[string]string{},
			wantErr: false,
		},
		{
			name:    "Valid key with empty value",
			labels:  map[string]string{"environment": ""},
			wantErr: false,
		},
		{
			name:    "Key starting with number",
			labels:  map[string]string{"1environment": "production"},
			wantErr: true,
			errMsg:  "invalid label key",
		},
		{
			name:    "Key with uppercase",
			labels:  map[string]string{"Environment": "production"},
			wantErr: true,
			errMsg:  "invalid label key",
		},
		{
			name:    "Key with special characters",
			labels:  map[string]string{"env@prod": "production"},
			wantErr: true,
			errMsg:  "invalid label key",
		},
		{
			name:    "Value with special characters",
			labels:  map[string]string{"environment": "prod@123"},
			wantErr: true,
			errMsg:  "invalid label value",
		},
		{
			name:    "Key too long",
			labels:  map[string]string{strings.Repeat("a", 64): "value"},
			wantErr: true,
			errMsg:  "invalid label key",
		},
		{
			name:    "Value too long",
			labels:  map[string]string{"key": strings.Repeat("a", 64)},
			wantErr: true,
			errMsg:  "invalid label value",
		},
		{
			name: "Too many labels",
			labels: func() map[string]string {
				labels := make(map[string]string)
				for i := 0; i < 65; i++ {
					labels[fmt.Sprintf("key%d", i)] = "value"
				}
				return labels
			}(),
			wantErr: true,
			errMsg:  "maximum 64 labels allowed",
		},
		{
			name:    "Valid underscore and dash",
			labels:  map[string]string{"env_name": "prod-01", "team-name": "backend_01"},
			wantErr: false,
		},
		{
			name:    "Valid mixed case value",
			labels:  map[string]string{"environment": "Production", "region": "US-East-1"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateLabels(tt.labels)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateIssueLabelsColor(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		labels  []*v1pb.Label
		wantErr bool
	}{
		{
			name: "missing color",
			labels: []*v1pb.Label{
				{Value: "review"},
			},
		},
		{
			name: "opaque color",
			labels: []*v1pb.Label{
				{Value: "review", Color: &colorpb.Color{Red: 0.1, Green: 0.2, Blue: 0.3}},
			},
		},
		{
			name: "explicit alpha one",
			labels: []*v1pb.Label{
				{Value: "review", Color: &colorpb.Color{Red: 0.1, Green: 0.2, Blue: 0.3, Alpha: wrapperspb.Float(1)}},
			},
		},
		{
			name: "channel out of range",
			labels: []*v1pb.Label{
				{Value: "review", Color: &colorpb.Color{Red: 1.1, Green: 0.2, Blue: 0.3}},
			},
			wantErr: true,
		},
		{
			name: "transparent color",
			labels: []*v1pb.Label{
				{Value: "review", Color: &colorpb.Color{Red: 0.1, Green: 0.2, Blue: 0.3, Alpha: wrapperspb.Float(0.5)}},
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validateIssueLabels(tc.labels)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestProjectPurgeCleansSampleBeforeDeletingMetadata(t *testing.T) {
	t.Parallel()
	ctx, stores, projectID, instanceID, _ := setupProjectInstanceLifecycleAPITest(t)
	manager := &sampleManagerStub{}
	cleanupErr := errors.New("sample cleanup failed")
	manager.projectPurge = func(ctx context.Context, workspaceID, gotProjectID string) error {
		require.Equal(t, "default", workspaceID)
		require.Equal(t, projectID, gotProjectID)
		instance, err := stores.GetInstance(ctx, &store.FindInstanceMessage{
			Workspace:  workspaceID,
			ProjectID:  &gotProjectID,
			ResourceID: &instanceID,
		})
		require.NoError(t, err)
		require.NotNil(t, instance)
		if cleanupErr != nil {
			return cleanupErr
		}
		return nil
	}
	projectService := NewProjectService(stores, nil, nil, manager)
	projectName := common.FormatProject(projectID)

	_, err := projectService.DeleteProject(ctx, connect.NewRequest(&v1pb.DeleteProjectRequest{Name: projectName}))
	require.NoError(t, err)
	_, err = projectService.DeleteProject(ctx, connect.NewRequest(&v1pb.DeleteProjectRequest{Name: projectName, Purge: true}))
	require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
	project, err := stores.GetProject(ctx, &store.FindProjectMessage{
		Workspace:   "default",
		ResourceID:  &projectID,
		ShowDeleted: true,
	})
	require.NoError(t, err)
	require.NotNil(t, project)

	cleanupErr = nil
	_, err = projectService.DeleteProject(ctx, connect.NewRequest(&v1pb.DeleteProjectRequest{Name: projectName, Purge: true}))
	require.NoError(t, err)
	require.Equal(t, []string{projectID, projectID}, manager.projectPurgeCalls)

	project, err = stores.GetProject(ctx, &store.FindProjectMessage{
		Workspace:   "default",
		ResourceID:  &projectID,
		ShowDeleted: true,
	})
	require.NoError(t, err)
	require.Nil(t, project)
}

func TestProjectPurgeRemovesSelfHostSample(t *testing.T) {
	t.Parallel()
	ctx, stores, projectID, instanceID, databaseName := setupProjectInstanceLifecycleAPITest(t)
	payload, err := protojson.Marshal(&storepb.SelfHostSampleInstanceSetupPayload{
		Instances: []*storepb.SelfHostSampleInstanceSetupPayload_Instance{{
			InstanceId:   instanceID,
			ProjectId:    &projectID,
			Title:        "Sample Project Instance",
			DatabaseName: databaseName,
			RoleName:     "sample-role",
		}},
	})
	require.NoError(t, err)
	const replicaID = "replica-a"
	_, created, err := stores.ReserveSampleInstanceSetup(ctx, &store.SampleInstanceSetupMessage{
		WorkspaceID: "default",
		ReplicaID:   replicaID,
		Payload:     payload,
	})
	require.NoError(t, err)
	require.True(t, created)
	activated, err := stores.ActivateSampleInstanceSetup(ctx, "default", replicaID, []string{projectID}, time.Now(), nil)
	require.NoError(t, err)
	require.True(t, activated)

	dataRoot := t.TempDir()
	dataDir := filepath.Join(dataRoot, "pgdata-sample-managed", instanceID)
	require.NoError(t, os.MkdirAll(dataDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dataDir, "sample-data"), []byte("sample"), 0o644))
	manager := selfhost.NewManager(
		stores,
		&config.Profile{DataDir: dataRoot, Port: 8080},
		nil,
		sample.ManagerOptions{ReplicaID: replicaID},
	)
	projectService := NewProjectService(stores, nil, nil, manager)
	projectName := common.FormatProject(projectID)

	_, err = projectService.DeleteProject(ctx, connect.NewRequest(&v1pb.DeleteProjectRequest{Name: projectName}))
	require.NoError(t, err)
	_, err = projectService.DeleteProject(ctx, connect.NewRequest(&v1pb.DeleteProjectRequest{Name: projectName, Purge: true}))
	require.NoError(t, err)

	_, err = os.Stat(dataDir)
	require.ErrorIs(t, err, os.ErrNotExist)
	setup, err := stores.GetSampleInstanceSetup(ctx, "default")
	require.NoError(t, err)
	require.NotNil(t, setup)
	require.NotNil(t, setup.DeletedAt)
	instances, err := manager.ListInstances(ctx, "default")
	require.NoError(t, err)
	require.Len(t, instances, 1)
	require.Equal(t, common.FormatProjectInstance(projectID, instanceID), instances[0].Name)
	require.Nil(t, instances[0].ExpireTime)
}
