package taskrun

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/store"
)

func TestCompletionWebhookEnvironmentUsesLastTask(t *testing.T) {
	tasks := []*store.TaskMessage{
		{ID: 1, Environment: "dev"},
		{ID: 2, Environment: "test"},
		{ID: 3, Environment: "prod"},
	}

	require.Equal(t, "prod", completionWebhookEnvironment(tasks))
}
