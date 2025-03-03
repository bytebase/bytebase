// Package utils is a utility library for server.
package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/pkg/errors"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/simplifiedchinese"
	textunicode "golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
	"google.golang.org/genproto/googleapis/type/expr"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// DataSourceFromInstanceWithType gets a typed data source from an instance.
func DataSourceFromInstanceWithType(instance *store.InstanceMessage, dataSourceType api.DataSourceType) *store.DataSourceMessage {
	for _, dataSource := range instance.DataSources {
		if dataSource.Type == dataSourceType {
			return dataSource
		}
	}
	return nil
}

// isMatchExpression checks whether a databases matches the query.
// labels is a mapping from database label key to value.
func isMatchExpression(labels map[string]string, expression *storepb.LabelSelectorRequirement) bool {
	switch expression.Operator {
	case storepb.LabelSelectorRequirement_IN:
		return checkLabelIn(labels, expression)
	case storepb.LabelSelectorRequirement_NOT_IN:
		return !checkLabelIn(labels, expression)
	case storepb.LabelSelectorRequirement_EXISTS:
		_, ok := labels[expression.Key]
		return ok
	default:
		return false
	}
}

func checkLabelIn(labels map[string]string, expression *storepb.LabelSelectorRequirement) bool {
	value, ok := labels[expression.Key]
	if !ok {
		return false
	}

	for _, exprValue := range expression.Values {
		if exprValue == value {
			return true
		}
	}
	return false
}

func isMatchExpressions(labels map[string]string, expressionList []*storepb.LabelSelectorRequirement) bool {
	// Empty expression list matches no databases.
	if len(expressionList) == 0 {
		return false
	}
	// Expressions are ANDed.
	for _, expression := range expressionList {
		if !isMatchExpression(labels, expression) {
			return false
		}
	}
	return true
}

// ValidateAndGetDeploymentSchedule validates and returns the deployment schedule.
// Note: this validation only checks whether the payloads is a valid json, however, invalid field name errors are ignored.
func ValidateDeploymentSchedule(schedule *storepb.Schedule) error {
	for _, d := range schedule.Deployments {
		if d.Title == "" {
			return common.Errorf(common.Invalid, "Deployment title must not be empty")
		}
		hasEnv := false
		for _, e := range d.Spec.Selector.MatchExpressions {
			switch e.Operator {
			case storepb.LabelSelectorRequirement_IN, storepb.LabelSelectorRequirement_NOT_IN:
				if len(e.Values) == 0 {
					return common.Errorf(common.Invalid, "expression key %q with %q operator should have at least one value", e.Key, e.Operator)
				}
			case storepb.LabelSelectorRequirement_EXISTS:
				if len(e.Values) > 0 {
					return common.Errorf(common.Invalid, "expression key %q with %q operator shouldn't have values", e.Key, e.Operator)
				}
			default:
				return common.Errorf(common.Invalid, "expression key %q has invalid operator %q", e.Key, e.Operator)
			}
			if e.Key == api.EnvironmentLabelKey {
				hasEnv = true
				if e.Operator != storepb.LabelSelectorRequirement_IN || len(e.Values) != 1 {
					return common.Errorf(common.Invalid, "label %q should must use operator %q with exactly one value", api.EnvironmentLabelKey, storepb.LabelSelectorRequirement_IN)
				}
			}
		}
		if !hasEnv {
			return common.Errorf(common.Invalid, "deployment should contain %q label", api.EnvironmentLabelKey)
		}
	}
	return nil
}

// GetDatabaseMatrixFromDeploymentSchedule gets a pipeline based on deployment schedule.
// The matrix will include the stage even if the stage has no database.
func GetDatabaseMatrixFromDeploymentSchedule(schedule *storepb.Schedule, databaseList []*store.DatabaseMessage) ([][]*store.DatabaseMessage, error) {
	var matrix [][]*store.DatabaseMessage

	// idToLabels maps database -> label key -> label value
	idToLabels := make(map[string]map[string]string)
	databaseMap := make(map[string]*store.DatabaseMessage)
	for _, database := range databaseList {
		databaseMap[database.String()] = database
		newMap := make(map[string]string)
		for k, v := range database.Metadata.Labels {
			newMap[k] = v
		}
		newMap[api.EnvironmentLabelKey] = database.EffectiveEnvironmentID

		idToLabels[database.String()] = newMap
	}

	// idsSeen records database which is already in a stage.
	idsSeen := make(map[string]bool)

	// For each stage, we loop over all databases to see if it is a match.
	for _, deployment := range schedule.Deployments {
		// For each stage, we will get a list of matched databases.
		var matchedDatabaseList []string
		// Loop over databaseList instead of idToLabels to get determinant results.
		for _, database := range databaseList {
			// Skip if the database is already in a stage.
			if _, ok := idsSeen[database.String()]; ok {
				continue
			}
			// Skip if the database is not found.
			if database.SyncState == api.NotFound {
				continue
			}

			if isMatchExpressions(idToLabels[database.String()], deployment.Spec.Selector.MatchExpressions) {
				matchedDatabaseList = append(matchedDatabaseList, database.String())
				idsSeen[database.String()] = true
			}
		}

		var databaseList []*store.DatabaseMessage
		for _, id := range matchedDatabaseList {
			databaseList = append(databaseList, databaseMap[id])
		}
		// sort databases in stage based on database name.
		if len(databaseList) > 0 {
			sort.Slice(databaseList, func(i, j int) bool {
				return databaseList[i].String() < databaseList[j].String()
			})
		}

		matrix = append(matrix, databaseList)
	}

	return matrix, nil
}

// GetTaskSheetID gets the sheetID of a task.
func GetTaskSheetID(taskPayload string) (int, error) {
	var taskSheetID struct {
		SheetID int `json:"sheetId"`
	}
	if err := json.Unmarshal([]byte(taskPayload), &taskSheetID); err != nil {
		return 0, err
	}
	return taskSheetID.SheetID, nil
}

// GetTaskSkipped gets skipped from a task.
func GetTaskSkipped(task *store.TaskMessage) (bool, error) {
	var payload struct {
		Skipped bool `json:"skipped,omitempty"`
	}
	if err := json.Unmarshal([]byte(task.Payload), &payload); err != nil {
		return false, err
	}
	return payload.Skipped, nil
}

// FindNextPendingStep finds the next pending step in the approval flow.
func FindNextPendingStep(template *storepb.ApprovalTemplate, approvers []*storepb.IssuePayloadApproval_Approver) *storepb.ApprovalStep {
	// We can do the finding like this for now because we are presuming that
	// one step is approved by one approver.
	// and the approver status is either
	// APPROVED or REJECTED.
	if len(approvers) >= len(template.Flow.Steps) {
		return nil
	}
	return template.Flow.Steps[len(approvers)]
}

// FindRejectedStep finds the rejected step in the approval flow.
func FindRejectedStep(template *storepb.ApprovalTemplate, approvers []*storepb.IssuePayloadApproval_Approver) *storepb.ApprovalStep {
	for i, approver := range approvers {
		if i >= len(template.Flow.Steps) {
			return nil
		}
		if approver.Status == storepb.IssuePayloadApproval_Approver_REJECTED {
			return template.Flow.Steps[i]
		}
	}
	return nil
}

// CheckApprovalApproved checks if the approval is approved.
func CheckApprovalApproved(approval *storepb.IssuePayloadApproval) (bool, error) {
	if approval == nil || !approval.ApprovalFindingDone {
		return false, nil
	}
	if approval.ApprovalFindingError != "" {
		return false, nil
	}
	if len(approval.ApprovalTemplates) == 0 {
		return true, nil
	}
	if len(approval.ApprovalTemplates) != 1 {
		return false, errors.Errorf("expecting one approval template but got %d", len(approval.ApprovalTemplates))
	}
	return FindRejectedStep(approval.ApprovalTemplates[0], approval.Approvers) == nil && FindNextPendingStep(approval.ApprovalTemplates[0], approval.Approvers) == nil, nil
}

// CheckIssueApproved checks if the issue is approved.
func CheckIssueApproved(issue *store.IssueMessage) (bool, error) {
	return CheckApprovalApproved(issue.Payload.Approval)
}

// HandleIncomingApprovalSteps handles incoming approval steps.
// - Blocks approval steps if no user can approve the step.
func HandleIncomingApprovalSteps(approval *storepb.IssuePayloadApproval) ([]*storepb.IssuePayloadApproval_Approver, error) {
	if len(approval.ApprovalTemplates) == 0 {
		return nil, nil
	}

	var approvers []*storepb.IssuePayloadApproval_Approver

	step := FindNextPendingStep(approval.ApprovalTemplates[0], approval.Approvers)
	if step == nil {
		return nil, nil
	}
	if len(step.Nodes) != 1 {
		return nil, errors.Errorf("expecting one node but got %v", len(step.Nodes))
	}
	if step.Type != storepb.ApprovalStep_ANY {
		return nil, errors.Errorf("expecting ANY step type but got %v", step.Type)
	}
	return approvers, nil
}

// UpdateProjectPolicyFromGrantIssue updates the project policy from grant issue.
func UpdateProjectPolicyFromGrantIssue(ctx context.Context, stores *store.Store, issue *store.IssueMessage, grantRequest *storepb.GrantRequest) error {
	policyMessage, err := stores.GetProjectIamPolicy(ctx, issue.Project.ResourceID)
	if err != nil {
		return errors.Wrapf(err, "failed to get project policy for project %s", issue.Project.ResourceID)
	}

	var newConditionExpr string
	if grantRequest.Condition != nil {
		newConditionExpr = grantRequest.Condition.Expression
	}
	updated := false

	userID, err := strconv.Atoi(strings.TrimPrefix(grantRequest.User, "users/"))
	if err != nil {
		return err
	}
	newUser, err := stores.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}
	if newUser == nil {
		return status.Errorf(codes.Internal, "user %v not found", userID)
	}
	for _, binding := range policyMessage.Policy.Bindings {
		if binding.Role != grantRequest.Role {
			continue
		}
		var oldConditionExpr string
		if binding.Condition != nil {
			oldConditionExpr = binding.Condition.Expression
		}
		if oldConditionExpr != newConditionExpr {
			continue
		}
		// Append
		binding.Members = append(binding.Members, common.FormatUserUID(newUser.ID))
		updated = true
		break
	}
	if !updated {
		condition := grantRequest.Condition
		if condition == nil {
			condition = &expr.Expr{}
		}
		condition.Description = fmt.Sprintf("#%d", issue.UID)
		policyMessage.Policy.Bindings = append(policyMessage.Policy.Bindings, &storepb.Binding{
			Role:      grantRequest.Role,
			Members:   []string{common.FormatUserUID(newUser.ID)},
			Condition: condition,
		})
	}

	policyPayload, err := protojson.Marshal(policyMessage.Policy)
	if err != nil {
		return err
	}
	if _, err := stores.CreatePolicyV2(ctx, &store.PolicyMessage{
		Resource:          common.FormatProject(issue.Project.ResourceID),
		ResourceType:      api.PolicyResourceTypeProject,
		Payload:           string(policyPayload),
		Type:              api.PolicyTypeIAM,
		InheritFromParent: false,
		// Enforce cannot be false while creating a policy.
		Enforce: true,
	}); err != nil {
		return err
	}

	return nil
}

// RenderStatement renders the given template statement with the given key-value map.
func RenderStatement(templateStatement string, secrets map[string]string) string {
	// Happy path for empty template statement.
	if templateStatement == "" {
		return ""
	}
	// Optimizations for databases without secrets.
	if len(secrets) == 0 {
		return templateStatement
	}
	// Don't render statement larger than 1MB.
	if len(templateStatement) > 1024*1024 {
		return templateStatement
	}

	// The regular expression consists of:
	// \${{: matches the string ${{, where $ is escaped with a backslash.
	// \s*: matches zero or more whitespace characters.
	// secrets\.: matches the string secrets., where . is escaped with a backslash.
	// (?P<name>[A-Z0-9_]+): uses a named capture group name to match the secret name. The capture group is defined using the syntax (?P<name>) and matches one or more uppercase letters, digits, or underscores.
	re := regexp.MustCompile(`\${{\s*secrets\.(?P<name>[A-Z0-9_]+)\s*}}`)
	matches := re.FindAllStringSubmatch(templateStatement, -1)
	for _, match := range matches {
		name := match[1]
		if value, ok := secrets[name]; ok {
			templateStatement = strings.ReplaceAll(templateStatement, match[0], value)
		}
	}
	return templateStatement
}

// GetSecretMapFromDatabaseMessage extracts the secret map from the given database message.
func GetSecretMapFromDatabaseMessage(databaseMessage *store.DatabaseMessage) map[string]string {
	materials := make(map[string]string)
	if databaseMessage.Secrets == nil || len(databaseMessage.Secrets.Items) == 0 {
		return materials
	}

	for _, item := range databaseMessage.Secrets.Items {
		materials[item.Name] = item.Value
	}
	return materials
}

// GetMatchedAndUnmatchedDatabasesInDatabaseGroup returns the matched and unmatched databases in the given database group.
func GetMatchedAndUnmatchedDatabasesInDatabaseGroup(ctx context.Context, databaseGroup *store.DatabaseGroupMessage, allDatabases []*store.DatabaseMessage) ([]*store.DatabaseMessage, []*store.DatabaseMessage, error) {
	var matches []*store.DatabaseMessage
	var unmatches []*store.DatabaseMessage

	// DONOT check bb.feature.database-grouping for instance. The API here is read-only in the frontend, we need to show if the instance is matched but missing required license.
	// The feature guard will works during issue creation.
	for _, database := range allDatabases {
		matched, err := CheckDatabaseGroupMatch(ctx, databaseGroup.Expression.Expression, database)
		if err != nil {
			return nil, nil, err
		}
		if matched {
			matches = append(matches, database)
		} else {
			unmatches = append(unmatches, database)
		}
	}
	return matches, unmatches, nil
}

func CheckDatabaseGroupMatch(ctx context.Context, expression string, database *store.DatabaseMessage) (bool, error) {
	prog, err := common.ValidateGroupCELExpr(expression)
	if err != nil {
		return false, err
	}

	res, _, err := prog.ContextEval(ctx, map[string]any{
		"resource": map[string]any{
			"database_name":    database.DatabaseName,
			"environment_name": common.FormatEnvironment(database.EffectiveEnvironmentID),
			"instance_id":      database.InstanceID,
			"labels":           database.Metadata.Labels,
		},
	})
	if err != nil {
		return false, status.Error(codes.Internal, err.Error())
	}

	val, err := res.ConvertToNative(reflect.TypeOf(false))
	if err != nil {
		return false, status.Errorf(codes.Internal, "expect bool result")
	}
	if boolVal, ok := val.(bool); ok && boolVal {
		return true, nil
	}
	return false, nil
}

func Uniq[T comparable](array []T) []T {
	res := make([]T, 0, len(array))
	seen := make(map[T]struct{}, len(array))

	for _, e := range array {
		if _, ok := seen[e]; ok {
			continue
		}
		seen[e] = struct{}{}
		res = append(res, e)
	}

	return res
}

// ConvertBytesToUTF8String tries to decode a byte slice into a UTF-8 string using common encodings.
func ConvertBytesToUTF8String(data []byte) (string, error) {
	encodings := []encoding.Encoding{
		textunicode.UTF8,
		simplifiedchinese.GBK,
		textunicode.UTF16(textunicode.LittleEndian, textunicode.UseBOM),
		textunicode.UTF16(textunicode.BigEndian, textunicode.UseBOM),
		charmap.ISO8859_1,
	}

	for _, enc := range encodings {
		reader := transform.NewReader(strings.NewReader(string(data)), enc.NewDecoder())
		decoded, err := io.ReadAll(reader)
		if err == nil && isUtf8(decoded) {
			return string(decoded), nil
		}
	}
	return "", errors.New("failed to decode the byte slice into a UTF-8 string")
}

func isUtf8(data []byte) bool {
	return !strings.Contains(string(data), string(unicode.ReplacementChar))
}

// IsSpaceOrSemicolon checks if the rune is a space or a semicolon.
func IsSpaceOrSemicolon(r rune) bool {
	if ok := unicode.IsSpace(r); ok {
		return true
	}
	return r == ';'
}
