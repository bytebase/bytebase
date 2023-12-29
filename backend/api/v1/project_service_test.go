package v1

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/store"
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
	for _, tt := range tests {
		err := validateBindings(tt.bindings, tt.roles)
		if tt.wantErr {
			a.Error(err)
		} else {
			a.NoError(err)
		}
	}
}

func TestValidateAndConvertToStoreDeploymentSchedule(t *testing.T) {
	tests := []struct {
		name    string
		apiCfg  *v1pb.DeploymentConfig
		wantCfg *store.DeploymentConfigMessage
		wantErr bool
	}{
		{
			name: "complexDeployments",
			apiCfg: &v1pb.DeploymentConfig{
				Title: "DeploymentConfig1",
				Schedule: &v1pb.Schedule{
					Deployments: []*v1pb.ScheduleDeployment{
						{
							Title: "Deployment1",
							Spec: &v1pb.DeploymentSpec{
								LabelSelector: &v1pb.LabelSelector{
									MatchExpressions: []*v1pb.LabelSelectorRequirement{
										{
											Key:      "environment",
											Operator: v1pb.OperatorType_OPERATOR_TYPE_IN,
											Values:   []string{"prod"},
										},
										{
											Key:      "location",
											Operator: v1pb.OperatorType_OPERATOR_TYPE_EXISTS,
											Values:   []string{},
										},
									},
								},
							},
						},
					},
				},
			},
			wantCfg: &store.DeploymentConfigMessage{
				Name: "DeploymentConfig1",
				Schedule: &store.Schedule{
					Deployments: []*store.Deployment{
						{
							Name: "Deployment1",
							Spec: &store.DeploymentSpec{
								Selector: &store.LabelSelector{
									MatchExpressions: []*store.LabelSelectorRequirement{
										{
											Key:      "environment",
											Operator: store.InOperatorType,
											Values:   []string{"prod"},
										},
										{
											Key:      "location",
											Operator: store.ExistsOperatorType,
											Values:   []string{},
										},
									},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "deployment title must not be empty",
			apiCfg: &v1pb.DeploymentConfig{
				Title: "DeploymentConfig1",
				Schedule: &v1pb.Schedule{
					Deployments: []*v1pb.ScheduleDeployment{
						{
							Title: "", // empty title
							Spec: &v1pb.DeploymentSpec{
								LabelSelector: &v1pb.LabelSelector{
									MatchExpressions: []*v1pb.LabelSelectorRequirement{
										{
											Key:      "environment",
											Operator: v1pb.OperatorType_OPERATOR_TYPE_IN,
											Values:   []string{"prod"},
										},
										{
											Key:      "location",
											Operator: v1pb.OperatorType_OPERATOR_TYPE_EXISTS,
											Values:   []string{},
										},
									},
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "In operator must have at least one value",
			apiCfg: &v1pb.DeploymentConfig{
				Title: "DeploymentConfig1",
				Schedule: &v1pb.Schedule{
					Deployments: []*v1pb.ScheduleDeployment{
						{
							Title: "Deployment1",
							Spec: &v1pb.DeploymentSpec{
								LabelSelector: &v1pb.LabelSelector{
									MatchExpressions: []*v1pb.LabelSelectorRequirement{
										{
											Key:      "environment",
											Operator: v1pb.OperatorType_OPERATOR_TYPE_IN,
											Values:   []string{},
										},
									},
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Exists operator must not have any values",
			apiCfg: &v1pb.DeploymentConfig{
				Title: "DeploymentConfig1",
				Schedule: &v1pb.Schedule{
					Deployments: []*v1pb.ScheduleDeployment{
						{
							Title: "Deployment1",
							Spec: &v1pb.DeploymentSpec{
								LabelSelector: &v1pb.LabelSelector{
									MatchExpressions: []*v1pb.LabelSelectorRequirement{
										{
											Key:      "location",
											Operator: v1pb.OperatorType_OPERATOR_TYPE_EXISTS,
											Values:   []string{"South America"},
										},
									},
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "environment label with In operator must have exactly one value",
			apiCfg: &v1pb.DeploymentConfig{
				Title: "DeploymentConfig1",
				Schedule: &v1pb.Schedule{
					Deployments: []*v1pb.ScheduleDeployment{
						{
							Title: "Deployment1",
							Spec: &v1pb.DeploymentSpec{
								LabelSelector: &v1pb.LabelSelector{
									MatchExpressions: []*v1pb.LabelSelectorRequirement{
										{
											Key:      "environment",
											Operator: v1pb.OperatorType_OPERATOR_TYPE_IN,
											Values:   []string{"prod", "test"},
										},
										{
											Key:      "location",
											Operator: v1pb.OperatorType_OPERATOR_TYPE_EXISTS,
											Values:   []string{},
										},
									},
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		cfg, err := validateAndConvertToStoreDeploymentSchedule(tc.apiCfg)
		if tc.wantErr {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
			require.Equal(t, tc.wantCfg, cfg)
		}
	}
}
