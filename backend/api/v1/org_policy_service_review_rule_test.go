package v1

import (
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/iam"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
)

// TestReviewRuleTypeEnumsMirror holds bytebase.v1.ReviewRuleType and
// bytebase.store.ReviewRuleType to the same numbering, which is what lets the
// converters cast between them.
func TestReviewRuleTypeEnumsMirror(t *testing.T) {
	t.Parallel()
	require.Equal(t, len(v1pb.ReviewRuleType_name), len(storepb.ReviewRuleType_name))
	for number, name := range v1pb.ReviewRuleType_name {
		require.Equal(t, name, storepb.ReviewRuleType_name[number], "value %d", number)
	}
}

// TestReviewMetadataPriorityEnumsMirror holds the v1 and store review result
// priorities to the same numbering for convertToIssueCommentReviewMetadata.
func TestReviewMetadataPriorityEnumsMirror(t *testing.T) {
	t.Parallel()
	require.Equal(t, len(v1pb.IssueComment_ReviewMetadata_Priority_name), len(storepb.IssueCommentPayload_ReviewMetadata_Priority_name))
	for number, name := range v1pb.IssueComment_ReviewMetadata_Priority_name {
		require.Equal(t, name, storepb.IssueCommentPayload_ReviewMetadata_Priority_name[number], "value %d", number)
	}
}

func TestConvertToIssueCommentReviewResult(t *testing.T) {
	t.Parallel()
	got := convertToIssueComment("projects/p/issues/101", &store.IssueCommentMessage{
		ResourceID: "c1",
		Payload: &storepb.IssueCommentPayload{
			Comment: "UPDATE without WHERE",
			ReviewMetadata: &storepb.IssueCommentPayload_ReviewMetadata{
				RunType:  storepb.ReviewRun_RULE,
				RuleType: storepb.ReviewRuleType_REQUIRE_WHERE,
				Priority: storepb.IssueCommentPayload_ReviewMetadata_P1,
				Targets:  []string{"instances/i/databases/a", "instances/i/databases/b"},
			},
		},
	})
	require.Empty(t, got.Creator, "a review result has no creator")
	require.Equal(t, &v1pb.IssueComment_ReviewMetadata{
		RunType:  v1pb.ReviewRun_RULE,
		RuleType: v1pb.ReviewRuleType_REQUIRE_WHERE,
		Priority: v1pb.IssueComment_ReviewMetadata_P1,
		Targets:  []string{"instances/i/databases/a", "instances/i/databases/b"},
	}, got.ReviewMetadata)

	person := convertToIssueComment("projects/p/issues/101", &store.IssueCommentMessage{
		ResourceID:   "c2",
		CreatorEmail: "dev@example.com",
		Payload:      &storepb.IssueCommentPayload{Comment: "looks fine"},
	})
	require.Equal(t, "users/dev@example.com", person.Creator)
	require.Nil(t, person.ReviewMetadata)
}

func TestReviewRulePolicyConverterRoundTrip(t *testing.T) {
	t.Parallel()
	in := &v1pb.ReviewRulePolicy{Rules: []v1pb.ReviewRuleType{
		v1pb.ReviewRuleType_SYNTAX,
		v1pb.ReviewRuleType_DISALLOW_DROP_OBJECT,
		v1pb.ReviewRuleType_REQUIRE_PRIMARY_KEY,
	}}

	payload := convertToStorePBReviewRulePolicy(in)
	require.Equal(t, []storepb.ReviewRuleType{
		storepb.ReviewRuleType_SYNTAX,
		storepb.ReviewRuleType_DISALLOW_DROP_OBJECT,
		storepb.ReviewRuleType_REQUIRE_PRIMARY_KEY,
	}, payload.Rules)

	payloadBytes, err := protojson.Marshal(payload)
	require.NoError(t, err)
	out, err := convertToV1PBReviewRulePolicy(string(payloadBytes))
	require.NoError(t, err)
	require.Equal(t, in.Rules, out.ReviewRulePolicy.Rules)

	empty := convertToStorePBReviewRulePolicy(nil)
	require.Empty(t, empty.Rules)
}

func TestReviewRulePolicyTypeConversion(t *testing.T) {
	t.Parallel()
	storeType, err := convertV1PBToStorePBPolicyType(v1pb.PolicyType_REVIEW_RULE)
	require.NoError(t, err)
	require.Equal(t, storepb.Policy_REVIEW_RULE, storeType)
	require.Equal(t, v1pb.PolicyType_REVIEW_RULE, convertStorePBToV1PBPolicyType(storepb.Policy_REVIEW_RULE))
}

func TestReviewRulePolicyResourceTypes(t *testing.T) {
	t.Parallel()
	require.NoError(t, validatePolicyType(storepb.Policy_REVIEW_RULE, storepb.Policy_WORKSPACE))
	require.NoError(t, validatePolicyType(storepb.Policy_REVIEW_RULE, storepb.Policy_PROJECT))
	require.Error(t, validatePolicyType(storepb.Policy_REVIEW_RULE, storepb.Policy_ENVIRONMENT))
}

func TestReviewRulePolicyPathMatchType(t *testing.T) {
	t.Parallel()
	require.True(t, pathMatchType("review_rule_policy", storepb.Policy_REVIEW_RULE))
	require.False(t, pathMatchType("query_data_policy", storepb.Policy_REVIEW_RULE))
	require.False(t, pathMatchType("review_rule_policy", storepb.Policy_QUERY_DATA))
}

func TestValidateReviewRulePolicyPayload(t *testing.T) {
	t.Parallel()
	reviewRule := func(rules ...v1pb.ReviewRuleType) *v1pb.Policy {
		return &v1pb.Policy{Policy: &v1pb.Policy_ReviewRulePolicy{
			ReviewRulePolicy: &v1pb.ReviewRulePolicy{Rules: rules},
		}}
	}
	for _, tc := range []struct {
		name   string
		policy *v1pb.Policy
		ok     bool
	}{
		{"empty list switches every rule off", reviewRule(), true},
		{"known rules", reviewRule(v1pb.ReviewRuleType_SYNTAX, v1pb.ReviewRuleType_REQUIRE_WHERE), true},
		{"unspecified", reviewRule(v1pb.ReviewRuleType_REVIEW_RULE_TYPE_UNSPECIFIED), false},
		{"unknown value", reviewRule(v1pb.ReviewRuleType(9999)), false},
		{"repeated rule", reviewRule(v1pb.ReviewRuleType_SYNTAX, v1pb.ReviewRuleType_SYNTAX), false},
		{"other payload", &v1pb.Policy{Policy: &v1pb.Policy_QueryDataPolicy{QueryDataPolicy: &v1pb.QueryDataPolicy{}}}, false},
		{"nil payload", &v1pb.Policy{Policy: &v1pb.Policy_ReviewRulePolicy{}}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validatePolicyPayload(storepb.Policy_REVIEW_RULE, tc.policy)
			if tc.ok {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

// TestDefaultReviewRulePolicy pins that the stand-in for a missing workspace
// row is every rule and survives the store-to-v1 conversion, and that a
// project has no stand-in.
func TestDefaultReviewRulePolicy(t *testing.T) {
	t.Parallel()
	message, err := getDefaultReviewRulePolicy("workspaces/w1")
	require.NoError(t, err)
	require.Equal(t, storepb.Policy_WORKSPACE, message.ResourceType)
	require.Equal(t, "workspaces/w1", message.Resource)
	require.Equal(t, storepb.Policy_REVIEW_RULE, message.Type)
	require.True(t, message.Enforce)

	policy, err := convertToPolicy(message)
	require.NoError(t, err)
	require.Equal(t, "workspaces/w1/policies/review_rule", policy.Name)
	require.Equal(t, v1pb.PolicyType_REVIEW_RULE, policy.Type)
	got := policy.GetReviewRulePolicy().Rules
	require.Len(t, got, len(v1pb.ReviewRuleType_name)-1)
	require.NotContains(t, got, v1pb.ReviewRuleType_REVIEW_RULE_TYPE_UNSPECIFIED)
	require.Equal(t, len(store.GetDefaultReviewRulePolicy().Rules), len(got))

	message, err = getDefaultReviewRulePolicy("projects/p1")
	require.NoError(t, err)
	require.Nil(t, message)
}

// TestReviewRulePolicyService walks the REVIEW_RULE policy through
// OrgPolicyService at both levels against PostgreSQL: a missing row, create,
// get, list, update, validation, delete, and the nearest-wins resolution the
// store exposes to the executor.
func TestReviewRulePolicyService(t *testing.T) {
	t.Parallel()
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	iamManager, err := iam.NewManager(stores, nil, false)
	require.NoError(t, err)
	service := NewOrgPolicyService(stores, nil, iamManager)

	const (
		workspaceID = "default"
		projectID   = "project-a"
	)
	projectPolicyName := common.FormatProject(projectID) + "/policies/review_rule"
	workspacePolicyName := common.FormatWorkspace(workspaceID) + "/policies/review_rule"
	everyRule := len(v1pb.ReviewRuleType_name) - 1

	rules := func(policy *v1pb.Policy) []v1pb.ReviewRuleType {
		return policy.GetReviewRulePolicy().GetRules()
	}
	reviewRule := func(list ...v1pb.ReviewRuleType) *v1pb.Policy {
		return &v1pb.Policy{
			Type:   v1pb.PolicyType_REVIEW_RULE,
			Policy: &v1pb.Policy_ReviewRulePolicy{ReviewRulePolicy: &v1pb.ReviewRulePolicy{Rules: list}},
		}
	}
	get := func(name string) *v1pb.Policy {
		t.Helper()
		resp, err := service.GetPolicy(ctx, connect.NewRequest(&v1pb.GetPolicyRequest{Name: name}))
		require.NoError(t, err)
		return resp.Msg
	}
	getCode := func(name string) connect.Code {
		_, err := service.GetPolicy(ctx, connect.NewRequest(&v1pb.GetPolicyRequest{Name: name}))
		return connect.CodeOf(err)
	}
	create := func(parent string, policy *v1pb.Policy) (*v1pb.Policy, error) {
		resp, err := service.CreatePolicy(ctx, connect.NewRequest(&v1pb.CreatePolicyRequest{Parent: parent, Policy: policy}))
		if err != nil {
			return nil, err
		}
		return resp.Msg, nil
	}
	update := func(name string, policy *v1pb.Policy) (*v1pb.Policy, error) {
		policy.Name = name
		resp, err := service.UpdatePolicy(ctx, connect.NewRequest(&v1pb.UpdatePolicyRequest{
			Policy:     policy,
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"review_rule_policy"}},
		}))
		if err != nil {
			return nil, err
		}
		return resp.Msg, nil
	}
	effective := func() []string {
		t.Helper()
		p, err := stores.GetEffectiveReviewRulePolicy(ctx, workspaceID, projectID)
		require.NoError(t, err)
		names := make([]string, 0, len(p.Rules))
		for _, r := range p.Rules {
			names = append(names, r.String())
		}
		return names
	}

	// No row at either level: the project has no policy of its own, and the
	// workspace stands in with every rule.
	require.Equal(t, connect.CodeNotFound, getCode(projectPolicyName))
	got := get(workspacePolicyName)
	require.Equal(t, workspacePolicyName, got.Name)
	require.Equal(t, v1pb.PolicyType_REVIEW_RULE, got.Type)
	require.Equal(t, v1pb.PolicyResourceType_WORKSPACE, got.ResourceType)
	require.Len(t, rules(got), everyRule)
	require.Len(t, effective(), everyRule)

	// Create on the project and read it back.
	created, err := create(common.FormatProject(projectID), reviewRule(v1pb.ReviewRuleType_SYNTAX, v1pb.ReviewRuleType_DISALLOW_DROP_OBJECT))
	require.NoError(t, err)
	require.Equal(t, projectPolicyName, created.Name)
	require.Equal(t, []v1pb.ReviewRuleType{v1pb.ReviewRuleType_SYNTAX, v1pb.ReviewRuleType_DISALLOW_DROP_OBJECT}, rules(created))
	require.Equal(t, rules(created), rules(get(projectPolicyName)))
	require.Equal(t, []string{"SYNTAX", "DISALLOW_DROP_OBJECT"}, effective())

	// List with the type filter finds it.
	listed, err := service.ListPolicies(ctx, connect.NewRequest(&v1pb.ListPoliciesRequest{
		Parent:     common.FormatProject(projectID),
		PolicyType: v1pb.PolicyType_REVIEW_RULE.Enum(),
	}))
	require.NoError(t, err)
	require.Len(t, listed.Msg.Policies, 1)
	require.Equal(t, projectPolicyName, listed.Msg.Policies[0].Name)

	// Update through the review_rule_policy mask path.
	updated, err := update(projectPolicyName, reviewRule(v1pb.ReviewRuleType_SYNTAX, v1pb.ReviewRuleType_REQUIRE_WHERE))
	require.NoError(t, err)
	require.Equal(t, []v1pb.ReviewRuleType{v1pb.ReviewRuleType_SYNTAX, v1pb.ReviewRuleType_REQUIRE_WHERE}, rules(updated))
	require.Equal(t, []string{"SYNTAX", "REQUIRE_WHERE"}, effective())

	// Unspecified and repeated rules are refused.
	for _, bad := range [][]v1pb.ReviewRuleType{
		{v1pb.ReviewRuleType_REVIEW_RULE_TYPE_UNSPECIFIED},
		{v1pb.ReviewRuleType_SYNTAX, v1pb.ReviewRuleType_SYNTAX},
	} {
		_, err = update(projectPolicyName, reviewRule(bad...))
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	}

	// The policy does not attach to an environment.
	_, err = create("environments/prod", reviewRule(v1pb.ReviewRuleType_SYNTAX))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))

	// Workspace level, and nearest wins: the project's policy while it has
	// one, the workspace's after the project's is deleted.
	created, err = create(common.FormatWorkspace(workspaceID), reviewRule(v1pb.ReviewRuleType_SYNTAX, v1pb.ReviewRuleType_DISALLOW_TRUNCATE))
	require.NoError(t, err)
	require.Equal(t, workspacePolicyName, created.Name)
	require.Equal(t, []string{"SYNTAX", "REQUIRE_WHERE"}, effective())

	_, err = service.DeletePolicy(ctx, connect.NewRequest(&v1pb.DeletePolicyRequest{Name: projectPolicyName}))
	require.NoError(t, err)
	require.Equal(t, []string{"SYNTAX", "DISALLOW_TRUNCATE"}, effective())
	require.Equal(t, connect.CodeNotFound, getCode(projectPolicyName))
	require.Equal(t, rules(created), rules(get(workspacePolicyName)))
}
