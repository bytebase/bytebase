package v1

import (
	"testing"

	"github.com/stretchr/testify/require"

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
					Role: v1pb.ProjectRole_PROJECT_ROLE_UNSPECIFIED,
				},
			},
			wantErr: true,
		},
		// Each binding must contain at least one member.
		{
			bindings: []*v1pb.Binding{
				{
					Role:    v1pb.ProjectRole_PROJECT_ROLE_OWNER,
					Members: []string{"user:bytebase"},
				},
				{
					Role:    v1pb.ProjectRole_PROJECT_ROLE_DEVELOPER,
					Members: []string{},
				},
			},
			wantErr: true,
		},
		// Must contain one owner binding.
		{
			bindings: []*v1pb.Binding{
				{
					Role:    v1pb.ProjectRole_PROJECT_ROLE_DEVELOPER,
					Members: []string{"user:bytebase"},
				},
			},
			wantErr: true,
		},
		// We have not merge the binding by the same role yet, so the roles in each binding must be unique.
		{
			bindings: []*v1pb.Binding{
				{
					Role:    v1pb.ProjectRole_PROJECT_ROLE_OWNER,
					Members: []string{"user:bytebase"},
				},
				{
					Role:    v1pb.ProjectRole_PROJECT_ROLE_OWNER,
					Members: []string{"user:foo"},
				},
			},
			wantErr: true,
		},
		// Valid case.
		{
			bindings: []*v1pb.Binding{
				{
					Role:    v1pb.ProjectRole_PROJECT_ROLE_OWNER,
					Members: []string{"user:bytebase"},
				},
			},
			wantErr: false,
		},
		{
			bindings: []*v1pb.Binding{
				{
					Role:    v1pb.ProjectRole_PROJECT_ROLE_OWNER,
					Members: []string{"user:bytebase"},
				},
				{
					Role:    v1pb.ProjectRole_PROJECT_ROLE_DEVELOPER,
					Members: []string{"user:foo"},
				},
			},
			wantErr: false,
		},
	}

	a := require.New(t)
	for _, tt := range tests {
		err := validateBindings(tt.bindings)
		if tt.wantErr {
			a.Error(err)
		} else {
			a.NoError(err)
		}
	}
}
