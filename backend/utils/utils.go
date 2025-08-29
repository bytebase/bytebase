// Package utils is a utility library for server.
//
//nolint:revive
package utils

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"unicode"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"google.golang.org/genproto/googleapis/type/expr"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

// DataSourceFromInstanceWithType gets a typed data source from an instance.
func DataSourceFromInstanceWithType(instance *store.InstanceMessage, dataSourceType storepb.DataSourceType) *storepb.DataSource {
	for _, dataSource := range instance.Metadata.GetDataSources() {
		if dataSource.GetType() == dataSourceType {
			return dataSource
		}
	}
	return nil
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
		return connect.NewError(connect.CodeInternal, errors.Errorf("user %v not found", userID))
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
		ResourceType:      storepb.Policy_PROJECT,
		Payload:           string(policyPayload),
		Type:              storepb.Policy_IAM,
		InheritFromParent: false,
		// Enforce cannot be false while creating a policy.
		Enforce: true,
	}); err != nil {
		return err
	}

	return nil
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

	effectiveEnvironmentID := ""
	if database.EffectiveEnvironmentID != nil {
		effectiveEnvironmentID = *database.EffectiveEnvironmentID
	}
	res, _, err := prog.ContextEval(ctx, map[string]any{
		"resource": map[string]any{
			"database_name":    database.DatabaseName,
			"environment_name": common.FormatEnvironment(effectiveEnvironmentID),
			"instance_id":      database.InstanceID,
			"labels":           database.Metadata.Labels,
		},
	})
	if err != nil {
		return false, connect.NewError(connect.CodeInternal, err)
	}

	val, err := res.ConvertToNative(reflect.TypeFor[bool]())
	if err != nil {
		return false, connect.NewError(connect.CodeInternal, errors.New("expect bool result"))
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

// IsSpaceOrSemicolon checks if the rune is a space or a semicolon.
func IsSpaceOrSemicolon(r rune) bool {
	if ok := unicode.IsSpace(r); ok {
		return true
	}
	return r == ';'
}
