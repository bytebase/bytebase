package v1

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"

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

// TestDefaultReviewRulePolicy pins that the stand-in for a missing row is
// every rule, and that it survives the store-to-v1 conversion.
func TestDefaultReviewRulePolicy(t *testing.T) {
	t.Parallel()
	message, err := getDefaultReviewRulePolicy("projects/p1")
	require.NoError(t, err)
	require.Equal(t, storepb.Policy_PROJECT, message.ResourceType)
	require.Equal(t, "projects/p1", message.Resource)
	require.Equal(t, storepb.Policy_REVIEW_RULE, message.Type)
	require.True(t, message.Enforce)

	policy, err := convertToPolicy(message)
	require.NoError(t, err)
	require.Equal(t, "projects/p1/policies/review_rule", policy.Name)
	require.Equal(t, v1pb.PolicyType_REVIEW_RULE, policy.Type)
	got := policy.GetReviewRulePolicy().Rules
	require.Len(t, got, len(v1pb.ReviewRuleType_name)-1)
	require.NotContains(t, got, v1pb.ReviewRuleType_REVIEW_RULE_TYPE_UNSPECIFIED)
	require.Equal(t, len(store.GetDefaultReviewRulePolicy().Rules), len(got))
}
