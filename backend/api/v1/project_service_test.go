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
											Key:      "bb.environment",
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
											Key:      "bb.environment",
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
											Key:      "bb.environment",
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
											Key:      "bb.environment",
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
											Key:      "bb.environment",
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
