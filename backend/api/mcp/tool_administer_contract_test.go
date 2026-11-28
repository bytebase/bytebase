package mcp

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/bytebase/bytebase/backend/common"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
)

// These tests check adminForm's contract: for a given stored state, two
// requests that would store different things get different forms.

// formsOf returns the form of each request that gets one, keyed by the
// request, and fails when two requests share a form.
func formsOf[R proto.Message](t *testing.T, requests map[string][]R, form func(requested []R) (string, bool)) map[string]string {
	t.Helper()
	forms, asked := map[string]string{}, map[string]string{}
	for name, requested := range requests {
		text, ok := form(requested)
		if !ok {
			continue
		}
		if other, taken := asked[text]; taken {
			require.Failf(t, "two requests get one form", "%s\nand\n%s\nboth get:\n%s", other, name, text)
		}
		asked[text], forms[name] = name, text
	}
	return forms
}

// writesOf returns every request that makes one change to stored: a rule
// removed, a rule moved to another place, a rule changed by one of changes, or
// one of added put at a place. Each is keyed by the list it requests, so two
// ways to reach one list are one request.
func writesOf[R proto.Message](stored, added []R, changes []func(rule R) bool) map[string][]R {
	clone := func() []R {
		rules := make([]R, len(stored))
		for i, rule := range stored {
			rules[i] = proto.CloneOf(rule)
		}
		return rules
	}
	key := func(rules []R) string {
		var sb strings.Builder
		for _, rule := range rules {
			sb.WriteString(protojson.MarshalOptions{}.Format(rule) + "\n")
		}
		return sb.String()
	}
	requests := map[string][]R{}
	for i := range stored {
		requests[key(slices.Delete(clone(), i, i+1))] = slices.Delete(clone(), i, i+1)
		for j := range stored {
			if rules := clone(); j != i {
				moved := rules[i]
				rules = slices.Insert(slices.Delete(rules, i, i+1), j, moved)
				requests[key(rules)] = rules
			}
		}
		for _, change := range changes {
			if rules := clone(); change(rules[i]) {
				requests[key(rules)] = rules
			}
		}
	}
	for _, rule := range added {
		for _, at := range []int{0, len(stored) / 2, len(stored)} {
			rules := slices.Insert(clone(), at, rule)
			requests[key(rules)] = rules
		}
	}
	return requests
}

func TestMaskingFormTellsEveryTwoWritesApart(t *testing.T) {
	rangeMask := func(end int32) *v1pb.Algorithm {
		return &v1pb.Algorithm{Mask: &v1pb.Algorithm_RangeMask_{RangeMask: &v1pb.Algorithm_RangeMask{Slices: []*v1pb.Algorithm_RangeMask_Slice{{Start: 0, End: end, Substitution: "*"}}}}}
	}
	// Two of the types read the same, and a rule can have a type the setting
	// does not hold.
	types, err := semanticTypesFromSetting([]*v1pb.SemanticTypeSetting_SemanticType{
		semanticType("card-4", "Card", rangeMask(4)),
		semanticType("card-12", "Card", rangeMask(12)),
		semanticType("email", "Email", &v1pb.Algorithm{Mask: &v1pb.Algorithm_Md5Mask{Md5Mask: &v1pb.Algorithm_MD5Mask{}}}),
	})
	require.NoError(t, err)
	typeIDs := []string{"bb.default", "bb.default-partial", "", "card-4", "card-12", "email", "gone"}
	// Twelve rules: more than the description lists. Two pairs read as one
	// scope, two rules share a title, and one condition has no scope.
	var stored []*v1pb.MaskingRulePolicy_MaskingRule
	for i := range 5 {
		stored = append(stored, maskingRule(fmt.Sprintf("f%d", i), "", fmt.Sprintf(`resource.table_name == "a_table_with_a_long_name_%02d" && resource.column_name == "a_column_with_a_long_name_%02d"`, i, i), "bb.default"))
	}
	stored = append(stored,
		maskingRule("r5", "", `resource.table_name == "pay" && resource.column_name == "amount"`, "bb.default"),
		maskingRule("r6", "Pay", `resource.table_name == "pay"`, "bb.default-partial"),
		maskingRule("r7", "", `resource.column_name in ['amount'] && resource.table_name == "pay"`, "bb.default"),
		maskingRule("r8", "Pay", `resource.table_name.startsWith("tmp_")`, "email"),
		maskingRule("r9", "Cards", `resource.column_name == "pan"`, "card-4"),
		maskingRule("r10", "Cards", `resource.column_name in ["pan"]`, "card-12"),
		maskingRule("r11", "", `resource.column_name == "ssn"`, ""))
	var changes []func(rule *v1pb.MaskingRulePolicy_MaskingRule) bool
	for _, title := range []string{"", "Pay", "Renamed"} {
		changes = append(changes, func(rule *v1pb.MaskingRulePolicy_MaskingRule) bool {
			changed := rule.GetCondition().GetTitle() != title
			rule.Condition.Title = title
			return changed
		})
	}
	for _, id := range typeIDs {
		changes = append(changes, func(rule *v1pb.MaskingRulePolicy_MaskingRule) bool {
			changed := rule.GetSemanticType() != id
			rule.SemanticType = id
			return changed
		})
	}
	// A new rule's ID and the spelling of its scope are not stated, so the new
	// rules have one ID and each its own scope, but for one that reads as a
	// stored rule.
	var added []*v1pb.MaskingRulePolicy_MaskingRule
	for _, condition := range []string{`resource.column_name == "new"`, `resource.table_name == "new"`, `resource.column_name == "ssn" // as r11`, `resource.column_name.endsWith("_id")`} {
		for _, title := range []string{"", "New"} {
			for _, id := range []string{"bb.default", "card-4", "card-12"} {
				added = append(added, maskingRule("r-new", title, condition, id))
			}
		}
	}
	alice := withUserEmail(context.Background(), "alice@example.com")
	requests := writesOf(stored, added, changes)
	forms := formsOf(t, requests, func(requested []*v1pb.MaskingRulePolicy_MaskingRule) (string, bool) {
		diff, err := diffMaskingRules(stored, requested, types, "LINK")
		require.NoError(t, err)
		form, err := maskingRulesForm(alice, diff, "LINK")
		if err != nil {
			return "", false
		}
		return form.message + "\n--\n" + form.description, true
	})
	// 12 removals, 132 moves less the 11 that two neighbors share, 26 new
	// titles, 72 new types and 72 added rules.
	require.Len(t, requests, 303)
	require.Greater(t, len(forms), 250, "most of them get a form")
	cut := 0
	for _, form := range forms {
		if strings.Contains(form, "more rules, tried after these.") || strings.Contains(form, "Their list has no room here.") {
			cut++
		}
	}
	require.Equal(t, len(forms), cut, "and no form lists the twelve rules whole")
}

func TestApprovalFormTellsEveryTwoWritesApart(t *testing.T) {
	flows := [][]string{nil, {"roles/workspaceDBA"}, {"roles/projectOwner"}, {"roles/projectOwner", "roles/workspaceDBA"}, {"roles/workspaceDBA", "roles/projectOwner"}, {"roles/custom-reviewer"}}
	// Twelve rules for database changes behind one fallback rule: more than
	// the description lists. Titles and approvers repeat.
	stored := []*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule()}
	for i := range 12 {
		rule := changeRule([]string{"", "Production", "Staging"}[i%3], fmt.Sprintf(`resource.project_id == "project-%02d" && resource.environment_id == "environment-%02d"`, i, i), flows[i%len(flows)]...)
		rule.Template.Description = []string{"", "Why"}[i%2]
		stored = append(stored, rule)
	}
	var changes []func(rule *v1pb.WorkspaceApprovalSetting_Rule) bool
	for _, title := range []string{"", "Production", "Renamed"} {
		changes = append(changes, func(rule *v1pb.WorkspaceApprovalSetting_Rule) bool {
			changed := rule.GetTemplate().GetTitle() != title
			rule.Template.Title = title
			return changed
		})
	}
	for _, description := range []string{"", "Why", "Because"} {
		changes = append(changes, func(rule *v1pb.WorkspaceApprovalSetting_Rule) bool {
			changed := rule.GetTemplate().GetDescription() != description
			rule.Template.Description = description
			return changed
		})
	}
	for _, flow := range flows {
		changes = append(changes, func(rule *v1pb.WorkspaceApprovalSetting_Rule) bool {
			changed := !slices.Equal(rule.GetTemplate().GetFlow().GetRoles(), flow)
			rule.Template.Flow = &v1pb.ApprovalFlow{Roles: flow}
			return changed
		})
	}
	var added []*v1pb.WorkspaceApprovalSetting_Rule
	for _, condition := range []string{`resource.project_id == "new"`, `resource.environment_id == "new"`} {
		for _, title := range []string{"", "New"} {
			for _, description := range []string{"", "Why"} {
				for _, flow := range flows[:3] {
					rule := changeRule(title, condition, flow...)
					rule.Template.Description = description
					added = append(added, rule)
				}
			}
		}
	}
	alice := withUserEmail(context.Background(), "alice@example.com")
	// The fallback rule stays first: a request sends it as stored.
	requests := map[string][]*v1pb.WorkspaceApprovalSetting_Rule{}
	for key, rules := range writesOf(stored[1:], added, changes) {
		requests[key] = append([]*v1pb.WorkspaceApprovalSetting_Rule{fallbackRule()}, rules...)
	}
	forms := formsOf(t, requests, func(requested []*v1pb.WorkspaceApprovalSetting_Rule) (string, bool) {
		diff, err := diffApprovalRules(stored, requested, "LINK")
		require.NoError(t, err)
		form, err := approvalRulesForm(alice, diff, "", "LINK")
		if err != nil {
			return "", false
		}
		return form.message + "\n--\n" + form.description, true
	})
	// 12 removals, 121 moves, 28 new titles, 24 new descriptions, 60 new flows
	// and 72 added rules.
	require.Len(t, requests, 317)
	require.Greater(t, len(forms), 250, "most of them get a form")
}

// A predefined role is named by its display name and any other role as
// written, so two roles never read the same: no display name is taken twice,
// and none reads as a role's name.
func TestRoleTitlesTellRolesApart(t *testing.T) {
	titles := map[string]string{}
	for _, role := range store.PredefinedRoles {
		name := common.FormatRole(role.ResourceID)
		title := roleTitle(name)
		require.Equal(t, role.Name, title)
		require.False(t, strings.HasPrefix(title, common.RolePrefix), title)
		require.Empty(t, titles[title], "%s and %s", titles[title], name)
		titles[title] = name
	}
	require.Equal(t, "roles/custom-reviewer", roleTitle("roles/custom-reviewer"))
	require.Equal(t, "roles/WorkspaceDBA", roleTitle("roles/WorkspaceDBA"), "a role's ID is matched as written")
}

// The instance form states every value its request sets, with the exceptions
// adminForm's comment names: each of those requests reads as journey 1's.
func TestInstanceFormStatesEveryValueTheRequestSets(t *testing.T) {
	ctx := withUserEmail(context.Background(), "alice@example.com")
	form := func(edit func(request, instance, dataSource map[string]any)) string {
		c, err := parseInstanceCreation(instanceBody(t, edit), "ws-test")
		require.NoError(t, err)
		require.NoError(t, c.check("LINK"))
		form, err := instanceForm(ctx, c, true, "LINK")
		require.NoError(t, err)
		return form.message + "\n--\n" + form.description
	}
	base := form(nil)
	seen := map[string]string{base: "journey 1's request"}
	for name, edit := range map[string]func(request, instance, dataSource map[string]any){
		"another instance ID": func(request, _, _ map[string]any) { request["instanceId"] = "hr-staging-2" },
		"another project":     func(request, _, _ map[string]any) { request["parent"] = "projects/finance" },
		"another title":       func(_, instance, _ map[string]any) { instance["title"] = "HR Staging 2" },
		"another environment": func(_, instance, _ map[string]any) { instance["environment"] = "environments/prod" },
		"no environment":      func(_, instance, _ map[string]any) { delete(instance, "environment") },
		"not activated":       func(_, instance, _ map[string]any) { instance["activation"] = false },
		"another host":        func(_, _, dataSource map[string]any) { dataSource["host"] = "db.internal" },
		"another port":        func(_, _, dataSource map[string]any) { dataSource["port"] = "5432" },
		"another user":        func(_, _, dataSource map[string]any) { dataSource["username"] = "bytebase" },
		"no password":         func(_, _, dataSource map[string]any) { delete(dataSource, "password") },
		"a database":          func(_, _, dataSource map[string]any) { dataSource["database"] = "hr" },
		"TLS required":        func(_, _, dataSource map[string]any) { dataSource["useSsl"] = true },
		"TLS required and verified": func(_, _, dataSource map[string]any) {
			dataSource["useSsl"], dataSource["verifyTlsCertificate"] = true, true
		},
	} {
		text := form(edit)
		require.Empty(t, seen[text], "%s reads as %s", name, seen[text])
		seen[text] = name
	}
	for name, edit := range map[string]func(request, instance, dataSource map[string]any){
		"another password":                  func(_, _, dataSource map[string]any) { dataSource["password"] = "another" },
		"another data source ID":            func(_, _, dataSource map[string]any) { dataSource["id"] = "0a1b2c3d-4e5f-4a6b-8c7d-9e0f1a2b3c4d" },
		"the sign-in type it has when none": func(_, _, dataSource map[string]any) { dataSource["authenticationType"] = "PASSWORD" },
		"verification with no TLS required": func(_, _, dataSource map[string]any) { dataSource["verifyTlsCertificate"] = true },
	} {
		require.Equal(t, base, form(edit), name)
	}
}

// The identity form states every value its request sets.
func TestIdentityFormStatesEveryValueTheRequestSets(t *testing.T) {
	ctx := withOAuth2ClientID(withUserEmail(withWorkspaceID(context.Background(), "ws-test"), "alice@example.com"), "client-A")
	form := func(edit func(request, config map[string]any)) string {
		c, err := parseIdentityCreation(identityBody(t, edit))
		require.NoError(t, err)
		require.NoError(t, c.check("LINK"))
		c.fill()
		form, err := identityForm(ctx, c, "LINK")
		require.NoError(t, err)
		return form.message + "\n--\n" + form.description
	}
	seen := map[string]string{form(nil): "journey 4's request"}
	for name, edit := range map[string]func(request, config map[string]any){
		"another identity ID": func(request, _ map[string]any) { request["workloadIdentityId"] = "hr-migrations-ci-2" },
		"another project":     func(request, _ map[string]any) { request["parent"] = "projects/finance" },
		"another title": func(request, _ map[string]any) {
			identity, ok := request["workloadIdentity"].(map[string]any)
			require.True(t, ok)
			identity["title"] = "HR migrations CI 2"
		},
		"another repository": func(_, config map[string]any) { config["subjectPattern"] = "repo:example-org/hr-schema:*" },
		"another owner":      func(_, config map[string]any) { config["subjectPattern"] = "repo:example-inc/hr-migrations:*" },
		"GitLab": func(_, config map[string]any) {
			config["providerType"], config["subjectPattern"] = "GITLAB", "project_path:example-org/hr-migrations:*"
		},
	} {
		text := form(edit)
		require.Empty(t, seen[text], "%s reads as %s", name, seen[text])
		seen[text] = name
	}
}
