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
func (s *TaskCheckScheduler) Run() error {
	go func() {
		s.l.Debug(fmt.Sprintf("Task check scheduler started and will run every %v", taskSchedulerInterval))
		runningTaskChecks := make(map[int]bool)
		mu := sync.RWMutex{}
		for {
			func() {
				defer func() {
					if r := recover(); r != nil {
						err, ok := r.(error)
						if !ok {
							err = fmt.Errorf("%v", r)
						}
						s.l.Error("Task check scheduler PANIC RECOVER", zap.Error(err))
					}
				}()

				ctx := context.Background()

				// Inspect all running task checks
				taskCheckRunStatusList := []api.TaskCheckRunStatus{api.TaskCheckRunRunning}
				taskCheckRunFind := &api.TaskCheckRunFind{
					StatusList: &taskCheckRunStatusList,
				}
				taskCheckRunList, err := s.server.TaskCheckRunService.FindTaskCheckRunList(ctx, taskCheckRunFind)
				if err != nil {
					s.l.Error("Failed to retrieve running tasks", zap.Error(err))
					return
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

			time.Sleep(taskSchedulerInterval)
		}
	}()

	return nil
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

// ScheduleCheckIfNeeded schedules a check if needed.
func (s *TaskCheckScheduler) ScheduleCheckIfNeeded(ctx context.Context, task *api.Task, creatorID int, skipIfAlreadyTerminated bool) (*api.Task, error) {
	// All task should pass timing task check.
	if task.NotBeforeTs != 0 {
		// Since Time is an auto-increment value, a task would generate a timing task check every time before executing.
		// However, this is an annoying behavior, and we adopt the following logic to ease this problem:
		// 		1. if no timing check has been scheduled yet, schedule one
		// 		2. if one has been scheduled before:
		//			a. succeed
		//				check again, since users are allowed to change this field
		//			b. failed
		//     			*** check if it has passed the not_before_ts ***
		// 					b-1. passed
		//						schedule a new timing task check
		// 					b-2. not passed
		//						do nothing
		// Notice that if no timing task check has been run before, the value of isTimingTaskCheckPassed would still be false
		isTimingTaskCheckPassed, err := s.server.passCheck(ctx, s.server, task, api.TaskCheckGeneralEarliestAllowedTime)
		if err != nil {
			return nil, err
		}
		if isTimingTaskCheckPassed {
			// Since we allowed user to modify 'notBeforeTs' after creating a task, it is possible previous passed timing check would fail this time.
			// So, we should create one if necessary
			if time.Now().Before(time.Unix(task.NotBeforeTs, 0)) {
				_, err = s.server.TaskCheckRunService.CreateTaskCheckRunIfNeeded(ctx, &api.TaskCheckRunCreate{
					CreatorID:               creatorID,
					TaskID:                  task.ID,
					Type:                    api.TaskCheckGeneralEarliestAllowedTime,
					SkipIfAlreadyTerminated: false,
				})
				if err != nil {
					return nil, err
				}
			}
		} else {
			if time.Now().Before(time.Unix(task.NotBeforeTs, 0)) {
				// Either no taskCheck had been scheduled or previous one had failed,
				// Create a taskCheck if not exist (by setting SkipIfAlreadyTerminated to true)
				_, err = s.server.TaskCheckRunService.CreateTaskCheckRunIfNeeded(ctx, &api.TaskCheckRunCreate{
					CreatorID:               creatorID,
					TaskID:                  task.ID,
					Type:                    api.TaskCheckGeneralEarliestAllowedTime,
					SkipIfAlreadyTerminated: true,
				})
				if err != nil {
					return nil, err
				}
				return task, nil
			}

			// If previous task check had failed and the time has passed now, schedule a new task check.
			// And this time it shall succeed.
			_, err = s.server.TaskCheckRunService.CreateTaskCheckRunIfNeeded(ctx, &api.TaskCheckRunCreate{
				CreatorID:               creatorID,
				TaskID:                  task.ID,
				Type:                    api.TaskCheckGeneralEarliestAllowedTime,
				SkipIfAlreadyTerminated: false,
			})
			if err != nil {
				return nil, err
			}
		}
	}

	if task.Type == api.TaskDatabaseSchemaUpdate {
		taskPayload := &api.TaskDatabaseSchemaUpdatePayload{}
		if err := json.Unmarshal([]byte(task.Payload), taskPayload); err != nil {
			return nil, fmt.Errorf("invalid database schema update payload: %w", err)
		}

		databaseFind := &api.DatabaseFind{
			ID: task.DatabaseID,
		}
		database, err := s.server.composeDatabaseByFind(ctx, databaseFind)
		if err != nil {
			return nil, err
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
				Statement: taskPayload.Statement,
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

		taskCheckRunFind := &api.TaskCheckRunFind{
			TaskID: &task.ID,
		}
		task.TaskCheckRunList, err = s.server.TaskCheckRunService.FindTaskCheckRunList(ctx, taskCheckRunFind)
		if err != nil {
			return nil, err
		}

		return task, err
	}
	return task, nil
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
			zap.String("task_check_type", string(api.TaskCheckDatabaseConnect)),
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
