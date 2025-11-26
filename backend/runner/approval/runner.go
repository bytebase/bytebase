// Package approval is the runner for finding approval templates for issues.
package approval

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"math"
	"sync"
	"time"

	"github.com/google/cel-go/cel"
	"github.com/pkg/errors"
	"go.uber.org/multierr"

	celtypes "github.com/google/cel-go/common/types"

	apiv1 "github.com/bytebase/bytebase/backend/api/v1"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/sheet"
	"github.com/bytebase/bytebase/backend/component/state"
	"github.com/bytebase/bytebase/backend/component/webhook"
	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
)

// Runner is the runner for finding approval templates for issues.
type Runner struct {
	store          *store.Store
	sheetManager   *sheet.Manager
	dbFactory      *dbfactory.DBFactory
	stateCfg       *state.State
	webhookManager *webhook.Manager
	licenseService *enterprise.LicenseService
}

// NewRunner creates a new runner.
func NewRunner(store *store.Store, sheetManager *sheet.Manager, dbFactory *dbfactory.DBFactory, stateCfg *state.State, webhookManager *webhook.Manager, licenseService *enterprise.LicenseService) *Runner {
	return &Runner{
		store:          store,
		sheetManager:   sheetManager,
		dbFactory:      dbFactory,
		stateCfg:       stateCfg,
		webhookManager: webhookManager,
		licenseService: licenseService,
	}
}

const approvalRunnerInterval = 1 * time.Second

// Run runs the runner.
func (r *Runner) Run(ctx context.Context, wg *sync.WaitGroup) {
	ticker := time.NewTicker(approvalRunnerInterval)
	defer ticker.Stop()
	defer wg.Done()
	slog.Debug(fmt.Sprintf("Approval runner started and will run every %v", approvalRunnerInterval))
	r.retryFindApprovalTemplate(ctx)

	for {
		select {
		case <-ticker.C:
			func() {
				defer func() {
					if r := recover(); r != nil {
						err, ok := r.(error)
						if !ok {
							err = errors.Errorf("%v", r)
						}
						slog.Error("Approval runner PANIC RECOVER", log.BBError(err), log.BBStack("panic-stack"))
					}
				}()
				r.runOnce(ctx)
			}()
		case <-ctx.Done():
			return
		}
	}
}

func (r *Runner) runOnce(ctx context.Context) {
	approvalSetting, err := r.store.GetWorkspaceApprovalSetting(ctx)
	if err != nil {
		slog.Error("failed to get workspace approval setting", log.BBError(err))
		return
	}

	r.stateCfg.ApprovalFinding.Range(func(key, value any) bool {
		issue, ok := value.(*store.IssueMessage)
		if !ok {
			return true
		}
		done, err := r.findApprovalTemplateForIssue(ctx, issue, approvalSetting)
		if err != nil {
			slog.Error("failed to find approval template for issue", slog.Int("issue", issue.UID), log.BBError(err))
		}
		if err != nil || done {
			r.stateCfg.ApprovalFinding.Delete(key)
		}
		return true
	})
}

func (r *Runner) retryFindApprovalTemplate(ctx context.Context) {
	issues, err := r.store.ListIssueV2(ctx, &store.FindIssueMessage{
		StatusList: []storepb.Issue_Status{storepb.Issue_OPEN},
	})
	if err != nil {
		err := errors.Wrap(err, "failed to list issues")
		slog.Error("failed to retry finding approval template", log.BBError(err))
	}
	for _, issue := range issues {
		payload := issue.Payload
		if payload.Approval == nil || !payload.Approval.ApprovalFindingDone {
			r.stateCfg.ApprovalFinding.Store(issue.UID, issue)
		}
	}
}

func (r *Runner) findApprovalTemplateForIssue(ctx context.Context, issue *store.IssueMessage, approvalSetting *storepb.WorkspaceApprovalSetting) (bool, error) {
	payload := issue.Payload
	if payload.Approval != nil && payload.Approval.ApprovalFindingDone {
		return true, nil
	}

	approvalTemplate, done, err := func() (*storepb.ApprovalTemplate, bool, error) {
		// no need to find if feature is not enabled
		if r.licenseService.IsFeatureEnabled(v1pb.PlanFeature_FEATURE_APPROVAL_WORKFLOW) != nil {
			// nolint:nilerr
			return nil, true, nil
		}

		// Step 1: Determine approval source from issue type
		approvalSource, err := r.getApprovalSourceFromIssue(ctx, issue)
		if err != nil {
			return nil, false, errors.Wrap(err, "failed to get approval source from issue")
		}
		if approvalSource == storepb.WorkspaceApprovalSetting_Rule_SOURCE_UNSPECIFIED {
			// Cannot determine source, no approval needed
			return nil, true, nil
		}

		// Step 2: Build CEL variables for evaluation
		celVarsList, done, err := r.buildCELVariablesForIssue(ctx, issue)
		if err != nil {
			return nil, false, errors.Wrap(err, "failed to build CEL variables for issue")
		}
		if !done {
			// Not ready yet (e.g., waiting for plan check runs)
			return nil, false, nil
		}

		// Step 3: Find matching approval template
		approvalTemplate, err := getApprovalTemplate(approvalSetting, approvalSource, celVarsList)
		if err != nil {
			return nil, false, errors.Wrapf(err, "failed to get approval template for source: %v", approvalSource)
		}

		return approvalTemplate, true, nil
	}()
	if err != nil {
		if updateErr := updateIssueApprovalPayload(ctx, r.store, issue, &storepb.IssuePayloadApproval{
			ApprovalFindingDone:  true,
			ApprovalFindingError: err.Error(),
		}); updateErr != nil {
			return false, multierr.Append(errors.Wrap(updateErr, "failed to update issue payload"), err)
		}
		return false, err
	}
	if !done {
		return false, nil
	}

	// Grant privilege and close issue similar to actions on issue approval.
	if issue.Type == storepb.Issue_GRANT_REQUEST && approvalTemplate == nil {
		if err := utils.UpdateProjectPolicyFromGrantIssue(ctx, r.store, issue, payload.GrantRequest); err != nil {
			return false, err
		}
		if err := webhook.ChangeIssueStatus(ctx, r.store, r.webhookManager, issue, storepb.Issue_DONE, r.store.GetSystemBotUser(ctx), ""); err != nil {
			return false, errors.Wrap(err, "failed to update issue status")
		}
	}

	payload.Approval = &storepb.IssuePayloadApproval{
		ApprovalFindingDone: true,
		RiskLevel:           storepb.RiskLevel_RISK_LEVEL_UNSPECIFIED, // Risk level no longer calculated after 3.13
		ApprovalTemplate:    approvalTemplate,
		Approvers:           nil,
	}

	if err := updateIssueApprovalPayload(ctx, r.store, issue, payload.Approval); err != nil {
		return false, errors.Wrap(err, "failed to update issue payload")
	}

	if err := func() error {
		if payload.Approval.ApprovalTemplate != nil {
			return nil
		}
		if issue.PipelineUID == nil {
			return nil
		}
		tasks, err := r.store.ListTasks(ctx, &store.TaskFind{PipelineID: issue.PipelineUID})
		if err != nil {
			return errors.Wrapf(err, "failed to list tasks")
		}
		if len(tasks) == 0 {
			return nil
		}
		// Get the first environment from tasks
		var firstEnvironment string
		for _, task := range tasks {
			firstEnvironment = task.Environment
			break
		}
		policy, err := apiv1.GetValidRolloutPolicyForEnvironment(ctx, r.store, firstEnvironment)
		if err != nil {
			return err
		}
		r.webhookManager.CreateEvent(ctx, &webhook.Event{
			Actor:   r.store.GetSystemBotUser(ctx),
			Type:    storepb.Activity_NOTIFY_PIPELINE_ROLLOUT,
			Comment: "",
			Issue:   webhook.NewIssue(issue),
			Project: webhook.NewProject(issue.Project),
			IssueRolloutReady: &webhook.EventIssueRolloutReady{
				RolloutPolicy: policy,
				StageName:     firstEnvironment,
			},
		})
		return nil
	}(); err != nil {
		slog.Error("failed to create rollout release notification activity", log.BBError(err))
	}

	func() {
		if payload.Approval.ApprovalTemplate == nil {
			return
		}
		role := utils.FindNextPendingRole(payload.Approval.ApprovalTemplate, payload.Approval.Approvers)
		if role == "" {
			return
		}
		r.webhookManager.CreateEvent(ctx, &webhook.Event{
			Actor:   r.store.GetSystemBotUser(ctx),
			Type:    storepb.Activity_ISSUE_APPROVAL_NOTIFY,
			Comment: "",
			Issue:   webhook.NewIssue(issue),
			Project: webhook.NewProject(issue.Project),
			IssueApprovalCreate: &webhook.EventIssueApprovalCreate{
				Role: role,
			},
		})
	}()

	return true, nil
}

// getApprovalTemplate finds the first matching approval template for the given source and CEL variables.
// After the risk layer removal (3.13), approval rules are directly evaluated with full CEL context.
// The rules are ordered by priority (HIGH risk rules first, then MODERATE, then LOW, then UNSPECIFIED).
// First matching rule wins.
//
// Parameters:
// - approvalSetting: workspace approval setting containing rules
// - riskSource: the approval source enum (DDL, DML, CREATE_DATABASE, EXPORT_DATA, REQUEST_ROLE)
// - celVarsList: list of CEL variable maps (one per task/component in the issue)
//
// For each rule filtered by source, we check if ANY of the celVars maps matches the condition.
// This preserves the "maximum risk" behavior: if any task matches a high-priority rule, that template is used.
func getApprovalTemplate(approvalSetting *storepb.WorkspaceApprovalSetting, riskSource storepb.WorkspaceApprovalSetting_Rule_Source, celVarsList []map[string]any) (*storepb.ApprovalTemplate, error) {
	if len(approvalSetting.Rules) == 0 {
		return nil, nil
	}
	if len(celVarsList) == 0 {
		return nil, nil
	}

	e, err := cel.NewEnv(common.ApprovalFactors...)
	if err != nil {
		return nil, err
	}

	// Rules are already ordered by priority (migration ensures this)
	// First matching rule wins
	for _, rule := range approvalSetting.Rules {
		// Filter by source
		if rule.Source != riskSource {
			continue
		}

		// Empty condition means this rule always matches (fallback rule)
		if rule.Condition == nil || rule.Condition.Expression == "" {
			continue
		}

		// Special case: "true" expression always matches (fallback from UNSPECIFIED level)
		if rule.Condition.Expression == "true" {
			return rule.Template, nil
		}

		ast, issues := e.Compile(rule.Condition.Expression)
		if issues != nil && issues.Err() != nil {
			return nil, issues.Err()
		}

		prg, err := e.Program(ast, cel.EvalOptions(cel.OptPartialEval))
		if err != nil {
			return nil, errors.Wrap(err, "failed to compile expression")
		}

		// Check if ANY of the CEL variable maps matches this rule's condition
		for _, celVars := range celVarsList {
			vars, err := e.PartialVars(celVars)
			if err != nil {
				return nil, errors.Wrap(err, "failed to create partial vars")
			}
			out, _, err := prg.Eval(vars)
			if err != nil {
				// Evaluation error - continue to next celVars map
				continue
			}
			if res, ok := out.Equal(celtypes.True).Value().(bool); ok && res {
				return rule.Template, nil
			}
		}
	}
	return nil, nil
}

// buildCELVariablesForIssue builds the CEL variable maps for evaluating approval rules.
// Returns a list of CEL variable maps (one per task/component), done flag, and error.
// done=false means the caller should retry later (e.g., waiting for plan check runs).
func (r *Runner) buildCELVariablesForIssue(ctx context.Context, issue *store.IssueMessage) ([]map[string]any, bool, error) {
	switch issue.Type {
	case storepb.Issue_GRANT_REQUEST:
		return r.buildCELVariablesForGrantRequest(ctx, issue)
	case storepb.Issue_DATABASE_CHANGE:
		return r.buildCELVariablesForDatabaseChange(ctx, issue)
	case storepb.Issue_DATABASE_EXPORT:
		return r.buildCELVariablesForDataExport(ctx, issue)
	default:
		return nil, false, errors.Errorf("unknown issue type %v", issue.Type)
	}
}

// buildCELVariablesForDatabaseChange builds CEL variables for DATABASE_CHANGE issues.
// This includes DDL and DML operations.
func (r *Runner) buildCELVariablesForDatabaseChange(ctx context.Context, issue *store.IssueMessage) ([]map[string]any, bool, error) {
	if issue.PlanUID == nil {
		return nil, false, errors.Errorf("expected plan UID in issue %v", issue.UID)
	}
	plan, err := r.store.GetPlan(ctx, &store.FindPlanMessage{UID: issue.PlanUID})
	if err != nil {
		return nil, false, errors.Wrapf(err, "failed to get plan %v", *issue.PlanUID)
	}
	if plan == nil {
		return nil, false, errors.Errorf("plan %v not found", *issue.PlanUID)
	}

	// Check if we can conclude the operation type
	riskSource := getRiskSourceFromPlan(plan.Config)
	if riskSource == store.RiskSourceUnknown {
		// Cannot determine source, return done=true with empty list (no approval needed)
		return []map[string]any{{}}, true, nil
	}

	// Check plan check runs status
	planCheckRuns, err := r.store.ListPlanCheckRuns(ctx, &store.FindPlanCheckRunMessage{
		PlanUID: &plan.UID,
		Type:    &[]store.PlanCheckRunType{store.PlanCheckDatabaseStatementSummaryReport},
	})
	if err != nil {
		return nil, false, errors.Wrapf(err, "failed to list plan check runs for plan %v", plan.UID)
	}

	type Key struct {
		InstanceID   string
		DatabaseName string
	}
	latestPlanCheckRun := map[Key]*store.PlanCheckRunMessage{}
	for _, run := range planCheckRuns {
		key := Key{
			InstanceID:   run.Config.InstanceId,
			DatabaseName: run.Config.DatabaseName,
		}
		latestPlanCheckRun[key] = run
	}

	// Wait for plan check runs to complete
	var planCheckRunDone int
	for _, run := range latestPlanCheckRun {
		if run.Status != store.PlanCheckRunStatusRunning {
			planCheckRunDone++
		}
	}
	planCheckRunCount := len(latestPlanCheckRun)
	if planCheckRunCount < common.MinimumCompletedPlanCheckRun && planCheckRunCount != planCheckRunDone {
		return nil, false, nil // Not ready yet, retry later
	}
	if planCheckRunCount >= common.MinimumCompletedPlanCheckRun && planCheckRunDone < common.MinimumCompletedPlanCheckRun {
		return nil, false, nil // Not ready yet, retry later
	}

	// Build CEL variables for each task
	pipelineCreate, err := apiv1.GetPipelineCreate(ctx, r.store, r.sheetManager, r.dbFactory, plan.Config.GetSpecs(), plan.Config.GetDeployment(), issue.Project)
	if err != nil {
		return nil, false, errors.Wrap(err, "failed to get pipeline create")
	}

	var celVarsList []map[string]any
	for _, task := range pipelineCreate.Tasks {
		instance, err := r.store.GetInstanceV2(ctx, &store.FindInstanceMessage{
			ResourceID: &task.InstanceID,
		})
		if err != nil {
			return nil, false, errors.Wrapf(err, "failed to get instance %v", task.InstanceID)
		}
		if instance.Deleted {
			continue
		}

		taskStatement := ""
		sheetUID := int(task.Payload.GetSheetId())
		if sheetUID != 0 {
			statement, err := r.store.GetSheetStatementByID(ctx, sheetUID)
			if err != nil {
				return nil, true, errors.Wrapf(err, "failed to get statement in sheet %v", sheetUID)
			}
			taskStatement = statement
		}

		var environmentID string
		var databaseName string
		if task.Type == storepb.Task_DATABASE_CREATE {
			databaseName = task.Payload.GetDatabaseName()
			environmentID = task.Payload.GetEnvironmentId()
		} else {
			database, err := r.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
				InstanceID:   &task.InstanceID,
				DatabaseName: task.DatabaseName,
			})
			if err != nil {
				return nil, false, err
			}
			databaseName = database.DatabaseName
			if database.EffectiveEnvironmentID != nil {
				environmentID = *database.EffectiveEnvironmentID
			}
		}

		// Base CEL variables
		celVars := map[string]any{
			common.CELAttributeResourceEnvironmentID: environmentID,
			common.CELAttributeResourceProjectID:     issue.Project.ResourceID,
			common.CELAttributeResourceInstanceID:    instance.ResourceID,
			common.CELAttributeResourceDatabaseName:  databaseName,
			common.CELAttributeResourceDBEngine:      instance.Metadata.GetEngine().String(),
			common.CELAttributeStatementText:         taskStatement,
		}

		// Add summary report data if available
		if run, ok := latestPlanCheckRun[Key{
			InstanceID:   instance.ResourceID,
			DatabaseName: databaseName,
		}]; ok {
			foundReport := false
			for _, result := range run.Result.Results {
				report := result.GetSqlSummaryReport()
				if report == nil {
					continue
				}
				foundReport = true

				// Calculate table rows from changed resources
				var tableRows int64
				var tableNames []string
				for _, db := range report.GetChangedResources().GetDatabases() {
					for _, sc := range db.GetSchemas() {
						for _, tb := range sc.GetTables() {
							tableRows += tb.GetTableRows()
							tableNames = append(tableNames, tb.Name)
						}
					}
				}

				celVars[common.CELAttributeStatementAffectedRows] = report.AffectedRows
				celVars[common.CELAttributeStatementTableRows] = tableRows

				// Create CEL variables for each statement type and table combination
				// to ensure we match the most appropriate approval rule
				if len(report.StatementTypes) > 0 && len(tableNames) > 0 {
					for _, statementType := range report.StatementTypes {
						for _, tableName := range tableNames {
							vars := maps.Clone(celVars)
							vars[common.CELAttributeStatementSQLType] = statementType
							vars[common.CELAttributeResourceTableName] = tableName
							celVarsList = append(celVarsList, vars)
						}
					}
				} else if len(report.StatementTypes) > 0 {
					for _, statementType := range report.StatementTypes {
						vars := maps.Clone(celVars)
						vars[common.CELAttributeStatementSQLType] = statementType
						celVarsList = append(celVarsList, vars)
					}
				} else {
					celVarsList = append(celVarsList, celVars)
				}
				break // Use first report
			}
			// If no valid report was found, still append basic celVars
			if !foundReport {
				celVarsList = append(celVarsList, celVars)
			}
		} else {
			celVarsList = append(celVarsList, celVars)
		}
	}

	// If no tasks, return empty list (no approval needed)
	if len(celVarsList) == 0 {
		celVarsList = append(celVarsList, map[string]any{})
	}

	return celVarsList, true, nil
}

// buildCELVariablesForDataExport builds CEL variables for DATABASE_EXPORT issues.
func (r *Runner) buildCELVariablesForDataExport(ctx context.Context, issue *store.IssueMessage) ([]map[string]any, bool, error) {
	if issue.PlanUID == nil {
		return nil, false, errors.Errorf("expected plan UID in issue %v", issue.UID)
	}
	plan, err := r.store.GetPlan(ctx, &store.FindPlanMessage{UID: issue.PlanUID})
	if err != nil {
		return nil, false, errors.Wrapf(err, "failed to get plan %v", *issue.PlanUID)
	}
	if plan == nil {
		return nil, false, errors.Errorf("plan %v not found", *issue.PlanUID)
	}

	pipelineCreate, err := apiv1.GetPipelineCreate(ctx, r.store, r.sheetManager, r.dbFactory, plan.Config.GetSpecs(), plan.Config.GetDeployment(), issue.Project)
	if err != nil {
		return nil, false, errors.Wrap(err, "failed to get pipeline create")
	}

	var celVarsList []map[string]any
	for _, task := range pipelineCreate.Tasks {
		if task.Type != storepb.Task_DATABASE_EXPORT {
			continue
		}
		instance, err := r.store.GetInstanceV2(ctx, &store.FindInstanceMessage{
			ResourceID: &task.InstanceID,
		})
		if err != nil {
			return nil, false, errors.Wrapf(err, "failed to get instance %v", task.InstanceID)
		}
		if instance.Deleted {
			continue
		}

		database, err := r.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
			InstanceID:   &task.InstanceID,
			DatabaseName: task.DatabaseName,
		})
		if err != nil {
			return nil, false, err
		}

		envID := ""
		if database.EffectiveEnvironmentID != nil {
			envID = *database.EffectiveEnvironmentID
		}

		celVars := map[string]any{
			common.CELAttributeResourceEnvironmentID: envID,
			common.CELAttributeResourceProjectID:     issue.Project.ResourceID,
			common.CELAttributeResourceInstanceID:    instance.ResourceID,
			common.CELAttributeResourceDatabaseName:  database.DatabaseName,
			common.CELAttributeResourceDBEngine:      instance.Metadata.GetEngine().String(),
		}
		celVarsList = append(celVarsList, celVars)
	}

	if len(celVarsList) == 0 {
		celVarsList = append(celVarsList, map[string]any{})
	}

	return celVarsList, true, nil
}

// buildCELVariablesForGrantRequest builds CEL variables for GRANT_REQUEST issues.
func (r *Runner) buildCELVariablesForGrantRequest(ctx context.Context, issue *store.IssueMessage) ([]map[string]any, bool, error) {
	payload := issue.Payload
	if payload.GrantRequest == nil {
		return nil, false, errors.New("grant request payload not found")
	}

	factors, err := common.GetQueryExportFactors(payload.GetGrantRequest().GetCondition().GetExpression())
	if err != nil {
		return nil, false, errors.Wrap(err, "failed to get query export factors")
	}

	// Default to max int if expiration is not set (no expiration)
	expirationDays := int64(math.MaxInt32)
	if payload.GrantRequest.Expiration != nil {
		expirationDays = int64(payload.GrantRequest.Expiration.AsDuration().Hours() / 24)
	}

	baseVars := map[string]any{
		common.CELAttributeResourceProjectID:     issue.Project.ResourceID,
		common.CELAttributeRequestExpirationDays: expirationDays,
		common.CELAttributeRequestRole:           payload.GrantRequest.Role,
	}

	// If no specific databases, create one entry per environment
	if len(factors.Databases) == 0 {
		environments, err := r.store.GetEnvironmentSetting(ctx)
		if err != nil {
			return nil, false, err
		}
		var celVarsList []map[string]any
		for _, environment := range environments.GetEnvironments() {
			celVars := make(map[string]any)
			for k, v := range baseVars {
				celVars[k] = v
			}
			celVars[common.CELAttributeResourceEnvironmentID] = environment.Id
			celVarsList = append(celVarsList, celVars)
		}
		if len(celVarsList) == 0 {
			celVarsList = append(celVarsList, baseVars)
		}
		return celVarsList, true, nil
	}

	// Build one entry per database
	databaseMap, err := r.getDatabaseMap(ctx, factors.Databases)
	if err != nil {
		return nil, false, errors.Wrap(err, "failed to retrieve database map")
	}

	var celVarsList []map[string]any
	for _, database := range databaseMap {
		celVars := make(map[string]any)
		for k, v := range baseVars {
			celVars[k] = v
		}
		if database.EffectiveEnvironmentID != nil {
			celVars[common.CELAttributeResourceEnvironmentID] = *database.EffectiveEnvironmentID
		} else {
			celVars[common.CELAttributeResourceEnvironmentID] = ""
		}
		celVarsList = append(celVarsList, celVars)
	}

	if len(celVarsList) == 0 {
		celVarsList = append(celVarsList, baseVars)
	}

	return celVarsList, true, nil
}

func (r *Runner) getDatabaseMap(ctx context.Context, databases []string) (map[string]*store.DatabaseMessage, error) {
	databaseMap := make(map[string]*store.DatabaseMessage)
	for _, database := range databases {
		instanceID, databaseName, err := common.GetInstanceDatabaseID(database)
		if err != nil {
			return nil, err
		}
		instance, err := r.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &instanceID})
		if err != nil {
			return nil, err
		}
		if instance == nil || instance.Deleted {
			continue
		}
		db, err := r.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
			InstanceID:      &instanceID,
			DatabaseName:    &databaseName,
			IsCaseSensitive: store.IsObjectCaseSensitive(instance),
		})
		if err != nil {
			return nil, err
		}
		if db == nil {
			continue
		}
		databaseMap[database] = db
	}
	return databaseMap, nil
}

func getRiskSourceFromPlan(config *storepb.PlanConfig) store.RiskSource {
	for _, spec := range config.GetSpecs() {
		switch v := spec.Config.(type) {
		case *storepb.PlanConfig_Spec_CreateDatabaseConfig:
			return store.RiskSourceDatabaseCreate
		case *storepb.PlanConfig_Spec_ChangeDatabaseConfig:
			switch v.ChangeDatabaseConfig.Type {
			case storepb.PlanConfig_ChangeDatabaseConfig_MIGRATE:
				// For MIGRATE type, check the migrate type
				switch v.ChangeDatabaseConfig.MigrateType {
				case storepb.MigrationType_DML:
					return store.RiskSourceDatabaseDataUpdate
				default:
					return store.RiskSourceDatabaseSchemaUpdate
				}
			case storepb.PlanConfig_ChangeDatabaseConfig_SDL:
				return store.RiskSourceDatabaseSchemaUpdate
			default:
				return store.RiskSourceDatabaseSchemaUpdate
			}
		}
	}
	return store.RiskSourceUnknown
}

// getApprovalSourceFromPlan determines the approval rule source enum from the plan config.
// This is used after the risk layer removal to directly filter approval rules by source.
func getApprovalSourceFromPlan(config *storepb.PlanConfig) storepb.WorkspaceApprovalSetting_Rule_Source {
	for _, spec := range config.GetSpecs() {
		switch v := spec.Config.(type) {
		case *storepb.PlanConfig_Spec_CreateDatabaseConfig:
			return storepb.WorkspaceApprovalSetting_Rule_CREATE_DATABASE
		case *storepb.PlanConfig_Spec_ChangeDatabaseConfig:
			switch v.ChangeDatabaseConfig.Type {
			case storepb.PlanConfig_ChangeDatabaseConfig_MIGRATE:
				switch v.ChangeDatabaseConfig.MigrateType {
				case storepb.MigrationType_DML:
					return storepb.WorkspaceApprovalSetting_Rule_DML
				default:
					return storepb.WorkspaceApprovalSetting_Rule_DDL
				}
			case storepb.PlanConfig_ChangeDatabaseConfig_SDL:
				return storepb.WorkspaceApprovalSetting_Rule_DDL
			default:
				return storepb.WorkspaceApprovalSetting_Rule_DDL
			}
		case *storepb.PlanConfig_Spec_ExportDataConfig:
			return storepb.WorkspaceApprovalSetting_Rule_EXPORT_DATA
		}
	}
	return storepb.WorkspaceApprovalSetting_Rule_SOURCE_UNSPECIFIED
}

// getApprovalSourceFromIssue determines the approval rule source enum from the issue type.
func (r *Runner) getApprovalSourceFromIssue(ctx context.Context, issue *store.IssueMessage) (storepb.WorkspaceApprovalSetting_Rule_Source, error) {
	switch issue.Type {
	case storepb.Issue_GRANT_REQUEST:
		return storepb.WorkspaceApprovalSetting_Rule_REQUEST_ROLE, nil
	case storepb.Issue_DATABASE_CHANGE:
		if issue.PlanUID == nil {
			return storepb.WorkspaceApprovalSetting_Rule_SOURCE_UNSPECIFIED, errors.Errorf("expected plan UID in issue %v", issue.UID)
		}
		plan, err := r.store.GetPlan(ctx, &store.FindPlanMessage{UID: issue.PlanUID})
		if err != nil {
			return storepb.WorkspaceApprovalSetting_Rule_SOURCE_UNSPECIFIED, errors.Wrapf(err, "failed to get plan %v", *issue.PlanUID)
		}
		if plan == nil {
			return storepb.WorkspaceApprovalSetting_Rule_SOURCE_UNSPECIFIED, errors.Errorf("plan %v not found", *issue.PlanUID)
		}
		return getApprovalSourceFromPlan(plan.Config), nil
	case storepb.Issue_DATABASE_EXPORT:
		return storepb.WorkspaceApprovalSetting_Rule_EXPORT_DATA, nil
	default:
		return storepb.WorkspaceApprovalSetting_Rule_SOURCE_UNSPECIFIED, errors.Errorf("unknown issue type %v", issue.Type)
	}
}

func updateIssueApprovalPayload(ctx context.Context, s *store.Store, issue *store.IssueMessage, approval *storepb.IssuePayloadApproval) error {
	if _, err := s.UpdateIssueV2(ctx, issue.UID, &store.UpdateIssueMessage{
		PayloadUpsert: &storepb.Issue{
			Approval: approval,
		},
	}); err != nil {
		return errors.Wrap(err, "failed to update issue payload")
	}
	return nil
}
