package v1

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/type/expr"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/durationpb"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
)

func TestValidateMembers(t *testing.T) {
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
		err := validateBindings(tt.bindings, tt.roles, nil)
		if tt.wantErr {
			a.Error(err)
		} else {
			a.NoError(err)
		}
	}
}

func TestValidateIAMPolicyExpression(t *testing.T) {
	timeNow := time.Now()
	tests := []struct {
		expr                  string
		maximumRoleExpiration *durationpb.Duration
		wantErr               bool
	}{
		{
			expr:                  fmt.Sprintf("request.time < timestamp(\"%s\")", timeNow.AddDate(0, 0, 15).Format(time.RFC3339)),
			maximumRoleExpiration: &durationpb.Duration{Seconds: 60 * 60 * 24 * 30}, // 30 days
		},
		{
			expr:                  fmt.Sprintf("request.time < timestamp(\"%s\")", timeNow.AddDate(0, 0, 60).Format(time.RFC3339)),
			maximumRoleExpiration: &durationpb.Duration{Seconds: 60 * 60 * 24 * 30},
			wantErr:               true,
		},
	}

	for _, tt := range tests {
		err := validateExpirationInExpression(tt.expr, tt.maximumRoleExpiration)
		if tt.wantErr {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
		}
	}
}

func TestFindIamPolicyDeltas(t *testing.T) {
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
