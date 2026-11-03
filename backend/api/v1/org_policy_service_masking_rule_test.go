package v1

import (
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/type/expr"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/bytebase/bytebase/backend/component/iam"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
)

func TestMaskingRulePolicyRefusesTypesThatMaskNoValue(t *testing.T) {
	t.Parallel()
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	iamManager, err := iam.NewManager(stores, nil, false)
	require.NoError(t, err)
	service := NewOrgPolicyService(stores, newEnterpriseLicenseService(ctx, t, stores), iamManager)

	_, err = stores.UpsertSetting(ctx, &store.SettingMessage{
		Name:      storepb.SettingName_SEMANTIC_TYPES,
		Workspace: "default",
		Value: &storepb.SemanticTypeSetting{Types: []*storepb.SemanticTypeSetting_SemanticType{
			{Id: "full", Title: "Full", Algorithm: &storepb.Algorithm{Mask: &storepb.Algorithm_FullMask_{FullMask: &storepb.Algorithm_FullMask{Substitution: "***"}}}},
			{Id: "no-method", Title: "No method"},
			{Id: "empty-range", Title: "Empty range", Algorithm: &storepb.Algorithm{Mask: &storepb.Algorithm_RangeMask_{RangeMask: &storepb.Algorithm_RangeMask{}}}},
			{Id: "no-margin", Title: "Outer with no margin", Algorithm: &storepb.Algorithm{Mask: &storepb.Algorithm_InnerOuterMask_{InnerOuterMask: &storepb.Algorithm_InnerOuterMask{Type: storepb.Algorithm_InnerOuterMask_OUTER}}}},
		}},
	})
	require.NoError(t, err)

	const policyName = "workspaces/default/policies/masking_rule"
	rule := func(id, semanticType string) *v1pb.MaskingRulePolicy_MaskingRule {
		return &v1pb.MaskingRulePolicy_MaskingRule{
			Id:           id,
			Condition:    &expr.Expr{Title: "Rule " + id, Expression: `resource.environment_id == "prod"`},
			SemanticType: semanticType,
		}
	}
	policy := func(rules ...*v1pb.MaskingRulePolicy_MaskingRule) *v1pb.Policy {
		return &v1pb.Policy{
			Name:   policyName,
			Type:   v1pb.PolicyType_MASKING_RULE,
			Policy: &v1pb.Policy_MaskingRulePolicy{MaskingRulePolicy: &v1pb.MaskingRulePolicy{Rules: rules}},
		}
	}
	save := func(rules ...*v1pb.MaskingRulePolicy_MaskingRule) error {
		_, err := service.UpdatePolicy(ctx, connect.NewRequest(&v1pb.UpdatePolicyRequest{
			Policy:       policy(rules...),
			AllowMissing: true,
			UpdateMask:   &fieldmaskpb.FieldMask{Paths: []string{"masking_rule_policy"}},
		}))
		return err
	}

	// No policy is stored yet, so these saves go through CreatePolicy.
	assert.ErrorContains(t, save(rule("a", "missing")), `masking rule "Rule a" (ID a) names semantic type "missing", which does not exist`)
	for _, semanticType := range []string{"no-method", "empty-range", "no-margin"} {
		assert.ErrorContains(t, save(rule("a", semanticType)), "which masks no value", semanticType)
	}
	require.NoError(t, save(rule("a", "full"), rule("b", "bb.default"), rule("c", "bb.default-partial")))

	// A rule stored before this check, naming a type that does not exist.
	stored, _, err := service.findPolicyMessage(ctx, policyName)
	require.NoError(t, err)
	payload, err := service.convertPolicyPayloadToString(ctx, policy(rule("a", "full"), rule("z", "missing")))
	require.NoError(t, err)
	_, err = stores.UpdatePolicy(ctx, &store.UpdatePolicyMessage{
		ResourceType: stored.ResourceType,
		Resource:     stored.Resource,
		Type:         stored.Type,
		Workspace:    stored.Workspace,
		Payload:      &payload,
	})
	require.NoError(t, err)

	// Moved ahead with its content unchanged, it would decide the columns both
	// rules match.
	assert.ErrorContains(t, save(rule("z", "missing"), rule("a", "full")), `"Rule z" (ID z)`)
	edited := rule("a", "full")
	edited.Condition.Expression = `resource.environment_id == "test"`
	assert.ErrorContains(t, save(edited, rule("z", "missing")), `"Rule z" (ID z)`)
	require.NoError(t, save(rule("a", "full")))

	var noCondition error
	assert.NotPanics(t, func() {
		noCondition = save(&v1pb.MaskingRulePolicy_MaskingRule{Id: "n", SemanticType: "full"})
	})
	assert.ErrorContains(t, noCondition, `masking rule "n" must have a condition`)
}
