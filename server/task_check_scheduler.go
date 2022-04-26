package server

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/db"
	"go.uber.org/zap"
)

// NewTaskCheckScheduler creates a task check scheduler.
func NewTaskCheckScheduler(logger *zap.Logger, server *Server) *TaskCheckScheduler {
	return &TaskCheckScheduler{
		l:         logger,
		executors: make(map[string]TaskCheckExecutor),
		server:    server,
	}
}

// TaskCheckScheduler is the task check scheduler.
type TaskCheckScheduler struct {
	l         *zap.Logger
	executors map[string]TaskCheckExecutor

	server *Server
}

// Run will run the task check scheduler once.
func (s *TaskCheckScheduler) Run(ctx context.Context, wg *sync.WaitGroup) {
	ticker := time.NewTicker(taskSchedulerInterval)
	defer ticker.Stop()
	defer wg.Done()
	s.l.Debug(fmt.Sprintf("Task check scheduler started and will run every %v", taskSchedulerInterval))
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
							err = fmt.Errorf("%v", r)
						}
						s.l.Error("Task check scheduler PANIC RECOVER", zap.Error(err), zap.Stack("stack"))
					}
				}()

				ctx := context.Background()

				// Inspect all running task checks
				taskCheckRunStatusList := []api.TaskCheckRunStatus{api.TaskCheckRunRunning}
				taskCheckRunFind := &api.TaskCheckRunFind{
					StatusList: &taskCheckRunStatusList,
				}
				taskCheckRunRawList, err := s.server.TaskCheckRunService.FindTaskCheckRunList(ctx, taskCheckRunFind)
				if err != nil {
					s.l.Error("Failed to retrieve running tasks", zap.Error(err))
					return
				}
				// TODO(dragonly): compose this
				var taskCheckRunList []*api.TaskCheckRun
				for _, raw := range taskCheckRunRawList {
					taskCheckRunList = append(taskCheckRunList, raw.ToTaskCheckRun())
				}
				for _, taskCheckRun := range taskCheckRunList {
					executor, ok := s.executors[string(taskCheckRun.Type)]
					if !ok {
						s.l.Error("Skip running task check run with unknown type",
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
								s.l.Error("Failed to marshal task check run result",
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
							_, err = s.server.TaskCheckRunService.PatchTaskCheckRunStatus(ctx, taskCheckRunStatusPatch)
							if err != nil {
								s.l.Error("Failed to mark task check run as DONE",
									zap.Int("id", taskCheckRun.ID),
									zap.Int("task_id", taskCheckRun.TaskID),
									zap.String("type", string(taskCheckRun.Type)),
									zap.Error(err),
								)
							}
						} else {
							s.l.Debug("Failed to run task check",
								zap.Int("id", taskCheckRun.ID),
								zap.Int("task_id", taskCheckRun.TaskID),
								zap.String("type", string(taskCheckRun.Type)),
								zap.Error(err),
							)
							bytes, marshalErr := json.Marshal(api.TaskCheckRunResultPayload{
								Detail: err.Error(),
							})
							if marshalErr != nil {
								s.l.Error("Failed to marshal task check run result",
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
							_, err = s.server.TaskCheckRunService.PatchTaskCheckRunStatus(ctx, taskCheckRunStatusPatch)
							if err != nil {
								s.l.Error("Failed to mark task check run as FAILED",
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
func (s *TaskCheckScheduler) Register(taskType string, executor TaskCheckExecutor) {
	if executor == nil {
		panic("scheduler: Register executor is nil for task type: " + taskType)
	}
	if _, dup := s.executors[taskType]; dup {
		panic("scheduler: Register called twice for task type: " + taskType)
	}
	s.executors[taskType] = executor
}

// Returns true if we meet either of the following conditions:
//   1. Task has a non-default value and no task check has run before (so we are about to kick of the check the first time)
//   2. The specified EarliestAllowedTs has elapsed, so we need to rerun the check to unblock the task.
// On the other hand, we would also rerun the check if user has modified EarliestAllowedTs. This is handled separately in the task patch handler.
func (s *TaskCheckScheduler) shouldScheduleTimingTaskCheck(ctx context.Context, task *api.Task, forceSchedule bool) (bool, error) {
	statusList := []api.TaskCheckRunStatus{api.TaskCheckRunDone, api.TaskCheckRunFailed, api.TaskCheckRunRunning}
	taskCheckType := api.TaskCheckGeneralEarliestAllowedTime
	taskCheckRunFind := &api.TaskCheckRunFind{
		TaskID:     &task.ID,
		Type:       &taskCheckType,
		StatusList: &statusList,
		Latest:     true,
	}
	taskCheckRunList, err := s.server.TaskCheckRunService.FindTaskCheckRunList(ctx, taskCheckRunFind)
	if err != nil {
		return false, err
	}

	// If there is not any task check scheduled before, we should only schedule one if user has specified a non-default value.
	if len(taskCheckRunList) == 0 {
		return task.EarliestAllowedTs != 0, nil
	}

	if forceSchedule {
		return true, nil
	}

	if time.Now().After(time.Unix(task.EarliestAllowedTs, 0)) {
		checkResult := &api.TaskCheckRunResultPayload{}
		if err := json.Unmarshal([]byte(taskCheckRunList[0].Result), checkResult); err != nil {
			return false, err
		}
		if checkResult.ResultList[0].Status == api.TaskCheckStatusSuccess {
			return false, nil
		}
		return true, nil
	}

	return false, nil
}

// ScheduleCheckIfNeeded schedules a check if needed.
func (s *TaskCheckScheduler) ScheduleCheckIfNeeded(ctx context.Context, task *api.Task, creatorID int, skipIfAlreadyTerminated bool) (*api.Task, error) {
	// the following block is for timing task check
	{
		// we only set skipIfAlreadyTerminated to false when user explicitly want to reschedule a taskCheck
		flag, err := s.shouldScheduleTimingTaskCheck(ctx, task, !skipIfAlreadyTerminated /* forceSchedule */)
		if err != nil {
			return nil, err
		}

		if flag {
			taskCheckPayload, err := json.Marshal(api.TaskCheckEarliestAllowedTimePayload{
				EarliestAllowedTs: task.EarliestAllowedTs,
			})
			if err != nil {
				return nil, err
			}
			_, err = s.server.TaskCheckRunService.CreateTaskCheckRunIfNeeded(ctx, &api.TaskCheckRunCreate{
				CreatorID:               creatorID,
				TaskID:                  task.ID,
				Type:                    api.TaskCheckGeneralEarliestAllowedTime,
				Payload:                 string(taskCheckPayload),
				SkipIfAlreadyTerminated: false,
			})
			if err != nil {
				return nil, err
			}
		}
	}

	if task.Type == api.TaskDatabaseSchemaUpdate || task.Type == api.TaskDatabaseDataUpdate {
		statement := ""

		if task.Type == api.TaskDatabaseSchemaUpdate {
			taskPayload := &api.TaskDatabaseSchemaUpdatePayload{}
			if err := json.Unmarshal([]byte(task.Payload), taskPayload); err != nil {
				return nil, fmt.Errorf("invalid database schema update payload: %w", err)
			}
			statement = taskPayload.Statement
		} else {
			taskPayload := &api.TaskDatabaseDataUpdatePayload{}
			if err := json.Unmarshal([]byte(task.Payload), taskPayload); err != nil {
				return nil, fmt.Errorf("invalid database data update payload: %w", err)
			}
			statement = taskPayload.Statement
		}

		databaseFind := &api.DatabaseFind{
			ID: task.DatabaseID,
		}
		database, err := s.server.composeDatabaseByFind(ctx, databaseFind)
		if err != nil {
			return nil, err
		}
		if database == nil {
			return nil, fmt.Errorf("database ID not found %v", task.DatabaseID)
		}

		_, err = s.server.TaskCheckRunService.CreateTaskCheckRunIfNeeded(ctx, &api.TaskCheckRunCreate{
			CreatorID:               creatorID,
			TaskID:                  task.ID,
			Type:                    api.TaskCheckDatabaseConnect,
			SkipIfAlreadyTerminated: skipIfAlreadyTerminated,
		})
		if err != nil {
			return nil, err
		}

		_, err = s.server.TaskCheckRunService.CreateTaskCheckRunIfNeeded(ctx, &api.TaskCheckRunCreate{
			CreatorID:               creatorID,
			TaskID:                  task.ID,
			Type:                    api.TaskCheckInstanceMigrationSchema,
			SkipIfAlreadyTerminated: skipIfAlreadyTerminated,
		})
		if err != nil {
			return nil, err
		}

		// For now we only supported MySQL dialect syntax and compatibility check
		if database.Instance.Engine == db.MySQL || database.Instance.Engine == db.TiDB {
			payload, err := json.Marshal(api.TaskCheckDatabaseStatementAdvisePayload{
				Statement: statement,
				DbType:    database.Instance.Engine,
				Charset:   database.CharacterSet,
				Collation: database.Collation,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to marshal statement advise payload: %v, err: %w", task.Name, err)
			}
			_, err = s.server.TaskCheckRunService.CreateTaskCheckRunIfNeeded(ctx, &api.TaskCheckRunCreate{
				CreatorID:               creatorID,
				TaskID:                  task.ID,
				Type:                    api.TaskCheckDatabaseStatementSyntax,
				Payload:                 string(payload),
				SkipIfAlreadyTerminated: skipIfAlreadyTerminated,
			})
			if err != nil {
				return nil, err
			}

			if s.server.feature(api.FeatureBackwardCompatibility) {
				_, err = s.server.TaskCheckRunService.CreateTaskCheckRunIfNeeded(ctx, &api.TaskCheckRunCreate{
					CreatorID:               creatorID,
					TaskID:                  task.ID,
					Type:                    api.TaskCheckDatabaseStatementCompatibility,
					Payload:                 string(payload),
					SkipIfAlreadyTerminated: skipIfAlreadyTerminated,
				})
				if err != nil {
					return nil, err
				}
			}
		}

		if s.server.feature(api.FeatureSchemaReviewPolicy) {
			err = s.server.createTaskCheckRunIfNeededBySchemaReviewPolicy(
				ctx,
				creatorID,
				task.ID,
				task.Instance.EnvironmentID,
				api.TaskCheckDatabaseStatementAdvisePayload{
					Statement: statement,
					DbType:    database.Instance.Engine,
					Charset:   database.CharacterSet,
					Collation: database.Collation,
				},
				skipIfAlreadyTerminated)
			if err != nil {
				return nil, err
			}
		}

		taskCheckRunFind := &api.TaskCheckRunFind{
			TaskID: &task.ID,
		}
		taskCheckRunRawList, err := s.server.TaskCheckRunService.FindTaskCheckRunList(ctx, taskCheckRunFind)
		if err != nil {
			return nil, err
		}
		// TODO(dragonly): compose this
		var taskCheckRunList []*api.TaskCheckRun
		for _, raw := range taskCheckRunRawList {
			taskCheckRunList = append(taskCheckRunList, raw.ToTaskCheckRun())
		}
		task.TaskCheckRunList = taskCheckRunList

		return task, err
	}
	return task, nil
}

// createTaskCheckRunIfNeededBySchemaReviewPolicy gets the corresponding SchemaReviewPolicy
// and generates specific TaskCheckRuns based on it.
func (s *Server) createTaskCheckRunIfNeededBySchemaReviewPolicy(
	ctx context.Context,
	creatotID int,
	taskID int,
	environmentID int,
	basePayload api.TaskCheckDatabaseStatementAdvisePayload,
	skipIfAlreadyTerminated bool,
) error {
	policy, err := s.store.GetSchemaReviewPolicyByEnvID(ctx, environmentID)
	if err != nil {
		return err
	}
	for _, rule := range policy.RuleList {
		if rule.Level == api.SchemaRuleLevelDisabled {
			continue
		}
		taskCheckType, payload, err := getTaskCheckTypeAndPayloadByRule(rule, basePayload)
		if err != nil {
			s.l.Warn(err.Error())
			continue
		}
		fmt.Println("taskCheckType: ", taskCheckType)
		_, err = s.TaskCheckRunService.CreateTaskCheckRunIfNeeded(ctx, &api.TaskCheckRunCreate{
			CreatorID:               creatotID,
			TaskID:                  taskID,
			Type:                    taskCheckType,
			Payload:                 payload,
			SkipIfAlreadyTerminated: skipIfAlreadyTerminated,
		})
		if err != nil {
			fmt.Println("Why?", err)
			return err
		}
	}
	return nil
}

// Returns true only if there is NO warning and error. User can still manually run the task if there is warning.
// But this method is used for gating the automatic run, so we are more cautious here.
func (s *Server) passCheck(ctx context.Context, server *Server, task *api.Task, checkType api.TaskCheckType) (bool, error) {
	statusList := []api.TaskCheckRunStatus{api.TaskCheckRunDone, api.TaskCheckRunFailed}
	taskCheckRunFind := &api.TaskCheckRunFind{
		TaskID:     &task.ID,
		Type:       &checkType,
		StatusList: &statusList,
		Latest:     true,
	}

	taskCheckRunList, err := server.TaskCheckRunService.FindTaskCheckRunList(ctx, taskCheckRunFind)
	if err != nil {
		return false, err
	}

	if len(taskCheckRunList) == 0 || taskCheckRunList[0].Status == api.TaskCheckRunFailed {
		server.l.Debug("Task is waiting for check to pass",
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
		if result.Status == api.TaskCheckStatusError || result.Status == api.TaskCheckStatusWarn {
			server.l.Debug("Task is waiting for check to pass",
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
