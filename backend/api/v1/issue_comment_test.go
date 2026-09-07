package v1

import (
	"fmt"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
)

func TestIssueCommentAPI(t *testing.T) {
	t.Parallel()
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	service := newIssueServiceForTest(t, stores)
	_, issue := createIssueServiceApprovalIssue(ctx, t, stores)
	parent := common.FormatIssue(issue.ProjectID, issue.UID)
	create := func(comment *v1pb.IssueComment) *v1pb.IssueComment {
		t.Helper()
		resp, err := service.CreateIssueComment(ctx, connect.NewRequest(&v1pb.CreateIssueCommentRequest{
			Parent: parent, IssueComment: comment,
		}))
		require.NoError(t, err)
		return resp.Msg
	}
	list := func(filter string) []*v1pb.IssueComment {
		t.Helper()
		var comments []*v1pb.IssueComment
		token := ""
		for {
			resp, err := service.ListIssueComments(ctx, connect.NewRequest(&v1pb.ListIssueCommentsRequest{
				Parent: parent, Filter: filter, PageSize: 1, PageToken: token,
			}))
			require.NoError(t, err)
			comments = append(comments, resp.Msg.IssueComments...)
			token = resp.Msg.NextPageToken
			if token == "" {
				return comments
			}
		}
	}
	update := func(comment *v1pb.IssueComment, paths ...string) (*v1pb.IssueComment, error) {
		t.Helper()
		resp, err := service.UpdateIssueComment(ctx, connect.NewRequest(&v1pb.UpdateIssueCommentRequest{
			Parent: parent, IssueComment: comment, UpdateMask: &fieldmaskpb.FieldMask{Paths: paths},
		}))
		if err != nil {
			return nil, err
		}
		return resp.Msg, nil
	}

	general := create(&v1pb.IssueComment{Comment: "general"})
	require.Nil(t, general.Root)
	require.Nil(t, general.ThreadState)
	require.Nil(t, general.StatementAnchor)
	sheets, err := stores.CreateSheets(ctx, issue.ProjectID, &store.SheetMessage{Statement: "SELECT 1;\nSELECT 2;\nSELECT 3;\nSELECT 4;"})
	require.NoError(t, err)
	anchor := &v1pb.StatementAnchor{
		Spec: "spec-1", SheetSha256: sheets[0].Sha256,
		StartPosition: &v1pb.Position{Line: 2}, EndPosition: &v1pb.Position{Line: 4},
	}
	root := create(&v1pb.IssueComment{Comment: "root", ThreadState: v1pb.IssueComment_OPEN.Enum(), StatementAnchor: anchor})
	require.True(t, proto.Equal(anchor, root.StatementAnchor))
	require.Equal(t, v1pb.IssueComment_OPEN, root.GetThreadState())
	unanchored := create(&v1pb.IssueComment{Comment: "unanchored thread", ThreadState: v1pb.IssueComment_OPEN.Enum()})
	require.Nil(t, unanchored.StatementAnchor)
	require.Equal(t, v1pb.IssueComment_OPEN, unanchored.GetThreadState())

	resolved, err := update(&v1pb.IssueComment{Name: root.Name, ThreadState: v1pb.IssueComment_RESOLVED.Enum()}, "thread_state")
	require.NoError(t, err)
	require.True(t, proto.Equal(root.UpdateTime, resolved.UpdateTime), "resolution is not a text edit")
	replyAnchor := proto.CloneOf(anchor)
	replyAnchor.StartPosition = &v1pb.Position{Line: 2, Column: 1}
	replyAnchor.EndPosition = &v1pb.Position{Line: 2, Column: 3}
	reply := create(&v1pb.IssueComment{Comment: "reply", Root: &root.Name, StatementAnchor: replyAnchor})
	require.Equal(t, root.Name, reply.GetRoot())
	require.Nil(t, reply.ThreadState)
	require.True(t, proto.Equal(replyAnchor, reply.StatementAnchor))
	replies := list(fmt.Sprintf("root == %q", root.Name))
	require.Len(t, replies, 1)
	require.Equal(t, reply.Name, replies[0].Name)
	require.Equal(t, []string{reply.Name}, commentNames(list(fmt.Sprintf("root in [%q, %q]", root.Name, unanchored.Name))))
	require.Empty(t, list("root in []"))

	event, err := stores.CreateIssueComments(ctx, "creator@example.com", &store.IssueCommentMessage{
		ProjectID: issue.ProjectID, IssueUID: issue.UID,
		Payload: &storepb.IssueCommentPayload{Event: &storepb.IssueCommentPayload_ReviewSubmission_{
			ReviewSubmission: &storepb.IssueCommentPayload_ReviewSubmission{},
		}},
	})
	require.NoError(t, err)
	eventName := common.FormatIssueComment(parent, event.ResourceID)
	timeline := list("root == null")
	require.Equal(t, []string{general.Name, root.Name, unanchored.Name, eventName}, commentNames(timeline))
	require.Equal(t, v1pb.IssueComment_RESOLVED, timeline[1].GetThreadState(), "reply must not reopen")
	require.Equal(t, commentNames(timeline), commentNames(list("")), "clients that predate threads keep the timeline")

	edited, err := update(&v1pb.IssueComment{Name: reply.Name, Comment: "edited"}, "comment")
	require.NoError(t, err)
	require.Equal(t, "edited", edited.Comment)
	require.Equal(t, root.Name, edited.GetRoot())
	require.True(t, proto.Equal(replyAnchor, edited.StatementAnchor))
	reopened, err := update(&v1pb.IssueComment{Name: root.Name, ThreadState: v1pb.IssueComment_OPEN.Enum()}, "thread_state")
	require.NoError(t, err)
	require.Equal(t, v1pb.IssueComment_OPEN, reopened.GetThreadState())

	for _, name := range []string{general.Name, reply.Name, eventName} {
		_, err := update(&v1pb.IssueComment{Name: name, ThreadState: v1pb.IssueComment_RESOLVED.Enum()}, "thread_state")
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
		_, err = service.CreateIssueComment(ctx, connect.NewRequest(&v1pb.CreateIssueCommentRequest{
			Parent: parent, IssueComment: &v1pb.IssueComment{Comment: "invalid reply", Root: &name},
		}))
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	}
	for _, path := range []string{"root", "statement_anchor", "event"} {
		_, err := update(&v1pb.IssueComment{Name: root.Name, Comment: "must not change"}, "comment", path)
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	}
	require.Equal(t, "root", list("root == null")[1].Comment)
	_, err = update(&v1pb.IssueComment{Name: general.Name}, "comment")
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	require.Equal(t, "general", list("root == null")[0].Comment)
	for _, state := range []*v1pb.IssueComment_ThreadState{nil, v1pb.IssueComment_THREAD_STATE_UNSPECIFIED.Enum(), v1pb.IssueComment_ThreadState(99).Enum()} {
		_, err := update(&v1pb.IssueComment{Name: root.Name, ThreadState: state}, "thread_state")
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	}

	_, otherIssue := createIssueServiceApprovalIssue(ctx, t, stores)
	otherParent := common.FormatIssue(issue.ProjectID, otherIssue.UID)
	for _, foreign := range []string{
		strings.Replace(root.Name, "projects/project-a/", "projects/project-b/", 1),
		strings.Replace(root.Name, parent+"/", otherParent+"/", 1),
	} {
		_, err := service.CreateIssueComment(ctx, connect.NewRequest(&v1pb.CreateIssueCommentRequest{
			Parent: parent, IssueComment: &v1pb.IssueComment{Comment: "foreign reply", Root: &foreign},
		}))
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
		_, err = update(&v1pb.IssueComment{Name: foreign, Comment: "foreign edit"}, "comment")
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	}

	missing := common.FormatIssueComment(parent, "no-such-comment")
	invalid := []*v1pb.IssueComment{
		nil, {}, {Comment: "empty root", Root: proto.String("")},
		{Comment: "dangling root", Root: &missing},
		{Comment: "resolved", ThreadState: v1pb.IssueComment_RESOLVED.Enum()},
		{Comment: "unspecified", ThreadState: v1pb.IssueComment_THREAD_STATE_UNSPECIFIED.Enum()},
		{Comment: "reply state", Root: &root.Name, ThreadState: v1pb.IssueComment_OPEN.Enum()},
		{Comment: "bad anchor", ThreadState: v1pb.IssueComment_OPEN.Enum(), StatementAnchor: &v1pb.StatementAnchor{}},
		{Comment: "unknown spec", StatementAnchor: &v1pb.StatementAnchor{Spec: "spec-9", SheetSha256: anchor.SheetSha256}},
		{Comment: "unknown sheet", StatementAnchor: &v1pb.StatementAnchor{Spec: "spec-1", SheetSha256: strings.Repeat("a", 64)}},
		{Comment: "anchor elsewhere", Root: &root.Name, StatementAnchor: &v1pb.StatementAnchor{Spec: "spec-2", SheetSha256: anchor.SheetSha256, StartPosition: anchor.StartPosition, EndPosition: anchor.EndPosition}},
		{Comment: "anchor on unanchored thread", Root: &unanchored.Name, StatementAnchor: anchor},
		{Comment: "event", Event: &v1pb.IssueComment_ReviewSubmission_{ReviewSubmission: &v1pb.IssueComment_ReviewSubmission{}}},
	}
	for _, comment := range invalid {
		_, err := service.CreateIssueComment(ctx, connect.NewRequest(&v1pb.CreateIssueCommentRequest{Parent: parent, IssueComment: comment}))
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	}
	derived := create(&v1pb.IssueComment{Comment: "anchor derives the thread", StatementAnchor: anchor})
	require.Equal(t, v1pb.IssueComment_OPEN, derived.GetThreadState())
}

func commentNames(comments []*v1pb.IssueComment) []string {
	names := make([]string, 0, len(comments))
	for _, comment := range comments {
		names = append(names, comment.Name)
	}
	return names
}

func TestIssueCommentAllowMissing(t *testing.T) {
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	service := newIssueServiceForTest(t, stores)
	_, issue := createIssueServiceApprovalIssue(ctx, t, stores)
	parent := common.FormatIssue(issue.ProjectID, issue.UID)
	sheets, err := stores.CreateSheets(ctx, issue.ProjectID, &store.SheetMessage{Statement: "SELECT 1;\nSELECT 2;\nSELECT 3;\nSELECT 4;\nSELECT 5;"})
	require.NoError(t, err)
	anchor := &v1pb.StatementAnchor{
		Spec: "spec-1", SheetSha256: sheets[0].Sha256,
		StartPosition: &v1pb.Position{Line: 2}, EndPosition: &v1pb.Position{Line: 4},
	}
	root, err := service.CreateIssueComment(ctx, connect.NewRequest(&v1pb.CreateIssueCommentRequest{
		Parent: parent, IssueComment: &v1pb.IssueComment{Comment: "root", StatementAnchor: anchor},
	}))
	require.NoError(t, err)
	missing := common.FormatIssueComment(parent, "missing")
	foreignRoot := strings.Replace(root.Msg.Name, "projects/project-a/", "projects/other/", 1)
	expanded := proto.CloneOf(anchor)
	expanded.EndPosition.Line = 5
	wrongSpec := proto.CloneOf(anchor)
	wrongSpec.Spec = "spec-2"

	count := func() int {
		t.Helper()
		comments, err := stores.ListIssueComment(ctx, &store.FindIssueCommentMessage{
			ProjectID: issue.ProjectID, IssueUID: &issue.UID,
		})
		require.NoError(t, err)
		return len(comments)
	}
	for _, tc := range []struct {
		name    string
		comment *v1pb.IssueComment
		invalid bool
	}{
		{"general", &v1pb.IssueComment{Comment: "general"}, false},
		{"open thread", &v1pb.IssueComment{Comment: "thread", ThreadState: v1pb.IssueComment_OPEN.Enum()}, false},
		{"anchored thread", &v1pb.IssueComment{Comment: "thread", ThreadState: v1pb.IssueComment_OPEN.Enum(), StatementAnchor: anchor}, false},
		{"derived thread", &v1pb.IssueComment{Comment: "thread", StatementAnchor: anchor}, false},
		{"reply", &v1pb.IssueComment{Comment: "reply", Root: &root.Msg.Name}, false},
		{"anchored reply", &v1pb.IssueComment{Comment: "reply", Root: &root.Msg.Name, StatementAnchor: anchor}, false},
		{"reply with state", &v1pb.IssueComment{Comment: "reply", Root: &root.Msg.Name, ThreadState: v1pb.IssueComment_OPEN.Enum()}, true},
		{"resolved on create", &v1pb.IssueComment{Comment: "thread", ThreadState: v1pb.IssueComment_RESOLVED.Enum()}, true},
		{"unspecified state", &v1pb.IssueComment{Comment: "thread", ThreadState: v1pb.IssueComment_THREAD_STATE_UNSPECIFIED.Enum()}, true},
		{"unknown state", &v1pb.IssueComment{Comment: "thread", ThreadState: v1pb.IssueComment_ThreadState(99).Enum()}, true},
		{"empty comment", &v1pb.IssueComment{StatementAnchor: anchor}, true},
		{"missing root", &v1pb.IssueComment{Comment: "reply", Root: &missing}, true},
		{"foreign root", &v1pb.IssueComment{Comment: "reply", Root: &foreignRoot}, true},
		{"malformed anchor", &v1pb.IssueComment{Comment: "thread", StatementAnchor: &v1pb.StatementAnchor{}}, true},
		{"expanded reply", &v1pb.IssueComment{Comment: "reply", Root: &root.Msg.Name, StatementAnchor: expanded}, true},
		{"different spec", &v1pb.IssueComment{Comment: "reply", Root: &root.Msg.Name, StatementAnchor: wrongSpec}, true},
		{"event", &v1pb.IssueComment{Comment: "event", Event: &v1pb.IssueComment_ReviewSubmission_{ReviewSubmission: &v1pb.IssueComment_ReviewSubmission{}}}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before := count()
			input := proto.CloneOf(tc.comment)
			input.Name = missing
			response, err := service.UpdateIssueComment(ctx, connect.NewRequest(&v1pb.UpdateIssueCommentRequest{
				Parent: parent, AllowMissing: true, IssueComment: input,
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"comment"}},
			}))
			if tc.invalid {
				require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
				require.Equal(t, before, count(), "invalid creation must not write a general comment")
				return
			}
			require.NoError(t, err)
			require.Equal(t, before+1, count())
			require.Equal(t, input.Comment, response.Msg.Comment)
			require.Equal(t, input.Root, response.Msg.Root)
			require.True(t, proto.Equal(input.StatementAnchor, response.Msg.StatementAnchor))
			state := input.ThreadState
			if input.Root == nil && input.StatementAnchor != nil {
				state = v1pb.IssueComment_OPEN.Enum()
			}
			require.Equal(t, state, response.Msg.ThreadState)

			_, _, id, err := common.GetProjectIDIssueUIDIssueCommentID(response.Msg.Name)
			require.NoError(t, err)
			persisted, err := stores.GetIssueComment(ctx, &store.FindIssueCommentMessage{
				ProjectID: issue.ProjectID, IssueUID: &issue.UID, ResourceID: &id,
			})
			require.NoError(t, err)
			require.True(t, proto.Equal(response.Msg, convertToIssueComment(parent, persisted)))
		})
	}

	// A present resource still applies only the mask, even with allow_missing.
	updated, err := service.UpdateIssueComment(ctx, connect.NewRequest(&v1pb.UpdateIssueCommentRequest{
		Parent: parent, AllowMissing: true, UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"comment"}},
		IssueComment: &v1pb.IssueComment{
			Name: root.Msg.Name, Comment: "edited",
			Root: &foreignRoot, ThreadState: v1pb.IssueComment_RESOLVED.Enum(), StatementAnchor: expanded,
		},
	}))
	require.NoError(t, err)
	require.Equal(t, "edited", updated.Msg.Comment)
	require.Nil(t, updated.Msg.Root)
	require.Equal(t, v1pb.IssueComment_OPEN, updated.Msg.GetThreadState())
	require.True(t, proto.Equal(anchor, updated.Msg.StatementAnchor))

	before := count()
	_, err = service.UpdateIssueComment(ctx, connect.NewRequest(&v1pb.UpdateIssueCommentRequest{
		Parent: parent, IssueComment: &v1pb.IssueComment{Name: missing, Comment: "missing", StatementAnchor: anchor},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"comment"}},
	}))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
	require.Equal(t, before, count())
}
