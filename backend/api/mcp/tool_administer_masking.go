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

	// maskingRulesEffect is the change's effect on the user: masking applies to
	// their own queries too.
	maskingRulesEffect             = "can unmask data, for you too"
	maskingRulesUnmask             = "This " + maskingRulesEffect + "."
	maskingRulesQuestion           = "Change the masking rules"
	maskingRulesQuestionWithEffect = maskingRulesQuestion + " (" + maskingRulesEffect + ")"
)

var maskingRules = adminRules{
	name:     "the masking rules",
	read:     "call_api OrgPolicyService/GetPolicy",
	question: maskingRulesQuestion,
	apply:    maskingRulesApply,
	replace:  "add a rule under a new UUID before removing the one it replaces (two rules cannot share an id)",
}

// maskingRulesPath is where the rules sit in an UpdatePolicyRequest, as
// maskingRequestFields names them.
const maskingRulesPath = "policy.masking_rule_policy.rules"

// maskingRequestFields are the UpdatePolicyRequest fields the masking entry
// takes: what the console's rule editor writes, and policy.enforce and
// policy.resource_type, which GetPolicy returns and this write ignores. A
// request that sets any other field is refused before the form, since the form
// could not state it.
var maskingRequestFields = map[string]bool{
	"policy":                                   true,
	"policy.name":                              true,
	"policy.type":                              true,
	"policy.enforce":                           true,
	"policy.resource_type":                     true,
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
	types, storedTypes, err := s.readSemanticTypes(ctx)
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
	digest := maskingFormDigest(form, change.request, stored, storedTypes)
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
	want := common.WorkspacePrefix + workspace + "/" + common.PolicyNamePrefix + "masking_rule"
	if name := request.GetPolicy().GetName(); name != want || request.GetPolicy().GetType() != v1pb.PolicyType_MASKING_RULE {
		return nil, &toolError{
			Code:       "UNSUPPORTED_CHANGE",
			Message:    fmt.Sprintf("administer_workspace does not change the policy %q of type %s", name, request.GetPolicy().GetType()),
			Suggestion: "it changes this workspace's masking rules, " + want + " with type MASKING_RULE; change other policies signed in to the Bytebase console",
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
// user is never asked about a write that cannot happen. It also refuses two
// rules with one id, which the handler stores but the console cannot edit: it
// finds a rule by its id.
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
	if resp.Status == http.StatusNotFound {
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
func (s *Server) readSemanticTypes(ctx context.Context) (map[string]*storepb.SemanticTypeSetting_SemanticType, []*v1pb.SemanticTypeSetting_SemanticType, error) {
	resp, err := s.apiRequest(ctx, v1connect.SettingServiceGetSettingProcedure, map[string]any{"name": common.SettingNamePrefix + v1pb.Setting_SEMANTIC_TYPES.String()})
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to read the semantic types")
	}
	if resp.Status == http.StatusNotFound {
		types, err := semanticTypesFromSetting(nil)
		return types, nil, err
	}
	if err := checkAPIResponse(resp, "read the semantic types", "bb.settings.get"); err != nil {
		return nil, nil, err
	}
	setting := &v1pb.Setting{}
	if err := common.ProtojsonUnmarshaler.Unmarshal(resp.Body, setting); err != nil {
		return nil, nil, errors.Wrap(err, "failed to parse the semantic types")
	}
	stored := setting.GetValue().GetSemanticType().GetTypes()
	types, err := semanticTypesFromSetting(stored)
	return types, stored, err
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

// maskingRulesDiff is how a request changes the masking rules.
type maskingRulesDiff struct {
	// changes are the form's change lines: the removed rules, then the
	// requested rules that differ, in the order masking tries them. A change
	// that can show data unmasked has the effect.
	changes []formChange
	// requested are the rules as the request orders them, and marks say of
	// each whether it is new, changed or kept as stored.
	requested []*v1pb.MaskingRulePolicy_MaskingRule
	marks     []string
	// types describe each rule's semantic type.
	types map[string]*storepb.SemanticTypeSetting_SemanticType
	// moves counts the kept rules the change lines call moved.
	moves int
}

// diffMaskingRules compares the requested rule list with the stored one. Rules
// are matched by their condition, the text masking evaluates, as the approval
// rules are; a kept rule must keep its stored ID, so the form states every
// change the write makes.
//
// Masking gives a column the type of the first rule that matches it, and a
// column no rule matches keeps its own type. So a removed or moved rule can
// hand columns to a rule that masks them less, and so can a rule given a type
// that is not a full mask; these changes say they can unmask data.
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
	storedByCondition, duplicate, shared := rulesByCondition(stored)
	if shared {
		return nil, &toolError{
			Code:       "DUPLICATE_CONDITION",
			Message:    fmt.Sprintf("two stored masking rules have the condition %q, so the form could not tell which one changes", duplicate),
			Suggestion: "change the rules signed in to the Bytebase console: " + link,
		}
	}
	requestedByCondition, duplicate, shared := rulesByCondition(requested)
	if shared {
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
	}

	diff := &maskingRulesDiff{requested: requested, types: types}
	var storedKept, requestedKept []string
	storedPlace := map[string]int{}
	for _, rule := range stored {
		condition := rule.GetCondition().GetExpression()
		if _, kept := requestedByCondition[condition]; kept {
			storedPlace[condition] = len(storedKept)
			storedKept = append(storedKept, condition)
			continue
		}
		diff.changes = append(diff.changes, formChange{
			text:   fmt.Sprintf("Remove, %s: %s", diff.typePhrase(rule.GetSemanticType()), maskingConditionText(condition)),
			effect: true,
		})
	}
	var kept []keptRule
	kinds := map[string]keptRule{}
	for i, rule := range requested {
		condition := rule.GetCondition().GetExpression()
		before := storedByCondition[condition]
		if before == nil {
			continue
		}
		requestedKept = append(requestedKept, condition)
		// A type's masker is built once for every rule that gives the type.
		entry, ok := kinds[rule.GetSemanticType()]
		if !ok {
			entry = keptRule{kind: len(kinds), weak: !diff.masksEveryValue(rule.GetSemanticType())}
			kinds[rule.GetSemanticType()] = entry
		}
		entry.was = storedPlace[condition]
		entry.name = func() string {
			if rule.GetCondition().GetTitle() == "" {
				return fmt.Sprintf("rule %d (%s)", i+1, diff.typePhrase(rule.GetSemanticType()))
			}
			return fmt.Sprintf("%s (%s)", maskingRuleTitle(rule), diff.typePhrase(rule.GetSemanticType()))
		}
		entry.line = func(moved bool, passed string) string { return diff.keptRuleLine(before, rule, moved, i+1, passed) }
		kept = append(kept, entry)
	}
	// A kept rule outside the common order moved. When two rules trade places
	// either one moved, so each rule that traded order is priced as moved and
	// the common order leaves the cheapest moves. Nothing comes before a
	// masking form's rule list, so its changes have every row.
	inPlace, moved, costs := keptRuleLines(kept, descriptionLines, typesPassed)
	inOrder := longestCommonOrder(storedKept, requestedKept, costs)
	next := 0
	for i, rule := range requested {
		condition := rule.GetCondition().GetExpression()
		before := storedByCondition[condition]
		if before == nil {
			// An added rule's place is part of its change.
			diff.changes = append(diff.changes, formChange{
				text:   fmt.Sprintf("Add rule %d, %s: %s", i+1, diff.typePhrase(rule.GetSemanticType()), maskingConditionText(condition)),
				effect: !diff.masksEveryValue(rule.GetSemanticType()),
			})
			diff.marks = append(diff.marks, "new")
			continue
		}
		change := formChange{text: inPlace[next], effect: rule.GetSemanticType() != before.GetSemanticType() && !diff.masksEveryValue(rule.GetSemanticType())}
		if !inOrder[next] {
			change = formChange{text: moved[next], effect: true}
			diff.moves++
		}
		next++
		if change.text == "" {
			diff.marks = append(diff.marks, "kept")
			continue
		}
		diff.changes = append(diff.changes, change)
		diff.marks = append(diff.marks, "changed")
	}
	return diff, nil
}

// typesPassed counts the rules a masking rule's move passes. Only a full mask
// hides every value, so a rule with none is counted apart.
var typesPassed = passedWords{same: "with the same type", weak: "with no full mask", other: "with other types"}

// keptRuleLine states how a kept rule changes, one clause per field the form
// shows: the type, the place and the title. It is empty for a rule that does
// not change. A move gives the rule's type, unless the line states a change of
// it, and passed, what rulesPassed says of the rules it traded order with: the
// first matching rule decides a column's mask, and the rule list under the
// choice can be cut.
func (d *maskingRulesDiff) keptRuleLine(before, rule *v1pb.MaskingRulePolicy_MaskingRule, moved bool, position int, passed string) string {
	var clauses []string
	typeChanged := rule.GetSemanticType() != before.GetSemanticType()
	if typeChanged {
		clauses = append(clauses, fmt.Sprintf("change type, %s to %s", d.typePhrase(before.GetSemanticType()), d.typePhrase(rule.GetSemanticType())))
	}
	if moved {
		clause := fmt.Sprintf("move to rule %d", position)
		if !typeChanged {
			clause += ", " + d.typePhrase(rule.GetSemanticType())
		}
		clauses = append(clauses, clause+passed)
	}
	if title := rule.GetCondition().GetTitle(); title != before.GetCondition().GetTitle() {
		clauses = append(clauses, "rename to "+maskingRuleTitle(rule))
	}
	if len(clauses) == 0 {
		return ""
	}
	line := strings.Join(clauses, "; ")
	return fmt.Sprintf("%s%s: %s", strings.ToUpper(line[:1]), line[1:], maskingConditionText(rule.GetCondition().GetExpression()))
}

// typePhrase names a rule's semantic type and what masking does with it. A
// rule with no type still matches, and leaves each column it matches to the
// column's own type.
func (d *maskingRulesDiff) typePhrase(id string) string {
	if id == "" {
		return "no type (each column's own type)"
	}
	semanticType, ok := d.types[id]
	if !ok {
		return fmt.Sprintf("%q (no such type, masks nothing)", id)
	}
	// A stored type cannot take a built-in ID (masker.WithBuiltinSemanticTypes
	// drops it), so the algorithm alone names a built-in type.
	if id == masker.DefaultSemanticTypeID || id == masker.DefaultPartialSemanticTypeID {
		return masker.AlgorithmName(semanticType.GetAlgorithm())
	}
	if m := d.typeMasker(id); m == nil || !masker.MasksValues(m) {
		return fmt.Sprintf("%q (masks nothing)", semanticType.GetTitle())
	}
	return fmt.Sprintf("%q (%s)", semanticType.GetTitle(), masker.AlgorithmName(semanticType.GetAlgorithm()))
}

// masksEveryValue reports whether the semantic type id hides every value of
// the columns a rule gives it.
func (d *maskingRulesDiff) masksEveryValue(id string) bool {
	return masker.MasksEveryValue(d.typeMasker(id))
}

// typeMasker returns the masker the semantic type id applies, or nil for no
// type, an unknown one, or one whose algorithm masking cannot build.
func (d *maskingRulesDiff) typeMasker(id string) masker.Masker {
	semanticType, ok := d.types[id]
	if id == "" || !ok {
		return nil
	}
	m, err := masker.ForSemanticType(semanticType)
	if err != nil {
		return nil
	}
	return m
}

func maskingRuleTitle(rule *v1pb.MaskingRulePolicy_MaskingRule) string {
	if title := rule.GetCondition().GetTitle(); title != "" {
		return fmt.Sprintf("%q", title)
	}
	return "untitled"
}

// maskingRulesForm builds the masking form. When a change can show data
// unmasked, the form says so on a line of its own, or in the question when
// that line does not fit.
func maskingRulesForm(ctx context.Context, diff *maskingRulesDiff, link string) (*adminForm, error) {
	rules := maskingRules
	if len(diff.requested) < 2 {
		rules.apply = ""
	}
	if form := rules.whole(ctx, maskingRulesQuestion, diff.changes, maskingRulesUnmask, maskingRuleSection(diff, false)); form != nil {
		return form, nil
	}
	// A removal shows the effect and adding a full-mask rule does not, so
	// without this the message could state a rule's add whole and only count
	// its removal. The question may pass messageLineWidth: Claude Code cuts a
	// long row, which costs a narrow terminal only the account at its end.
	if form := rules.whole(ctx, maskingRulesQuestionWithEffect, diff.changes, "", maskingRuleSection(diff, false)); form != nil {
		return form, nil
	}
	alone := func(change formChange) bool {
		return messageHolds(ctx, maskingRulesQuestion, change, maskingRulesUnmask) || messageHolds(ctx, maskingRulesQuestionWithEffect, change, "")
	}
	// A step is a request of its own, and its rules decide which of two rules
	// that trade places it calls moved. So a change with a move is offered no
	// steps, as on the approval form.
	eachAlone := diff.moves == 0 && !slices.ContainsFunc(diff.changes, func(change formChange) bool { return !alone(change) })
	return rules.listed(ctx, diff.changes, countLine(len(diff.changes), diff.marks, 0), maskingRulesUnmask, maskingRuleSection(diff, true), link, eachAlone)
}

// maskingRuleSection is what the description says of the rules a change leaves.
// Every line names its rule's type: the order decides which rule masks a
// column. A kept rule, which no change line shows, gives its scope. A new or changed one gives the CEL it stores,
// unless the description lists the changes itself, right above it.
func maskingRuleSection(diff *maskingRulesDiff, listed bool) ruleSection {
	if len(diff.requested) == 0 {
		return ruleSection{empty: "No masking rule is left, so each column keeps its own semantic type."}
	}
	return ruleSection{name: "Masking rules", header: "The masking rules, in the order masking tries them:", total: len(diff.requested), rule: func(i int) string {
		rule := diff.requested[i]
		line := fmt.Sprintf("%d. %s (%s), %s", i+1, maskingRuleTitle(rule), diff.marks[i], diff.typePhrase(rule.GetSemanticType()))
		switch {
		case diff.marks[i] == "kept":
			line += ": " + maskingConditionText(rule.GetCondition().GetExpression())
		case !listed:
			line += ": " + conditionText(rule.GetCondition().GetExpression())
		default:
		}
		return line
	}}
}

// maskingFormDigest binds an answer to the form, the request, and the stored
// rules and semantic types the form was built from. A kept rule can move to
// another type that reads the same, and a type's settings show only as its
// algorithm's name, so the form alone does not cover them.
func maskingFormDigest(form *adminForm, request *v1pb.UpdatePolicyRequest, stored []*v1pb.MaskingRulePolicy_MaskingRule, types []*v1pb.SemanticTypeSetting_SemanticType) []byte {
	h := sha256.New()
	_, _ = h.Write(adminFormDigest(form, request))
	_, _ = h.Write(auth.MCPValueDigest(&v1pb.MaskingRulePolicy{Rules: stored}))
	_, _ = h.Write(auth.MCPValueDigest(&v1pb.SemanticTypeSetting{Types: types}))
	return h.Sum(nil)
}
