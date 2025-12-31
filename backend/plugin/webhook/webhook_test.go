package webhook

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestContext_getMetaList(t *testing.T) {
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
