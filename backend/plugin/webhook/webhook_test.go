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
		want := []Meta{
			{
				Name:  "Task",
				Value: "task name",
			},
			{
				Name:  "Status",
				Value: "PENDING_APPROVAL",
			},
		}
		a.Equal(want, context.GetMetaList())
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
		want := []Meta{
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
		a.Equal(want, context.GetMetaList())
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
		want := []Meta{
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
		a.Equal(want, context.GetMetaList())
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
		want := []Meta{
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
		a.Equal(want, context.GetMetaList())
	})
	t.Run("Issue with Creator", func(t *testing.T) {
		a := require.New(t)
		context := Context{
			Issue: &Issue{
				Name:        "Issue 101",
				Description: "Fix critical bug",
				Creator: Creator{
					Name:  "Alice",
					Email: "alice@example.com",
				},
			},
		}
		want := []Meta{
			{
				Name:  "Issue",
				Value: "Issue 101",
			},
			{
				Name:  "Issue Creator",
				Value: "Alice (alice@example.com)",
			},
			{
				Name:  "Issue Description",
				Value: "Fix critical bug",
			},
		}
		a.Equal(want, context.GetMetaList())
	})

	t.Run("Issue with Creator Zh", func(t *testing.T) {
		a := require.New(t)
		context := Context{
			Issue: &Issue{
				Name:        "Issue 101",
				Description: "Fix critical bug",
				Creator: Creator{
					Name:  "Alice",
					Email: "alice@example.com",
				},
			},
		}
		want := []Meta{
			{
				Name:  "工单",
				Value: "Issue 101",
			},
			{
				Name:  "工单创建者",
				Value: "Alice (alice@example.com)",
			},
			{
				Name:  "工单描述",
				Value: "Fix critical bug",
			},
		}
		a.Equal(want, context.GetMetaListZh())
	})
}
