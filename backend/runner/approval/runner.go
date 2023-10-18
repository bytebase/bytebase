// Package approval is the runner for finding approval templates for issues.
package approval

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/cel-go/cel"
	"github.com/pkg/errors"
	"go.uber.org/multierr"
	"google.golang.org/protobuf/encoding/protojson"

	v1 "github.com/bytebase/bytebase/backend/api/v1"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/activity"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/state"
	enterpriseAPI "github.com/bytebase/bytebase/backend/enterprise/api"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/runner/relay"
	"github.com/bytebase/bytebase/backend/utils"

	"github.com/bytebase/bytebase/backend/store"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// Runner is the runner for finding approval templates for issues.
type Runner struct {
	store           *store.Store
	dbFactory       *dbfactory.DBFactory
	stateCfg        *state.State
	activityManager *activity.Manager
	relayRunner     *relay.Runner
	licenseService  enterpriseAPI.LicenseService
}

// NewRunner creates a new runner.
func NewRunner(store *store.Store, dbFactory *dbfactory.DBFactory, stateCfg *state.State, activityManager *activity.Manager, relayRunner *relay.Runner, licenseService enterpriseAPI.LicenseService) *Runner {
	return &Runner{
		store:           store,
		dbFactory:       dbFactory,
		stateCfg:        stateCfg,
		activityManager: activityManager,
		relayRunner:     relayRunner,
		licenseService:  licenseService,
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
			r.runOnce(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (r *Runner) runOnce(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			err, ok := r.(error)
			if !ok {
				err = errors.Errorf("%v", r)
			}
			slog.Error("Approval runner PANIC RECOVER", log.BBError(err), log.BBStack("panic-stack"))
		}
	}()

	if err := func() error {
		risks, err := r.store.ListRisks(ctx)
		if err != nil {
			return errors.Wrap(err, "failed to list risks")
		}
		approvalSetting, err := r.store.GetWorkspaceApprovalSetting(ctx)
		if err != nil {
			return errors.Wrap(err, "failed to get workspace approval setting")
		}

		var errs error
		r.stateCfg.ApprovalFinding.Range(func(key, value any) bool {
			issue := value.(*store.IssueMessage)
			done, err := r.findApprovalTemplateForIssue(ctx, issue, risks, approvalSetting)
			if err != nil {
				errs = multierr.Append(errs, errors.Wrapf(err, "failed to find approval template for issue %v", issue.UID))
			}
			if err != nil || done {
				r.stateCfg.ApprovalFinding.Delete(key)
			}
			return true
		})

		return errs
	}(); err != nil {
		slog.Error("approval runner", log.BBError(err))
	}
}

func (r *Runner) retryFindApprovalTemplate(ctx context.Context) {
	issues, err := r.store.ListIssueV2(ctx, &store.FindIssueMessage{
		StatusList: []api.IssueStatus{api.IssueOpen},
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

	approvalTemplate, done, err := func() (*storepb.ApprovalTemplate, bool, error) {
		// no need to find if
		// - feature is not enabled
		// - approval setting rules are empty
		if r.licenseService.IsFeatureEnabled(api.FeatureCustomApproval) != nil || len(approvalSetting.Rules) == 0 {
			// nolint:nilerr
			return nil, true, nil
		}

		riskLevel, riskSource, done, err := getIssueRisk(ctx, r.store, r.licenseService, r.dbFactory, issue, risks)
		if err != nil {
			err = errors.Wrap(err, "failed to get issue risk level")
			return nil, false, err
		}
		if !done {
			return nil, false, nil
		}

		approvalTemplate, err := getApprovalTemplate(approvalSetting, riskLevel, riskSource)
		if err != nil {
			err = errors.Wrapf(err, "failed to get approval template, riskLevel: %v", riskLevel)
		}

		return approvalTemplate, true, err
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
	if issue.Type == api.IssueGrantRequest && approvalTemplate == nil {
		if err := utils.UpdateProjectPolicyFromGrantIssue(ctx, r.store, issue, payload.GrantRequest); err != nil {
			return false, err
		}
		userID, err := strconv.Atoi(strings.TrimPrefix(payload.GrantRequest.User, "users/"))
		if err != nil {
			return false, err
		}
		newUser, err := r.store.GetUserByID(ctx, userID)
		if err != nil {
			return false, err
		}
		// Post project IAM policy update activity.
		if _, err := r.activityManager.CreateActivity(ctx, &store.ActivityMessage{
			CreatorUID:   api.SystemBotID,
			ContainerUID: issue.Project.UID,
			Type:         api.ActivityProjectMemberCreate,
			Level:        api.ActivityInfo,
			Comment:      fmt.Sprintf("Granted %s to %s (%s).", newUser.Name, newUser.Email, payload.GrantRequest.Role),
		}, &activity.Metadata{}); err != nil {
			slog.Warn("Failed to create project activity", log.BBError(err))
		}
		if err := utils.ChangeIssueStatus(ctx, r.store, r.activityManager, issue, api.IssueDone, api.SystemBotID, ""); err != nil {
			return false, errors.Wrap(err, "failed to update issue status")
		}
	}

	payload.Approval = &storepb.IssuePayloadApproval{
		ApprovalFindingDone: true,
		ApprovalTemplates:   nil,
		Approvers:           nil,
	}
	if approvalTemplate != nil {
		payload.Approval.ApprovalTemplates = []*storepb.ApprovalTemplate{approvalTemplate}
	}

	newApprovers, activityCreates, err := utils.HandleIncomingApprovalSteps(ctx, r.store, r.relayRunner.Client, issue, payload.Approval)
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

	// It's ok to fail to create activity.
	if err := func() error {
		for _, create := range activityCreates {
			if _, err := r.activityManager.CreateActivity(ctx, create, &activity.Metadata{}); err != nil {
				return err
			}
		}

		return nil
	}(); err != nil {
		slog.Error("failed to create activity after approving review", log.BBError(err))
	}

	if err := func() error {
		if len(payload.Approval.ApprovalTemplates) != 1 {
			return nil
		}
		approvalStep := utils.FindNextPendingStep(payload.Approval.ApprovalTemplates[0], payload.Approval.Approvers)
		if approvalStep == nil {
			return nil
		}
		protoPayload, err := protojson.Marshal(&storepb.ActivityIssueApprovalNotifyPayload{
			ApprovalStep: approvalStep,
		})
		if err != nil {
			return err
		}
		activityPayload, err := json.Marshal(api.ActivityIssueApprovalNotifyPayload{
			ProtoPayload: string(protoPayload),
		})
		if err != nil {
			return err
		}

		create := &store.ActivityMessage{
			CreatorUID:   api.SystemBotID,
			ContainerUID: issue.UID,
			Type:         api.ActivityIssueApprovalNotify,
			Level:        api.ActivityInfo,
			Comment:      "",
			Payload:      string(activityPayload),
		}
		if _, err := r.activityManager.CreateActivity(ctx, create, &activity.Metadata{Issue: issue}); err != nil {
			return err
		}

		return nil
	}(); err != nil {
		slog.Error("failed to create approval step pending activity after creating review", log.BBError(err))
	}

	return true, nil
}

func getApprovalTemplate(approvalSetting *storepb.WorkspaceApprovalSetting, riskLevel int64, riskSource store.RiskSource) (*storepb.ApprovalTemplate, error) {
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

		res, _, err := prg.Eval(map[string]any{
			"level":  riskLevel,
			"source": int64(convertToSource(riskSource)),
		})
		if err != nil {
			return nil, err
		}

		val, err := res.ConvertToNative(reflect.TypeOf(false))
		if err != nil {
			return nil, errors.Wrap(err, "expect bool result")
		}
		if boolVal, ok := val.(bool); ok && boolVal {
			return rule.Template, nil
		}
	}
	return nil, nil
}

func getIssueRisk(ctx context.Context, s *store.Store, licenseService enterpriseAPI.LicenseService, dbFactory *dbfactory.DBFactory, issue *store.IssueMessage, risks []*store.RiskMessage) (int64, store.RiskSource, bool, error) {
	switch issue.Type {
	case api.IssueGrantRequest:
		return getGrantRequestIssueRisk(ctx, s, issue, risks)
	case api.IssueDatabaseGeneral:
		return getDatabaseGeneralIssueRisk(ctx, s, licenseService, dbFactory, issue, risks)
	default:
		return 0, store.RiskSourceUnknown, false, errors.Errorf("unknown issue type %v", issue.Type)
	}
}

func getDatabaseGeneralIssueRisk(ctx context.Context, s *store.Store, licenseService enterpriseAPI.LicenseService, dbFactory *dbfactory.DBFactory, issue *store.IssueMessage, risks []*store.RiskMessage) (int64, store.RiskSource, bool, error) {
	if issue.PlanUID == nil {
		return 0, store.RiskSourceUnknown, false, errors.Errorf("expected plan UID in issue %v", issue.UID)
	}
	plan, err := s.GetPlan(ctx, &store.FindPlanMessage{UID: issue.PlanUID})
	if err != nil {
		return 0, store.RiskSourceUnknown, false, errors.Wrapf(err, "failed to get plan %v", *issue.PlanUID)
	}
	if plan == nil {
		return 0, store.RiskSourceUnknown, false, errors.Errorf("plan %v not found", *issue.PlanUID)
	}

	planCheckRuns, err := s.ListPlanCheckRuns(ctx, &store.FindPlanCheckRunMessage{
		PlanUID: &plan.UID,
		Type:    &[]store.PlanCheckRunType{store.PlanCheckDatabaseStatementSummaryReport},
	})
	if err != nil {
		return 0, store.RiskSourceUnknown, false, errors.Wrapf(err, "failed to list plan check runs for plan %v", plan.UID)
	}
	type Key struct {
		InstanceUID  int
		DatabaseName string
	}
	latestPlanCheckRun := map[Key]*store.PlanCheckRunMessage{}
	for _, run := range planCheckRuns {
		key := Key{
			InstanceUID:  int(run.Config.InstanceUid),
			DatabaseName: run.Config.DatabaseName,
		}
		oldValue, ok := latestPlanCheckRun[key]
		if !ok || oldValue.UID < run.UID {
			latestPlanCheckRun[key] = run
		}
	}
	for _, run := range latestPlanCheckRun {
		// the latest plan check run is not done yet, return done=false
		if run.Status != store.PlanCheckRunStatusDone {
			return 0, store.RiskSourceUnknown, false, nil
		}
	}

	pipelineCreate, err := v1.GetPipelineCreate(ctx, s, licenseService, dbFactory, plan.Config.Steps, issue.Project)
	if err != nil {
		return 0, store.RiskSourceUnknown, false, errors.Wrap(err, "failed to get pipeline create")
	}

	// Conclude risk source from task types.
	seenTaskType := map[api.TaskType]bool{}
	for _, stage := range pipelineCreate.Stages {
		for _, task := range stage.TaskList {
			seenTaskType[task.Type] = true
		}
	}
	riskSource := func() store.RiskSource {
		if seenTaskType[api.TaskDatabaseCreate] {
			return store.RiskSourceDatabaseCreate
		}
		if seenTaskType[api.TaskDatabaseSchemaUpdate] || seenTaskType[api.TaskDatabaseSchemaUpdateSDL] || seenTaskType[api.TaskDatabaseSchemaUpdateGhostSync] {
			return store.RiskSourceDatabaseSchemaUpdate
		}
		if seenTaskType[api.TaskDatabaseDataUpdate] {
			return store.RiskSourceDatabaseDataUpdate
		}
		return store.RiskSourceUnknown
	}()

	// cannot conclude risk source
	if riskSource == store.RiskSourceUnknown {
		return 0, store.RiskSourceUnknown, true, nil
	}

	e, err := cel.NewEnv(common.RiskFactors...)
	if err != nil {
		return 0, store.RiskSourceUnknown, false, err
	}

	var maxRiskLevel int64
	for _, stage := range pipelineCreate.Stages {
		for _, task := range stage.TaskList {
			instance, err := s.GetInstanceV2(ctx, &store.FindInstanceMessage{
				UID: &task.InstanceID,
			})
			if err != nil {
				return 0, store.RiskSourceUnknown, false, errors.Wrapf(err, "failed to get instance %v", task.InstanceID)
			}
			if instance.Deleted {
				continue
			}

			environmentID := instance.EnvironmentID
			var databaseName string
			if task.Type == api.TaskDatabaseCreate {
				payload := &api.TaskDatabaseCreatePayload{}
				if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
					return 0, store.RiskSourceUnknown, false, err
				}
				databaseName = payload.DatabaseName
				if payload.EnvironmentID != "" {
					environmentID = payload.EnvironmentID
				}
			} else {
				database, err := s.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
					UID: task.DatabaseID,
				})
				if err != nil {
					return 0, store.RiskSourceUnknown, false, err
				}
				databaseName = database.DatabaseName
				environmentID = database.EffectiveEnvironmentID
			}

			risk, err := func() (int64, error) {
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
					prg, err := e.Program(ast)
					if err != nil {
						return 0, err
					}
					args := map[string]any{
						"environment_id": environmentID,
						"project_id":     issue.Project.ResourceID,
						"database_name":  databaseName,
						// convert to string type otherwise cel-go will complain that storepb.Engine is not string type.
						"db_engine":     string(instance.Engine),
						"sql_type":      "UNKNOWN",
						"affected_rows": math.MaxInt32,
					}

					if run, ok := latestPlanCheckRun[Key{
						InstanceUID:  instance.UID,
						DatabaseName: databaseName,
					}]; ok {
						for _, result := range run.Result.Results {
							report := result.GetSqlSummaryReport()
							if report == nil {
								continue
							}
							args["affected_rows"] = report.AffectedRows
							for _, statementType := range report.StatementTypes {
								args["sql_type"] = statementType
								res, _, err := prg.Eval(args)
								if err != nil {
									return 0, err
								}
								val, err := res.ConvertToNative(reflect.TypeOf(false))
								if err != nil {
									return 0, errors.Wrap(err, "expect bool result")
								}
								if boolVal, ok := val.(bool); ok && boolVal {
									return risk.Level, nil
								}
							}
						}
					}

					args["sql_type"] = "UNKNOWN"
					args["affected_rows"] = math.MaxInt32
					res, _, err := prg.Eval(args)
					if err != nil {
						return 0, err
					}
					val, err := res.ConvertToNative(reflect.TypeOf(false))
					if err != nil {
						return 0, errors.Wrap(err, "expect bool result")
					}
					if boolVal, ok := val.(bool); ok && boolVal {
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
		}
	}

	return maxRiskLevel, riskSource, true, nil
}

func getGrantRequestIssueRisk(ctx context.Context, s *store.Store, issue *store.IssueMessage, risks []*store.RiskMessage) (int64, store.RiskSource, bool, error) {
	payload := issue.Payload
	if payload.GrantRequest == nil {
		return 0, store.RiskSourceUnknown, false, errors.New("grant request payload not found")
	}

	var riskSource store.RiskSource
	switch payload.GrantRequest.Role {
	case "roles/EXPORTER":
		riskSource = store.RiskRequestExport
	case "roles/QUERIER":
		riskSource = store.RiskRequestQuery
	default:
		return 0, store.RiskSourceUnknown, false, errors.Errorf("unknown grant request role %v", payload.GrantRequest.Role)
	}

	// fast path, no risks so return the DEFAULT risk level "0"
	if len(risks) == 0 {
		return 0, riskSource, true, nil
	}

	// higher risks go first
	sort.Slice(risks, func(i, j int) bool {
		return risks[i].Level > risks[j].Level
	})

	e, err := cel.NewEnv(common.RiskFactors...)
	if err != nil {
		return 0, store.RiskSourceUnknown, false, err
	}

	factors, err := common.GetQueryExportFactors(payload.GetGrantRequest().GetCondition().GetExpression())
	if err != nil {
		return 0, store.RiskSourceUnknown, false, errors.Wrap(err, "failed to get query export factors")
	}
	expirationDays := payload.GrantRequest.Expiration.AsDuration().Hours() / 24
	databases := []*store.DatabaseMessage{}
	if len(factors.DatabaseNames) == 0 {
		list, err := s.ListDatabases(ctx, &store.FindDatabaseMessage{
			ProjectID: &issue.Project.ResourceID,
		})
		if err != nil {
			return 0, store.RiskSourceUnknown, false, errors.Wrap(err, "failed to list databases")
		}
		databases = list
	} else {
		for _, dbName := range factors.DatabaseNames {
			instanceID, databaseName, err := common.GetInstanceDatabaseID(dbName)
			if err != nil {
				return 0, store.RiskSourceUnknown, false, errors.Wrap(err, "failed to get instance database id")
			}

			instance, err := s.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &instanceID})
			if err != nil {
				return 0, store.RiskSourceUnknown, false, errors.Wrap(err, "failed to get instance")
			}
			database, err := s.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
				ProjectID:           &issue.Project.ResourceID,
				InstanceID:          &instanceID,
				DatabaseName:        &databaseName,
				IgnoreCaseSensitive: store.IgnoreDatabaseAndTableCaseSensitive(instance),
			})
			if err != nil {
				return 0, store.RiskSourceUnknown, false, errors.Wrap(err, "failed to get database")
			}
			if database == nil {
				return 0, store.RiskSourceUnknown, false, errors.Errorf("database %q not found", databaseName)
			}
			databases = append(databases, database)
		}
	}

	var maxRisk int64
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
			return 0, store.RiskSourceUnknown, false, errors.Errorf("failed to parse expression: %v", issues.Err())
		}
		prg, err := e.Program(ast)
		if err != nil {
			return 0, store.RiskSourceUnknown, false, err
		}

		if riskSource == store.RiskRequestExport {
			for _, database := range databases {
				instance, err := s.GetInstanceV2(ctx, &store.FindInstanceMessage{
					ResourceID: &database.InstanceID,
				})
				if err != nil {
					return 0, store.RiskSourceUnknown, false, errors.Wrap(err, "failed to get instance")
				}
				args := map[string]any{
					"environment_id": database.EffectiveEnvironmentID,
					"project_id":     issue.Project.ResourceID,
					"database_name":  database.DatabaseName,
					// convert to string type otherwise cel-go will complain that storepb.Engine is not string type.
					"db_engine":       string(instance.Engine),
					"export_rows":     factors.ExportRows,
					"expiration_days": expirationDays,
				}
				res, _, err := prg.Eval(args)
				if err != nil {
					return 0, store.RiskSourceUnknown, false, err
				}

				val, err := res.ConvertToNative(reflect.TypeOf(false))
				if err != nil {
					return 0, store.RiskSourceUnknown, false, errors.Wrap(err, "expect bool result")
				}
				if boolVal, ok := val.(bool); ok && boolVal {
					if risk.Level > maxRisk {
						maxRisk = risk.Level
					}
				}
			}
		} else if riskSource == store.RiskRequestQuery {
			for _, database := range databases {
				instance, err := s.GetInstanceV2(ctx, &store.FindInstanceMessage{
					ResourceID: &database.InstanceID,
				})
				if err != nil {
					return 0, store.RiskSourceUnknown, false, errors.Wrap(err, "failed to get instance")
				}
				args := map[string]any{
					"environment_id": database.EffectiveEnvironmentID,
					"project_id":     issue.Project.ResourceID,
					"database_name":  database.DatabaseName,
					// convert to string type otherwise cel-go will complain that storepb.Engine is not string type.
					"db_engine":       string(instance.Engine),
					"expiration_days": expirationDays,
				}
				res, _, err := prg.Eval(args)
				if err != nil {
					return 0, store.RiskSourceUnknown, false, err
				}

				val, err := res.ConvertToNative(reflect.TypeOf(false))
				if err != nil {
					return 0, store.RiskSourceUnknown, false, errors.Wrap(err, "expect bool result")
				}
				if boolVal, ok := val.(bool); ok && boolVal {
					if risk.Level > maxRisk {
						maxRisk = risk.Level
					}
				}
			}
		}
	}

	return maxRisk, riskSource, true, nil
}

func updateIssueApprovalPayload(ctx context.Context, s *store.Store, issue *store.IssueMessage, approval *storepb.IssuePayloadApproval) error {
	if _, err := s.UpdateIssueV2(ctx, issue.UID, &store.UpdateIssueMessage{
		PayloadUpsert: &storepb.IssuePayload{
			Approval: approval,
		},
	}, api.SystemBotID); err != nil {
		return errors.Wrap(err, "failed to update issue payload")
	}
	return nil
}

func convertToSource(source store.RiskSource) v1pb.Risk_Source {
	switch source {
	case store.RiskSourceDatabaseCreate:
		return v1pb.Risk_CREATE_DATABASE
	case store.RiskSourceDatabaseSchemaUpdate:
		return v1pb.Risk_DDL
	case store.RiskSourceDatabaseDataUpdate:
		return v1pb.Risk_DML
	case store.RiskRequestQuery:
		return v1pb.Risk_QUERY
	case store.RiskRequestExport:
		return v1pb.Risk_EXPORT
	}
	return v1pb.Risk_SOURCE_UNSPECIFIED
}
