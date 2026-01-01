// Package approval is the runner for finding approval templates for issues.
package approval

import (
	"context"
	"log/slog"
	"maps"
	"math"
	"sync"

	"github.com/google/cel-go/cel"
	celtypes "github.com/google/cel-go/common/types"
	"github.com/pkg/errors"

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

// FindAndApplyApprovalTemplate finds and applies the approval template for an issue.
// This is a utility function that can be called synchronously (from issue creation)
// or asynchronously (from the event handler).
func FindAndApplyApprovalTemplate(ctx context.Context, stores *store.Store, webhookManager *webhook.Manager, licenseService *enterprise.LicenseService, issue *store.IssueMessage) error {
	approvalSetting, err := stores.GetWorkspaceApprovalSetting(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to get workspace approval setting")
	}

	// Find approval template - errors are logged, not persisted
	err = findApprovalTemplateForIssue(ctx, stores, webhookManager, licenseService, issue, approvalSetting)
	if err != nil {
		return errors.Wrap(err, "failed to find approval template")
	}
	return nil
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

	if err := findApprovalTemplateForIssue(ctx, r.store, r.webhookManager, r.licenseService, issue, approvalSetting); err != nil {
		slog.Error("failed to find approval template",
			slog.Int64("issue_uid", issueUID),
			slog.String("issue_title", issue.Title),
			log.BBError(err))
		// Don't persist error - user can rerun plan check to retry
	}
}

func findApprovalTemplateForIssue(ctx context.Context, stores *store.Store, webhookManager *webhook.Manager, licenseService *enterprise.LicenseService, issue *store.IssueMessage, approvalSetting *storepb.WorkspaceApprovalSetting) error {
	payload := issue.Payload
	if payload.Approval != nil && payload.Approval.ApprovalFindingDone {
		return nil
	}

	project, err := stores.GetProject(ctx, &store.FindProjectMessage{ResourceID: &issue.ProjectID})
	if err != nil {
		return errors.Wrap(err, "failed to get project")
	}
	if project == nil {
		return errors.Errorf("project %s not found", issue.ProjectID)
	}

	approvalTemplate, celVarsList, done, err := func() (*storepb.ApprovalTemplate, []map[string]any, bool, error) {
		// no need to find if feature is not enabled
		if licenseService.IsFeatureEnabled(v1pb.PlanFeature_FEATURE_APPROVAL_WORKFLOW) != nil {
			// nolint:nilerr
			return nil, nil, true, nil
		}

		// Step 1: Determine approval source from issue type
		approvalSource, err := getApprovalSourceFromIssue(ctx, stores, issue)
		if err != nil {
			return nil, nil, false, errors.Wrap(err, "failed to get approval source from issue")
		}
		if approvalSource == storepb.WorkspaceApprovalSetting_Rule_SOURCE_UNSPECIFIED {
			// Cannot determine source, no approval needed
			return nil, nil, true, nil
		}

		// Step 2: Build CEL variables for evaluation
		celVarsList, done, err := buildCELVariablesForIssue(ctx, stores, issue)
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
		return err
	}
	if !done {
		return nil
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

	if _, err := stores.UpdateIssue(ctx, issue.UID, &store.UpdateIssueMessage{
		PayloadUpsert: &storepb.Issue{
			Approval:  payload.Approval,
			RiskLevel: riskLevel,
		},
	}); err != nil {
		return errors.Wrap(err, "failed to update issue payload")
	}

	NotifyApprovalRequested(ctx, stores, webhookManager, issue, project)

	return nil
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
func buildCELVariablesForIssue(ctx context.Context, stores *store.Store, issue *store.IssueMessage) ([]map[string]any, bool, error) {
	switch issue.Type {
	case storepb.Issue_GRANT_REQUEST:
		return buildCELVariablesForGrantRequest(ctx, stores, issue)
	case storepb.Issue_DATABASE_CHANGE:
		return buildCELVariablesForDatabaseChange(ctx, stores, issue)
	case storepb.Issue_DATABASE_EXPORT:
		return buildCELVariablesForDataExport(ctx, stores, issue)
	default:
		return nil, false, errors.Errorf("unknown issue type %v", issue.Type)
	}
}

// specTarget represents a database target extracted from a spec.
type specTarget struct {
	database    *store.DatabaseMessage
	sheetSha256 string
}

// unfoldDatabaseTargets unfolds database groups and returns all database targets.
// If the targets list contains a single database group reference, it unfolds it to individual databases.
// Otherwise, it returns the targets as-is.
func unfoldDatabaseTargets(ctx context.Context, stores *store.Store, dbTargets []string, projectID string, allDatabases []*store.DatabaseMessage) ([]string, error) {
	// Check if this is a database group (single target)
	if len(dbTargets) == 1 {
		if _, databaseGroupID, err := common.GetProjectIDDatabaseGroupID(dbTargets[0]); err == nil {
			// This is a database group - unfold it
			databaseGroup, err := stores.GetDatabaseGroup(ctx, &store.FindDatabaseGroupMessage{
				ResourceID: &databaseGroupID,
				ProjectID:  &projectID,
			})
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get database group")
			}
			if databaseGroup == nil {
				return nil, errors.Errorf("database group %q not found", dbTargets[0])
			}

			matchedDatabases, err := utils.GetMatchedDatabasesInDatabaseGroup(ctx, databaseGroup, allDatabases)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get matched databases in database group %q", databaseGroupID)
			}

			// Replace dbTargets with unfolded databases
			var unfolded []string
			for _, db := range matchedDatabases {
				unfolded = append(unfolded, common.FormatDatabase(db.InstanceID, db.DatabaseName))
			}
			return unfolded, nil
		}
	}
	return dbTargets, nil
}

// unfoldSpecTargets unfolds database groups in specs and returns all database targets.
func unfoldSpecTargets(ctx context.Context, stores *store.Store, specs []*storepb.PlanConfig_Spec, projectID string) ([]specTarget, error) {
	// Batch fetch all databases for the project
	allDatabases, err := stores.ListDatabases(ctx, &store.FindDatabaseMessage{ProjectID: &projectID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list databases for project %q", projectID)
	}

	// Build a map for quick lookup: instanceID/databaseName -> DatabaseMessage
	databaseMap := make(map[string]*store.DatabaseMessage)
	for _, db := range allDatabases {
		key := common.FormatDatabase(db.InstanceID, db.DatabaseName)
		databaseMap[key] = db
	}

	var targets []specTarget

	for _, spec := range specs {
		switch config := spec.Config.(type) {
		case *storepb.PlanConfig_Spec_CreateDatabaseConfig:
			instanceID, err := common.GetInstanceID(config.CreateDatabaseConfig.Target)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to parse instance from target %q", config.CreateDatabaseConfig.Target)
			}
			// For CREATE_DATABASE, create a synthetic database message
			// since the database doesn't exist yet
			envID := config.CreateDatabaseConfig.Environment
			targets = append(targets, specTarget{
				database: &store.DatabaseMessage{
					InstanceID:             instanceID,
					DatabaseName:           config.CreateDatabaseConfig.Database,
					EffectiveEnvironmentID: &envID,
				},
				sheetSha256: "",
			})

		case *storepb.PlanConfig_Spec_ChangeDatabaseConfig:
			dbTargets, err := unfoldDatabaseTargets(ctx, stores, config.ChangeDatabaseConfig.Targets, projectID, allDatabases)
			if err != nil {
				return nil, err
			}

			for _, target := range dbTargets {
				db := databaseMap[target]
				if db == nil {
					return nil, errors.Errorf("database %q not found", target)
				}
				targets = append(targets, specTarget{
					database:    db,
					sheetSha256: config.ChangeDatabaseConfig.SheetSha256,
				})
			}

		case *storepb.PlanConfig_Spec_ExportDataConfig:
			dbTargets, err := unfoldDatabaseTargets(ctx, stores, config.ExportDataConfig.Targets, projectID, allDatabases)
			if err != nil {
				return nil, err
			}

			for _, target := range dbTargets {
				db := databaseMap[target]
				if db == nil {
					return nil, errors.Errorf("database %q not found", target)
				}
				targets = append(targets, specTarget{
					database:    db,
					sheetSha256: config.ExportDataConfig.SheetSha256,
				})
			}
		}
	}

	return targets, nil
}

// buildCELVariablesForDatabaseChange builds CEL variables for DATABASE_CHANGE issues.
// This includes DDL and DML operations.
func buildCELVariablesForDatabaseChange(ctx context.Context, stores *store.Store, issue *store.IssueMessage) ([]map[string]any, bool, error) {
	if issue.PlanUID == nil {
		return nil, false, errors.Errorf("expected plan UID in issue %v", issue.UID)
	}
	plan, err := stores.GetPlan(ctx, &store.FindPlanMessage{UID: issue.PlanUID})
	if err != nil {
		return nil, false, errors.Wrapf(err, "failed to get plan %v", *issue.PlanUID)
	}
	if plan == nil {
		return nil, false, errors.Errorf("plan %v not found", *issue.PlanUID)
	}

	planCheckRun, err := stores.GetPlanCheckRun(ctx, plan.UID)
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

	// Unfold database groups and get all targets
	targets, err := unfoldSpecTargets(ctx, stores, plan.Config.GetSpecs(), issue.ProjectID)
	if err != nil {
		return nil, false, errors.Wrap(err, "failed to unfold spec targets")
	}

	var celVarsList []map[string]any
	for _, target := range targets {
		taskStatement := ""
		if target.sheetSha256 != "" {
			sheet, err := stores.GetSheetFull(ctx, target.sheetSha256)
			if err != nil {
				return nil, true, errors.Wrapf(err, "failed to get statement in sheet %v", target.sheetSha256)
			}
			if sheet == nil {
				return nil, true, errors.Errorf("sheet %v not found", target.sheetSha256)
			}
			taskStatement = sheet.Statement
		}

		environmentID := ""
		if target.database.EffectiveEnvironmentID != nil {
			environmentID = *target.database.EffectiveEnvironmentID
		}

		// Base CEL variables
		celVars := map[string]any{
			common.CELAttributeResourceEnvironmentID: environmentID,
			common.CELAttributeResourceProjectID:     issue.ProjectID,
			common.CELAttributeResourceInstanceID:    target.database.InstanceID,
			common.CELAttributeResourceDatabaseName:  target.database.DatabaseName,
			common.CELAttributeResourceDBEngine:      target.database.Engine.String(),
			common.CELAttributeStatementText:         taskStatement,
		}

		// Add summary report data if available
		result, ok := latestPlanCheckRun[Key{
			InstanceID:   target.database.InstanceID,
			DatabaseName: target.database.DatabaseName,
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
func buildCELVariablesForDataExport(ctx context.Context, stores *store.Store, issue *store.IssueMessage) ([]map[string]any, bool, error) {
	if issue.PlanUID == nil {
		return nil, false, errors.Errorf("expected plan UID in issue %v", issue.UID)
	}
	plan, err := stores.GetPlan(ctx, &store.FindPlanMessage{UID: issue.PlanUID})
	if err != nil {
		return nil, false, errors.Wrapf(err, "failed to get plan %v", *issue.PlanUID)
	}
	if plan == nil {
		return nil, false, errors.Errorf("plan %v not found", *issue.PlanUID)
	}

	// Unfold database groups and get all targets (only EXPORT_DATA targets)
	targets, err := unfoldSpecTargets(ctx, stores, plan.Config.GetSpecs(), issue.ProjectID)
	if err != nil {
		return nil, false, errors.Wrap(err, "failed to unfold spec targets")
	}

	var celVarsList []map[string]any
	for _, target := range targets {
		envID := ""
		if target.database.EffectiveEnvironmentID != nil {
			envID = *target.database.EffectiveEnvironmentID
		}

		celVars := map[string]any{
			common.CELAttributeResourceEnvironmentID: envID,
			common.CELAttributeResourceProjectID:     issue.ProjectID,
			common.CELAttributeResourceInstanceID:    target.database.InstanceID,
			common.CELAttributeResourceDatabaseName:  target.database.DatabaseName,
			common.CELAttributeResourceDBEngine:      target.database.Engine.String(),
		}
		celVarsList = append(celVarsList, celVars)
	}

	if len(celVarsList) == 0 {
		celVarsList = append(celVarsList, map[string]any{})
	}

	return celVarsList, true, nil
}

// buildCELVariablesForGrantRequest builds CEL variables for GRANT_REQUEST issues.
func buildCELVariablesForGrantRequest(ctx context.Context, stores *store.Store, issue *store.IssueMessage) ([]map[string]any, bool, error) {
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
		environments, err := stores.GetEnvironment(ctx)
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
	databases, err := getDatabasesForGrantRequest(ctx, stores, issue.ProjectID, factors.Databases)
	if err != nil {
		return nil, false, errors.Wrap(err, "failed to retrieve databases for grant request")
	}

	var celVarsList []map[string]any
	for _, database := range databases {
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

// getDatabasesForGrantRequest fetches database messages for the given database identifiers.
// Used exclusively by grant request approval flow to retrieve database information.
// Returns a deduplicated list of databases.
func getDatabasesForGrantRequest(ctx context.Context, stores *store.Store, projectID string, databaseIdentifiers []string) ([]*store.DatabaseMessage, error) {
	// Parse and deduplicate database identifiers
	type dbKey struct {
		instanceID   string
		databaseName string
	}
	requestedDBs := make(map[dbKey]bool)
	for _, identifier := range databaseIdentifiers {
		instanceID, databaseName, err := common.GetInstanceDatabaseID(identifier)
		if err != nil {
			return nil, err
		}
		requestedDBs[dbKey{instanceID: instanceID, databaseName: databaseName}] = true
	}

	// Batch fetch all databases in the project
	allDatabases, err := stores.ListDatabases(ctx, &store.FindDatabaseMessage{ProjectID: &projectID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list databases for project %q", projectID)
	}

	// Filter to only requested databases
	var result []*store.DatabaseMessage
	for _, db := range allDatabases {
		key := dbKey{instanceID: db.InstanceID, databaseName: db.DatabaseName}
		if requestedDBs[key] {
			result = append(result, db)
		}
	}

	return result, nil
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
func getApprovalSourceFromIssue(ctx context.Context, stores *store.Store, issue *store.IssueMessage) (storepb.WorkspaceApprovalSetting_Rule_Source, error) {
	switch issue.Type {
	case storepb.Issue_GRANT_REQUEST:
		return storepb.WorkspaceApprovalSetting_Rule_REQUEST_ROLE, nil
	case storepb.Issue_DATABASE_CHANGE:
		if issue.PlanUID == nil {
			return storepb.WorkspaceApprovalSetting_Rule_SOURCE_UNSPECIFIED, errors.Errorf("expected plan UID in issue %v", issue.UID)
		}
		plan, err := stores.GetPlan(ctx, &store.FindPlanMessage{UID: issue.PlanUID})
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
func getApproversForRole(ctx context.Context, stores *store.Store, projectID string, role string) ([]webhook.User, error) {
	// Get project IAM policy
	projectIAM, err := stores.GetProjectIamPolicy(ctx, projectID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get project IAM policy")
	}

	// Get workspace IAM policy
	workspaceIAM, err := stores.GetWorkspaceIamPolicy(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get workspace IAM policy")
	}

	// Get all users with the specified role
	users := utils.GetUsersByRoleInIAMPolicy(ctx, stores, role, projectIAM.Policy, workspaceIAM.Policy)

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

// NotifyApprovalRequested sends the ISSUE_APPROVAL_REQUESTED webhook event for the next pending approval stage.
// It finds the next pending role, retrieves approvers for that role, and triggers the webhook.
// This should be called after:
// - Creating an issue with an approval template
// - Approving an issue stage (to notify next stage approvers)
// - Sending back an issue (to notify the current stage approvers again)
func NotifyApprovalRequested(ctx context.Context, stores *store.Store, webhookManager *webhook.Manager, issue *store.IssueMessage, project *store.ProjectMessage) {
	role := utils.FindNextPendingRole(issue.Payload.Approval)
	if role == "" {
		return
	}

	// Get issue creator as actor
	creator, err := stores.GetUserByEmail(ctx, issue.CreatorEmail)
	if err != nil {
		slog.Warn("failed to get issue creator, using system bot", log.BBError(err))
		creator = store.SystemBotUser
	}

	// Get approvers for this role
	approvers, err := getApproversForRole(ctx, stores, issue.ProjectID, role)
	if err != nil {
		slog.Warn("failed to get approvers", log.BBError(err))
		approvers = []webhook.User{} // Continue with empty list
	}

	// Trigger ISSUE_APPROVAL_REQUESTED webhook
	webhookManager.CreateEvent(ctx, &webhook.Event{
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
}
