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

func TestRunReviewRefusesRuleReviewWithoutSyntax(t *testing.T) {
	t.Parallel()
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	_, issue := createIssueServiceApprovalIssue(ctx, t, stores)
	service := newIssueServiceForTest(t, stores)

	setProjectRules := func(rules ...storepb.ReviewRuleType) {
		t.Helper()
		payload, err := protojson.Marshal(&storepb.ReviewRulePolicy{Rules: rules})
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
	}
	runRuleReview := func() (*v1pb.ReviewRun, error) {
		resp, err := service.RunReview(ctx, connect.NewRequest(&v1pb.RunReviewRequest{
			Name: fmt.Sprintf("projects/project-a/issues/%d/reviewRuns/rule", issue.UID),
		}))
		if err != nil {
			return nil, err
		}
		return resp.Msg, nil
	}

	setProjectRules(storepb.ReviewRuleType_REQUIRE_WHERE)
	_, err := runRuleReview()
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))

	setProjectRules(storepb.ReviewRuleType_SYNTAX, storepb.ReviewRuleType_REQUIRE_WHERE)
	run, err := runRuleReview()
	require.NoError(t, err)
	require.Equal(t, v1pb.ReviewRun_RULE, run.Type)
	require.Equal(t, v1pb.ReviewRun_AVAILABLE, run.Status)
}
