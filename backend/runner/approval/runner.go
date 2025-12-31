// Package approval is the runner for finding approval templates for issues.
package approval

import (
	"context"
	"log/slog"
	"maps"
	"math"
	"sync"

	"github.com/google/cel-go/cel"
	"github.com/pkg/errors"

	celtypes "github.com/google/cel-go/common/types"

	apiv1 "github.com/bytebase/bytebase/backend/api/v1"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/bus"
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
	bus            *bus.Bus
	webhookManager *webhook.Manager
	licenseService *enterprise.LicenseService
}

// NewRunner creates a new runner.
func NewRunner(store *store.Store, bus *bus.Bus, webhookManager *webhook.Manager, licenseService *enterprise.LicenseService) *Runner {
	return &Runner{
		store:          store,
		bus:            bus,
		webhookManager: webhookManager,
		licenseService: licenseService,
	}
}

// Run runs the runner.
func (r *Runner) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	slog.Debug("Approval runner started (event-driven)")

	for {
		select {
		case issueUID := <-r.bus.ApprovalCheckChan:
			r.processIssue(ctx, issueUID)
		case <-ctx.Done():
			return
		}
	}
}

func (r *Runner) processIssue(ctx context.Context, issueUID int64) {
	// Get fresh issue from database
	uid := int(issueUID)
	issue, err := r.store.GetIssue(ctx, &store.FindIssueMessage{UID: &uid})
	if err != nil {
		slog.Error("failed to get issue for approval check",
			slog.Int64("issue_uid", issueUID), log.BBError(err))
		return
	}
	if issue == nil {
		return // Issue deleted, nothing to do
	}

	approvalSetting, err := r.store.GetWorkspaceApprovalSetting(ctx)
	if err != nil {
		slog.Error("failed to get workspace approval setting", log.BBError(err))
		return
	}

	// Find approval template - errors are logged, not persisted
	_, err = r.findApprovalTemplateForIssue(ctx, issue, approvalSetting)
	if err != nil {
		slog.Error("failed to find approval template",
			slog.Int64("issue_uid", issueUID),
			slog.String("issue_title", issue.Title),
			log.BBError(err))
		// Don't persist error - user can rerun plan check to retry
	}
}

func (r *Runner) findApprovalTemplateForIssue(ctx context.Context, issue *store.IssueMessage, approvalSetting *storepb.WorkspaceApprovalSetting) (bool, error) {
	payload := issue.Payload
	if payload.Approval != nil && payload.Approval.ApprovalFindingDone {
		return true, nil
	}

	project, err := r.store.GetProject(ctx, &store.FindProjectMessage{ResourceID: &issue.ProjectID})
	if err != nil {
		return false, errors.Wrap(err, "failed to get project")
	}
	if project == nil {
		return false, errors.Errorf("project %s not found", issue.ProjectID)
	}

	approvalTemplate, celVarsList, done, err := func() (*storepb.ApprovalTemplate, []map[string]any, bool, error) {
		// no need to find if feature is not enabled
		if r.licenseService.IsFeatureEnabled(v1pb.PlanFeature_FEATURE_APPROVAL_WORKFLOW) != nil {
			// nolint:nilerr
			return nil, nil, true, nil
		}

		// Step 1: Determine approval source from issue type
		approvalSource, err := r.getApprovalSourceFromIssue(ctx, issue)
		if err != nil {
			return nil, nil, false, errors.Wrap(err, "failed to get approval source from issue")
		}
		if approvalSource == storepb.WorkspaceApprovalSetting_Rule_SOURCE_UNSPECIFIED {
			// Cannot determine source, no approval needed
			return nil, nil, true, nil
		}

		// Step 2: Build CEL variables for evaluation
		celVarsList, done, err := r.buildCELVariablesForIssue(ctx, issue)
		if err != nil {
			return nil, nil, false, errors.Wrap(err, "failed to build CEL variables for issue")
		}
		if !done {
			// Not ready yet (e.g., waiting for plan check runs)
			return nil, nil, false, nil
		}

		// Step 3: Inject risk level into CEL variables for CHANGE_DATABASE issues
		// Risk level is calculated from statement types and injected so approval rules
		// can use conditions like: risk_level == "HIGH"
		if approvalSource == storepb.WorkspaceApprovalSetting_Rule_CHANGE_DATABASE {
			riskLevel := calculateRiskLevelFromCELVars(celVarsList)
			injectRiskLevelIntoCELVars(celVarsList, riskLevel)
		}

		// Step 4: Find matching approval template
		approvalTemplate, err := getApprovalTemplate(approvalSetting, approvalSource, celVarsList)
		if err != nil {
			return nil, nil, false, errors.Wrapf(err, "failed to get approval template for source: %v", approvalSource)
		}

		return approvalTemplate, celVarsList, true, nil
	}()
	if err != nil {
		// Don't persist error - it will be logged by caller
		// User can rerun plan check to retry
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
		if err := func() error {
			newStatus := storepb.Issue_DONE
			updatedIssue, err := r.store.UpdateIssue(ctx, issue.UID, &store.UpdateIssueMessage{Status: &newStatus})
			if err != nil {
				return errors.Wrapf(err, "failed to update issue %q's status", issue.Title)
			}

			if _, err := r.store.CreateIssueComments(ctx, common.SystemBotEmail, &store.IssueCommentMessage{
				IssueUID: issue.UID,
				Payload: &storepb.IssueCommentPayload{
					Event: &storepb.IssueCommentPayload_IssueUpdate_{
						IssueUpdate: &storepb.IssueCommentPayload_IssueUpdate{
							FromStatus: &issue.Status,
							ToStatus:   &updatedIssue.Status,
						},
					},
				},
			}); err != nil {
				return errors.Wrapf(err, "failed to create issue comment after changing the issue status")
			}

			return nil
		}(); err != nil {
			return false, errors.Wrap(err, "failed to update issue status")
		}
	}

	// Calculate risk level separately from approval flow
	// TODO(p0ny): maybe move risk calculation to another runner in the future
	riskLevel := calculateRiskLevelFromCELVars(celVarsList)

	payload.Approval = &storepb.IssuePayloadApproval{
		ApprovalFindingDone: true,
		ApprovalTemplate:    approvalTemplate,
		Approvers:           nil,
	}
	payload.RiskLevel = riskLevel

	if err := updateIssueApprovalPayload(ctx, r.store, issue, payload.Approval, riskLevel); err != nil {
		return false, errors.Wrap(err, "failed to update issue payload")
	}

	func() {
		if payload.Approval.ApprovalTemplate == nil {
			return
		}
		role := utils.FindNextPendingRole(payload.Approval.ApprovalTemplate, payload.Approval.Approvers)
		if role == "" {
			return
		}

		// Get issue creator as actor
		creator, err := r.store.GetUserByEmail(ctx, issue.CreatorEmail)
		if err != nil {
			slog.Warn("failed to get issue creator, using system bot", log.BBError(err))
			creator = store.SystemBotUser
		}

		// Get approvers for this role
		approvers, err := r.getApproversForRole(ctx, issue.ProjectID, role)
		if err != nil {
			slog.Warn("failed to get approvers", log.BBError(err))
			approvers = []webhook.User{} // Continue with empty list
		}

		// Trigger ISSUE_APPROVAL_REQUESTED webhook
		r.webhookManager.CreateEvent(ctx, &webhook.Event{
			Type:    storepb.Activity_ISSUE_APPROVAL_REQUESTED,
			Project: webhook.NewProject(project),
			ApprovalRequested: &webhook.EventIssueApprovalRequested{
				Creator: &webhook.User{
					Name:  creator.Name,
					Email: creator.Email,
				},
				Issue:     webhook.NewIssue(issue),
				Approvers: approvers,
			},
		})
	}()

	return true, nil
}

// calculateRiskLevelFromCELVars calculates the risk level from CEL variables.
// This is separated from approval flow generation to allow independent evolution.
func calculateRiskLevelFromCELVars(celVarsList []map[string]any) storepb.RiskLevel {
	if celVarsList == nil {
		return storepb.RiskLevel_LOW
	}
	statementTypes := collectStatementTypes(celVarsList)
	return common.GetRiskLevelFromStatementTypes(statementTypes)
}

// injectRiskLevelIntoCELVars adds the risk level to all CEL variable maps.
// This allows approval rules to use conditions like: risk_level == "HIGH"
func injectRiskLevelIntoCELVars(celVarsList []map[string]any, riskLevel storepb.RiskLevel) {
	riskLevelStr := riskLevelToString(riskLevel)
	for _, celVars := range celVarsList {
		celVars[common.CELAttributeRiskLevel] = riskLevelStr
	}
}

// riskLevelToString converts a RiskLevel enum to its string representation for CEL.
func riskLevelToString(level storepb.RiskLevel) string {
	switch level {
	case storepb.RiskLevel_LOW:
		return "LOW"
	case storepb.RiskLevel_MODERATE:
		return "MODERATE"
	case storepb.RiskLevel_HIGH:
		return "HIGH"
	default:
		return "LOW"
	}
}

// getApprovalTemplate finds the first matching approval template for the given source and CEL variables.
// Uses two-phase matching:
// Phase 1: Try source-specific rules (filtered by riskSource)
// Phase 2: Try SOURCE_UNSPECIFIED fallback rules (with limited CEL variables)
//
// Parameters:
// - approvalSetting: workspace approval setting containing rules
// - riskSource: the approval source enum (DDL, DML, CREATE_DATABASE, EXPORT_DATA, REQUEST_ROLE)
// - celVarsList: list of CEL variable maps (one per task/component in the issue)
//
// For each rule, we check if ANY of the celVars maps matches the condition.
// First matching rule wins within each phase.
func getApprovalTemplate(approvalSetting *storepb.WorkspaceApprovalSetting, riskSource storepb.WorkspaceApprovalSetting_Rule_Source, celVarsList []map[string]any) (*storepb.ApprovalTemplate, error) {
	if len(approvalSetting.Rules) == 0 {
		return nil, nil
	}
	if len(celVarsList) == 0 {
		return nil, nil
	}

	// Phase 1: Try source-specific rules
	template, err := matchRulesForSource(approvalSetting.Rules, riskSource, celVarsList, common.ApprovalFactors)
	if err != nil {
		return nil, err
	}
	if template != nil {
		return template, nil
	}

	// Phase 2: Try SOURCE_UNSPECIFIED fallback rules with limited CEL variables
	fallbackVars := buildFallbackCELVars(celVarsList)
	template, err = matchRulesForSource(approvalSetting.Rules, storepb.WorkspaceApprovalSetting_Rule_SOURCE_UNSPECIFIED, fallbackVars, common.FallbackApprovalFactors)
	if err != nil {
		return nil, err
	}
	return template, nil
}

// matchRulesForSource evaluates rules for a specific source type.
func matchRulesForSource(rules []*storepb.WorkspaceApprovalSetting_Rule, source storepb.WorkspaceApprovalSetting_Rule_Source, celVarsList []map[string]any, celFactors []cel.EnvOption) (*storepb.ApprovalTemplate, error) {
	e, err := cel.NewEnv(celFactors...)
	if err != nil {
		return nil, err
	}

	for _, rule := range rules {
		// Filter by source
		if rule.Source != source {
			continue
		}

		// Empty condition means skip (not a catch-all)
		if rule.Condition == nil || rule.Condition.Expression == "" {
			continue
		}

		// Special case: "true" expression always matches
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

// buildFallbackCELVars extracts only the project_id from the first CEL vars map.
// Fallback rules can only use resource.project_id.
func buildFallbackCELVars(celVarsList []map[string]any) []map[string]any {
	if len(celVarsList) == 0 {
		return nil
	}

	// Extract project_id from the first vars map
	firstVars := celVarsList[0]
	projectID, ok := firstVars[common.CELAttributeResourceProjectID]
	if !ok {
		return nil
	}

	return []map[string]any{
		{common.CELAttributeResourceProjectID: projectID},
	}
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

	planCheckRun, err := r.store.GetPlanCheckRun(ctx, plan.UID)
	if err != nil {
		return nil, false, errors.Wrapf(err, "failed to get plan check run for plan %v", plan.UID)
	}

	type Key struct {
		InstanceID   string
		DatabaseName string
	}
	latestPlanCheckRun := map[Key]*storepb.PlanCheckRunResult_Result{}

	// Wait for plan check to complete if running
	if planCheckRun != nil && planCheckRun.Status == store.PlanCheckRunStatusRunning {
		return nil, false, nil // Not ready yet, retry later
	}

	// Build map from results, filtering for STATEMENT_SUMMARY_REPORT
	if planCheckRun != nil {
		for _, result := range planCheckRun.Result.Results {
			if result.Type != storepb.PlanCheckType_PLAN_CHECK_TYPE_STATEMENT_SUMMARY_REPORT {
				continue
			}
			instanceID, databaseName, err := common.GetInstanceDatabaseID(result.Target)
			if err != nil {
				continue
			}
			key := Key{
				InstanceID:   instanceID,
				DatabaseName: databaseName,
			}
			latestPlanCheckRun[key] = result
		}
	}

	// Build CEL variables for each task
	tasks, err := apiv1.GetPipelineCreate(ctx, r.store, plan.Config.GetSpecs(), issue.ProjectID)
	if err != nil {
		return nil, false, errors.Wrap(err, "failed to get pipeline create")
	}

	var celVarsList []map[string]any
	for _, task := range tasks {
		instance, err := r.store.GetInstance(ctx, &store.FindInstanceMessage{
			ResourceID: &task.InstanceID,
		})
		if err != nil {
			return nil, false, errors.Wrapf(err, "failed to get instance %v", task.InstanceID)
		}
		if instance.Deleted {
			continue
		}

		taskStatement := ""
		sheetSha256 := task.Payload.GetSheetSha256()
		if sheetSha256 != "" {
			sheet, err := r.store.GetSheetFull(ctx, sheetSha256)
			if err != nil {
				return nil, true, errors.Wrapf(err, "failed to get statement in sheet %v", sheetSha256)
			}
			if sheet == nil {
				return nil, true, errors.Errorf("sheet %v not found", sheetSha256)
			}
			taskStatement = sheet.Statement
		}

		var environmentID string
		var databaseName string
		if task.Type == storepb.Task_DATABASE_CREATE {
			databaseName = task.Payload.GetDatabaseName()
			environmentID = task.Payload.GetEnvironmentId()
		} else {
			database, err := r.store.GetDatabase(ctx, &store.FindDatabaseMessage{
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
			common.CELAttributeResourceProjectID:     issue.ProjectID,
			common.CELAttributeResourceInstanceID:    instance.ResourceID,
			common.CELAttributeResourceDatabaseName:  databaseName,
			common.CELAttributeResourceDBEngine:      instance.Metadata.GetEngine().String(),
			common.CELAttributeStatementText:         taskStatement,
		}

		// Add summary report data if available
		result, ok := latestPlanCheckRun[Key{
			InstanceID:   instance.ResourceID,
			DatabaseName: databaseName,
		}]
		if !ok {
			celVarsList = append(celVarsList, celVars)
			continue
		}

		report := result.GetSqlSummaryReport()
		if report == nil {
			celVarsList = append(celVarsList, celVars)
			continue
		}

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

		celVarsList = append(celVarsList, expandCELVars(celVars, report.StatementTypes, tableNames)...)
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

	tasks, err := apiv1.GetPipelineCreate(ctx, r.store, plan.Config.GetSpecs(), issue.ProjectID)
	if err != nil {
		return nil, false, errors.Wrap(err, "failed to get pipeline create")
	}

	var celVarsList []map[string]any
	for _, task := range tasks {
		if task.Type != storepb.Task_DATABASE_EXPORT {
			continue
		}
		instance, err := r.store.GetInstance(ctx, &store.FindInstanceMessage{
			ResourceID: &task.InstanceID,
		})
		if err != nil {
			return nil, false, errors.Wrapf(err, "failed to get instance %v", task.InstanceID)
		}
		if instance.Deleted {
			continue
		}

		database, err := r.store.GetDatabase(ctx, &store.FindDatabaseMessage{
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
			common.CELAttributeResourceProjectID:     issue.ProjectID,
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
		common.CELAttributeResourceProjectID:     issue.ProjectID,
		common.CELAttributeRequestExpirationDays: expirationDays,
		common.CELAttributeRequestRole:           payload.GrantRequest.Role,
	}

	// If no specific databases, create one entry per environment
	if len(factors.Databases) == 0 {
		environments, err := r.store.GetEnvironment(ctx)
		if err != nil {
			return nil, false, err
		}
		var celVarsList []map[string]any
		for _, environment := range environments.GetEnvironments() {
			celVars := maps.Clone(baseVars)
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
		celVars := maps.Clone(baseVars)
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
		instance, err := r.store.GetInstance(ctx, &store.FindInstanceMessage{ResourceID: &instanceID})
		if err != nil {
			return nil, err
		}
		if instance == nil || instance.Deleted {
			continue
		}
		db, err := r.store.GetDatabase(ctx, &store.FindDatabaseMessage{
			InstanceID:   &instanceID,
			DatabaseName: &databaseName,
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

// getApprovalSourceFromPlan determines the approval rule source enum from the plan config.
// This is used after the risk layer removal to directly filter approval rules by source.
func getApprovalSourceFromPlan(config *storepb.PlanConfig) storepb.WorkspaceApprovalSetting_Rule_Source {
	for _, spec := range config.GetSpecs() {
		switch spec.Config.(type) {
		case *storepb.PlanConfig_Spec_CreateDatabaseConfig:
			return storepb.WorkspaceApprovalSetting_Rule_CREATE_DATABASE
		case *storepb.PlanConfig_Spec_ChangeDatabaseConfig:
			return storepb.WorkspaceApprovalSetting_Rule_CHANGE_DATABASE
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

func updateIssueApprovalPayload(ctx context.Context, s *store.Store, issue *store.IssueMessage, approval *storepb.IssuePayloadApproval, riskLevel storepb.RiskLevel) error {
	if _, err := s.UpdateIssue(ctx, issue.UID, &store.UpdateIssueMessage{
		PayloadUpsert: &storepb.Issue{
			Approval:  approval,
			RiskLevel: riskLevel,
		},
	}); err != nil {
		return errors.Wrap(err, "failed to update issue payload")
	}
	return nil
}

// expandCELVars creates CEL variable maps for each combination of statement types and table names.
func expandCELVars(base map[string]any, statementTypes, tableNames []string) []map[string]any {
	if len(statementTypes) == 0 {
		return []map[string]any{base}
	}

	// Use empty string as sentinel when no table names
	if len(tableNames) == 0 {
		tableNames = []string{""}
	}

	var result []map[string]any
	for _, statementType := range statementTypes {
		for _, tableName := range tableNames {
			vars := maps.Clone(base)
			vars[common.CELAttributeStatementSQLType] = statementType
			if tableName != "" {
				vars[common.CELAttributeResourceTableName] = tableName
			}
			result = append(result, vars)
		}
	}
	return result
}

// collectStatementTypes extracts all statement types from CEL variables list.
func collectStatementTypes(celVarsList []map[string]any) []string {
	seen := make(map[string]bool)
	var result []string
	for _, vars := range celVarsList {
		if sqlType, ok := vars[common.CELAttributeStatementSQLType].(string); ok && sqlType != "" {
			if !seen[sqlType] {
				seen[sqlType] = true
				result = append(result, sqlType)
			}
		}
	}
	return result
}

// getApproversForRole retrieves the list of users who have the specified role
// for the given project. It queries both project and workspace IAM policies.
// Only returns END_USER type principals (excludes service accounts, system bots, etc).
func (r *Runner) getApproversForRole(ctx context.Context, projectID string, role string) ([]webhook.User, error) {
	// Get project IAM policy
	projectIAM, err := r.store.GetProjectIamPolicy(ctx, projectID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get project IAM policy")
	}

	// Get workspace IAM policy
	workspaceIAM, err := r.store.GetWorkspaceIamPolicy(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get workspace IAM policy")
	}

	// Get all users with the specified role
	users := utils.GetUsersByRoleInIAMPolicy(ctx, r.store, role, projectIAM.Policy, workspaceIAM.Policy)

	// Convert to webhook.User format, filtering by END_USER principal type
	approvers := make([]webhook.User, 0, len(users))
	for _, user := range users {
		// Only include END_USER principals as approvers
		if user.Type != storepb.PrincipalType_END_USER {
			continue
		}
		approvers = append(approvers, webhook.User{
			Name:  user.Name,
			Email: user.Email,
		})
	}

	return approvers, nil
}
