package v1

import (
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/iam"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

func TestReviewAIPolicyConverterRoundTrip(t *testing.T) {
	t.Parallel()
	const text = "Every table has a primary key.\n\nAn index on a table over 1 million rows is created CONCURRENTLY.\n表名用蛇形命名。"

	payload := convertToStorePBReviewAIPolicy(&v1pb.ReviewAIPolicy{Content: text})
	require.Equal(t, text, payload.Content)

	payloadBytes, err := protojson.Marshal(payload)
	require.NoError(t, err)
	out, err := convertToV1PBReviewAIPolicy(string(payloadBytes))
	require.NoError(t, err)
	require.Equal(t, text, out.ReviewAiPolicy.Content)

	require.Empty(t, convertToStorePBReviewAIPolicy(nil).Content)
	_, err = convertToV1PBReviewAIPolicy("not json")
	require.Error(t, err)
}

func TestReviewAIPolicyTypeConversion(t *testing.T) {
	t.Parallel()
	storeType, err := convertV1PBToStorePBPolicyType(v1pb.PolicyType_REVIEW_AI)
	require.NoError(t, err)
	require.Equal(t, storepb.Policy_REVIEW_AI, storeType)
	require.Equal(t, v1pb.PolicyType_REVIEW_AI, convertStorePBToV1PBPolicyType(storepb.Policy_REVIEW_AI))
}

func TestReviewAIPolicyResourceTypes(t *testing.T) {
	t.Parallel()
	require.NoError(t, validatePolicyType(storepb.Policy_REVIEW_AI, storepb.Policy_WORKSPACE))
	require.NoError(t, validatePolicyType(storepb.Policy_REVIEW_AI, storepb.Policy_PROJECT))
	require.Error(t, validatePolicyType(storepb.Policy_REVIEW_AI, storepb.Policy_ENVIRONMENT))
}

func TestReviewAIPolicyPathMatchType(t *testing.T) {
	t.Parallel()
	require.True(t, pathMatchType("review_ai_policy", storepb.Policy_REVIEW_AI))
	require.False(t, pathMatchType("review_rule_policy", storepb.Policy_REVIEW_AI))
	require.False(t, pathMatchType("review_ai_policy", storepb.Policy_REVIEW_RULE))
}

func TestValidateReviewAIPolicyPayload(t *testing.T) {
	t.Parallel()
	reviewAI := func(content string) *v1pb.Policy {
		return &v1pb.Policy{Policy: &v1pb.Policy_ReviewAiPolicy{
			ReviewAiPolicy: &v1pb.ReviewAIPolicy{Content: content},
		}}
	}
	for _, tc := range []struct {
		name   string
		policy *v1pb.Policy
		ok     bool
	}{
		{"one sentence", reviewAI("Every table has a primary key."), true},
		{"many lines", reviewAI("# Standards\n\n- Every table has a primary key.\n- No column is dropped in production.\n"), true},
		{"at the limit", reviewAI(strings.Repeat("a", maxReviewAIPolicyBytes)), true},
		{"empty", reviewAI(""), false},
		{"whitespace only", reviewAI(" \n\t"), false},
		{"over the limit", reviewAI(strings.Repeat("a", maxReviewAIPolicyBytes+1)), false},
		{"other payload", &v1pb.Policy{Policy: &v1pb.Policy_QueryDataPolicy{QueryDataPolicy: &v1pb.QueryDataPolicy{}}}, false},
		{"nil payload", &v1pb.Policy{Policy: &v1pb.Policy_ReviewAiPolicy{}}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validatePolicyPayload(storepb.Policy_REVIEW_AI, tc.policy)
			if tc.ok {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

// TestReviewAIPolicyService walks the REVIEW_AI policy through the service
// against a real store at both levels: NotFound on a missing row, create,
// get, list, update, validation, enforce, and delete. The effective lookup
// is the store's behavior and is tested there.
func TestReviewAIPolicyService(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	iamManager, err := iam.NewManager(stores, nil, false)
	a.NoError(err)
	service := NewOrgPolicyService(stores, nil, iamManager)

	const projectName = "projects/project-a"
	projectPolicyName := projectName + "/policies/review_ai"
	workspaceName := common.FormatWorkspace("default")
	workspacePolicyName := workspaceName + "/policies/review_ai"
	content := func(policy *v1pb.Policy) string {
		return policy.GetReviewAiPolicy().GetContent()
	}
	reviewAI := func(text string) *v1pb.Policy {
		return &v1pb.Policy{
			Type:   v1pb.PolicyType_REVIEW_AI,
			Policy: &v1pb.Policy_ReviewAiPolicy{ReviewAiPolicy: &v1pb.ReviewAIPolicy{Content: text}},
		}
	}
	get := func(name string) (*v1pb.Policy, error) {
		response, err := service.GetPolicy(ctx, connect.NewRequest(&v1pb.GetPolicyRequest{Name: name}))
		if err != nil {
			return nil, err
		}
		return response.Msg, nil
	}
	// No row: NotFound. A missing policy has no default, unlike the rule policy.
	_, err = get(projectPolicyName)
	a.Equal(connect.CodeNotFound, connect.CodeOf(err))

	// Create on the project.
	const projectText = "Every new column on orders has a comment."
	created, err := service.CreatePolicy(ctx, connect.NewRequest(&v1pb.CreatePolicyRequest{
		Parent: projectName,
		Policy: reviewAI(projectText),
	}))
	a.NoError(err)
	a.Equal(projectPolicyName, created.Msg.Name)
	a.Equal(v1pb.PolicyType_REVIEW_AI, created.Msg.Type)
	a.Equal(v1pb.PolicyResourceType_PROJECT, created.Msg.ResourceType)
	a.Equal(projectText, content(created.Msg))
	a.True(created.Msg.Enforce)

	got, err := get(projectPolicyName)
	a.NoError(err)
	a.Equal(projectText, content(got))

	// List with the type filter finds it.
	listed, err := service.ListPolicies(ctx, connect.NewRequest(&v1pb.ListPoliciesRequest{
		Parent:     projectName,
		PolicyType: v1pb.PolicyType_REVIEW_AI.Enum(),
	}))
	a.NoError(err)
	a.Len(listed.Msg.Policies, 1)
	a.Equal(projectPolicyName, listed.Msg.Policies[0].Name)

	// Update through the review_ai_policy mask path.
	const updatedText = "Every new column on orders has a comment, and no column of orders is dropped."
	updatedPolicy := reviewAI(updatedText)
	updatedPolicy.Name = projectPolicyName
	updated, err := service.UpdatePolicy(ctx, connect.NewRequest(&v1pb.UpdatePolicyRequest{
		Policy:     updatedPolicy,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"review_ai_policy"}},
	}))
	a.NoError(err)
	a.Equal(updatedText, content(updated.Msg))

	// An update that declares another type is refused, and the row keeps its
	// content instead of storing the other type's empty payload.
	mismatched := reviewAI(updatedText)
	mismatched.Name = projectPolicyName
	mismatched.Type = v1pb.PolicyType_REVIEW_RULE
	_, err = service.UpdatePolicy(ctx, connect.NewRequest(&v1pb.UpdatePolicyRequest{
		Policy:     mismatched,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"review_ai_policy"}},
	}))
	a.Equal(connect.CodeInvalidArgument, connect.CodeOf(err))
	got, err = get(projectPolicyName)
	a.NoError(err)
	a.Equal(updatedText, content(got))

	// Validation: blank or oversized policy content is refused.
	for _, bad := range []string{"", " \n\t", strings.Repeat("x", maxReviewAIPolicyBytes+1)} {
		badPolicy := reviewAI(bad)
		badPolicy.Name = projectPolicyName
		_, err = service.UpdatePolicy(ctx, connect.NewRequest(&v1pb.UpdatePolicyRequest{
			Policy:     badPolicy,
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"review_ai_policy"}},
		}))
		a.Equal(connect.CodeInvalidArgument, connect.CodeOf(err))
	}

	// The policy does not attach to an environment.
	_, err = service.CreatePolicy(ctx, connect.NewRequest(&v1pb.CreatePolicyRequest{
		Parent: "environments/prod",
		Policy: reviewAI(projectText),
	}))
	a.Equal(connect.CodeInvalidArgument, connect.CodeOf(err))

	// Workspace level: NotFound, then create.
	_, err = get(workspacePolicyName)
	a.Equal(connect.CodeNotFound, connect.CodeOf(err))
	const workspaceText = "An index on a table over 1 million rows is created CONCURRENTLY."
	created, err = service.CreatePolicy(ctx, connect.NewRequest(&v1pb.CreatePolicyRequest{
		Parent: workspaceName,
		Policy: reviewAI(workspaceText),
	}))
	a.NoError(err)
	a.Equal(workspacePolicyName, created.Msg.Name)
	a.Equal(v1pb.PolicyResourceType_WORKSPACE, created.Msg.ResourceType)
	got, err = get(workspacePolicyName)
	a.NoError(err)
	a.Equal(workspaceText, content(got))

	// Enforce is switched through its own mask path.
	off := reviewAI(updatedText)
	off.Name = projectPolicyName
	off.Enforce = false
	updated, err = service.UpdatePolicy(ctx, connect.NewRequest(&v1pb.UpdatePolicyRequest{
		Policy:     off,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"enforce"}},
	}))
	a.NoError(err)
	a.False(updated.Msg.Enforce)

	// Deleted: NotFound, and the workspace policy is untouched.
	_, err = service.DeletePolicy(ctx, connect.NewRequest(&v1pb.DeletePolicyRequest{Name: projectPolicyName}))
	a.NoError(err)
	_, err = get(projectPolicyName)
	a.Equal(connect.CodeNotFound, connect.CodeOf(err))
	got, err = get(workspacePolicyName)
	a.NoError(err)
	a.Equal(workspaceText, content(got))
}
