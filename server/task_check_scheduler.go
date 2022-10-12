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
)

// NewTaskCheckScheduler creates a task check scheduler.
func NewTaskCheckScheduler(server *Server) *TaskCheckScheduler {
	return &TaskCheckScheduler{
		executors: make(map[api.TaskCheckType]TaskCheckExecutor),
		server:    server,
	}
}

// TaskCheckScheduler is the task check scheduler.
type TaskCheckScheduler struct {
	executors map[api.TaskCheckType]TaskCheckExecutor

	server *Server
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
				taskCheckRunList, err := s.server.store.FindTaskCheckRun(ctx, taskCheckRunFind)
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
						checkResultList, err := executor.Run(ctx, s.server, taskCheckRun)

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
							_, err = s.server.store.PatchTaskCheckRunStatus(ctx, taskCheckRunStatusPatch)
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
							_, err = s.server.store.PatchTaskCheckRunStatus(ctx, taskCheckRunStatusPatch)
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

// ScheduleCheckIfNeeded schedules a check if needed.
func (s *TaskCheckScheduler) ScheduleCheckIfNeeded(ctx context.Context, task *api.Task, creatorID int) (*api.Task, error) {
	if err := s.scheduleLGTMTaskCheck(ctx, task, creatorID); err != nil {
		return nil, errors.Wrap(err, "failed to schedule LGTM task check")
	}

	if err := s.schedulePITRTaskCheck(ctx, task, creatorID); err != nil {
		return nil, errors.Wrap(err, "failed to schedule backup/PITR task check")
	}

	if task.Type != api.TaskDatabaseSchemaUpdate && task.Type != api.TaskDatabaseSchemaUpdateSDL && task.Type != api.TaskDatabaseDataUpdate && task.Type != api.TaskDatabaseSchemaUpdateGhostSync {
		return task, nil
	}

	if err := s.scheduleGeneralTaskCheck(ctx, task, creatorID); err != nil {
		return nil, errors.Wrap(err, "failed to schedule general task check")
	}

	if err := s.scheduleGhostTaskCheck(ctx, task, creatorID); err != nil {
		return nil, errors.Wrap(err, "failed to schedule gh-ost task check")
	}

	statement, err := s.getStatement(task)
	if err != nil {
		return nil, err
	}
	database, err := s.server.store.GetDatabase(ctx, &api.DatabaseFind{ID: task.DatabaseID})
	if err != nil {
		return nil, err
	}
	if database == nil {
		return nil, errors.Errorf("database ID not found %v", task.DatabaseID)
	}

	if err := s.scheduleSyntaxCheckTaskCheck(ctx, task, creatorID, database, statement); err != nil {
		return nil, errors.Wrap(err, "failed to schedule syntax check task check")
	}

	if err := s.scheduleSQLReviewTaskCheck(ctx, task, creatorID, database, statement); err != nil {
		return nil, errors.Wrap(err, "failed to schedule SQL review task check")
	}

	if err := s.scheduleStmtTypeTaskCheck(ctx, task, creatorID, database, statement); err != nil {
		return nil, errors.Wrap(err, "failed to schedule statement type task check")
	}

	taskCheckRunFind := &api.TaskCheckRunFind{
		TaskID: &task.ID,
	}
	taskCheckRunList, err := s.server.store.FindTaskCheckRun(ctx, taskCheckRunFind)
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
		taskPayload := &api.TaskDatabaseSchemaUpdatePayload{}
		if err := json.Unmarshal([]byte(task.Payload), taskPayload); err != nil {
			return "", errors.Wrap(err, "invalid TaskDatabaseSchemaUpdatePayload")
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

func (s *TaskCheckScheduler) scheduleStmtTypeTaskCheck(ctx context.Context, task *api.Task, creatorID int, database *api.Database, statement string) error {
	if !api.IsStatementTypeCheckSupported(database.Instance.Engine) {
		return nil
	}
	payload, err := json.Marshal(api.TaskCheckDatabaseStatementTypePayload{
		Statement: statement,
		DbType:    database.Instance.Engine,
		Charset:   database.CharacterSet,
		Collation: database.Collation,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to marshal statement type payload: %v", task.Name)
	}
	if _, err := s.server.store.CreateTaskCheckRunIfNeeded(ctx, &api.TaskCheckRunCreate{
		CreatorID: creatorID,
		TaskID:    task.ID,
		Type:      api.TaskCheckDatabaseStatementType,
		Payload:   string(payload),
	}); err != nil {
		return err
	}
	return nil
}

func (s *TaskCheckScheduler) scheduleSQLReviewTaskCheck(ctx context.Context, task *api.Task, creatorID int, database *api.Database, statement string) error {
	if !api.IsSQLReviewSupported(database.Instance.Engine, s.server.profile.Mode) {
		return nil
	}
	policyID, err := s.server.store.GetSQLReviewPolicyIDByEnvID(ctx, task.Instance.EnvironmentID)
	if err != nil {
		return errors.Wrapf(err, "failed to get SQL review policy ID for task: %v, in environment: %v", task.Name, task.Instance.EnvironmentID)
	}
	payload, err := json.Marshal(api.TaskCheckDatabaseStatementAdvisePayload{
		Statement: statement,
		DbType:    database.Instance.Engine,
		Charset:   database.CharacterSet,
		Collation: database.Collation,
		PolicyID:  policyID,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to marshal statement advise payload: %v", task.Name)
	}
	if _, err := s.server.store.CreateTaskCheckRunIfNeeded(ctx, &api.TaskCheckRunCreate{
		CreatorID: creatorID,
		TaskID:    task.ID,
		Type:      api.TaskCheckDatabaseStatementAdvise,
		Payload:   string(payload),
	}); err != nil {
		return err
	}
	return nil
}

func (s *TaskCheckScheduler) scheduleSyntaxCheckTaskCheck(ctx context.Context, task *api.Task, creatorID int, database *api.Database, statement string) error {
	if !api.IsSyntaxCheckSupported(database.Instance.Engine, s.server.profile.Mode) {
		return nil
	}
	payload, err := json.Marshal(api.TaskCheckDatabaseStatementAdvisePayload{
		Statement: statement,
		DbType:    database.Instance.Engine,
		Charset:   database.CharacterSet,
		Collation: database.Collation,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to marshal statement advise payload: %v", task.Name)
	}
	if _, err := s.server.store.CreateTaskCheckRunIfNeeded(ctx, &api.TaskCheckRunCreate{
		CreatorID: creatorID,
		TaskID:    task.ID,
		Type:      api.TaskCheckDatabaseStatementSyntax,
		Payload:   string(payload),
	}); err != nil {
		return err
	}
	return nil
}

func (s *TaskCheckScheduler) scheduleGeneralTaskCheck(ctx context.Context, task *api.Task, creatorID int) error {
	if _, err := s.server.store.CreateTaskCheckRunIfNeeded(ctx, &api.TaskCheckRunCreate{
		CreatorID: creatorID,
		TaskID:    task.ID,
		Type:      api.TaskCheckDatabaseConnect,
	}); err != nil {
		return err
	}

	if _, err := s.server.store.CreateTaskCheckRunIfNeeded(ctx, &api.TaskCheckRunCreate{
		CreatorID: creatorID,
		TaskID:    task.ID,
		Type:      api.TaskCheckInstanceMigrationSchema,
	}); err != nil {
		return err
	}

	return nil
}

func (s *TaskCheckScheduler) scheduleGhostTaskCheck(ctx context.Context, task *api.Task, creatorID int) error {
	if task.Type != api.TaskDatabaseSchemaUpdateGhostSync {
		return nil
	}
	if _, err := s.server.store.CreateTaskCheckRunIfNeeded(ctx, &api.TaskCheckRunCreate{
		CreatorID: creatorID,
		TaskID:    task.ID,
		Type:      api.TaskCheckGhostSync,
	}); err != nil {
		return err
	}
	return nil
}

func (s *TaskCheckScheduler) schedulePITRTaskCheck(ctx context.Context, task *api.Task, creatorID int) error {
	if task.Type != api.TaskDatabaseRestorePITRRestore {
		return nil
	}
	if _, err := s.server.store.CreateTaskCheckRunIfNeeded(ctx, &api.TaskCheckRunCreate{
		CreatorID: creatorID,
		TaskID:    task.ID,
		Type:      api.TaskCheckPITRMySQL,
	}); err != nil {
		return err
	}
	return nil
}

func (s *TaskCheckScheduler) scheduleLGTMTaskCheck(ctx context.Context, task *api.Task, creatorID int) error {
	if !s.server.feature(api.FeatureLGTM) {
		return nil
	}
	issue, err := s.server.store.GetIssueByPipelineID(ctx, task.PipelineID)
	if err != nil {
		return err
	}
	if issue == nil {
		// skip if no containing issue.
		return nil
	}
	if issue.Project.LGTMCheckSetting.Value == api.LGTMValueDisabled {
		// don't schedule LGTM check if it's disabled.
		return nil
	}
	if _, err := s.server.store.CreateTaskCheckRunIfNeeded(ctx, &api.TaskCheckRunCreate{
		CreatorID: creatorID,
		TaskID:    task.ID,
		Type:      api.TaskCheckIssueLGTM,
	}); err != nil {
		return err
	}
	return nil
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
