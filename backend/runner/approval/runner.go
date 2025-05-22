// Package approval is the runner for finding approval templates for issues.
package approval

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/google/cel-go/cel"
	"github.com/pkg/errors"
	"go.uber.org/multierr"

	celtypes "github.com/google/cel-go/common/types"

	apiv1 "github.com/bytebase/bytebase/backend/api/v1"
	"github.com/bytebase/bytebase/backend/base"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/sheet"
	"github.com/bytebase/bytebase/backend/component/state"
	"github.com/bytebase/bytebase/backend/component/webhook"
	enterprise "github.com/bytebase/bytebase/backend/enterprise/api"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// Runner is the runner for finding approval templates for issues.
type Runner struct {
	store          *store.Store
	sheetManager   *sheet.Manager
	dbFactory      *dbfactory.DBFactory
	stateCfg       *state.State
	webhookManager *webhook.Manager
	licenseService enterprise.LicenseService
}

// NewRunner creates a new runner.
func NewRunner(store *store.Store, sheetManager *sheet.Manager, dbFactory *dbfactory.DBFactory, stateCfg *state.State, webhookManager *webhook.Manager, licenseService enterprise.LicenseService) *Runner {
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
	risks, err := r.store.ListRisks(ctx)
	if err != nil {
		slog.Error("failed to list risks", log.BBError(err))
		return
	}
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
		done, err := r.findApprovalTemplateForIssue(ctx, issue, risks, approvalSetting)
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
		StatusList: []base.IssueStatus{base.IssueOpen},
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

func (r *Runner) findApprovalTemplateForIssue(ctx context.Context, issue *store.IssueMessage, risks []*store.RiskMessage, approvalSetting *storepb.WorkspaceApprovalSetting) (bool, error) {
	payload := issue.Payload
	if payload.Approval != nil && payload.Approval.ApprovalFindingDone {
		return true, nil
	}

	approvalTemplate, riskLevel, done, err := func() (*storepb.ApprovalTemplate, storepb.IssuePayloadApproval_RiskLevel, bool, error) {
		// no need to find if
		// - feature is not enabled
		// - approval setting rules are empty
		if r.licenseService.IsFeatureEnabled(base.FeatureCustomApproval) != nil || len(approvalSetting.Rules) == 0 {
			// nolint:nilerr
			return nil, 0, true, nil
		}

		riskLevel, riskSource, done, err := r.getIssueRisk(ctx, issue, risks)
		if err != nil {
			err = errors.Wrap(err, "failed to get issue risk level")
			return nil, 0, false, err
		}
		if !done {
			return nil, 0, false, nil
		}

		approvalTemplate, err := getApprovalTemplate(approvalSetting, riskLevel, riskSource)
		if err != nil {
			return nil, 0, false, errors.Wrapf(err, "failed to get approval template, riskLevel: %v", riskLevel)
		}

		riskLevelEnum, err := convertRiskLevel(riskLevel)
		if err != nil {
			return nil, 0, false, errors.Wrap(err, "failed to convert risk level")
		}
		return approvalTemplate, riskLevelEnum, true, nil
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
	if issue.Type == base.IssueGrantRequest && approvalTemplate == nil {
		if err := utils.UpdateProjectPolicyFromGrantIssue(ctx, r.store, issue, payload.GrantRequest); err != nil {
			return false, err
		}
		if err := webhook.ChangeIssueStatus(ctx, r.store, r.webhookManager, issue, base.IssueDone, r.store.GetSystemBotUser(ctx), ""); err != nil {
			return false, errors.Wrap(err, "failed to update issue status")
		}
	}

	payload.Approval = &storepb.IssuePayloadApproval{
		ApprovalFindingDone: true,
		RiskLevel:           riskLevel,
		ApprovalTemplates:   nil,
		Approvers:           nil,
	}
	if approvalTemplate != nil {
		payload.Approval.ApprovalTemplates = []*storepb.ApprovalTemplate{approvalTemplate}
	}

	newApprovers, err := utils.HandleIncomingApprovalSteps(payload.Approval)
	if err != nil {
		err = errors.Wrapf(err, "failed to handle incoming approval steps")
		if updateErr := updateIssueApprovalPayload(ctx, r.store, issue, &storepb.IssuePayloadApproval{
			ApprovalFindingDone:  true,
			ApprovalFindingError: err.Error(),
		}); updateErr != nil {
			return false, multierr.Append(errors.Wrap(updateErr, "failed to update issue payload"), err)
		}
		return false, err
	}
	payload.Approval.Approvers = append(payload.Approval.Approvers, newApprovers...)

	if err := updateIssueApprovalPayload(ctx, r.store, issue, payload.Approval); err != nil {
		return false, errors.Wrap(err, "failed to update issue payload")
	}

	if err := func() error {
		if len(payload.Approval.ApprovalTemplates) != 0 {
			return nil
		}
		if issue.PipelineUID == nil {
			return nil
		}
		stages, err := r.store.ListStageV2(ctx, *issue.PipelineUID)
		if err != nil {
			return errors.Wrapf(err, "failed to list stages")
		}
		if len(stages) == 0 {
			return nil
		}
		policy, err := apiv1.GetValidRolloutPolicyForStage(ctx, r.store, stages[0])
		if err != nil {
			return err
		}
		r.webhookManager.CreateEvent(ctx, &webhook.Event{
			Actor:   r.store.GetSystemBotUser(ctx),
			Type:    webhook.EventTypeIssueRolloutReady,
			Comment: "",
			Issue:   webhook.NewIssue(issue),
			Project: webhook.NewProject(issue.Project),
			IssueRolloutReady: &webhook.EventIssueRolloutReady{
				RolloutPolicy: policy,
				StageName:     stages[0].Environment,
			},
		})
		return nil
	}(); err != nil {
		slog.Error("failed to create rollout release notification activity", log.BBError(err))
	}

	if err := func() error {
		if len(payload.Approval.ApprovalTemplates) != 1 {
			return nil
		}
		approvalStep := utils.FindNextPendingStep(payload.Approval.ApprovalTemplates[0], payload.Approval.Approvers)
		if approvalStep == nil {
			return nil
		}
		r.webhookManager.CreateEvent(ctx, &webhook.Event{
			Actor:   r.store.GetSystemBotUser(ctx),
			Type:    webhook.EventTypeIssueApprovalCreate,
			Comment: "",
			Issue:   webhook.NewIssue(issue),
			Project: webhook.NewProject(issue.Project),
			IssueApprovalCreate: &webhook.EventIssueApprovalCreate{
				ApprovalStep: approvalStep,
			},
		})
		return nil
	}(); err != nil {
		slog.Error("failed to create approval step pending activity after creating review", log.BBError(err))
	}

	return true, nil
}

func getApprovalTemplate(approvalSetting *storepb.WorkspaceApprovalSetting, riskLevel int32, riskSource store.RiskSource) (*storepb.ApprovalTemplate, error) {
	e, err := cel.NewEnv(common.ApprovalFactors...)
	if err != nil {
		return nil, err
	}
	for _, rule := range approvalSetting.Rules {
		if rule.Condition == nil || rule.Condition.Expression == "" {
			continue
		}
		ast, issues := e.Compile(rule.Condition.Expression)
		if issues != nil && issues.Err() != nil {
			return nil, issues.Err()
		}

		prg, err := e.Program(ast)
		if err != nil {
			return nil, errors.Wrap(err, "failed to compile expression")
		}

		out, _, err := prg.Eval(map[string]any{
			"level":  riskLevel,
			"source": apiv1.ConvertToV1Source(riskSource).String(),
		})
		if err != nil {
			return nil, err
		}
		if res, ok := out.Equal(celtypes.True).Value().(bool); ok && res {
			return rule.Template, nil
		}
	}
	return nil, nil
}

func (r *Runner) getIssueRisk(ctx context.Context, issue *store.IssueMessage, risks []*store.RiskMessage) (int32, store.RiskSource, bool, error) {
	// sort by level DESC, higher risks go first.
	sort.Slice(risks, func(i, j int) bool {
		return risks[i].Level > risks[j].Level
	})

	switch issue.Type {
	case base.IssueGrantRequest:
		return r.getGrantRequestIssueRisk(ctx, issue, risks)
	case base.IssueDatabaseGeneral:
		return r.getDatabaseGeneralIssueRisk(ctx, issue, risks)
	case base.IssueDatabaseDataExport:
		return r.getDatabaseDataExportIssueRisk(ctx, issue, risks)
	default:
		return 0, store.RiskSourceUnknown, false, errors.Errorf("unknown issue type %v", issue.Type)
	}
}

func (r *Runner) getDatabaseGeneralIssueRisk(ctx context.Context, issue *store.IssueMessage, risks []*store.RiskMessage) (int32, store.RiskSource, bool, error) {
	if issue.PlanUID == nil {
		return 0, store.RiskSourceUnknown, false, errors.Errorf("expected plan UID in issue %v", issue.UID)
	}
	plan, err := r.store.GetPlan(ctx, &store.FindPlanMessage{UID: issue.PlanUID})
	if err != nil {
		return 0, store.RiskSourceUnknown, false, errors.Wrapf(err, "failed to get plan %v", *issue.PlanUID)
	}
	if plan == nil {
		return 0, store.RiskSourceUnknown, false, errors.Errorf("plan %v not found", *issue.PlanUID)
	}

	// Conclude risk source from task types.
	riskSource := getRiskSourceFromPlan(plan.Config)
	// Cannot conclude risk source.
	if riskSource == store.RiskSourceUnknown {
		return 0, store.RiskSourceUnknown, true, nil
	}

	planCheckRuns, err := r.store.ListPlanCheckRuns(ctx, &store.FindPlanCheckRunMessage{
		PlanUID: &plan.UID,
		Type:    &[]store.PlanCheckRunType{store.PlanCheckDatabaseStatementSummaryReport},
	})
	if err != nil {
		return 0, store.RiskSourceUnknown, false, errors.Wrapf(err, "failed to list plan check runs for plan %v", plan.UID)
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

	// Get the max risk level of the same risk source.
	// risks is sorted by level DESC, so we just need to return the 1st matched risk.
	var maxRiskSourceRiskLevel int32
	for _, risk := range risks {
		if !risk.Active {
			continue
		}
		if risk.Source != riskSource {
			continue
		}
		maxRiskSourceRiskLevel = risk.Level
		break
	}

	// If any plan check run is skipped because of large SQL,
	// return the max risk level in the risks of the same risk source.
	for _, run := range latestPlanCheckRun {
		for _, result := range run.Result.GetResults() {
			if result.GetCode() == common.SizeExceeded.Int32() {
				return maxRiskSourceRiskLevel, riskSource, true, nil
			}
		}
	}

	var planCheckRunDone int
	// the latest plan check run is not done yet, return done=false
	for _, run := range latestPlanCheckRun {
		if run.Status == store.PlanCheckRunStatusDone {
			planCheckRunDone++
		}
	}
	planCheckRunCount := len(latestPlanCheckRun)
	// We have less than 5 planCheckRuns in total.
	// We wait for all of them to finish.
	if planCheckRunCount < common.MinimumCompletedPlanCheckRun && planCheckRunCount != planCheckRunDone {
		return 0, store.RiskSourceUnknown, false, nil
	}
	// We have not less than 5 planCheckRuns in total.
	// We need at least 5 completed plan check run.
	if planCheckRunCount >= common.MinimumCompletedPlanCheckRun && planCheckRunDone < common.MinimumCompletedPlanCheckRun {
		return 0, store.RiskSourceUnknown, false, nil
	}

	pipelineCreate, err := apiv1.GetPipelineCreate(ctx, r.store, r.sheetManager, r.licenseService, r.dbFactory, plan.Name, plan.Config.GetSteps(), plan.Config.GetDeployment(), issue.Project)
	if err != nil {
		return 0, store.RiskSourceUnknown, false, errors.Wrap(err, "failed to get pipeline create")
	}

	var maxRiskLevel int32
	for _, stage := range pipelineCreate.Stages {
		for _, task := range stage.TaskList {
			instance, err := r.store.GetInstanceV2(ctx, &store.FindInstanceMessage{
				ResourceID: &task.InstanceID,
			})
			if err != nil {
				return 0, store.RiskSourceUnknown, false, errors.Wrapf(err, "failed to get instance %v", task.InstanceID)
			}
			if instance.Deleted {
				continue
			}
			if r.licenseService.IsFeatureEnabledForInstance(base.FeatureCustomApproval, instance) != nil {
				// nolint:nilerr
				return 0, store.RiskSourceUnknown, true, nil
			}

			taskStatement := ""
			sheetUID := int(task.Payload.GetSheetId())
			if sheetUID != 0 {
				statement, err := r.store.GetSheetStatementByID(ctx, sheetUID)
				if err != nil {
					return 0, store.RiskSourceUnknown, true, errors.Wrapf(err, "failed to get statement in sheet %v", sheetUID)
				}
				taskStatement = statement
			}

			var environmentID string
			var databaseName string
			if task.Type == base.TaskDatabaseCreate {
				databaseName = task.Payload.GetDatabaseName()
				environmentID = task.Payload.GetEnvironmentId()
			} else {
				database, err := r.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
					InstanceID:   &task.InstanceID,
					DatabaseName: task.DatabaseName,
				})
				if err != nil {
					return 0, store.RiskSourceUnknown, false, err
				}
				databaseName = database.DatabaseName
				environmentID = database.EffectiveEnvironmentID
			}

			commonArgs := map[string]any{
				"environment_id": environmentID,
				"project_id":     issue.Project.ResourceID,
				"database_name":  databaseName,
				// convert to string type otherwise cel-go will complain that storepb.Engine is not string type.
				"db_engine":     instance.Metadata.GetEngine().String(),
				"sql_statement": taskStatement,
			}
			risk, err := func() (int32, error) {
				if run, ok := latestPlanCheckRun[Key{
					InstanceID:   instance.ResourceID,
					DatabaseName: databaseName,
				}]; ok {
					for _, result := range run.Result.Results {
						report := result.GetSqlSummaryReport()
						if report == nil {
							continue
						}
						riskLevel, err := apiv1.CalculateRiskLevelWithSummaryReport(ctx, risks, commonArgs, riskSource, report)
						if err != nil {
							return 0, err
						}
						if riskLevel == 0 {
							continue
						}
						return riskLevel, nil
					}
				}
				return 0, nil
			}()
			if err != nil {
				return 0, store.RiskSourceUnknown, false, errors.Wrapf(err, "failed to evaluate risk expression for risk source %v", riskSource)
			}

			if maxRiskLevel < risk {
				maxRiskLevel = risk
			}
			if maxRiskLevel == maxRiskSourceRiskLevel {
				return maxRiskLevel, riskSource, true, nil
			}
		}
	}

	return maxRiskLevel, riskSource, true, nil
}

func (r *Runner) getDatabaseDataExportIssueRisk(ctx context.Context, issue *store.IssueMessage, risks []*store.RiskMessage) (int32, store.RiskSource, bool, error) {
	if issue.PlanUID == nil {
		return 0, store.RiskSourceUnknown, false, errors.Errorf("expected plan UID in issue %v", issue.UID)
	}
	plan, err := r.store.GetPlan(ctx, &store.FindPlanMessage{UID: issue.PlanUID})
	if err != nil {
		return 0, store.RiskSourceUnknown, false, errors.Wrapf(err, "failed to get plan %v", *issue.PlanUID)
	}
	if plan == nil {
		return 0, store.RiskSourceUnknown, false, errors.Errorf("plan %v not found", *issue.PlanUID)
	}

	pipelineCreate, err := apiv1.GetPipelineCreate(ctx, r.store, r.sheetManager, r.licenseService, r.dbFactory, plan.Name, plan.Config.GetSteps(), plan.Config.GetDeployment(), issue.Project)
	if err != nil {
		return 0, store.RiskSourceUnknown, false, errors.Wrap(err, "failed to get pipeline create")
	}

	riskSource := store.RiskSourceDatabaseDataExport

	e, err := cel.NewEnv(common.RiskFactors...)
	if err != nil {
		return 0, store.RiskSourceUnknown, false, err
	}

	var maxRiskLevel int32
	for _, stage := range pipelineCreate.Stages {
		for _, task := range stage.TaskList {
			if task.Type != base.TaskDatabaseDataExport {
				continue
			}
			instance, err := r.store.GetInstanceV2(ctx, &store.FindInstanceMessage{
				ResourceID: &task.InstanceID,
			})
			if err != nil {
				return 0, store.RiskSourceUnknown, false, errors.Wrapf(err, "failed to get instance %v", task.InstanceID)
			}
			if instance.Deleted {
				continue
			}
			if r.licenseService.IsFeatureEnabledForInstance(base.FeatureCustomApproval, instance) != nil {
				// nolint:nilerr
				return 0, store.RiskSourceUnknown, true, nil
			}

			database, err := r.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
				InstanceID:   &task.InstanceID,
				DatabaseName: task.DatabaseName,
			})
			if err != nil {
				return 0, store.RiskSourceUnknown, false, err
			}
			databaseName := database.DatabaseName
			environmentID := database.EffectiveEnvironmentID

			risk, err := func() (int32, error) {
				for _, risk := range risks {
					if !risk.Active {
						continue
					}
					if risk.Source != riskSource {
						continue
					}
					if risk.Expression == nil || risk.Expression.Expression == "" {
						continue
					}
					ast, issues := e.Parse(risk.Expression.Expression)
					if issues != nil && issues.Err() != nil {
						return 0, errors.Errorf("failed to parse expression: %v", issues.Err())
					}
					prg, err := e.Program(ast, cel.EvalOptions(cel.OptPartialEval))
					if err != nil {
						return 0, err
					}
					args := map[string]any{
						"environment_id": environmentID,
						"project_id":     issue.Project.ResourceID,
						"database_name":  databaseName,
						"db_engine":      instance.Metadata.GetEngine().String(),
					}

					vars, err := e.PartialVars(args)
					if err != nil {
						return 0, errors.Wrapf(err, "failed to get vars")
					}
					out, _, err := prg.Eval(vars)
					if err != nil {
						return 0, errors.Wrapf(err, "failed to eval expression")
					}
					if res, ok := out.Equal(celtypes.True).Value().(bool); ok && res {
						return risk.Level, nil
					}
				}
				return 0, nil
			}()
			if err != nil {
				return 0, store.RiskSourceUnknown, false, errors.Wrapf(err, "failed to evaluate risk expression for risk source %v", riskSource)
			}

			if maxRiskLevel < risk {
				maxRiskLevel = risk
			}
			if level, _ := convertRiskLevel(maxRiskLevel); level == storepb.IssuePayloadApproval_HIGH {
				return maxRiskLevel, riskSource, true, nil
			}
		}
	}

	return maxRiskLevel, riskSource, true, nil
}

func (r *Runner) getDatabaseListByRequestRole(ctx context.Context, projectID string, databaseNames []string) ([]*store.DatabaseMessage, error) {
	if len(databaseNames) == 0 {
		databases, err := r.store.ListDatabases(ctx, &store.FindDatabaseMessage{
			ProjectID: &projectID,
		})
		if err != nil {
			return nil, errors.Wrap(err, "failed to list databases")
		}
		return databases, nil
	}

	var databases []*store.DatabaseMessage
	for _, dbName := range databaseNames {
		instanceID, databaseName, err := common.GetInstanceDatabaseID(dbName)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get instance database id")
		}

		instance, err := r.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &instanceID})
		if err != nil {
			return nil, errors.Wrap(err, "failed to get instance")
		}
		if instance == nil || instance.Deleted {
			continue
		}
		database, err := r.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
			ProjectID:       &projectID,
			InstanceID:      &instanceID,
			DatabaseName:    &databaseName,
			IsCaseSensitive: store.IsObjectCaseSensitive(instance),
		})
		if err != nil {
			return nil, errors.Wrap(err, "failed to get database")
		}
		if database == nil {
			return nil, errors.Errorf("database %q not found", databaseName)
		}
		databases = append(databases, database)
	}
	return databases, nil
}

func (r *Runner) getGrantRequestIssueRisk(ctx context.Context, issue *store.IssueMessage, risks []*store.RiskMessage) (int32, store.RiskSource, bool, error) {
	payload := issue.Payload
	if payload.GrantRequest == nil {
		return 0, store.RiskSourceUnknown, false, errors.New("grant request payload not found")
	}

	// fast path, no risks so return the DEFAULT risk level "0"
	if len(risks) == 0 {
		return 0, store.RiskRequestRole, true, nil
	}

	e, err := cel.NewEnv(common.RiskFactors...)
	if err != nil {
		return 0, store.RiskSourceUnknown, false, err
	}

	factors, err := common.GetQueryExportFactors(payload.GetGrantRequest().GetCondition().GetExpression())
	if err != nil {
		return 0, store.RiskSourceUnknown, false, errors.Wrap(err, "failed to get query export factors")
	}
	// Default to max float64 if expiration is not set. AKA no expiration.
	expirationDays := math.MaxFloat64
	if payload.GrantRequest.Expiration != nil {
		expirationDays = payload.GrantRequest.Expiration.AsDuration().Hours() / 24
	}
	databases, err := r.getDatabaseListByRequestRole(ctx, issue.Project.ResourceID, factors.DatabaseNames)
	if err != nil {
		return 0, store.RiskSourceUnknown, false, errors.Wrap(err, "failed to list databases")
	}

	var maxRisk int32
	for _, risk := range risks {
		if !risk.Active {
			continue
		}
		if risk.Source != store.RiskRequestRole {
			continue
		}
		if risk.Expression == nil || risk.Expression.Expression == "" {
			continue
		}

		ast, issues := e.Parse(risk.Expression.Expression)
		if issues != nil && issues.Err() != nil {
			return 0, store.RiskSourceUnknown, false, errors.Errorf("failed to parse expression: %v", issues.Err())
		}
		prg, err := e.Program(ast)
		if err != nil {
			return 0, store.RiskSourceUnknown, false, err
		}

		if len(databases) == 0 {
			args := map[string]any{
				"project_id":      issue.Project.ResourceID,
				"expiration_days": expirationDays,
				"role":            payload.GrantRequest.Role,
			}
			out, _, err := prg.Eval(args)
			if err != nil {
				return 0, store.RiskSourceUnknown, false, err
			}
			if res, ok := out.Equal(celtypes.True).Value().(bool); ok && res {
				maxRisk = risk.Level
			}
			if maxRisk > 0 {
				break
			}
			continue
		}

		for _, database := range databases {
			instance, err := r.store.GetInstanceV2(ctx, &store.FindInstanceMessage{
				ResourceID: &database.InstanceID,
			})
			if err != nil {
				return 0, store.RiskSourceUnknown, false, errors.Wrap(err, "failed to get instance")
			}
			if instance == nil || instance.Deleted {
				continue
			}
			if r.licenseService.IsFeatureEnabledForInstance(base.FeatureCustomApproval, instance) != nil {
				// nolint:nilerr
				return 0, store.RiskSourceUnknown, true, nil
			}

			args := map[string]any{
				"environment_id": database.EffectiveEnvironmentID,
				"project_id":     issue.Project.ResourceID,
				"database_name":  database.DatabaseName,
				// convert to string type otherwise cel-go will complain that storepb.Engine is not string type.
				"db_engine":       instance.Metadata.GetEngine().String(),
				"expiration_days": expirationDays,
				"export_rows":     factors.ExportRows,
				"role":            payload.GrantRequest.Role,
			}
			out, _, err := prg.Eval(args)
			if err != nil {
				return 0, store.RiskSourceUnknown, false, err
			}
			if res, ok := out.Equal(celtypes.True).Value().(bool); ok && res {
				maxRisk = risk.Level
			}
			if maxRisk > 0 {
				break
			}
		}

		// We can stop the loop because the risk list is sorted by level DESC.
		if maxRisk > 0 {
			break
		}
	}

	return maxRisk, store.RiskRequestRole, true, nil
}

func getRiskSourceFromPlan(config *storepb.PlanConfig) store.RiskSource {
	for _, step := range config.GetSteps() {
		for _, spec := range step.GetSpecs() {
			switch v := spec.Config.(type) {
			case *storepb.PlanConfig_Spec_CreateDatabaseConfig:
				return store.RiskSourceDatabaseCreate
			case *storepb.PlanConfig_Spec_ChangeDatabaseConfig:
				switch v.ChangeDatabaseConfig.Type {
				case storepb.PlanConfig_ChangeDatabaseConfig_MIGRATE, storepb.PlanConfig_ChangeDatabaseConfig_MIGRATE_GHOST, storepb.PlanConfig_ChangeDatabaseConfig_MIGRATE_SDL:
					return store.RiskSourceDatabaseSchemaUpdate
				case storepb.PlanConfig_ChangeDatabaseConfig_DATA:
					return store.RiskSourceDatabaseDataUpdate
				}
			}
		}
	}
	return store.RiskSourceUnknown
}

func updateIssueApprovalPayload(ctx context.Context, s *store.Store, issue *store.IssueMessage, approval *storepb.IssuePayloadApproval) error {
	if _, err := s.UpdateIssueV2(ctx, issue.UID, &store.UpdateIssueMessage{
		PayloadUpsert: &storepb.IssuePayload{
			Approval: approval,
		},
	}); err != nil {
		return errors.Wrap(err, "failed to update issue payload")
	}
	return nil
}

func convertRiskLevel(riskLevel int32) (storepb.IssuePayloadApproval_RiskLevel, error) {
	switch riskLevel {
	case 0:
		return storepb.IssuePayloadApproval_RISK_LEVEL_UNSPECIFIED, nil
	case 100:
		return storepb.IssuePayloadApproval_LOW, nil
	case 200:
		return storepb.IssuePayloadApproval_MODERATE, nil
	case 300:
		return storepb.IssuePayloadApproval_HIGH, nil
	}
	return storepb.IssuePayloadApproval_RISK_LEVEL_UNSPECIFIED, errors.Errorf("unexpected risk level %d", riskLevel)
}
