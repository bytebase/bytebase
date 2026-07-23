package taskrun

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

func TestCompletionWebhookEnvironmentUsesLastEnvironmentOrder(t *testing.T) {
	tasks := []*store.TaskMessage{
		{ID: 1, Environment: "dev"},
		{ID: 2, Environment: "prod"},
		{ID: 3, Environment: "test"},
	}
	environmentOrderMap := common.EnvironmentOrderMap([]*storepb.EnvironmentSetting_Environment{
		{Id: "dev"},
		{Id: "test"},
		{Id: "prod"},
	})

	require.Equal(t, "prod", completionWebhookEnvironment(tasks, environmentOrderMap))
}
