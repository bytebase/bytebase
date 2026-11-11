package plancheck

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

func TestDeriveCheckTargetsAppliesCISampling(t *testing.T) {
	ctx := context.Background()
	project := &store.ProjectMessage{
		ResourceID: "p",
		Setting:    &storepb.Project{},
	}
	plan := &store.PlanMessage{
		Config: &storepb.PlanConfig{
			Specs: []*storepb.PlanConfig_Spec{{
				Id: "spec-1",
				Config: &storepb.PlanConfig_Spec_ChangeDatabaseConfig{
					ChangeDatabaseConfig: &storepb.PlanConfig_ChangeDatabaseConfig{
						Targets: []string{
							"instances/i/databases/a",
							"instances/i/databases/b",
							"instances/i/databases/c",
						},
					},
				},
			}},
		},
	}

	full, err := DeriveCheckTargets(ctx, nil, project, plan, nil)
	require.NoError(t, err)
	require.Len(t, full, 3, "a project with no sampling limit covers every target")
	for _, target := range full {
		require.Equal(t, "spec-1", target.SpecID, "a review result anchors on its spec")
	}

	project.Setting.CiSamplingSize = 1
	sampled, err := DeriveCheckTargets(ctx, nil, project, plan, nil)
	require.NoError(t, err)
	require.Len(t, sampled, 1)
	require.Equal(t, "instances/i/databases/a", sampled[0].Target, "the sample is the first targets of the spec")
}
