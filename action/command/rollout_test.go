package command

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/action/world"
)

func TestNewRolloutCommandRegistersPlanCustomizationFlags(t *testing.T) {
	t.Parallel()

	w := world.NewWorld()
	cmd := NewRolloutCommand(w)

	require.NotNil(t, cmd)
	require.NotNil(t, cmd.Flags().Lookup("plan-title"))
	require.NotNil(t, cmd.Flags().Lookup("plan-description"))
	require.NotNil(t, cmd.Flags().Lookup("release-id-template"))
	require.NotNil(t, cmd.Flags().Lookup("release-id-timezone"))
	require.NotNil(t, cmd.Flags().Lookup("target-stage"))
}

func TestRolloutUsesConfiguredPlanMetadata(t *testing.T) {
	t.Parallel()

	w := world.NewWorld()
	w.PlanTitle = "custom title"
	w.PlanDescription = "custom description"
	w.ReleaseIDTemplate = "release_{date}"
	w.ReleaseIDTimezone = "America/Los_Angeles"
	w.TargetStage = "environments/prod"

	require.Equal(t, "custom title", w.PlanTitle)
	require.Equal(t, "custom description", w.PlanDescription)
	require.Equal(t, "release_{date}", w.ReleaseIDTemplate)
	require.Equal(t, "America/Los_Angeles", w.ReleaseIDTimezone)
	require.Equal(t, "environments/prod", w.TargetStage)
}
