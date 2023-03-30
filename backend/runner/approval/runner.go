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
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
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
	store          *store.Store
	dbFactory      *dbfactory.DBFactory
	profile        config.Profile
	licenseService enterpriseAPI.LicenseService
}

// NewRunner creates a new runner.
func NewRunner(store *store.Store, dbFactory *dbfactory.DBFactory, profile config.Profile, licenseService enterpriseAPI.LicenseService) *Runner {
	return &Runner{
		store:          store,
		dbFactory:      dbFactory,
		profile:        profile,
		licenseService: licenseService,
	}
}

// Run runs the runner.
func (r *Runner) Run(ctx context.Context, wg *sync.WaitGroup) {
	ticker := time.NewTicker(r.profile.ApprovalRunnerInterval)
	defer ticker.Stop()
	defer wg.Done()
	log.Debug(fmt.Sprintf("Approval runner started and will run every %v", r.profile.ApprovalRunnerInterval))
	for {
		select {
		case <-ticker.C:
			err := func() error {
				issues, err := r.store.ListIssueV2(ctx, &store.FindIssueMessage{
					StatusList: []api.IssueStatus{api.IssueOpen},
				})
				if err != nil {
					return errors.Wrap(err, "failed to list issues")
				}
				risks, err := r.store.ListRisks(ctx)
				if err != nil {
					return errors.Wrap(err, "failed to list risks")
				}
				approvalSetting, err := r.store.GetWorkspaceApprovalSetting(ctx)
				if err != nil {
					return errors.Wrap(err, "failed to get workspace approval setting")
				}
				for _, issue := range issues {
					payload := &storepb.IssuePayload{}
					if err := protojson.Unmarshal([]byte(issue.Payload), payload); err != nil {
						log.Error("failed to unmarshal issue payload", zap.Error(err))
						continue
					}
					if payload.Approval != nil && payload.Approval.ApprovalFindingDone {
						continue
					}

					// no need to find if
					// - feature is not enabled
					// - risk source is RiskSourceUnknown
					// - risks are empty
					// - approval setting rules are empty
					if !r.licenseService.IsFeatureEnabled(api.FeatureCustomApproval) || issueTypeToRiskSource[issue.Type] == store.RiskSourceUnknown || len(risks) == 0 || len(approvalSetting.Rules) == 0 {
						payload := &storepb.IssuePayload{
							Approval: &storepb.IssuePayloadApproval{
								ApprovalFindingDone: true,
								ApprovalTemplates:   nil,
								Approvers:           nil,
							},
						}
						payloadBytes, err := protojson.Marshal(payload)
						if err != nil {
							log.Error("failed to marshal issue payload", zap.Error(err))
							continue
						}
						payloadStr := string(payloadBytes)
						if _, err := r.store.UpdateIssueV2(ctx, issue.UID, &store.UpdateIssueMessage{
							Payload: &payloadStr,
						}, api.SystemBotID); err != nil {
							log.Error("failed to update issue payload", zap.Error(err))
							continue
						}
						continue
					}

					riskLevel, done, err := getIssueRiskLevel(ctx, r.store, issue, risks)
					if err != nil {
						log.Error("failed to get issue risk level", zap.Int("issueID", issue.UID), zap.Error(err))
						continue
					}
					if !done {
						continue
					}
					approvalTemplate, err := getApprovalTemplate(approvalSetting, riskLevel, issueTypeToRiskSource[issue.Type])
					if err != nil {
						log.Error("failed to get approval template", zap.Int64("riskLevel", riskLevel), zap.String("issueType", string(issue.Type)), zap.Error(err))
						continue
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

					payloadBytes, err := protojson.Marshal(payload)
					if err != nil {
						log.Error("failed to marshal issue payload", zap.Error(err))
						continue
					}
					payloadStr := string(payloadBytes)
					if _, err := r.store.UpdateIssueV2(ctx, issue.UID, &store.UpdateIssueMessage{
						Payload: &payloadStr,
					}, api.SystemBotID); err != nil {
						log.Error("failed to update issue payload", zap.Error(err))
						continue
					}
				}

				return nil
			}()
			if err != nil {
				log.Error("approval runner", zap.Error(err))
			}

		case <-ctx.Done():
			return
		}
	}
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

		res, _, err := prg.Eval(map[string]interface{}{
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
	if err := json.Unmarshal([]byte(lastReport.Payload), payload); err != nil {
		return nil, false, err
	}
	return payload.ResultList, true, nil
}

func getTaskRiskLevel(ctx context.Context, s *store.Store, issue *store.IssueMessage, task *store.TaskMessage, risks []*store.RiskMessage) (int64, bool, error) {
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
		args := map[string]interface{}{
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
