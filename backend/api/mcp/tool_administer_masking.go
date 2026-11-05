package mcp

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strings"

	"connectrpc.com/connect"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/masker"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
)

const (
	// maskingRulesApply is the fixed text of the masking form. It is one line,
	// so it goes in the message whenever a line is left.
	maskingRulesApply = "First matching rule decides; unmatched columns keep their own type."

	// maskingRulesUnmask is the change's effect on the user: masking applies to
	// their own queries too.
	maskingRulesUnmask = "This can unmask data, for you too."
)

var maskingRules = adminRules{name: "the masking rules", read: "call_api OrgPolicyService/GetPolicy"}

// maskingRulesPath is where the rules sit in an UpdatePolicyRequest, as
// maskingRequestFields names them.
const maskingRulesPath = "policy.masking_rule_policy.rules"

// maskingRequestFields are the UpdatePolicyRequest fields the masking entry
// takes: what the console's rule editor writes. A request that sets any other
// field is refused before the form, since the form could not state it.
var maskingRequestFields = map[string]bool{
	"policy":                                   true,
	"policy.name":                              true,
	"policy.type":                              true,
	"policy.masking_rule_policy":               true,
	maskingRulesPath:                           true,
	maskingRulesPath + ".id":                   true,
	maskingRulesPath + ".condition":            true,
	maskingRulesPath + ".condition.expression": true,
	maskingRulesPath + ".condition.title":      true,
	maskingRulesPath + ".semantic_type":        true,
	"update_mask":                              true,
	"update_mask.paths":                        true,
	"allow_missing":                            true,
}

// maskingRulesChange is a request to replace the workspace's masking rules.
type maskingRulesChange struct {
	request *v1pb.UpdatePolicyRequest
}

func (*maskingRulesChange) confirmedProcedure() string {
	return v1connect.OrgPolicyServiceUpdatePolicyProcedure
}

// confirmedResource is the policy name as the request writes it, since the
// gate matches the confirmation against the request.
func (c *maskingRulesChange) confirmedResource() string { return c.request.GetPolicy().GetName() }

func (s *Server) administerMaskingRules(ctx context.Context, req *mcp.CallToolRequest, input AdministerWorkspaceInput) (*mcp.CallToolResult, any, error) {
	link := s.consoleLink(ctx, "global-masking")
	change, err := parseMaskingRulesChange(input.Body, getWorkspaceID(ctx), link)
	if err != nil {
		return formatToolError(err), nil, nil
	}
	if !canShowConfirmation(req) {
		return formatToolError(&toolError{
			Code:       "CONFIRMATION_UNAVAILABLE",
			Message:    "this MCP client cannot show the confirmation form administer_workspace needs, so the masking rules were not changed",
			Suggestion: "change them signed in to the Bytebase console: " + link,
		}), nil, nil
	}

	// Read on both rounds: the form describes the rules and types as stored now,
	// and an answer is acted on only while they read as they did when the user
	// was asked.
	stored, found, err := s.readMaskingRules(ctx, change.confirmedResource())
	if err != nil {
		return formatToolError(err), nil, nil
	}
	if !found && !change.request.GetAllowMissing() {
		return formatToolError(&toolError{
			Code:       "INVALID_REQUEST",
			Message:    "this workspace has no masking rules stored yet, so the policy is created and the request needs allowMissing set",
			Suggestion: "send the same request with allowMissing: true",
		}), nil, nil
	}
	types, err := s.readSemanticTypes(ctx)
	if err != nil {
		return formatToolError(err), nil, nil
	}
	diff, err := diffMaskingRules(stored, change.request.GetPolicy().GetMaskingRulePolicy().GetRules(), types, link)
	if err != nil {
		return formatToolError(err), nil, nil
	}
	if len(diff.changes) == 0 {
		text := "Nothing to change: the request holds the masking rules as they are stored.\nMasking rules: " + link
		return textResult(text), &AdministerWorkspaceOutput{OperationID: input.OperationID, Message: text, Link: link}, nil
	}
	form, err := maskingRulesForm(ctx, diff, link)
	if err != nil {
		return formatToolError(err), nil, nil
	}
	digest := maskingRulesDigest(form, stored, change.request)
	if result := s.confirmAdminChange(ctx, req, change, digest, form, maskingRules, "the masking rules, the semantic types or this request"); result != nil {
		return result, nil, nil
	}

	confirmedCtx := auth.WithMCPConfirmedDecision(ctx, auth.MCPConfirmedDecision{
		Procedure: change.confirmedProcedure(),
		Resource:  change.confirmedResource(),
	})
	body, err := protojson.Marshal(change.request)
	if err != nil {
		return formatToolError(errors.Wrap(err, "failed to build the update")), nil, nil
	}
	resp, err := s.apiRequest(confirmedCtx, change.confirmedProcedure(), json.RawMessage(body))
	if err != nil {
		return formatToolError(errors.Wrap(err, "failed to update the masking rules")), nil, nil
	}
	if resp.Status >= http.StatusBadRequest {
		return formatToolError(errors.Errorf("failed to update the masking rules: HTTP %d: %s", resp.Status, parseError(resp.Body))), nil, nil
	}
	text := fmt.Sprintf("Applied: the masking rules now hold %s. %s\nMasking rules: %s", countRules(len(diff.requested)), maskingRulesApply, link)
	return textResult(text), &AdministerWorkspaceOutput{OperationID: input.OperationID, Changed: true, Message: text, Link: link}, nil
}

func parseMaskingRulesChange(body map[string]any, workspace, link string) (*maskingRulesChange, error) {
	// The masking rules change through one update mask path, which the tool
	// sets. Protojson reads a field mask only in its JSON form, so the path as
	// the API names it is taken too.
	body = maps.Clone(body)
	for _, key := range []string{"updateMask", "update_mask"} {
		if body[key] == "masking_rule_policy" {
			body[key] = "maskingRulePolicy"
		}
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return nil, &toolError{Code: "INVALID_REQUEST", Message: "the request body could not be read: " + err.Error()}
	}
	request := &v1pb.UpdatePolicyRequest{}
	if err := common.ProtojsonUnmarshaler.Unmarshal(encoded, request); err != nil {
		return nil, &toolError{Code: "INVALID_REQUEST", Message: "the request body is not an UpdatePolicyRequest: " + err.Error()}
	}
	// The handler reads the policy type in any letter case.
	want := common.WorkspacePrefix + workspace + "/" + common.PolicyNamePrefix
	name := request.GetPolicy().GetName()
	if kind, ok := strings.CutPrefix(name, want); !ok || !strings.EqualFold(kind, "masking_rule") || request.GetPolicy().GetType() != v1pb.PolicyType_MASKING_RULE {
		return nil, &toolError{
			Code:       "UNSUPPORTED_CHANGE",
			Message:    fmt.Sprintf("administer_workspace does not change the policy %q of type %s", name, request.GetPolicy().GetType()),
			Suggestion: "it changes this workspace's masking rules, " + want + "masking_rule with type MASKING_RULE; change other policies signed in to the Bytebase console",
		}
	}
	if request.GetPolicy().GetMaskingRulePolicy() == nil {
		return nil, &toolError{
			Code:       "INVALID_REQUEST",
			Message:    "the request has no policy.maskingRulePolicy, so it holds no rule list",
			Suggestion: "send the whole rule list in policy.maskingRulePolicy.rules: the rules GetPolicy returns, with the user's change made",
		}
	}
	if paths := request.GetUpdateMask().GetPaths(); len(paths) > 0 && !slices.Equal(paths, []string{"masking_rule_policy"}) {
		return nil, &toolError{
			Code:       "INVALID_REQUEST",
			Message:    fmt.Sprintf("the request's updateMask is %q, and the masking rules change through masking_rule_policy alone", paths),
			Suggestion: "leave updateMask out: the tool sets it",
		}
	}
	request.UpdateMask = &fieldmaskpb.FieldMask{Paths: []string{"masking_rule_policy"}}
	if field := unlistedField(request.ProtoReflect(), "", maskingRequestFields); field != "" {
		return nil, &toolError{
			Code:       "FIELD_NOT_SUPPORTED",
			Message:    fmt.Sprintf("the request sets %s, which administer_workspace does not change", field),
			Suggestion: "send policy.name, policy.type, updateMask, allowMissing and each rule's id, condition.expression, condition.title and semanticType only, or change the rules signed in to the Bytebase console: " + link,
		}
	}
	if err := checkMaskingRules(request.GetPolicy().GetMaskingRulePolicy().GetRules()); err != nil {
		return nil, err
	}
	return &maskingRulesChange{request: request}, nil
}

// checkMaskingRules refuses what the handler would refuse or fail on, so the
// user is never asked about a write that cannot happen.
func checkMaskingRules(rules []*v1pb.MaskingRulePolicy_MaskingRule) error {
	ids := make(map[string]bool, len(rules))
	for i, rule := range rules {
		switch {
		case rule.GetId() == "":
			return &toolError{
				Code:       "INVALID_REQUEST",
				Message:    fmt.Sprintf("rule %d has no id", i+1),
				Suggestion: "keep each stored rule's id, and give each new rule a new UUID",
			}
		case ids[rule.GetId()]:
			return &toolError{
				Code:       "INVALID_REQUEST",
				Message:    fmt.Sprintf("two rules have the id %q", rule.GetId()),
				Suggestion: "keep each stored rule's id, and give each new rule a new UUID",
			}
		case rule.GetCondition() == nil:
			return &toolError{
				Code:       "INVALID_REQUEST",
				Message:    fmt.Sprintf("rule %d has no condition", i+1),
				Suggestion: "give each rule a condition.expression, the CEL the console's rule editor writes",
			}
		}
		ids[rule.GetId()] = true
		if _, err := common.ValidateMaskingRuleCELExpr(rule.GetCondition().GetExpression()); err != nil {
			return &toolError{
				Code:       "INVALID_REQUEST",
				Message:    fmt.Sprintf("rule %d's condition %q is not a masking rule condition: %v", i+1, rule.GetCondition().GetExpression(), err),
				Suggestion: "use the variables the masking rule condition takes, such as resource.environment_id, resource.database_name, resource.table_name and resource.column_name",
			}
		}
	}
	return nil
}

// readMaskingRules reads the stored masking rules with the user's permissions.
// found is false for a workspace that never saved one.
func (s *Server) readMaskingRules(ctx context.Context, name string) ([]*v1pb.MaskingRulePolicy_MaskingRule, bool, error) {
	resp, err := s.apiRequest(ctx, v1connect.OrgPolicyServiceGetPolicyProcedure, map[string]any{"name": name})
	if err != nil {
		return nil, false, errors.Wrap(err, "failed to read the masking rules")
	}
	if resp.Status == http.StatusNotFound || connectCodeOf(resp.Body) == connect.CodeNotFound {
		return nil, false, nil
	}
	if err := checkAPIResponse(resp, "read the masking rules", "bb.policies.getMaskingRulePolicy"); err != nil {
		return nil, false, err
	}
	policy := &v1pb.Policy{}
	if err := common.ProtojsonUnmarshaler.Unmarshal(resp.Body, policy); err != nil {
		return nil, false, errors.Wrap(err, "failed to parse the masking rules")
	}
	return policy.GetMaskingRulePolicy().GetRules(), true, nil
}

// readSemanticTypes reads the workspace's semantic types with the user's
// permissions. A workspace that stores none has only the built-in types.
func (s *Server) readSemanticTypes(ctx context.Context) (map[string]*storepb.SemanticTypeSetting_SemanticType, error) {
	resp, err := s.apiRequest(ctx, v1connect.SettingServiceGetSettingProcedure, map[string]any{"name": common.SettingNamePrefix + v1pb.Setting_SEMANTIC_TYPES.String()})
	if err != nil {
		return nil, errors.Wrap(err, "failed to read the semantic types")
	}
	if resp.Status == http.StatusNotFound || connectCodeOf(resp.Body) == connect.CodeNotFound {
		return semanticTypesFromSetting(nil)
	}
	if err := checkAPIResponse(resp, "read the semantic types", "bb.settings.get"); err != nil {
		return nil, err
	}
	setting := &v1pb.Setting{}
	if err := common.ProtojsonUnmarshaler.Unmarshal(resp.Body, setting); err != nil {
		return nil, errors.Wrap(err, "failed to parse the semantic types")
	}
	return semanticTypesFromSetting(setting.GetValue().GetSemanticType().GetTypes())
}

// semanticTypesFromSetting maps the semantic types masking reads by their ID:
// the stored types and the built-in ones.
func semanticTypesFromSetting(stored []*v1pb.SemanticTypeSetting_SemanticType) (map[string]*storepb.SemanticTypeSetting_SemanticType, error) {
	converted := make([]*storepb.SemanticTypeSetting_SemanticType, 0, len(stored))
	for _, semanticType := range stored {
		c, err := storeSemanticType(semanticType)
		if err != nil {
			return nil, err
		}
		converted = append(converted, c)
	}
	types := make(map[string]*storepb.SemanticTypeSetting_SemanticType)
	for _, semanticType := range masker.WithBuiltinSemanticTypes(converted) {
		types[semanticType.GetId()] = semanticType
	}
	return types, nil
}

// storeSemanticType converts an API semantic type to the store's message,
// which has the same fields, so the masker builds it as masking does.
func storeSemanticType(semanticType *v1pb.SemanticTypeSetting_SemanticType) (*storepb.SemanticTypeSetting_SemanticType, error) {
	encoded, err := protojson.Marshal(semanticType)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read a semantic type")
	}
	stored := &storepb.SemanticTypeSetting_SemanticType{}
	if err := common.ProtojsonUnmarshaler.Unmarshal(encoded, stored); err != nil {
		return nil, errors.Wrap(err, "failed to read a semantic type")
	}
	return stored, nil
}

// maskingRulesDigest binds an answer to the form the user read, the rules
// stored when they read it, and the request. The form covers the semantic
// types, since it states what each rule's type does.
func maskingRulesDigest(form *adminForm, stored []*v1pb.MaskingRulePolicy_MaskingRule, request *v1pb.UpdatePolicyRequest) []byte {
	h := sha256.New()
	_, _ = h.Write([]byte(form.message))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(form.description))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write(auth.MCPValueDigest(&v1pb.MaskingRulePolicy{Rules: stored}))
	_, _ = h.Write(auth.MCPValueDigest(request))
	return h.Sum(nil)
}

// maskingRulesDiff is how a request changes the masking rules.
type maskingRulesDiff struct {
	// changes are the form's change lines: the removed rules, then the
	// requested rules that differ, in the order masking tries them.
	changes []string
	// requested are the rules as the request orders them.
	requested []*v1pb.MaskingRulePolicy_MaskingRule
	// unmasks reports a change that can show data unmasked: a rule removed,
	// moved or given another type, or an added rule that masks nothing.
	unmasks bool
	// types describe each rule's semantic type.
	types map[string]*storepb.SemanticTypeSetting_SemanticType
}

// diffMaskingRules compares the requested rule list with the stored one. Rules
// are matched by their condition, the text masking evaluates, as the approval
// rules are; a kept rule must keep its stored ID, so the form states every
// change the write makes.
func diffMaskingRules(stored, requested []*v1pb.MaskingRulePolicy_MaskingRule, types map[string]*storepb.SemanticTypeSetting_SemanticType, link string) (*maskingRulesDiff, error) {
	// The request cannot carry such a field, so the write would drop it unstated.
	for _, rule := range stored {
		if field := unlistedField(rule.ProtoReflect(), maskingRulesPath, maskingRequestFields); field != "" {
			return nil, &toolError{
				Code:       "FIELD_NOT_SUPPORTED",
				Message:    fmt.Sprintf("a stored masking rule sets %s, which the confirmation form cannot show and the write would drop, so the masking rules were not changed", field),
				Suggestion: "change the rules signed in to the Bytebase console: " + link,
			}
		}
	}
	storedByCondition, duplicate := maskingRulesByCondition(stored)
	if duplicate != "" {
		return nil, &toolError{
			Code:       "DUPLICATE_CONDITION",
			Message:    fmt.Sprintf("two stored masking rules have the condition %q, so the form could not tell which one changes", duplicate),
			Suggestion: "change the rules signed in to the Bytebase console: " + link,
		}
	}
	requestedByCondition, duplicate := maskingRulesByCondition(requested)
	if duplicate != "" {
		return nil, &toolError{
			Code:       "DUPLICATE_CONDITION",
			Message:    fmt.Sprintf("two requested masking rules have the condition %q, and only the first could ever match", duplicate),
			Suggestion: "keep one rule per condition",
		}
	}
	for _, rule := range requested {
		if before := storedByCondition[rule.GetCondition().GetExpression()]; before != nil && before.GetId() != rule.GetId() {
			return nil, &toolError{
				Code:       "INVALID_REQUEST",
				Message:    fmt.Sprintf("the rule for %q has the id %q, and the stored rule for that condition has the id %q", rule.GetCondition().GetExpression(), rule.GetId(), before.GetId()),
				Suggestion: "keep each stored rule's id, and give each new rule a new UUID",
			}
		}
		if _, err := semanticTypeMasker(rule.GetSemanticType(), types); err != nil {
			return nil, err
		}
	}

	diff := &maskingRulesDiff{requested: requested, types: types}
	var storedKept, requestedKept []string
	for _, rule := range stored {
		condition := rule.GetCondition().GetExpression()
		if _, kept := requestedByCondition[condition]; kept {
			storedKept = append(storedKept, condition)
			continue
		}
		diff.unmasks = true
		diff.changes = append(diff.changes, fmt.Sprintf("Remove, %s: %s", diff.typePhrase(rule.GetSemanticType()), maskingConditionText(condition)))
	}
	for _, rule := range requested {
		if condition := rule.GetCondition().GetExpression(); storedByCondition[condition] != nil {
			requestedKept = append(requestedKept, condition)
		}
	}
	// A kept rule outside the longest common order moved: the first matching
	// rule decides, so a move changes which rule masks some columns.
	inOrder := longestCommonOrder(storedKept, requestedKept)
	kept := 0
	for i, rule := range requested {
		condition := rule.GetCondition().GetExpression()
		before := storedByCondition[condition]
		if before == nil {
			if !diff.masks(rule.GetSemanticType()) {
				diff.unmasks = true
			}
			// The first matching rule decides, so an added rule's place is part
			// of the change.
			diff.changes = append(diff.changes, fmt.Sprintf("Add rule %d, %s: %s", i+1, diff.typePhrase(rule.GetSemanticType()), maskingConditionText(condition)))
			continue
		}
		moved := !inOrder[kept]
		kept++
		if clauses := diff.keptRuleClauses(before, rule, moved, i+1); len(clauses) > 0 {
			line := strings.Join(clauses, "; ")
			diff.changes = append(diff.changes, fmt.Sprintf("%s%s: %s", strings.ToUpper(line[:1]), line[1:], maskingConditionText(condition)))
		}
	}
	return diff, nil
}

// keptRuleClauses states how a kept rule changes, one clause per field the
// form shows: the type, the place and the title.
func (d *maskingRulesDiff) keptRuleClauses(before, rule *v1pb.MaskingRulePolicy_MaskingRule, moved bool, position int) []string {
	var clauses []string
	if rule.GetSemanticType() != before.GetSemanticType() {
		d.unmasks = true
		clauses = append(clauses, fmt.Sprintf("change type, %s to %s", d.typePhrase(before.GetSemanticType()), d.typePhrase(rule.GetSemanticType())))
	}
	if moved {
		d.unmasks = true
		clauses = append(clauses, fmt.Sprintf("move to rule %d", position))
	}
	if title := rule.GetCondition().GetTitle(); title != before.GetCondition().GetTitle() {
		clauses = append(clauses, "rename to "+maskingRuleTitle(rule))
	}
	return clauses
}

// typePhrase names a rule's semantic type and what masking does with it. The
// first matching rule decides, so a rule whose type masks nothing shows the
// columns it matches unmasked, and a rule with no type leaves each column to its
// own type and stops every later rule.
func (d *maskingRulesDiff) typePhrase(id string) string {
	if id == "" {
		return "no type (each column's own type)"
	}
	semanticType, ok := d.types[id]
	if !ok {
		return fmt.Sprintf("%q (no such type, masks nothing)", id)
	}
	title := semanticType.GetTitle()
	if title == "" {
		title = id
	}
	if !d.masks(id) {
		return fmt.Sprintf("%q (masks nothing)", title)
	}
	return fmt.Sprintf("%q (%s)", title, masker.AlgorithmName(semanticType.GetAlgorithm()))
}

// masks reports whether the semantic type id masks the columns a rule matches.
// A rule with no type leaves each column to its own type, so it counts as
// masking nothing.
func (d *maskingRulesDiff) masks(id string) bool {
	m, err := semanticTypeMasker(id, d.types)
	return err == nil && m != nil && masker.MasksValues(m)
}

// semanticTypeMasker returns the masker the semantic type id applies, or nil
// for an empty or unknown ID. A type whose algorithm masking cannot build is
// refused, since masking fails on it.
func semanticTypeMasker(id string, types map[string]*storepb.SemanticTypeSetting_SemanticType) (masker.Masker, error) {
	semanticType, ok := types[id]
	if id == "" || !ok {
		return nil, nil
	}
	m, err := masker.ForSemanticType(semanticType)
	if err != nil {
		return nil, &toolError{
			Code:       "INVALID_SEMANTIC_TYPE",
			Message:    fmt.Sprintf("the semantic type %q has an algorithm masking cannot use: %v", id, err),
			Suggestion: "fix the semantic type signed in to the Bytebase console",
		}
	}
	return m, nil
}

// maskingRulesByCondition maps rules by their condition. It returns the first
// condition two rules share, if any.
func maskingRulesByCondition(rules []*v1pb.MaskingRulePolicy_MaskingRule) (map[string]*v1pb.MaskingRulePolicy_MaskingRule, string) {
	byCondition := make(map[string]*v1pb.MaskingRulePolicy_MaskingRule, len(rules))
	for _, rule := range rules {
		condition := rule.GetCondition().GetExpression()
		if _, ok := byCondition[condition]; ok {
			return nil, condition
		}
		byCondition[condition] = rule
	}
	return byCondition, ""
}

// maskingConditionText shows a masking rule's condition. Masking evaluates an
// empty condition as true, so it matches every column.
func maskingConditionText(condition string) string {
	if condition == "" {
		return "every column"
	}
	return conditionText(condition)
}

func maskingRuleTitle(rule *v1pb.MaskingRulePolicy_MaskingRule) string {
	if title := rule.GetCondition().GetTitle(); title != "" {
		return fmt.Sprintf("%q", title)
	}
	return "untitled"
}

// maskingRulesForm builds the masking form in the lines Claude Code shows: the
// question, a line or more per rule change, and the change's effect on the
// user. A change that does not fit is refused, never cut. The fixed sentence
// goes in the message when a line is left, and in the description otherwise;
// the description also lists every rule.
func maskingRulesForm(ctx context.Context, diff *maskingRulesDiff, link string) (*adminForm, error) {
	question := "Change the masking rules"
	if email := getUserEmail(ctx); email != "" {
		question += ", as " + email
	}
	effect := ""
	if diff.unmasks {
		effect = maskingRulesUnmask
	}
	lines, tooLong := fitMessage(question+"?", diff.changes, effect)
	if len(lines) > messageLines {
		return nil, changeTooLong(len(lines), tooLong, maskingRules, link)
	}

	var sb strings.Builder
	if len(diff.requested) == 0 {
		sb.WriteString("No masking rule is left, so each column keeps its own semantic type.")
	} else {
		sb.WriteString("The masking rules, in the order masking tries them:")
		for i, rule := range diff.requested {
			fmt.Fprintf(&sb, "\n%d. %s, %s: %s", i+1, maskingRuleTitle(rule), diff.typePhrase(rule.GetSemanticType()), maskingConditionText(rule.GetCondition().GetExpression()))
		}
	}
	if len(lines) < messageLines {
		lines = append(lines, maskingRulesApply)
	} else {
		sb.WriteString("\n" + maskingRulesApply)
	}
	return &adminForm{message: strings.Join(lines, "\n"), description: sb.String()}, nil
}
