package server

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/common/log"
	enterpriseAPI "github.com/bytebase/bytebase/enterprise/api"
	"github.com/bytebase/bytebase/store"
)

// NewTaskCheckScheduler creates a task check scheduler.
func NewTaskCheckScheduler(server *Server, store *store.Store, licenseService enterpriseAPI.LicenseService) *TaskCheckScheduler {
	return &TaskCheckScheduler{
		server:         server,
		store:          store,
		licenseService: licenseService,
		executors:      make(map[api.TaskCheckType]TaskCheckExecutor),
	}
}

// TaskCheckScheduler is the task check scheduler.
type TaskCheckScheduler struct {
	server         *Server
	store          *store.Store
	licenseService enterpriseAPI.LicenseService
	executors      map[api.TaskCheckType]TaskCheckExecutor
}

// Run will run the task check scheduler once.
func (s *TaskCheckScheduler) Run(ctx context.Context, wg *sync.WaitGroup) {
	ticker := time.NewTicker(taskSchedulerInterval)
	defer ticker.Stop()
	defer wg.Done()
	log.Debug(fmt.Sprintf("Task check scheduler started and will run every %v", taskSchedulerInterval))
	runningTaskChecks := make(map[int]bool)
	mu := sync.RWMutex{}
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
						log.Error("Task check scheduler PANIC RECOVER", zap.Error(err), zap.Stack("panic-stack"))
					}
				}()

				ctx := context.Background()

				// Inspect all running task checks
				taskCheckRunStatusList := []api.TaskCheckRunStatus{api.TaskCheckRunRunning}
				taskCheckRunFind := &api.TaskCheckRunFind{
					StatusList: &taskCheckRunStatusList,
				}
				taskCheckRunList, err := s.store.FindTaskCheckRun(ctx, taskCheckRunFind)
				if err != nil {
					log.Error("Failed to retrieve running tasks", zap.Error(err))
					return
				}
				for _, taskCheckRun := range taskCheckRunList {
					executor, ok := s.executors[taskCheckRun.Type]
					if !ok {
						log.Error("Skip running task check run with unknown type",
							zap.Int("id", taskCheckRun.ID),
							zap.Int("task_id", taskCheckRun.TaskID),
							zap.String("type", string(taskCheckRun.Type)),
						)
						continue
					}

					mu.Lock()
					if _, ok := runningTaskChecks[taskCheckRun.ID]; ok {
						mu.Unlock()
						continue
					}
					runningTaskChecks[taskCheckRun.ID] = true
					mu.Unlock()

					go func(taskCheckRun *api.TaskCheckRun) {
						defer func() {
							mu.Lock()
							delete(runningTaskChecks, taskCheckRun.ID)
							mu.Unlock()
						}()
						checkResultList, err := executor.Run(ctx, taskCheckRun)

						if err == nil {
							bytes, err := json.Marshal(api.TaskCheckRunResultPayload{
								ResultList: checkResultList,
							})
							if err != nil {
								log.Error("Failed to marshal task check run result",
									zap.Int("id", taskCheckRun.ID),
									zap.Int("task_id", taskCheckRun.TaskID),
									zap.String("type", string(taskCheckRun.Type)),
									zap.Error(err),
								)
								return
							}

							taskCheckRunStatusPatch := &api.TaskCheckRunStatusPatch{
								ID:        &taskCheckRun.ID,
								UpdaterID: api.SystemBotID,
								Status:    api.TaskCheckRunDone,
								Code:      common.Ok,
								Result:    string(bytes),
							}
							_, err = s.store.PatchTaskCheckRunStatus(ctx, taskCheckRunStatusPatch)
							if err != nil {
								log.Error("Failed to mark task check run as DONE",
									zap.Int("id", taskCheckRun.ID),
									zap.Int("task_id", taskCheckRun.TaskID),
									zap.String("type", string(taskCheckRun.Type)),
									zap.Error(err),
								)
							}
						} else {
							log.Warn("Failed to run task check",
								zap.Int("id", taskCheckRun.ID),
								zap.Int("task_id", taskCheckRun.TaskID),
								zap.String("type", string(taskCheckRun.Type)),
								zap.Error(err),
							)
							bytes, marshalErr := json.Marshal(api.TaskCheckRunResultPayload{
								Detail: err.Error(),
							})
							if marshalErr != nil {
								log.Error("Failed to marshal task check run result",
									zap.Int("id", taskCheckRun.ID),
									zap.Int("task_id", taskCheckRun.TaskID),
									zap.String("type", string(taskCheckRun.Type)),
									zap.Error(marshalErr),
								)
								return
							}

							taskCheckRunStatusPatch := &api.TaskCheckRunStatusPatch{
								ID:        &taskCheckRun.ID,
								UpdaterID: api.SystemBotID,
								Status:    api.TaskCheckRunFailed,
								Code:      common.ErrorCode(err),
								Result:    string(bytes),
							}
							_, err = s.store.PatchTaskCheckRunStatus(ctx, taskCheckRunStatusPatch)
							if err != nil {
								log.Error("Failed to mark task check run as FAILED",
									zap.Int("id", taskCheckRun.ID),
									zap.Int("task_id", taskCheckRun.TaskID),
									zap.String("type", string(taskCheckRun.Type)),
									zap.Error(err),
								)
							}
						}
					}(taskCheckRun)
				}
			}()
		case <-ctx.Done(): // if cancel() execute
			return
		}
	}
}

// Register will register the task check executor.
func (s *TaskCheckScheduler) Register(taskType api.TaskCheckType, executor TaskCheckExecutor) {
	if executor == nil {
		panic("scheduler: Register executor is nil for task type: " + taskType)
	}
	if _, dup := s.executors[taskType]; dup {
		panic("scheduler: Register called twice for task type: " + taskType)
	}
	s.executors[taskType] = executor
}

func (s *TaskCheckScheduler) getTaskCheck(ctx context.Context, task *api.Task, creatorID int) ([]*api.TaskCheckRunCreate, error) {
	var createList []*api.TaskCheckRunCreate

	create, err := s.getLGTMTaskCheck(ctx, task, creatorID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to schedule LGTM task check")
	}
	createList = append(createList, create...)

	create, err = s.getPITRTaskCheck(ctx, task, creatorID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to schedule backup/PITR task check")
	}
	createList = append(createList, create...)

	if task.Type != api.TaskDatabaseSchemaUpdate && task.Type != api.TaskDatabaseSchemaUpdateSDL && task.Type != api.TaskDatabaseDataUpdate && task.Type != api.TaskDatabaseSchemaUpdateGhostSync {
		return createList, nil
	}

	create, err = s.getGeneralTaskCheck(ctx, task, creatorID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to schedule general task check")
	}
	createList = append(createList, create...)

	create, err = s.getGhostTaskCheck(ctx, task, creatorID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to schedule gh-ost task check")
	}
	createList = append(createList, create...)

	statement, err := s.getStatement(task)
	if err != nil {
		return nil, err
	}
	database, err := s.store.GetDatabase(ctx, &api.DatabaseFind{ID: task.DatabaseID})
	if err != nil {
		return nil, err
	}
	if database == nil {
		return nil, errors.Errorf("database ID not found %v", task.DatabaseID)
	}

	create, err = getSyntaxCheckTaskCheck(ctx, task, creatorID, database, statement)
	if err != nil {
		return nil, errors.Wrap(err, "failed to schedule syntax check task check")
	}
	createList = append(createList, create...)

	create, err = s.getSQLReviewTaskCheck(ctx, task, creatorID, database, statement)
	if err != nil {
		return nil, errors.Wrap(err, "failed to schedule SQL review task check")
	}
	createList = append(createList, create...)

	create, err = s.getStmtTypeTaskCheck(ctx, task, creatorID, database, statement)
	if err != nil {
		return nil, errors.Wrap(err, "failed to schedule statement type task check")
	}
	createList = append(createList, create...)

	return createList, nil
}

// ScheduleCheck schedules variouse task checks depending on the task type.
func (s *TaskCheckScheduler) ScheduleCheck(ctx context.Context, task *api.Task, creatorID int) (*api.Task, error) {
	createList, err := s.getTaskCheck(ctx, task, creatorID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to getTaskCheck")
	}
	taskCheckRunList, err := s.store.BatchCreateTaskCheckRun(ctx, createList)
	if err != nil {
		return nil, err
	}
	task.TaskCheckRunList = taskCheckRunList
	return task, err
}

func (*TaskCheckScheduler) getStatement(task *api.Task) (string, error) {
	switch task.Type {
	case api.TaskDatabaseSchemaUpdate:
		taskPayload := &api.TaskDatabaseSchemaUpdatePayload{}
		if err := json.Unmarshal([]byte(task.Payload), taskPayload); err != nil {
			return "", errors.Wrap(err, "invalid TaskDatabaseSchemaUpdatePayload")
		}
		return taskPayload.Statement, nil
	case api.TaskDatabaseSchemaUpdateSDL:
		taskPayload := &api.TaskDatabaseSchemaUpdateSDLPayload{}
		if err := json.Unmarshal([]byte(task.Payload), taskPayload); err != nil {
			return "", errors.Wrap(err, "invalid TaskDatabaseSchemaUpdateSDLPayload")
		}
		return taskPayload.Statement, nil
	case api.TaskDatabaseDataUpdate:
		taskPayload := &api.TaskDatabaseDataUpdatePayload{}
		if err := json.Unmarshal([]byte(task.Payload), taskPayload); err != nil {
			return "", errors.Wrap(err, "invalid TaskDatabaseDataUpdatePayload")
		}
		return taskPayload.Statement, nil
	case api.TaskDatabaseSchemaUpdateGhostSync:
		taskPayload := &api.TaskDatabaseSchemaUpdateGhostSyncPayload{}
		if err := json.Unmarshal([]byte(task.Payload), taskPayload); err != nil {
			return "", errors.Wrap(err, "invalid TaskDatabaseSchemaUpdateGhostSyncPayload")
		}
		return taskPayload.Statement, nil
	default:
		return "", errors.Errorf("invalid task type %s", task.Type)
	}
}

func (*TaskCheckScheduler) getStmtTypeTaskCheck(_ context.Context, task *api.Task, creatorID int, database *api.Database, statement string) ([]*api.TaskCheckRunCreate, error) {
	if !api.IsStatementTypeCheckSupported(database.Instance.Engine) {
		return nil, nil
	}
	payload, err := json.Marshal(api.TaskCheckDatabaseStatementTypePayload{
		Statement: statement,
		DbType:    database.Instance.Engine,
		Charset:   database.CharacterSet,
		Collation: database.Collation,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal statement type payload: %v", task.Name)
	}
	return []*api.TaskCheckRunCreate{
		{
			CreatorID: creatorID,
			TaskID:    task.ID,
			Type:      api.TaskCheckDatabaseStatementType,
			Payload:   string(payload),
		},
	}, nil
}

func (s *TaskCheckScheduler) getSQLReviewTaskCheck(ctx context.Context, task *api.Task, creatorID int, database *api.Database, statement string) ([]*api.TaskCheckRunCreate, error) {
	if !api.IsSQLReviewSupported(database.Instance.Engine) {
		return nil, nil
	}
	policyID, err := s.store.GetSQLReviewPolicyIDByEnvID(ctx, task.Instance.EnvironmentID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get SQL review policy ID for task: %v, in environment: %v", task.Name, task.Instance.EnvironmentID)
	}
	payload, err := json.Marshal(api.TaskCheckDatabaseStatementAdvisePayload{
		Statement: statement,
		DbType:    database.Instance.Engine,
		Charset:   database.CharacterSet,
		Collation: database.Collation,
		PolicyID:  policyID,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal statement advise payload: %v", task.Name)
	}
	return []*api.TaskCheckRunCreate{
		{
			CreatorID: creatorID,
			TaskID:    task.ID,
			Type:      api.TaskCheckDatabaseStatementAdvise,
			Payload:   string(payload),
		},
	}, nil
}

func getSyntaxCheckTaskCheck(_ context.Context, task *api.Task, creatorID int, database *api.Database, statement string) ([]*api.TaskCheckRunCreate, error) {
	if !api.IsSyntaxCheckSupported(database.Instance.Engine) {
		return nil, nil
	}
	payload, err := json.Marshal(api.TaskCheckDatabaseStatementAdvisePayload{
		Statement: statement,
		DbType:    database.Instance.Engine,
		Charset:   database.CharacterSet,
		Collation: database.Collation,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal statement advise payload: %v", task.Name)
	}
	return []*api.TaskCheckRunCreate{
		{
			CreatorID: creatorID,
			TaskID:    task.ID,
			Type:      api.TaskCheckDatabaseStatementSyntax,
			Payload:   string(payload),
		},
	}, nil
}

func (*TaskCheckScheduler) getGeneralTaskCheck(_ context.Context, task *api.Task, creatorID int) ([]*api.TaskCheckRunCreate, error) {
	return []*api.TaskCheckRunCreate{
		{
			CreatorID: creatorID,
			TaskID:    task.ID,
			Type:      api.TaskCheckDatabaseConnect,
		},
		{
			CreatorID: creatorID,
			TaskID:    task.ID,
			Type:      api.TaskCheckInstanceMigrationSchema,
		},
	}, nil
}

func (*TaskCheckScheduler) getGhostTaskCheck(_ context.Context, task *api.Task, creatorID int) ([]*api.TaskCheckRunCreate, error) {
	if task.Type != api.TaskDatabaseSchemaUpdateGhostSync {
		return nil, nil
	}
	return []*api.TaskCheckRunCreate{
		{
			CreatorID: creatorID,
			TaskID:    task.ID,
			Type:      api.TaskCheckGhostSync,
		},
	}, nil
}

func (*TaskCheckScheduler) getPITRTaskCheck(_ context.Context, task *api.Task, creatorID int) ([]*api.TaskCheckRunCreate, error) {
	if task.Type != api.TaskDatabaseRestorePITRRestore {
		return nil, nil
	}
	return []*api.TaskCheckRunCreate{
		{
			CreatorID: creatorID,
			TaskID:    task.ID,
			Type:      api.TaskCheckPITRMySQL,
		},
	}, nil
}

func (s *TaskCheckScheduler) getLGTMTaskCheck(ctx context.Context, task *api.Task, creatorID int) ([]*api.TaskCheckRunCreate, error) {
	if !s.licenseService.IsFeatureEnabled(api.FeatureLGTM) {
		return nil, nil
	}
	issues, err := s.store.FindIssueStripped(ctx, &api.IssueFind{PipelineID: &task.PipelineID})
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if len(issues) > 1 {
		return nil, errors.Errorf("expect to find 0 or 1 issue, get %d", len(issues))
	}
	if len(issues) == 0 {
		// skip if no containing issue.
		return nil, nil
	}
	if issues[0].Project.LGTMCheckSetting.Value == api.LGTMValueDisabled {
		// don't schedule LGTM check if it's disabled.
		return nil, nil
	}
	approvalPolicy, err := s.store.GetPipelineApprovalPolicy(ctx, task.Instance.EnvironmentID)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if approvalPolicy.Value == api.PipelineApprovalValueManualNever {
		// don't schedule LGTM check if the approval policy is auto-approval.
		return nil, nil
	}
	return []*api.TaskCheckRunCreate{
		{
			CreatorID: creatorID,
			TaskID:    task.ID,
			Type:      api.TaskCheckIssueLGTM,
		},
	}, nil
}

// Returns true only if the task check run result is at least the minimum required level.
// For PendingApproval->Pending transitions, the minimum level is SUCCESS.
// For Pending->Running transitions, the minimum level is WARN.
// TODO(dragonly): refactor arguments.
func (s *Server) passCheck(ctx context.Context, task *api.Task, checkType api.TaskCheckType, allowedStatus api.TaskCheckStatus) (bool, error) {
	statusList := []api.TaskCheckRunStatus{api.TaskCheckRunDone, api.TaskCheckRunFailed}
	taskCheckRunFind := &api.TaskCheckRunFind{
		TaskID:     &task.ID,
		Type:       &checkType,
		StatusList: &statusList,
		Latest:     true,
	}

	taskCheckRunList, err := s.store.FindTaskCheckRun(ctx, taskCheckRunFind)
	if err != nil {
		return false, err
	}

	if len(taskCheckRunList) == 0 || taskCheckRunList[0].Status == api.TaskCheckRunFailed {
		log.Debug("Task is waiting for check to pass",
			zap.Int("task_id", task.ID),
			zap.String("task_name", task.Name),
			zap.String("task_type", string(task.Type)),
			zap.String("task_check_type", string(checkType)),
		)
		return false, nil
	}

	checkResult := &api.TaskCheckRunResultPayload{}
	if err := json.Unmarshal([]byte(taskCheckRunList[0].Result), checkResult); err != nil {
		return false, err
	}
	for _, result := range checkResult.ResultList {
		if result.Status.LessThan(allowedStatus) {
			log.Debug("Task is waiting for check to pass",
				zap.Int("task_id", task.ID),
				zap.String("task_name", task.Name),
				zap.String("task_type", string(task.Type)),
				zap.String("task_check_type", string(api.TaskCheckDatabaseConnect)),
			)
			return false, nil
		}
	}

	return true, nil
}
