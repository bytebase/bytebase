package tests

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/bytebase/bytebase/backend/common"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// TestReviewRulePolicy walks the REVIEW_RULE policy through the org policy
// service at both levels: the all-rules default on a missing row, create,
// get, list, update, validation, and delete.
func TestReviewRulePolicy(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl, ctx := startWorkspace(ctx, t)

	projectPolicyName := ctl.project.Name + "/policies/review_rule"
	rules := func(policy *v1pb.Policy) []v1pb.ReviewRuleType {
		return policy.GetReviewRulePolicy().GetRules()
	}
	reviewRule := func(list ...v1pb.ReviewRuleType) *v1pb.Policy {
		return &v1pb.Policy{
			Type:   v1pb.PolicyType_REVIEW_RULE,
			Policy: &v1pb.Policy_ReviewRulePolicy{ReviewRulePolicy: &v1pb.ReviewRulePolicy{Rules: list}},
		}
	}

	// No row: the default is every rule, not NotFound.
	got, err := ctl.orgPolicyServiceClient.GetPolicy(ctx, connect.NewRequest(&v1pb.GetPolicyRequest{Name: projectPolicyName}))
	a.NoError(err)
	a.Equal(projectPolicyName, got.Msg.Name)
	a.Equal(v1pb.PolicyType_REVIEW_RULE, got.Msg.Type)
	a.Equal(v1pb.PolicyResourceType_PROJECT, got.Msg.ResourceType)
	a.Len(rules(got.Msg), len(v1pb.ReviewRuleType_name)-1)

	// Create on the project.
	created, err := ctl.orgPolicyServiceClient.CreatePolicy(ctx, connect.NewRequest(&v1pb.CreatePolicyRequest{
		Parent: ctl.project.Name,
		Policy: reviewRule(v1pb.ReviewRuleType_SYNTAX, v1pb.ReviewRuleType_DISALLOW_DROP_OBJECT),
	}))
	a.NoError(err)
	a.Equal(projectPolicyName, created.Msg.Name)
	a.Equal([]v1pb.ReviewRuleType{v1pb.ReviewRuleType_SYNTAX, v1pb.ReviewRuleType_DISALLOW_DROP_OBJECT}, rules(created.Msg))

	got, err = ctl.orgPolicyServiceClient.GetPolicy(ctx, connect.NewRequest(&v1pb.GetPolicyRequest{Name: projectPolicyName}))
	a.NoError(err)
	a.Equal(rules(created.Msg), rules(got.Msg))

	// List with the type filter finds it.
	listed, err := ctl.orgPolicyServiceClient.ListPolicies(ctx, connect.NewRequest(&v1pb.ListPoliciesRequest{
		Parent:     ctl.project.Name,
		PolicyType: v1pb.PolicyType_REVIEW_RULE.Enum(),
	}))
	a.NoError(err)
	a.Len(listed.Msg.Policies, 1)
	a.Equal(projectPolicyName, listed.Msg.Policies[0].Name)

	// Update through the review_rule_policy mask path.
	updatedPolicy := reviewRule(v1pb.ReviewRuleType_REQUIRE_WHERE)
	updatedPolicy.Name = projectPolicyName
	updated, err := ctl.orgPolicyServiceClient.UpdatePolicy(ctx, connect.NewRequest(&v1pb.UpdatePolicyRequest{
		Policy:     updatedPolicy,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"review_rule_policy"}},
	}))
	a.NoError(err)
	a.Equal([]v1pb.ReviewRuleType{v1pb.ReviewRuleType_REQUIRE_WHERE}, rules(updated.Msg))

	// Validation: unspecified and repeated rules are refused on update and create.
	for _, bad := range [][]v1pb.ReviewRuleType{
		{v1pb.ReviewRuleType_REVIEW_RULE_TYPE_UNSPECIFIED},
		{v1pb.ReviewRuleType_SYNTAX, v1pb.ReviewRuleType_SYNTAX},
	} {
		badPolicy := reviewRule(bad...)
		badPolicy.Name = projectPolicyName
		_, err = ctl.orgPolicyServiceClient.UpdatePolicy(ctx, connect.NewRequest(&v1pb.UpdatePolicyRequest{
			Policy:     badPolicy,
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"review_rule_policy"}},
		}))
		a.Equal(connect.CodeInvalidArgument, connect.CodeOf(err))
	}

	// The policy does not attach to an environment.
	_, err = ctl.orgPolicyServiceClient.CreatePolicy(ctx, connect.NewRequest(&v1pb.CreatePolicyRequest{
		Parent: "environments/prod",
		Policy: reviewRule(v1pb.ReviewRuleType_SYNTAX),
	}))
	a.Equal(connect.CodeInvalidArgument, connect.CodeOf(err))

	// Workspace level.
	stores := getStore(t, ctl.server)
	workspaceID, err := stores.GetWorkspaceID(ctx)
	a.NoError(err)
	workspacePolicyName := common.FormatWorkspace(workspaceID) + "/policies/review_rule"
	got, err = ctl.orgPolicyServiceClient.GetPolicy(ctx, connect.NewRequest(&v1pb.GetPolicyRequest{Name: workspacePolicyName}))
	a.NoError(err)
	a.Equal(v1pb.PolicyResourceType_WORKSPACE, got.Msg.ResourceType)
	a.Len(rules(got.Msg), len(v1pb.ReviewRuleType_name)-1)
	created, err = ctl.orgPolicyServiceClient.CreatePolicy(ctx, connect.NewRequest(&v1pb.CreatePolicyRequest{
		Parent: common.FormatWorkspace(workspaceID),
		Policy: reviewRule(v1pb.ReviewRuleType_DISALLOW_TRUNCATE),
	}))
	a.NoError(err)
	a.Equal(workspacePolicyName, created.Msg.Name)

	// The effective set is the nearest policy: the project's while it has one,
	// the workspace's after the project's is deleted.
	projectID, err := common.GetProjectID(ctl.project.Name)
	a.NoError(err)
	effective, err := stores.GetEffectiveReviewRulePolicy(ctx, workspaceID, projectID)
	a.NoError(err)
	a.Equal("REQUIRE_WHERE", effective.Rules[0].String())
	a.Len(effective.Rules, 1)

	_, err = ctl.orgPolicyServiceClient.DeletePolicy(ctx, connect.NewRequest(&v1pb.DeletePolicyRequest{Name: projectPolicyName}))
	a.NoError(err)
	effective, err = stores.GetEffectiveReviewRulePolicy(ctx, workspaceID, projectID)
	a.NoError(err)
	a.Equal("DISALLOW_TRUNCATE", effective.Rules[0].String())
	a.Len(effective.Rules, 1)

	// Deleted: back to the default.
	got, err = ctl.orgPolicyServiceClient.GetPolicy(ctx, connect.NewRequest(&v1pb.GetPolicyRequest{Name: projectPolicyName}))
	a.NoError(err)
	a.Len(rules(got.Msg), len(v1pb.ReviewRuleType_name)-1)
}
