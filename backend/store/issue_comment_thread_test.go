package store_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/migrator"
	"github.com/bytebase/bytebase/backend/store"

	_ "github.com/bytebase/bytebase/backend/plugin/db/pg"
)

// TestIssueCommentThreads covers the store-level thread surface: reply
// creation with its root invariants, resolve/reopen on the root only, the
// timeline and reply list filters, and statement anchor round-trips.
func TestIssueCommentThreads(t *testing.T) {
	const (
		workspaceID  = "thread-ws"
		projectID    = "thread-p"
		otherProject = "thread-p2"
		issueUID     = int64(101)
		otherIssue   = int64(102)
		creator      = "users/thread@example.com"
	)

	ctx := context.Background()
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(ctx) })
	db := container.GetDB()
	require.NoError(t, migrator.MigrateSchema(ctx, db))

	_, err := db.ExecContext(ctx,
		`INSERT INTO workspace (resource_id) VALUES ($1)`, workspaceID)
	require.NoError(t, err)
	for _, p := range []string{projectID, otherProject} {
		_, err = db.ExecContext(ctx,
			`INSERT INTO project (resource_id, workspace, name) VALUES ($1, $2, $1)`,
			p, workspaceID)
		require.NoError(t, err)
	}
	for _, id := range []int64{issueUID, otherIssue} {
		_, err = db.ExecContext(ctx, `
			INSERT INTO issue (id, project, creator, name, status, type, description)
			VALUES ($1, $2, $3, 'issue', 'OPEN', 'DATABASE_CHANGE', '')`,
			id, projectID, creator)
		require.NoError(t, err)
	}

	pgURL := fmt.Sprintf(
		"host=%s port=%s user=postgres password=root-password database=postgres",
		container.GetHost(), container.GetPort(),
	)
	stores, err := store.New(ctx, pgURL, false)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, stores.Close()) })

	anchor := &storepb.IssueCommentPayload_StatementAnchor{
		SpecId:        "spec-1",
		SheetSha256:   "0be1f01d6ee8e6f6c6a2ce9b418ba10ea9d16c9b9bfae5548b8fa0e26c04a5e0",
		StartPosition: &storepb.Position{Line: 5},
		EndPosition:   &storepb.Position{Line: 8},
	}
	root, err := stores.CreateIssueComments(ctx, creator, &store.IssueCommentMessage{
		ProjectID: projectID,
		IssueUID:  issueUID,
		Payload:   &storepb.IssueCommentPayload{Comment: "root", StatementAnchor: anchor},
	})
	require.NoError(t, err)
	require.Nil(t, root.ParentID)
	require.NotNil(t, root.ThreadState)
	require.Equal(t, store.ThreadStateOpen, *root.ThreadState)

	event, err := stores.CreateIssueComments(ctx, creator, &store.IssueCommentMessage{
		ProjectID: projectID,
		IssueUID:  issueUID,
		Payload: &storepb.IssueCommentPayload{
			Event: &storepb.IssueCommentPayload_IssueUpdate_{
				IssueUpdate: &storepb.IssueCommentPayload_IssueUpdate{},
			},
		},
	})
	require.NoError(t, err)
	require.Nil(t, event.ThreadState, "events must stay outside threads")

	// Reply invariants.
	reply, err := stores.CreateIssueCommentReply(ctx, creator, &store.IssueCommentMessage{
		ProjectID: projectID,
		IssueUID:  issueUID,
		ParentID:  &root.ResourceID,
		Payload:   &storepb.IssueCommentPayload{Comment: "re"},
	})
	require.NoError(t, err)
	require.Equal(t, root.ResourceID, *reply.ParentID)
	require.Nil(t, reply.ThreadState, "replies carry no thread state")
	storedReply, err := stores.GetIssueComment(ctx, &store.FindIssueCommentMessage{
		ProjectID:  projectID,
		ResourceID: &reply.ResourceID,
	})
	require.NoError(t, err)
	require.Equal(t, root.ResourceID, *storedReply.ParentID, "the stored row must reference the root")
	require.Nil(t, storedReply.ThreadState)

	_, err = stores.CreateIssueCommentReply(ctx, creator, &store.IssueCommentMessage{
		ProjectID: projectID,
		IssueUID:  issueUID,
		ParentID:  &reply.ResourceID,
		Payload:   &storepb.IssueCommentPayload{Comment: "nested"},
	})
	require.Equal(t, common.Invalid, common.ErrorCode(err), "a reply to a reply must be rejected, not re-rooted")

	_, err = stores.CreateIssueCommentReply(ctx, creator, &store.IssueCommentMessage{
		ProjectID: projectID,
		IssueUID:  issueUID,
		ParentID:  &event.ResourceID,
		Payload:   &storepb.IssueCommentPayload{Comment: "on event"},
	})
	require.Equal(t, common.Invalid, common.ErrorCode(err), "events cannot be replied to")

	missing := "no-such-comment"
	_, err = stores.CreateIssueCommentReply(ctx, creator, &store.IssueCommentMessage{
		ProjectID: projectID,
		IssueUID:  issueUID,
		ParentID:  &missing,
		Payload:   &storepb.IssueCommentPayload{Comment: "orphan"},
	})
	require.Equal(t, common.NotFound, common.ErrorCode(err))

	_, err = stores.CreateIssueCommentReply(ctx, creator, &store.IssueCommentMessage{
		ProjectID: projectID,
		IssueUID:  otherIssue,
		ParentID:  &root.ResourceID,
		Payload:   &storepb.IssueCommentPayload{Comment: "cross-issue"},
	})
	require.Equal(t, common.NotFound, common.ErrorCode(err), "a reply must target a root in its own issue")

	_, err = stores.CreateIssueCommentReply(ctx, creator, &store.IssueCommentMessage{
		ProjectID: otherProject,
		IssueUID:  issueUID,
		ParentID:  &root.ResourceID,
		Payload:   &storepb.IssueCommentPayload{Comment: "cross-project"},
	})
	require.Equal(t, common.NotFound, common.ErrorCode(err), "a reply must target a root in its own project")

	_, err = stores.CreateIssueCommentReply(ctx, creator, &store.IssueCommentMessage{
		ProjectID: projectID,
		IssueUID:  issueUID,
		ParentID:  &root.ResourceID,
		Payload: &storepb.IssueCommentPayload{
			Event: &storepb.IssueCommentPayload_IssueUpdate_{
				IssueUpdate: &storepb.IssueCommentPayload_IssueUpdate{},
			},
		},
	})
	require.Equal(t, common.Invalid, common.ErrorCode(err), "a reply cannot carry an event")

	preset := store.ThreadStateResolved
	_, err = stores.CreateIssueComments(ctx, creator, &store.IssueCommentMessage{
		ProjectID:   projectID,
		IssueUID:    issueUID,
		Payload:     &storepb.IssueCommentPayload{Comment: "preset"},
		ThreadState: &preset,
	})
	require.Equal(t, common.Invalid, common.ErrorCode(err), "thread state is not settable on create")
	_, err = stores.CreateIssueCommentReply(ctx, creator, &store.IssueCommentMessage{
		ProjectID:   projectID,
		IssueUID:    issueUID,
		ParentID:    &root.ResourceID,
		ThreadState: &preset,
		Payload:     &storepb.IssueCommentPayload{Comment: "re"},
	})
	require.Equal(t, common.Invalid, common.ErrorCode(err), "thread state is not settable on a reply")

	_, err = stores.CreateIssueComments(ctx, creator, &store.IssueCommentMessage{
		ProjectID: projectID,
		IssueUID:  issueUID,
		ParentID:  &root.ResourceID,
		Payload:   &storepb.IssueCommentPayload{Comment: "misrouted"},
	})
	require.Equal(t, common.Invalid, common.ErrorCode(err), "CreateIssueComments must not create replies")

	_, err = stores.CreateIssueComments(ctx, creator, &store.IssueCommentMessage{
		ProjectID: projectID,
		IssueUID:  issueUID,
		Payload: &storepb.IssueCommentPayload{
			StatementAnchor: anchor,
			Event: &storepb.IssueCommentPayload_IssueUpdate_{
				IssueUpdate: &storepb.IssueCommentPayload_IssueUpdate{},
			},
		},
	})
	require.Equal(t, common.Invalid, common.ErrorCode(err), "an event cannot carry a statement anchor")

	_, err = stores.CreateIssueComments(ctx, creator, &store.IssueCommentMessage{
		ProjectID: projectID,
		IssueUID:  issueUID,
		Payload: &storepb.IssueCommentPayload{
			Comment: "bad anchor",
			StatementAnchor: &storepb.IssueCommentPayload_StatementAnchor{
				SpecId:        "spec-1",
				SheetSha256:   anchor.SheetSha256,
				StartPosition: &storepb.Position{Line: 5},
				EndPosition:   &storepb.Position{Line: 8, Column: 4},
			},
		},
	})
	require.Equal(t, common.Invalid, common.ErrorCode(err), "an anchor range must set both columns or neither")

	// List filters.
	uid := issueUID
	all, err := stores.ListIssueComment(ctx, &store.FindIssueCommentMessage{
		ProjectID: projectID,
		IssueUID:  &uid,
	})
	require.NoError(t, err)
	require.Len(t, all, 3, "unfiltered list returns roots, events, and replies")

	topLevel, err := stores.ListIssueComment(ctx, &store.FindIssueCommentMessage{
		ProjectID:    projectID,
		IssueUID:     &uid,
		TopLevelOnly: true,
	})
	require.NoError(t, err)
	require.Len(t, topLevel, 2, "timeline entries are the root and the event")
	for _, comment := range topLevel {
		require.Nil(t, comment.ParentID)
	}

	parentIDs := []string{root.ResourceID}
	replies, err := stores.ListIssueComment(ctx, &store.FindIssueCommentMessage{
		ProjectID: projectID,
		ParentIDs: &parentIDs,
	})
	require.NoError(t, err)
	require.Len(t, replies, 1)
	require.Equal(t, reply.ResourceID, replies[0].ResourceID)

	crossProjectReplies, err := stores.ListIssueComment(ctx, &store.FindIssueCommentMessage{
		ProjectID: otherProject,
		ParentIDs: &parentIDs,
	})
	require.NoError(t, err)
	require.Empty(t, crossProjectReplies, "reply reads are project-scoped")

	// Anchor round-trip through the jsonb payload.
	fetched, err := stores.GetIssueComment(ctx, &store.FindIssueCommentMessage{
		ProjectID:  projectID,
		ResourceID: &root.ResourceID,
	})
	require.NoError(t, err)
	got := fetched.Payload.GetStatementAnchor()
	require.NotNil(t, got)
	require.Equal(t, anchor.SpecId, got.SpecId)
	require.Equal(t, anchor.SheetSha256, got.SheetSha256)
	require.Equal(t, anchor.StartPosition.Line, got.StartPosition.Line)
	require.Equal(t, anchor.EndPosition.Line, got.EndPosition.Line)

	// Resolve and reopen live on the root only.
	resolved := store.ThreadStateResolved
	require.NoError(t, stores.UpdateIssueComment(ctx, &store.UpdateIssueCommentMessage{
		ProjectID:   projectID,
		ResourceID:  root.ResourceID,
		ThreadState: &resolved,
	}))
	fetched, err = stores.GetIssueComment(ctx, &store.FindIssueCommentMessage{
		ProjectID:  projectID,
		ResourceID: &root.ResourceID,
	})
	require.NoError(t, err)
	require.Equal(t, store.ThreadStateResolved, *fetched.ThreadState)

	// Replying to a resolved thread must not reopen it.
	_, err = stores.CreateIssueCommentReply(ctx, creator, &store.IssueCommentMessage{
		ProjectID: projectID,
		IssueUID:  issueUID,
		ParentID:  &root.ResourceID,
		Payload:   &storepb.IssueCommentPayload{Comment: "late reply"},
	})
	require.NoError(t, err)
	fetched, err = stores.GetIssueComment(ctx, &store.FindIssueCommentMessage{
		ProjectID:  projectID,
		ResourceID: &root.ResourceID,
	})
	require.NoError(t, err)
	require.Equal(t, store.ThreadStateResolved, *fetched.ThreadState)
	replies, err = stores.ListIssueComment(ctx, &store.FindIssueCommentMessage{
		ProjectID: projectID,
		ParentIDs: &parentIDs,
	})
	require.NoError(t, err)
	require.Len(t, replies, 2, "the late reply must land in the thread")
	for _, r := range replies {
		require.Equal(t, root.ResourceID, *r.ParentID)
	}

	// A text edit leaves the anchor and thread state alone; only text edits
	// bump updated_at.
	edited := "root (edited)"
	require.NoError(t, stores.UpdateIssueComment(ctx, &store.UpdateIssueCommentMessage{
		ProjectID:  projectID,
		ResourceID: root.ResourceID,
		Comment:    &edited,
	}))
	fetched, err = stores.GetIssueComment(ctx, &store.FindIssueCommentMessage{
		ProjectID:  projectID,
		ResourceID: &root.ResourceID,
	})
	require.NoError(t, err)
	require.Equal(t, edited, fetched.Payload.GetComment())
	require.Equal(t, anchor.SpecId, fetched.Payload.GetStatementAnchor().GetSpecId(), "a comment patch must not touch the anchor")
	require.Equal(t, anchor.SheetSha256, fetched.Payload.GetStatementAnchor().GetSheetSha256())
	require.Equal(t, store.ThreadStateResolved, *fetched.ThreadState)
	editStamp := fetched.UpdatedAt

	nope := "text on event"
	err = stores.UpdateIssueComment(ctx, &store.UpdateIssueCommentMessage{
		ProjectID:  projectID,
		ResourceID: event.ResourceID,
		Comment:    &nope,
	})
	require.Equal(t, common.NotFound, common.ErrorCode(err), "a pure event cannot gain comment text")

	hybrid, err := stores.CreateIssueComments(ctx, creator, &store.IssueCommentMessage{
		ProjectID: projectID,
		IssueUID:  issueUID,
		Payload: &storepb.IssueCommentPayload{
			Comment: "approved",
			Event: &storepb.IssueCommentPayload_Approval_{
				Approval: &storepb.IssueCommentPayload_Approval{},
			},
		},
	})
	require.NoError(t, err)
	require.Nil(t, hybrid.ThreadState)
	hybridEdit := "approved (edited)"
	require.NoError(t, stores.UpdateIssueComment(ctx, &store.UpdateIssueCommentMessage{
		ProjectID:  projectID,
		ResourceID: hybrid.ResourceID,
		Comment:    &hybridEdit,
	}), "hybrid event+comment rows keep their text editable")

	err = stores.UpdateIssueComment(ctx, &store.UpdateIssueCommentMessage{
		ProjectID:   projectID,
		ResourceID:  reply.ResourceID,
		ThreadState: &resolved,
	})
	require.Equal(t, common.NotFound, common.ErrorCode(err), "replies carry no thread state")
	err = stores.UpdateIssueComment(ctx, &store.UpdateIssueCommentMessage{
		ProjectID:   projectID,
		ResourceID:  event.ResourceID,
		ThreadState: &resolved,
	})
	require.Equal(t, common.NotFound, common.ErrorCode(err), "events carry no thread state")

	open := store.ThreadStateOpen
	require.NoError(t, stores.UpdateIssueComment(ctx, &store.UpdateIssueCommentMessage{
		ProjectID:   projectID,
		ResourceID:  root.ResourceID,
		ThreadState: &open,
	}))
	openRoots, err := stores.ListIssueComment(ctx, &store.FindIssueCommentMessage{
		ProjectID:   projectID,
		IssueUID:    &uid,
		ThreadState: &open,
	})
	require.NoError(t, err)
	require.Len(t, openRoots, 1)
	require.Equal(t, root.ResourceID, openRoots[0].ResourceID)
	require.True(t, editStamp.Equal(openRoots[0].UpdatedAt),
		"resolve and reopen must not mark the comment as edited")
}
