package webhook

import (
	"strings"
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

// TestContext_IssueApproved_MetaList tests the meta list rendering for an ISSUE_APPROVED
// webhook context. This simulates the Context that getWebhookContextFromEvent builds
// when processing an ISSUE_APPROVED event.
func TestContext_IssueApproved_MetaList(t *testing.T) {
	// Build a Context that matches what the manager produces for ISSUE_APPROVED:
	// - Level: WebhookSuccess
	// - Issue with creator (looked up from store)
	// - Description: "<approver> approved the issue"
	// - Project set
	// - Link set
	approvedCtx := Context{
		Level:       WebhookSuccess,
		EventType:   "ISSUE_APPROVED",
		Title:       "Issue approved",
		TitleZh:     "工单审批通过",
		Description: "Bob approved the issue",
		Link:        "https://bb.example.com/projects/proj-1/issues/42",
		ActorName:   "Bob",
		ActorEmail:  "bob@example.com",
		Project: &Project{
			Name:  "projects/proj-1",
			Title: "My Project",
		},
		Issue: &Issue{
			ID:          42,
			Name:        "Grant read access to prod",
			Status:      "OPEN",
			Type:        "DATABASE_CHANGE",
			Description: "Engineers need read access to production database for debugging",
			Creator: Creator{
				Name:  "Alice",
				Email: "alice@example.com",
			},
		},
	}

	t.Run("GetMetaList includes project and issue", func(t *testing.T) {
		a := require.New(t)
		meta := approvedCtx.GetMetaList()

		// Should have: Project Title, Project ID, Issue, Issue Creator, Issue Description
		a.Len(meta, 5)
		a.Equal("Project Title", meta[0].Name)
		a.Equal("My Project", meta[0].Value)
		a.Equal("Project ID", meta[1].Name)
		a.Equal("projects/proj-1", meta[1].Value)
		a.Equal("Issue", meta[2].Name)
		a.Equal("Grant read access to prod", meta[2].Value)
		a.Equal("Issue Creator", meta[3].Name)
		a.Equal("Alice (alice@example.com)", meta[3].Value)
		a.Equal("Issue Description", meta[4].Name)
		a.Contains(meta[4].Value, "Engineers need read access")
	})

	t.Run("GetMetaListZh includes project and issue", func(t *testing.T) {
		a := require.New(t)
		meta := approvedCtx.GetMetaListZh()

		a.Len(meta, 5)
		a.Equal("项目名称", meta[0].Name)
		a.Equal("My Project", meta[0].Value)
		a.Equal("项目 ID", meta[1].Name)
		a.Equal("projects/proj-1", meta[1].Value)
		a.Equal("工单", meta[2].Name)
		a.Equal("Grant read access to prod", meta[2].Value)
		a.Equal("工单创建者", meta[3].Name)
		a.Equal("Alice (alice@example.com)", meta[3].Value)
		a.Equal("工单描述", meta[4].Name)
		a.Contains(meta[4].Value, "Engineers need read access")
	})

	t.Run("Level is WebhookSuccess", func(t *testing.T) {
		a := require.New(t)
		a.Equal(WebhookSuccess, approvedCtx.Level)
	})

	t.Run("Title and TitleZh are correct", func(t *testing.T) {
		a := require.New(t)
		a.Equal("Issue approved", approvedCtx.Title)
		a.Equal("工单审批通过", approvedCtx.TitleZh)
	})

	t.Run("Description contains approver name", func(t *testing.T) {
		a := require.New(t)
		a.Equal("Bob approved the issue", approvedCtx.Description)
	})
}

// TestContext_IssueApproved_NilEvent tests the ISSUE_APPROVED context when
// the event data is nil (e.IssueApproved == nil). In this case, the manager
// sets only title/titleZh and all other fields remain zero-valued.
func TestContext_IssueApproved_NilEvent(t *testing.T) {
	a := require.New(t)

	// Simulates what the manager produces when e.IssueApproved is nil
	ctx := Context{
		Level:     WebhookSuccess,
		EventType: "ISSUE_APPROVED",
		Title:     "Issue approved",
		TitleZh:   "工单审批通过",
	}

	meta := ctx.GetMetaList()
	a.Empty(meta, "no meta when issue is nil")

	metaZh := ctx.GetMetaListZh()
	a.Empty(metaZh, "no meta zh when issue is nil")
}

// TestContext_IssueApproved_LongDescription tests that long issue descriptions
// are truncated in the meta list rendering.
func TestContext_IssueApproved_LongDescription(t *testing.T) {
	a := require.New(t)

	longDesc := strings.Repeat("This is a long description. ", 50) // ~1400 chars
	ctx := Context{
		Issue: &Issue{
			Name:        "Issue with long description",
			Description: longDesc,
			Creator:     Creator{Name: "Alice", Email: "alice@example.com"},
		},
	}

	meta := ctx.GetMetaList()
	descMeta := meta[2]
	a.Equal("Issue Description", descMeta.Name)
	// TruncateStringWithDescription truncates at 450 chars and appends "... (view details in Bytebase)"
	a.True(len(descMeta.Value) < len(longDesc), "description should be truncated")
	a.Contains(descMeta.Value, "view details in Bytebase")
}

// TestContext_IssueVsRollout_Exclusive tests that Issue and Rollout are mutually
// exclusive in the meta list — matching the `else if` logic in GetMetaList.
func TestContext_IssueVsRollout_Exclusive(t *testing.T) {
	t.Run("Issue takes precedence over Rollout", func(t *testing.T) {
		a := require.New(t)
		ctx := Context{
			Issue: &Issue{
				Name:    "My Issue",
				Creator: Creator{Name: "Alice", Email: "alice@example.com"},
			},
			Rollout: &Rollout{
				UID:   1,
				Title: "My Rollout",
			},
		}
		meta := ctx.GetMetaList()
		// Should contain Issue fields, not Rollout
		hasIssue := false
		hasRollout := false
		for _, m := range meta {
			if m.Name == "Issue" {
				hasIssue = true
			}
			if m.Name == "Rollout" {
				hasRollout = true
			}
		}
		a.True(hasIssue, "should have Issue meta")
		a.False(hasRollout, "should not have Rollout meta when Issue is set")
	})

	t.Run("Rollout shown when no Issue", func(t *testing.T) {
		a := require.New(t)
		ctx := Context{
			Rollout: &Rollout{
				UID:   1,
				Title: "Deploy v2.0",
			},
			Environment: "environments/prod",
		}
		meta := ctx.GetMetaList()
		a.Len(meta, 2)
		a.Equal("Rollout", meta[0].Name)
		a.Equal("Deploy v2.0", meta[0].Value)
		a.Equal("Environment", meta[1].Name)
		a.Equal("environments/prod", meta[1].Value)
	})
}

// TestContext_AllWebhookLevels verifies all webhook level constants.
func TestContext_AllWebhookLevels(t *testing.T) {
	a := require.New(t)
	a.Equal(Level("INFO"), WebhookInfo)
	a.Equal(Level("SUCCESS"), WebhookSuccess)
	a.Equal(Level("WARN"), WebhookWarn)
	a.Equal(Level("ERROR"), WebhookError)
}
