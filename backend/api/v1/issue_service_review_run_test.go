package v1

import (
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
)

// TestRunReviewCreatesRuleRunWithReviewOff pins that switching every rule off
// does not refuse a rule run: the run completes with no findings, which
// resolves the findings of earlier runs.
func TestRunReviewCreatesRuleRunWithReviewOff(t *testing.T) {
	t.Parallel()
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	_, issue := createIssueServiceApprovalIssue(ctx, t, stores)
	service := newIssueServiceForTest(t, stores)

	// REQUIRE_WHERE without SYNTAX is review switched off.
	payload, err := protojson.Marshal(&storepb.ReviewRulePolicy{Rules: []storepb.ReviewRuleType{storepb.ReviewRuleType_REQUIRE_WHERE}})
	require.NoError(t, err)
	_, err = stores.CreatePolicy(ctx, &store.PolicyMessage{
		Workspace:    "default",
		ResourceType: storepb.Policy_PROJECT,
		Resource:     common.FormatProject("project-a"),
		Type:         storepb.Policy_REVIEW_RULE,
		Payload:      string(payload),
		Enforce:      true,
	})
	require.NoError(t, err)

	resp, err := service.RunReview(ctx, connect.NewRequest(&v1pb.RunReviewRequest{
		Name: fmt.Sprintf("projects/project-a/issues/%d/reviewRuns/rule", issue.UID),
	}))
	require.NoError(t, err)
	require.Equal(t, v1pb.ReviewRun_RULE, resp.Msg.Type)
	require.Equal(t, v1pb.ReviewRun_AVAILABLE, resp.Msg.Status)
}
