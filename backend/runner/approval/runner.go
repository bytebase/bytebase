// Package approval is the runner for finding approval templates for issues.
package approval

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/google/cel-go/cel"
	"github.com/pkg/errors"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/state"
	enterpriseAPI "github.com/bytebase/bytebase/backend/enterprise/api"
	api "github.com/bytebase/bytebase/backend/legacyapi"

	"github.com/bytebase/bytebase/backend/store"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// RiskFactors are the variables when evaluating the risk level.
var RiskFactors = []cel.EnvOption{
	// string factors
	// use environment.resource_id
	cel.Variable("environment_id", cel.StringType),
	// use project.resource_id
	cel.Variable("project_id", cel.StringType),
	cel.Variable("database_name", cel.StringType),
	cel.Variable("db_engine", cel.StringType),
	cel.Variable("sql_type", cel.StringType),

	// number factors
	cel.Variable("affected_rows", cel.IntType),
}

// ApprovalFactors are the variables when finding the approval template.
var ApprovalFactors = []cel.EnvOption{
	cel.Variable("level", cel.IntType),
	cel.Variable("source", cel.IntType),
}

var issueTypeToRiskSource = map[api.IssueType]store.RiskSource{
	// RiskSourceDatabaseSchemaUpdate
	api.IssueDatabaseSchemaUpdate:      store.RiskSourceDatabaseSchemaUpdate,
	api.IssueDatabaseSchemaUpdateGhost: store.RiskSourceDatabaseSchemaUpdate,
	// RiskSourceDatabaseDataUpdate
	api.IssueDatabaseDataUpdate: store.RiskSourceDatabaseDataUpdate,
	// RiskSourceDatabaseCreate
	api.IssueDatabaseCreate: store.RiskSourceDatabaseCreate,
	// RiskSourceUnknown
	api.IssueGeneral:             store.RiskSourceUnknown,
	api.IssueDatabaseRestorePITR: store.RiskSourceUnknown,
}

// Runner is the runner for finding approval templates for issues.
type Runner struct {
	store     *store.Store
	dbFactory *dbfactory.DBFactory
	stateCfg  *state.State

	licenseService enterpriseAPI.LicenseService
}

// NewRunner creates a new runner.
func NewRunner(store *store.Store, dbFactory *dbfactory.DBFactory, stateCfg *state.State, licenseService enterpriseAPI.LicenseService) *Runner {
	return &Runner{
		store:          store,
		dbFactory:      dbFactory,
		stateCfg:       stateCfg,
		licenseService: licenseService,
	}
}

const approvalRunnerInterval = 1 * time.Second

// Run runs the runner.
func (r *Runner) Run(ctx context.Context, wg *sync.WaitGroup) {
	ticker := time.NewTicker(approvalRunnerInterval)
	defer ticker.Stop()
	defer wg.Done()
	log.Debug(fmt.Sprintf("Approval runner started and will run every %v", approvalRunnerInterval))
	r.retryFindApprovalTemplate(ctx)
	for {
		select {
		case <-ticker.C:
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
				log.Error("approval runner", zap.Error(err))
			}

		case <-ctx.Done():
			return
		}
	}
}

func (r *Runner) retryFindApprovalTemplate(ctx context.Context) {
	issues, err := r.store.ListIssueV2(ctx, &store.FindIssueMessage{
		StatusList: []api.IssueStatus{api.IssueOpen},
	})
	if err != nil {
		err := errors.Wrap(err, "failed to list issues")
		log.Error("failed to retry finding approval template", zap.Error(err))
	}
	for _, issue := range issues {
		payload := &storepb.IssuePayload{}
		if err := protojson.Unmarshal([]byte(issue.Payload), payload); err != nil {
			log.Error("failed to retry finding approval template", zap.Int("issueID", issue.UID), zap.Error(err))
			continue
		}
		if payload.Approval == nil || !payload.Approval.ApprovalFindingDone {
			r.stateCfg.ApprovalFinding.Store(issue.UID, issue)
		}
	}
}

func (r *Runner) findApprovalTemplateForIssue(ctx context.Context, issue *store.IssueMessage, risks []*store.RiskMessage, approvalSetting *storepb.WorkspaceApprovalSetting) (bool, error) {
	payload := &storepb.IssuePayload{}
	if err := protojson.Unmarshal([]byte(issue.Payload), payload); err != nil {
		return false, errors.Wrap(err, "failed to unmarshal issue payload")
	}
	if payload.Approval != nil && payload.Approval.ApprovalFindingDone {
		return true, nil
	}

	// no need to find if
	// - feature is not enabled
	// - risk source is RiskSourceUnknown
	// - approval setting rules are empty
	if !r.licenseService.IsFeatureEnabled(api.FeatureCustomApproval) || issueTypeToRiskSource[issue.Type] == store.RiskSourceUnknown || len(approvalSetting.Rules) == 0 {
		if err := updateIssuePayload(ctx, r.store, issue.UID, &storepb.IssuePayload{
			Approval: &storepb.IssuePayloadApproval{
				ApprovalFindingDone: true,
				ApprovalTemplates:   nil,
				Approvers:           nil,
			},
		}); err != nil {
			return false, errors.Wrap(err, "failed to update issue payload")
		}
		return true, nil
	}

	riskLevel, done, err := getIssueRiskLevel(ctx, r.store, issue, risks)
	if err != nil {
		err = errors.Wrap(err, "failed to get issue risk level")
		if updateErr := updateIssuePayload(ctx, r.store, issue.UID, &storepb.IssuePayload{
			Approval: &storepb.IssuePayloadApproval{
				ApprovalFindingDone:  true,
				ApprovalFindingError: err.Error(),
			},
		}); updateErr != nil {
			return false, multierr.Append(errors.Wrap(updateErr, "failed to update issue payload"), err)
		}
		return false, err
	}
	if !done {
		return false, nil
	}

	approvalTemplate, err := getApprovalTemplate(approvalSetting, riskLevel, issueTypeToRiskSource[issue.Type])
	if err != nil {
		err = errors.Wrapf(err, "failed to get approval template, riskLevel: %v", riskLevel)
		if updateErr := updateIssuePayload(ctx, r.store, issue.UID, &storepb.IssuePayload{
			Approval: &storepb.IssuePayloadApproval{
				ApprovalFindingDone:  true,
				ApprovalFindingError: err.Error(),
			},
		}); updateErr != nil {
			return false, multierr.Append(errors.Wrap(updateErr, "failed to update issue payload"), err)
		}
		return false, err
	}

	payload = &storepb.IssuePayload{
		Approval: &storepb.IssuePayloadApproval{
			ApprovalFindingDone: true,
			ApprovalTemplates:   nil,
			Approvers:           nil,
		},
	}
	if approvalTemplate != nil {
		payload.Approval.ApprovalTemplates = append(payload.Approval.ApprovalTemplates, approvalTemplate)
	}

	if err := updateIssuePayload(ctx, r.store, issue.UID, payload); err != nil {
		return false, errors.Wrap(err, "failed to update issue payload")
	}
	return true, nil
}

func getApprovalTemplate(approvalSetting *storepb.WorkspaceApprovalSetting, riskLevel int64, riskSource store.RiskSource) (*storepb.ApprovalTemplate, error) {
	e, err := cel.NewEnv(ApprovalFactors...)
	if err != nil {
		return nil, err
	}
	for _, rule := range approvalSetting.Rules {
		if rule.Expression == nil || rule.Expression.Expr == nil {
			continue
		}
		ast := cel.ParsedExprToAst(rule.Expression)
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

func getIssueRiskLevel(ctx context.Context, s *store.Store, issue *store.IssueMessage, risks []*store.RiskMessage) (int64, bool, error) {
	tasks, err := s.ListTasks(ctx, &api.TaskFind{
		PipelineID: &issue.PipelineUID,
		StatusList: &[]api.TaskStatus{api.TaskPendingApproval},
	})
	if err != nil {
		return 0, false, err
	}

	var maxRiskLevel int64
	for _, task := range tasks {
		riskLevel, done, err := getTaskRiskLevel(ctx, s, issue, task, risks)
		if err != nil {
			return 0, false, errors.Wrapf(err, "failed to get task risk level for task %v", task.ID)
		}
		if !done {
			return 0, false, nil
		}
		if riskLevel > maxRiskLevel {
			maxRiskLevel = riskLevel
		}
	}

	return maxRiskLevel, true, nil
}

func getReportResult(ctx context.Context, s *store.Store, task *store.TaskMessage, taskCheckType api.TaskCheckType) ([]api.TaskCheckResult, bool, error) {
	reports, err := s.ListTaskCheckRuns(ctx, &store.TaskCheckRunFind{
		TaskID: &task.ID,
		Type:   &taskCheckType,
	})
	if err != nil {
		return nil, false, err
	}
	if len(reports) == 0 {
		return nil, false, nil
	}
	lastReport := reports[0]
	for i, report := range reports {
		if report.ID > lastReport.ID {
			lastReport = reports[i]
		}
	}
	if lastReport.Status != api.TaskCheckRunDone {
		return nil, false, nil
	}

	payload := &api.TaskCheckRunResultPayload{}
	if err := json.Unmarshal([]byte(lastReport.Result), payload); err != nil {
		return nil, false, err
	}
	return payload.ResultList, true, nil
}

func getTaskRiskLevel(ctx context.Context, s *store.Store, issue *store.IssueMessage, task *store.TaskMessage, risks []*store.RiskMessage) (int64, bool, error) {
	// Fall through to "DEFAULT" risk level if risks are empty.
	if len(risks) == 0 {
		return 0, true, nil
	}

	instance, err := s.GetInstanceV2(ctx, &store.FindInstanceMessage{
		UID: &task.InstanceID,
	})
	if err != nil {
		return 0, false, err
	}

	var affectedRowsReportResult, statementTypeReportResult []api.TaskCheckResult
	if api.IsTaskCheckReportSupported(instance.Engine) && api.IsTaskCheckReportNeededForTaskType(task.Type) {
		affectedRowsReportResultInner, done, err := getReportResult(ctx, s, task, api.TaskCheckDatabaseStatementAffectedRowsReport)
		if err != nil {
			return 0, false, err
		}
		if !done {
			return 0, false, nil
		}
		affectedRowsReportResult = affectedRowsReportResultInner

		statementTypeReportResultInner, done, err := getReportResult(ctx, s, task, api.TaskCheckDatabaseStatementTypeReport)
		if err != nil {
			return 0, false, err
		}
		if !done {
			return 0, false, nil
		}
		statementTypeReportResult = statementTypeReportResultInner
	}

	if len(affectedRowsReportResult) != len(statementTypeReportResult) {
		return 0, false, errors.New("affected rows report result and statement type report result length mismatch")
	}

	var databaseName string
	if task.Type == api.TaskDatabaseCreate {
		payload := &api.TaskDatabaseCreatePayload{}
		if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
			return 0, false, err
		}
		databaseName = payload.DatabaseName
	} else {
		database, err := s.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
			UID: task.DatabaseID,
		})
		if err != nil {
			return 0, false, err
		}
		databaseName = database.DatabaseName
	}

	e, err := cel.NewEnv(RiskFactors...)
	if err != nil {
		return 0, false, err
	}

	// higher risks go first
	sort.Slice(risks, func(i, j int) bool {
		return risks[i].Level > risks[j].Level
	})
	var maxRisk int64
	for _, risk := range risks {
		if !risk.Active {
			continue
		}
		if risk.Source != issueTypeToRiskSource[issue.Type] {
			continue
		}
		if risk.Expression == nil || risk.Expression.Expr == nil {
			continue
		}

		ast := cel.ParsedExprToAst(risk.Expression)
		prg, err := e.Program(ast)
		if err != nil {
			return 0, false, err
		}
		args := map[string]any{
			"environment_id": instance.EnvironmentID,
			"project_id":     issue.Project.ResourceID,
			"database_name":  databaseName,
			// convert to string type otherwise cel-go will complain that db.Type is not string type.
			"db_engine":     string(instance.Engine),
			"sql_type":      "UNKNOWN",
			"affected_rows": 0,
		}

		// eval for each statement
		if len(affectedRowsReportResult) > 0 {
			for i := range affectedRowsReportResult {
				args := args
				if affectedRowsReportResult[i].Code == common.Ok.Int() {
					affectedRows, err := strconv.ParseInt(affectedRowsReportResult[i].Content, 10, 64)
					if err != nil {
						log.Warn("failed to convert affectedRows to int64, will use 0 as the value of affected_rows", zap.Error(err))
					} else {
						args["affected_rows"] = affectedRows
					}
				}
				if statementTypeReportResult[i].Code == common.Ok.Int() {
					args["sql_type"] = statementTypeReportResult[i].Content
				}

				res, _, err := prg.Eval(args)
				if err != nil {
					return 0, false, err
				}

				val, err := res.ConvertToNative(reflect.TypeOf(false))
				if err != nil {
					return 0, false, errors.Wrap(err, "expect bool result")
				}
				if boolVal, ok := val.(bool); ok && boolVal {
					if risk.Level > maxRisk {
						maxRisk = risk.Level
					}
				}
			}
		} else {
			res, _, err := prg.Eval(args)
			if err != nil {
				return 0, false, err
			}
			val, err := res.ConvertToNative(reflect.TypeOf(false))
			if err != nil {
				return 0, false, errors.Wrap(err, "expect bool result")
			}
			if boolVal, ok := val.(bool); ok && boolVal {
				return risk.Level, true, nil
			}
		}
	}

	return maxRisk, true, nil
}

func updateIssuePayload(ctx context.Context, s *store.Store, issueID int, payload *storepb.IssuePayload) error {
	payloadBytes, err := protojson.Marshal(payload)
	if err != nil {
		return errors.Wrap(err, "failed to marshal issue payload")
	}
	payloadStr := string(payloadBytes)
	if _, err := s.UpdateIssueV2(ctx, issueID, &store.UpdateIssueMessage{
		Payload: &payloadStr,
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
	}
	return v1pb.Risk_SOURCE_UNSPECIFIED
}
