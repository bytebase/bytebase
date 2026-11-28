package validation

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/action/world"
)

func TestValidateFlagsAcceptsValidPlanCustomizationAndTargets(t *testing.T) {
	t.Setenv("BYTEBASE_ACCESS_TOKEN", "env-token")
	w := &world.World{
		URL:             "https://bytebase.example.com/",
		Project:         "projects/demo",
		Targets:         []string{"instances/instance-a/databases/db-a"},
		PlanTitle:       "Release plan",
		PlanDescription: "Describe the rollout",
	}

	err := ValidateFlags(w)
	require.NoError(t, err)
	require.Equal(t, "env-token", w.AccessToken)
	require.Equal(t, "https://bytebase.example.com", w.URL)
}

func TestValidateFlagsRejectsPlanCustomizationLimits(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		mutate    func(*world.World)
		wantError string
	}{
		{
			name: "plan title too long",
			mutate: func(w *world.World) {
				w.PlanTitle = strings.Repeat("a", 201)
			},
			wantError: "--plan-title",
		},
		{
			name: "plan description too long",
			mutate: func(w *world.World) {
				w.PlanDescription = strings.Repeat("b", 10001)
			},
			wantError: "--plan-description",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			w := &world.World{
				URL:         "https://bytebase.example.com",
				Project:     "projects/demo",
				Targets:     []string{"instances/instance-a/databases/db-a"},
				AccessToken: "token",
			}
			tt.mutate(w)

			err := ValidateFlags(w)
			require.Error(t, err)
			require.ErrorContains(t, err, tt.wantError)
		})
	}
}

func TestValidateFlagsRejectsInvalidTargets(t *testing.T) {
	t.Parallel()

	w := &world.World{
		URL:         "https://bytebase.example.com",
		Project:     "projects/demo",
		Targets:     []string{"not-a-target"},
		AccessToken: "token",
	}

	err := ValidateFlags(w)
	require.Error(t, err)
	require.ErrorContains(t, err, "invalid target format")
}
