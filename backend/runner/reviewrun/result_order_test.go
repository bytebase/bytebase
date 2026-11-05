package reviewrun

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

func TestSortReviewResults(t *testing.T) {
	t.Parallel()
	const (
		p0          = storepb.IssueCommentPayload_ReviewMetadata_P0
		p2          = storepb.IssueCommentPayload_ReviewMetadata_P2
		unspecified = storepb.IssueCommentPayload_ReviewMetadata_PRIORITY_UNSPECIFIED
	)
	result := func(comment, specID string, line int32, priority storepb.IssueCommentPayload_ReviewMetadata_Priority) *store.IssueCommentMessage {
		payload := &storepb.IssueCommentPayload{
			Comment:        comment,
			ReviewMetadata: &storepb.IssueCommentPayload_ReviewMetadata{Priority: priority},
		}
		if specID != "" {
			payload.StatementAnchor = &storepb.IssueCommentPayload_StatementAnchor{
				SpecId:        specID,
				StartPosition: &storepb.Position{Line: line},
				EndPosition:   &storepb.Position{Line: line},
			}
		}
		return &store.IssueCommentMessage{Payload: payload}
	}
	results := []*store.IssueCommentMessage{
		result("b line 1", "spec-b", 1, p0),
		result("whole change", "", 0, p0),
		result("a line 28 P2", "spec-a", 28, p2),
		result("a line 16", "spec-a", 16, unspecified),
		result("a line 28 unspecified", "spec-a", 28, unspecified),
		result("a line 28 P0", "spec-a", 28, p0),
		result("a line 28 P2 again", "spec-a", 28, p2),
		result("removed spec", "spec-removed", 1, p0),
	}

	sortReviewResults([]*storepb.PlanConfig_Spec{{Id: "spec-a"}, {Id: "spec-b"}}, results)

	var comments []string
	for _, r := range results {
		comments = append(comments, r.Payload.Comment)
	}
	require.Equal(t, []string{
		"a line 16",
		"a line 28 P0",
		"a line 28 P2",
		"a line 28 P2 again",
		"a line 28 unspecified",
		"b line 1",
		"whole change",
		"removed spec",
	}, comments)
}
