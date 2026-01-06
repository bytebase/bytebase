// Package utils is a utility library for server.
//
//nolint:revive
package utils

import (
	"context"
	"fmt"
	"reflect"
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

// FindNextPendingRole finds the next pending role in the approval flow.
func FindNextPendingRole(approval *storepb.IssuePayloadApproval) string {
	if approval == nil || approval.ApprovalTemplate == nil {
		return ""
	}
	// We can do the finding like this for now because we are presuming that
	// one role is approved by one approver.
	// and the approver status is either
	// APPROVED or REJECTED.
	if len(approval.Approvers) >= len(approval.ApprovalTemplate.Flow.Roles) {
		return ""
	}
	return approval.ApprovalTemplate.Flow.Roles[len(approval.Approvers)]
}

// FindRejectedRole finds the rejected role in the approval flow.
func FindRejectedRole(approval *storepb.IssuePayloadApproval) string {
	if approval == nil || approval.ApprovalTemplate == nil {
		return ""
	}
	for i, approver := range approval.Approvers {
		if i >= len(approval.ApprovalTemplate.Flow.Roles) {
			return ""
		}
		if approver.Status == storepb.IssuePayloadApproval_Approver_REJECTED {
			return approval.ApprovalTemplate.Flow.Roles[i]
		}
	}
	return ""
}

// CheckApprovalApproved checks if the approval is approved.
func CheckApprovalApproved(approval *storepb.IssuePayloadApproval) (bool, error) {
	if approval == nil || !approval.ApprovalFindingDone {
		return false, nil
	}
	if approval.ApprovalTemplate == nil {
		return true, nil
	}
	return FindRejectedRole(approval) == "" && FindNextPendingRole(approval) == "", nil
}

// CheckIssueApproved checks if the issue is approved.
func CheckIssueApproved(issue *store.IssueMessage) (bool, error) {
	return CheckApprovalApproved(issue.Payload.Approval)
}

// UpdateProjectPolicyFromGrantIssue updates the project policy from grant issue.
func UpdateProjectPolicyFromGrantIssue(ctx context.Context, stores *store.Store, issue *store.IssueMessage, grantRequest *storepb.GrantRequest) error {
	policyMessage, err := stores.GetProjectIamPolicy(ctx, issue.ProjectID)
	if err != nil {
		return errors.Wrapf(err, "failed to get project policy for project %s", issue.ProjectID)
	}

	var newConditionExpr string
	if grantRequest.Condition != nil {
		newConditionExpr = grantRequest.Condition.Expression
	}
	updated := false

	email := strings.TrimPrefix(grantRequest.User, "users/")
	if email == "" {
		return errors.New("invalid empty user identifier")
	}
	newUser, err := stores.GetUserByEmail(ctx, email)
	if err != nil {
		return errors.Wrapf(err, "failed to find user %s", email)
	}
	if newUser == nil {
		return connect.NewError(connect.CodeInternal, errors.Errorf("user %s not found", email))
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
		binding.Members = append(binding.Members, common.FormatUserEmail(newUser.Email))
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
			Members:   []string{common.FormatUserEmail(newUser.Email)},
			Condition: condition,
		})
	}

	policyPayload, err := protojson.Marshal(policyMessage.Policy)
	if err != nil {
		return err
	}
	if _, err := stores.CreatePolicy(ctx, &store.PolicyMessage{
		Resource:          common.FormatProject(issue.ProjectID),
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

// GetMatchedDatabasesInDatabaseGroup returns the matched databases in the given database group.
func GetMatchedDatabasesInDatabaseGroup(ctx context.Context, databaseGroup *store.DatabaseGroupMessage, allDatabases []*store.DatabaseMessage) ([]*store.DatabaseMessage, error) {
	var matches []*store.DatabaseMessage

	// DONOT check bb.feature.database-grouping for instance. The API here is read-only in the frontend, we need to show if the instance is matched but missing required license.
	// The feature guard will works during issue creation.
	for _, database := range allDatabases {
		matched, err := CheckDatabaseGroupMatch(ctx, databaseGroup.Expression.Expression, database)
		if err != nil {
			return nil, err
		}
		if matched {
			matches = append(matches, database)
		}
	}
	return matches, nil
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
		common.CELAttributeResourceDatabaseName:   database.DatabaseName,
		common.CELAttributeResourceEnvironmentID:  effectiveEnvironmentID,
		common.CELAttributeResourceInstanceID:     database.InstanceID,
		common.CELAttributeResourceDatabaseLabels: database.Metadata.Labels,
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

// IsSpaceOrSemicolon checks if the rune is a space or a semicolon.
func IsSpaceOrSemicolon(r rune) bool {
	if ok := unicode.IsSpace(r); ok {
		return true
	}
	return r == ';'
}
