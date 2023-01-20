package webhook

import (
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
)

func TestContext_getMetaList(t *testing.T) {
	t.Run("task1", func(t *testing.T) {
		a := require.New(t)
		context := Context{
			TaskResult: &TaskResult{
				Name:   "task name",
				Status: "PENDING_APPROVAL",
			},
		}
		want := []meta{
			{
				Name:  "Task",
				Value: "task name",
			},
			{
				Name:  "Status",
				Value: "PENDING_APPROVAL",
			},
		}
		a.Equal(want, context.getMetaList())
	})
	t.Run("task2", func(t *testing.T) {
		a := require.New(t)
		context := Context{
			TaskResult: &TaskResult{
				Name:   "task name",
				Status: "FAILED",
				Detail: "SQL STATE",
			},
		}
		want := []meta{
			{
				Name:  "Task",
				Value: "task name",
			},
			{
				Name:  "Status",
				Value: "FAILED",
			},
			{
				Name:  "Result Detail",
				Value: "SQL STATE",
			},
		}
		a.Equal(want, context.getMetaList())
	})
	t.Run("task3", func(t *testing.T) {
		a := require.New(t)

		// generate random string
		b := make([]byte, 300)
		_, err := rand.Read(b)
		a.NoError(err)
		detail := string(b)
		a.Equal(300, len(detail))

		context := Context{
			TaskResult: &TaskResult{
				Name:   "task name",
				Status: "FAILED",
				Detail: detail,
			},
		}
		want := []meta{
			{
				Name:  "Task",
				Value: "task name",
			},
			{
				Name:  "Status",
				Value: "FAILED",
			},
			{
				Name:  "Result Detail",
				Value: common.TruncateStringWithDescription(detail),
			},
		}
		a.Equal(want, context.getMetaList())
	})
	t.Run("task4", func(t *testing.T) {
		a := require.New(t)

		// generate random string
		b := make([]byte, 600)
		_, err := rand.Read(b)
		a.NoError(err)
		detail := string(b)
		a.Equal(600, len(detail))

		context := Context{
			TaskResult: &TaskResult{
				Name:   "task name",
				Status: "FAILED",
				Detail: detail,
			},
		}
		want := []meta{
			{
				Name:  "Task",
				Value: "task name",
			},
			{
				Name:  "Status",
				Value: "FAILED",
			},
			{
				Name:  "Result Detail",
				Value: common.TruncateStringWithDescription(detail),
			},
		}
		a.Equal(want, context.getMetaList())
	})
}
