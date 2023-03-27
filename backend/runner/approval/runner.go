// Package approval is the runner for finding approval templates for issues.
package approval

import (
	"context"
	"encoding/json"
	"reflect"
	"sort"
	"sync"
	"time"

	"github.com/google/cel-go/cel"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

const approvalRunnerInterval = 30 * time.Second

var predefinedVariables = []cel.EnvOption{
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

var riskVariables = []cel.EnvOption{
	cel.Variable("risk", cel.IntType),
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
}

// NewRunner creates a new runner.
func NewRunner(store *store.Store, dbFactory *dbfactory.DBFactory) *Runner {
	return &Runner{
		store:     store,
		dbFactory: dbFactory,
	}
}

// Run runs the runner.
func (r *Runner) Run(ctx context.Context, wg *sync.WaitGroup) {
	ticker := time.NewTicker(approvalRunnerInterval)
	defer ticker.Stop()
	defer wg.Done()
	for {
		select {
		case <-ticker.C:
			err := func() error {
				issues, err := r.store.ListIssueV2(ctx, &store.FindIssueMessage{
					StatusList: []api.IssueStatus{api.IssueOpen},
				})
				if err != nil {
					return err
				}
				risks, err := r.store.ListRisks(ctx)
				if err != nil {
					return err
				}
				approvalSetting, err := r.store.GetWorkspaceApprovalSetting(ctx)
				if err != nil {
					return err
				}
				for _, issue := range issues {
					payload := &storepb.IssuePayload{}
					if err := protojson.Unmarshal([]byte(issue.Payload), payload); err != nil {
						return err
					}
					if payload.Approval != nil && payload.Approval.ApprovalFindingDone {
						continue
					}

					riskLevel, err := getIssueRiskLevel(ctx, r.store, issue, risks)
					if err != nil {
						return err
					}
					if riskLevel == 0 {
						continue
					}
					approvalTemplate, err := getApprovalTemplate(approvalSetting, riskLevel, issueTypeToRiskSource[issue.Type])
					if err != nil {
						return err
					}
					payload = &storepb.IssuePayload{
						Approval: &storepb.IssuePayloadApproval{
							ApprovalFindingDone: true,
							ApprovalTemplates:   []*storepb.ApprovalTemplate{approvalTemplate},
							Approvers:           nil,
						},
					}
					payloadBytes, err := protojson.Marshal(payload)
					if err != nil {
						return err
					}
					payloadStr := string(payloadBytes)
					if _, err := r.store.UpdateIssueV2(ctx, issue.UID, &store.UpdateIssueMessage{
						Payload: &payloadStr,
					}, api.SystemBotID); err != nil {
						return err
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
	e, err := cel.NewEnv(riskVariables...)
	if err != nil {
		return nil, err
	}
	for _, rule := range approvalSetting.Rules {
		if rule.Expression == nil {
			continue
		}
		ast := cel.ParsedExprToAst(rule.Expression)
		prg, err := e.Program(ast)
		if err != nil {
			return nil, err
		}

		res, _, err := prg.Eval(map[string]interface{}{
			"risk":   riskLevel,
			"source": convertToSource(riskSource),
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

func getIssueRiskLevel(ctx context.Context, s *store.Store, issue *store.IssueMessage, risks []*store.RiskMessage) (int64, error) {
	tasks, err := s.ListTasks(ctx, &api.TaskFind{
		PipelineID: &issue.PipelineUID,
		StatusList: &[]api.TaskStatus{api.TaskPendingApproval},
	})
	if err != nil {
		return 0, err
	}

	// all tasks must have passed task checks.
	for _, task := range tasks {
		instance, err := s.GetInstanceV2(ctx, &store.FindInstanceMessage{
			UID: &task.InstanceID,
		})
		if err != nil {
			return 0, err
		}
		pass, err := utils.PassAllCheck(task, api.TaskCheckStatusWarn, task.TaskCheckRunRawList, instance.Engine)
		if err != nil {
			return 0, err
		}
		if !pass {
			return 0, nil
		}
	}

	var maxRiskLevel int64
	for _, task := range tasks {
		riskLevel, err := getTaskRiskLevel(ctx, s, issue, task, risks)
		if err != nil {
			return 0, err
		}
		if riskLevel > maxRiskLevel {
			maxRiskLevel = riskLevel
		}
	}

	return maxRiskLevel, nil
}

func getTaskRiskLevel(ctx context.Context, s *store.Store, issue *store.IssueMessage, task *store.TaskMessage, risks []*store.RiskMessage) (int64, error) {
	instance, err := s.GetInstanceV2(ctx, &store.FindInstanceMessage{
		UID: &task.InstanceID,
	})
	if err != nil {
		return 0, err
	}

	var databaseName string
	if task.Type == api.TaskDatabaseCreate {
		payload := &api.TaskDatabaseCreatePayload{}
		if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
			return 0, err
		}
		databaseName = payload.DatabaseName
	} else {
		database, err := s.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
			UID: task.DatabaseID,
		})
		if err != nil {
			return 0, err
		}
		databaseName = database.DatabaseName
	}

	e, err := cel.NewEnv(predefinedVariables...)
	if err != nil {
		return 0, err
	}

	// higher risks go first
	sort.Slice(risks, func(i, j int) bool {
		return risks[i].Level > risks[j].Level
	})
	for _, risk := range risks {
		if risk.Source != issueTypeToRiskSource[issue.Type] {
			continue
		}

		ast := cel.ParsedExprToAst(risk.Expression)
		prg, err := e.Program(ast)
		if err != nil {
			return 0, err
		}

		// TODO(p0ny): approval, impl other factors.
		res, _, err := prg.Eval(map[string]interface{}{
			"environment_id": instance.EnvironmentID,
			"project_id":     issue.Project.ResourceID,
			"database_name":  databaseName,
			"db_engine":      instance.Engine,
			"sql_type":       "UNKNOWN",
			"affected_rows":  0,
		})
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
