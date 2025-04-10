package v1

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/type/expr"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/bytebase/bytebase/backend/base"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
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
		roles    []*v1pb.Role
		wantErr  bool
	}{
		// Empty binding list.
		{
			bindings: []*v1pb.Binding{},
			wantErr:  true,
		},
		// Invalid project role.
		{
			bindings: []*v1pb.Binding{
				{
					Role: "roles/haha",
				},
			},
			roles: []*v1pb.Role{
				{
					Name: "roles/projectOwner",
				},
			},
			wantErr: true,
		},
		// Each binding must contain at least one member.
		{
			bindings: []*v1pb.Binding{
				{
					Role:    "roles/projectOwner",
					Members: []string{"user:bytebase"},
				},
				{
					Role:    "role/projectDeveloper",
					Members: []string{},
				},
			},
			roles: []*v1pb.Role{
				{
					Name: "roles/projectOwner",
				},
				{
					Name: "role/projectDeveloper",
				},
			},
			wantErr: true,
		},
		// Must contain one owner binding.
		{
			bindings: []*v1pb.Binding{
				{
					Role:    "role/projectDeveloper",
					Members: []string{"user:bytebase"},
				},
			},
			roles: []*v1pb.Role{
				{
					Name: "roles/projectOwner",
				},
				{
					Name: "role/projectDeveloper",
				},
			},
			wantErr: true,
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
			roles: []*v1pb.Role{
				{
					Name: "roles/projectOwner",
				},
				{
					Name: "role/projectDeveloper",
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
			roles: []*v1pb.Role{
				{
					Name: "roles/projectOwner",
				},
				{
					Name: "role/projectDeveloper",
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
					Role:    "role/projectDeveloper",
					Members: []string{"user:foo"},
				},
			},
			roles: []*v1pb.Role{
				{
					Name: "roles/projectOwner",
				},
				{
					Name: "role/projectDeveloper",
				},
			},
			wantErr: false,
		},
	}

	a := require.New(t)
	// Mock an empty project service to test the validateBindings function.
	projectService := NewProjectService(nil, nil, nil, nil)
	for _, tt := range tests {
		err := projectService.validateBindings(tt.bindings, tt.roles, nil)
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
		err := validateIAMPolicyExpression(tt.expr, tt.maximumRoleExpiration)
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

	for _, test := range tests {
		deltas := findIamPolicyDeltas(test.oldPolicy, test.newIamPolicy)
		if !cmp.Equal(test.want, deltas, cmpopts.IgnoreUnexported(v1pb.BindingDelta{}, expr.Expr{})) {
			t.Fatalf("%v != %v", test.want, deltas)
		}
	}
}

func TestListProjectFilter(t *testing.T) {
	testCases := []struct {
		input string
		want  *store.ListResourceFilter
		error error
	}{
		{
			input: `title == "sample project"`,
			error: status.Errorf(codes.InvalidArgument, "unsupport variable %q", "title"),
		},
		{
			input: `name == "sample project"`,
			want: &store.ListResourceFilter{
				Where: `(project.name = $1)`,
				Args:  []any{"sample project"},
			},
		},
		{
			input: `name.matches("Sample")`,
			want: &store.ListResourceFilter{
				Where: `(LOWER(project.name) LIKE '%sample%')`,
			},
		},
		{
			input: `resource_id == "sample-project"`,
			want: &store.ListResourceFilter{
				Where: `(project.resource_id = $1)`,
				Args:  []any{"sample-project"},
			},
		},
		{
			input: `resource_id.matches("sample")`,
			want: &store.ListResourceFilter{
				Where: `(LOWER(project.resource_id) LIKE '%sample%')`,
			},
		},
		{
			input: `exclude_default == true`,
			want: &store.ListResourceFilter{
				Where: `(project.resource_id != $1)`,
				Args:  []any{base.DefaultProjectID},
			},
		},
		{
			input: `(name.matches("sample") || resource_id.matches("Sample")) && exclude_default == true`,
			want: &store.ListResourceFilter{
				Where: `(((LOWER(project.name) LIKE '%sample%') OR (LOWER(project.resource_id) LIKE '%sample%')) AND (project.resource_id != $1))`,
				Args:  []any{base.DefaultProjectID},
			},
		},
	}

	for _, tc := range testCases {
		filter, err := getListProjectFilter(tc.input)
		if tc.error != nil {
			require.Error(t, err)
			require.Equal(t, tc.error, err)
		} else {
			require.NoError(t, err)
			require.Equal(t, tc.want.Where, filter.Where)
		}
	}
}
