// Package utils is a utility library for server.
package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
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
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/state"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/app/relay"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
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
func isMatchExpression(labels map[string]string, expression *store.LabelSelectorRequirement) bool {
	switch expression.Operator {
	case store.InOperatorType:
		return checkLabelIn(labels, expression)
	case store.NotInOperatorType:
		return !checkLabelIn(labels, expression)
	case store.ExistsOperatorType:
		_, ok := labels[expression.Key]
		return ok
	default:
		return false
	}
}

func checkLabelIn(labels map[string]string, expression *store.LabelSelectorRequirement) bool {
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

func isMatchExpressions(labels map[string]string, expressionList []*store.LabelSelectorRequirement) bool {
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
func ValidateDeploymentSchedule(schedule *store.Schedule) error {
	for _, d := range schedule.Deployments {
		if d.Name == "" {
			return common.Errorf(common.Invalid, "Deployment name must not be empty")
		}
		hasEnv := false
		for _, e := range d.Spec.Selector.MatchExpressions {
			switch e.Operator {
			case store.InOperatorType, store.NotInOperatorType:
				if len(e.Values) == 0 {
					return common.Errorf(common.Invalid, "expression key %q with %q operator should have at least one value", e.Key, e.Operator)
				}
			case store.ExistsOperatorType:
				if len(e.Values) > 0 {
					return common.Errorf(common.Invalid, "expression key %q with %q operator shouldn't have values", e.Key, e.Operator)
				}
			default:
				return common.Errorf(common.Invalid, "expression key %q has invalid operator %q", e.Key, e.Operator)
			}
			if e.Key == api.EnvironmentLabelKey {
				hasEnv = true
				if e.Operator != store.InOperatorType || len(e.Values) != 1 {
					return common.Errorf(common.Invalid, "label %q should must use operator %q with exactly one value", api.EnvironmentLabelKey, store.InOperatorType)
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
func GetDatabaseMatrixFromDeploymentSchedule(schedule *store.Schedule, databaseList []*store.DatabaseMessage) ([][]*store.DatabaseMessage, error) {
	var matrix [][]*store.DatabaseMessage

	// idToLabels maps databaseID -> label key -> label value
	idToLabels := make(map[int]map[string]string)
	databaseMap := make(map[int]*store.DatabaseMessage)
	for _, database := range databaseList {
		databaseMap[database.UID] = database
		newMap := make(map[string]string)
		for k, v := range database.Metadata.Labels {
			newMap[k] = v
		}
		newMap[api.EnvironmentLabelKey] = database.EffectiveEnvironmentID

		idToLabels[database.UID] = newMap
	}

	// idsSeen records database id which is already in a stage.
	idsSeen := make(map[int]bool)

	// For each stage, we loop over all databases to see if it is a match.
	for _, deployment := range schedule.Deployments {
		// For each stage, we will get a list of matched databases.
		var matchedDatabaseList []int
		// Loop over databaseList instead of idToLabels to get determinant results.
		for _, database := range databaseList {
			// Skip if the database is already in a stage.
			if _, ok := idsSeen[database.UID]; ok {
				continue
			}
			// Skip if the database is not found.
			if database.SyncState == api.NotFound {
				continue
			}

			if isMatchExpressions(idToLabels[database.UID], deployment.Spec.Selector.MatchExpressions) {
				matchedDatabaseList = append(matchedDatabaseList, database.UID)
				idsSeen[database.UID] = true
			}
		}

		var databaseList []*store.DatabaseMessage
		for _, id := range matchedDatabaseList {
			databaseList = append(databaseList, databaseMap[id])
		}
		// sort databases in stage based on IDs.
		if len(databaseList) > 0 {
			sort.Slice(databaseList, func(i, j int) bool {
				return databaseList[i].UID < databaseList[j].UID
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

// MergeTaskCreateLists merges a matrix of taskCreate and taskIndexDAG to a list of taskCreate and taskIndexDAG.
// The index of returned taskIndexDAG list is set regarding the merged taskCreate.
func MergeTaskCreateLists(taskCreateLists [][]*store.TaskMessage, taskIndexDAGLists [][]store.TaskIndexDAG) ([]*store.TaskMessage, []store.TaskIndexDAG, error) {
	if len(taskCreateLists) != len(taskIndexDAGLists) {
		return nil, nil, errors.Errorf("expect taskCreateLists and taskIndexDAGLists to have the same length, get %d, %d respectively", len(taskCreateLists), len(taskIndexDAGLists))
	}
	var resTaskCreateList []*store.TaskMessage
	var resTaskIndexDAGList []store.TaskIndexDAG
	offset := 0
	for i := range taskCreateLists {
		taskCreateList := taskCreateLists[i]
		taskIndexDAGList := taskIndexDAGLists[i]

		resTaskCreateList = append(resTaskCreateList, taskCreateList...)
		for _, dag := range taskIndexDAGList {
			resTaskIndexDAGList = append(resTaskIndexDAGList, store.TaskIndexDAG{
				FromIndex: dag.FromIndex + offset,
				ToIndex:   dag.ToIndex + offset,
			})
		}
		offset += len(taskCreateList)
	}
	return resTaskCreateList, resTaskIndexDAGList, nil
}

// ExecuteMigrationDefault executes migration.
func ExecuteMigrationDefault(ctx context.Context, driverCtx context.Context, store *store.Store, stateCfg *state.State, taskRunUID int, driver db.Driver, mi *db.MigrationInfo, statement string, sheetID *int, opts db.ExecuteOptions) (migrationHistoryID string, updatedSchema string, resErr error) {
	execFunc := func(ctx context.Context, execStatement string) error {
		if _, err := driver.Execute(ctx, execStatement, opts); err != nil {
			return err
		}
		return nil
	}
	return ExecuteMigrationWithFunc(ctx, driverCtx, store, stateCfg, taskRunUID, driver, mi, statement, sheetID, execFunc, opts)
}

// ExecuteMigrationWithFunc executes the migration with custom migration function.
func ExecuteMigrationWithFunc(ctx context.Context, driverCtx context.Context, s *store.Store, stateCfg *state.State, taskRunUID int, driver db.Driver, m *db.MigrationInfo, statement string, sheetID *int, execFunc func(ctx context.Context, execStatement string) error, opts db.ExecuteOptions) (migrationHistoryID string, updatedSchema string, resErr error) {
	opts.LogSchemaDumpStart()
	var prevSchemaBuf bytes.Buffer
	// Don't record schema if the database hasn't existed yet or is schemaless, e.g. MongoDB.
	// For baseline migration, we also record the live schema to detect the schema drift.
	// See https://bytebase.com/blog/what-is-database-schema-drift
	if _, err := driver.Dump(ctx, &prevSchemaBuf); err != nil {
		opts.LogSchemaDumpEnd(err.Error())
		return "", "", err
	}
	opts.LogSchemaDumpEnd("")

	insertedID, err := BeginMigration(ctx, s, m, prevSchemaBuf.String(), statement, sheetID)
	if err != nil {
		return "", "", errors.Wrapf(err, "failed to begin migration")
	}

	startedNs := time.Now().UnixNano()

	defer func() {
		if err := EndMigration(ctx, s, startedNs, insertedID, updatedSchema, prevSchemaBuf.String(), sheetID, resErr == nil /* isDone */); err != nil {
			slog.Error("Failed to update migration history record",
				log.BBError(err),
				slog.String("migration_id", migrationHistoryID),
			)
		}
	}()

	// Phase 3 - Executing migration
	// Branch migration type always has empty sql.
	// Baseline migration type could has non-empty sql but will not execute.
	// https://github.com/bytebase/bytebase/issues/394
	doMigrate := true
	if statement == "" || m.Type == db.Baseline {
		doMigrate = false
	}
	if doMigrate {
		var renderedStatement = statement
		// The m.DatabaseID is nil means the migration is a instance level migration
		if m.DatabaseID != nil {
			database, err := s.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
				UID: m.DatabaseID,
			})
			if err != nil {
				return "", "", err
			}
			if database == nil {
				return "", "", errors.Errorf("database %d not found", *m.DatabaseID)
			}
			materials := GetSecretMapFromDatabaseMessage(database)
			// To avoid leak the rendered statement, the error message should use the original statement and not the rendered statement.
			renderedStatement = RenderStatement(statement, materials)
		}

		if stateCfg != nil {
			stateCfg.TaskRunExecutionStatuses.Store(taskRunUID,
				state.TaskRunExecutionStatus{
					ExecutionStatus: v1pb.TaskRun_EXECUTING,
					UpdateTime:      time.Now(),
				})
		}

		if err := execFunc(driverCtx, renderedStatement); err != nil {
			return "", "", err
		}
	}

	if stateCfg != nil {
		stateCfg.TaskRunExecutionStatuses.Store(taskRunUID,
			state.TaskRunExecutionStatus{
				ExecutionStatus: v1pb.TaskRun_POST_EXECUTING,
				UpdateTime:      time.Now(),
			})
	}

	opts.LogSchemaDumpStart()
	// Phase 4 - Dump the schema after migration
	var afterSchemaBuf bytes.Buffer
	if _, err := driver.Dump(ctx, &afterSchemaBuf); err != nil {
		// We will ignore the dump error if the database is dropped.
		if strings.Contains(err.Error(), "not found") {
			return insertedID, "", nil
		}
		opts.LogSchemaDumpEnd(err.Error())
		return "", "", err
	}
	opts.LogSchemaDumpEnd("")

	return insertedID, afterSchemaBuf.String(), nil
}

// BeginMigration checks before executing migration and inserts a migration history record with pending status.
func BeginMigration(ctx context.Context, stores *store.Store, m *db.MigrationInfo, prevSchema, statement string, sheetID *int) (string, error) {
	// Phase 1 - Pre-check before executing migration
	// Check if the same migration version has already been applied.
	if list, err := stores.ListInstanceChangeHistory(ctx, &store.FindInstanceChangeHistoryMessage{
		InstanceID: m.InstanceID,
		DatabaseID: m.DatabaseID,
		Version:    &m.Version,
	}); err != nil {
		return "", errors.Wrap(err, "failed to check duplicate version")
	} else if len(list) > 0 {
		migrationHistory := list[0]
		switch migrationHistory.Status {
		case db.Done:
			return "", common.Errorf(common.MigrationAlreadyApplied, "database %q has already applied version %s, hint: the version might be duplicate, please check the version", m.Database, m.Version.Version)
		case db.Pending:
			err := errors.Errorf("database %q version %s migration is already in progress", m.Database, m.Version.Version)
			slog.Debug(err.Error())
			// For force migration, we will ignore the existing migration history and continue to migration.
			return migrationHistory.UID, nil
		case db.Failed:
			err := errors.Errorf("database %q version %s migration has failed, please check your database to make sure things are fine and then start a new migration using a new version ", m.Database, m.Version.Version)
			slog.Debug(err.Error())
			// For force migration, we will ignore the existing migration history and continue to migration.
			return migrationHistory.UID, nil
		}
	}

	// Phase 2 - Record migration history as PENDING.
	statementRecord, _ := common.TruncateString(statement, common.MaxSheetSize)
	insertedID, err := stores.CreatePendingInstanceChangeHistory(ctx, prevSchema, m, statementRecord, sheetID)
	if err != nil {
		return "", err
	}

	return insertedID, nil
}

// EndMigration updates the migration history record to DONE or FAILED depending on migration is done or not.
func EndMigration(ctx context.Context, storeInstance *store.Store, startedNs int64, insertedID string, updatedSchema, schemaPrev string, sheetID *int, isDone bool) error {
	migrationDurationNs := time.Now().UnixNano() - startedNs
	update := &store.UpdateInstanceChangeHistoryMessage{
		ID:                  insertedID,
		ExecutionDurationNs: &migrationDurationNs,
		// Update the sheet ID just in case it has been updated.
		Sheet: sheetID,
		// Update schemaPrev because we might be re-using a previous change history entry.
		SchemaPrev: &schemaPrev,
	}
	if isDone {
		// Upon success, update the migration history as 'DONE', execution_duration_ns, updated schema.
		status := db.Done
		update.Status = &status
		update.Schema = &updatedSchema
	} else {
		// Otherwise, update the migration history as 'FAILED', execution_duration.
		status := db.Failed
		update.Status = &status
	}
	return storeInstance.UpdateInstanceChangeHistory(ctx, update)
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
// - creates external approvals for external approval nodes.
func HandleIncomingApprovalSteps(ctx context.Context, s *store.Store, relayClient *relay.Client, issue *store.IssueMessage, approval *storepb.IssuePayloadApproval) ([]*storepb.IssuePayloadApproval_Approver, []*store.IssueCommentMessage, error) {
	if len(approval.ApprovalTemplates) == 0 {
		return nil, nil, nil
	}

	var approvers []*storepb.IssuePayloadApproval_Approver
	var issueComments []*store.IssueCommentMessage

	step := FindNextPendingStep(approval.ApprovalTemplates[0], approval.Approvers)
	if step == nil {
		return nil, nil, nil
	}
	if len(step.Nodes) != 1 {
		return nil, nil, errors.Errorf("expecting one node but got %v", len(step.Nodes))
	}
	if step.Type != storepb.ApprovalStep_ANY {
		return nil, nil, errors.Errorf("expecting ANY step type but got %v", step.Type)
	}
	node := step.Nodes[0]
	if v, ok := node.GetPayload().(*storepb.ApprovalNode_ExternalNodeId); ok {
		if err := handleApprovalNodeExternalNode(ctx, s, relayClient, issue, v.ExternalNodeId); err != nil {
			approvers = append(approvers, &storepb.IssuePayloadApproval_Approver{
				Status:      storepb.IssuePayloadApproval_Approver_REJECTED,
				PrincipalId: api.SystemBotID,
			})

			issueComments = append(issueComments, &store.IssueCommentMessage{
				IssueUID: issue.UID,
				Payload: &storepb.IssueCommentPayload{
					Event: &storepb.IssueCommentPayload_Approval_{
						Approval: &storepb.IssueCommentPayload_Approval{
							Status: storepb.IssueCommentPayload_Approval_APPROVED,
						},
					},
				},
			})
		}
	}
	return approvers, issueComments, nil
}

func handleApprovalNodeExternalNode(ctx context.Context, s *store.Store, relayClient *relay.Client, issue *store.IssueMessage, externalNodeID string) error {
	getExternalApprovalByID := func(ctx context.Context, s *store.Store, externalApprovalID string) (*storepb.ExternalApprovalSetting_Node, error) {
		setting, err := s.GetWorkspaceExternalApprovalSetting(ctx)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get workspace external approval setting")
		}
		for _, node := range setting.Nodes {
			if node.Id == externalApprovalID {
				return node, nil
			}
		}
		return nil, nil
	}
	node, err := getExternalApprovalByID(ctx, s, externalNodeID)
	if err != nil {
		return errors.Wrapf(err, "failed to get external approval node %s", externalNodeID)
	}
	if node == nil {
		return errors.Errorf("external approval node %s not found", externalNodeID)
	}
	id, err := relayClient.Create(node.Endpoint, &relay.CreatePayload{
		IssueID:     fmt.Sprintf("%d", issue.UID),
		Title:       issue.Title,
		Description: issue.Description,
		Project:     issue.Project.ResourceID,
		CreateTime:  issue.CreatedTime,
		Creator:     issue.Creator.Email,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to create external approval")
	}
	payload, err := json.Marshal(&api.ExternalApprovalPayloadRelay{
		ExternalApprovalNodeID: node.Id,
		ID:                     id,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to marshal external approval payload")
	}
	if _, err := s.CreateExternalApprovalV2(ctx, &store.ExternalApprovalMessage{
		IssueUID:     issue.UID,
		ApproverUID:  api.SystemBotID,
		Type:         api.ExternalApprovalTypeRelay,
		Payload:      string(payload),
		RequesterUID: api.SystemBotID,
	}); err != nil {
		return errors.Wrapf(err, "failed to create external approval")
	}
	return nil
}

// UpdateProjectPolicyFromGrantIssue updates the project policy from grant issue.
func UpdateProjectPolicyFromGrantIssue(ctx context.Context, stores *store.Store, issue *store.IssueMessage, grantRequest *storepb.GrantRequest) error {
	policy, err := stores.GetProjectIamPolicy(ctx, issue.Project.UID)
	if err != nil {
		return errors.Wrapf(err, "failed to get project policy for project %q", issue.Project.UID)
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
	for _, binding := range policy.Bindings {
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
		policy.Bindings = append(policy.Bindings, &storepb.Binding{
			Role:      grantRequest.Role,
			Members:   []string{common.FormatUserUID(newUser.ID)},
			Condition: condition,
		})
	}

	policyPayload, err := protojson.Marshal(policy)
	if err != nil {
		return err
	}
	if _, err := stores.CreatePolicyV2(ctx, &store.PolicyMessage{
		ResourceUID:       issue.Project.UID,
		ResourceType:      api.PolicyResourceTypeProject,
		Payload:           string(policyPayload),
		Type:              api.PolicyTypeProjectIAM,
		InheritFromParent: false,
		// Enforce cannot be false while creating a policy.
		Enforce: true,
	}, api.SystemBotID); err != nil {
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
		matched, err := CheckDatabaseGroupMatch(ctx, databaseGroup, database)
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

func CheckDatabaseGroupMatch(ctx context.Context, databaseGroup *store.DatabaseGroupMessage, database *store.DatabaseMessage) (bool, error) {
	prog, err := common.ValidateGroupCELExpr(databaseGroup.Expression.Expression)
	if err != nil {
		return false, err
	}

	res, _, err := prog.ContextEval(ctx, map[string]any{
		"resource": map[string]any{
			"database_name":    database.DatabaseName,
			"environment_name": common.FormatEnvironment(database.EffectiveEnvironmentID),
			"instance_id":      database.InstanceID,
		},
	})
	if err != nil {
		return false, status.Errorf(codes.Internal, err.Error())
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

func uniq[T comparable](array []T) []T {
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
