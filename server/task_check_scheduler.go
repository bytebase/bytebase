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

const (
	taskCheckScheduleInterval = time.Duration(1) * time.Second
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
