package plancheck

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

// Plan checks are CI validation and honor the project's CI sampling limit;
// a review run's DONE means every (spec, target) unit was evaluated, so
// review derivation must not sample.
func TestDeriveReviewTargetsSkipsCISampling(t *testing.T) {
	ctx := context.Background()
	project := &store.ProjectMessage{
		ResourceID: "p",
		Setting:    &storepb.Project{CiSamplingSize: 1},
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

	sampled, err := DeriveCheckTargets(ctx, nil, project, plan, nil)
	require.NoError(t, err)
	require.Len(t, sampled, 1, "plan checks apply the CI sampling limit")

	full, err := DeriveReviewTargets(ctx, nil, project, plan, nil)
	require.NoError(t, err)
	require.Len(t, full, 3, "review must evaluate every target regardless of CI sampling")
	for _, target := range full {
		require.Equal(t, "spec-1", target.SpecID, "a review result anchors on its spec")
	}
}
