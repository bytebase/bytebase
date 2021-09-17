package server

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/bytebase/bytebase/api"
	"go.uber.org/zap"
)

const (
	TASK_CHECK_SCHEDULE_INTERVAL = time.Duration(1) * time.Second
)

func NewTaskCheckScheduler(logger *zap.Logger, server *Server) *TaskCheckScheduler {
	return &TaskCheckScheduler{
		l:         logger,
		executors: make(map[string]TaskCheckExecutor),
		server:    server,
	}
}

type TaskCheckScheduler struct {
	l         *zap.Logger
	executors map[string]TaskCheckExecutor

	server *Server
}

func (s *TaskCheckScheduler) Run() error {
	go func() {
		s.l.Debug(fmt.Sprintf("Task check scheduler started and will run every %v", TASK_SCHEDULE_INTERVAL))
		runningTaskChecks := make(map[int]bool)
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

				// Inspect all running task checks
				taskCheckRunStatusList := []api.TaskCheckRunStatus{api.TaskCheckRunRunning}
				taskCheckRunFind := &api.TaskCheckRunFind{
					StatusList: &taskCheckRunStatusList,
				}
				taskCheckRunList, err := s.server.TaskCheckRunService.FindTaskCheckRunList(context.Background(), taskCheckRunFind)
				if err != nil {
					s.l.Error("Failed to retrieve running tasks", zap.Error(err))
				}

				for _, taskCheckRun := range taskCheckRunList {
					executor, ok := s.executors[string(taskCheckRun.Type)]
					if !ok {
						s.l.Error("Skip running task check run with unknown type",
							zap.Int("id", taskCheckRun.ID),
							zap.Int("task_id", taskCheckRun.TaskId),
							zap.String("type", string(taskCheckRun.Type)),
						)
						continue
					}

					if _, ok := runningTaskChecks[taskCheckRun.ID]; ok {
						continue
					}

					runningTaskChecks[taskCheckRun.ID] = true

					go func(taskCheckRun *api.TaskCheckRun) {
						defer func() {
							delete(runningTaskChecks, taskCheckRun.ID)
						}()
						checkResultList, err := executor.Run(context.Background(), s.server, taskCheckRun)

						if err == nil {
							bytes, err := json.Marshal(api.TaskCheckRunResultPayload{
								ResultList: checkResultList,
							})
							if err != nil {
								s.l.Error("Failed to marshal task check run result",
									zap.Int("id", taskCheckRun.ID),
									zap.Int("task_id", taskCheckRun.TaskId),
									zap.String("type", string(taskCheckRun.Type)),
									zap.Error(err),
								)
								return
							}

							taskCheckRunStatusPatch := &api.TaskCheckRunStatusPatch{
								ID:        &taskCheckRun.ID,
								UpdaterId: api.SYSTEM_BOT_ID,
								Status:    api.TaskCheckRunDone,
								Result:    string(bytes),
							}
							_, err = s.server.TaskCheckRunService.PatchTaskCheckRunStatus(context.Background(), taskCheckRunStatusPatch)
							if err != nil {
								s.l.Error("Failed to mark task check run as DONE",
									zap.Int("id", taskCheckRun.ID),
									zap.Int("task_id", taskCheckRun.TaskId),
									zap.String("type", string(taskCheckRun.Type)),
									zap.Error(err),
								)
							}
						} else {
							s.l.Debug("Failed to run task check",
								zap.Int("id", taskCheckRun.ID),
								zap.Int("task_id", taskCheckRun.TaskId),
								zap.String("type", string(taskCheckRun.Type)),
								zap.Error(err),
							)
							taskCheckRunStatusPatch := &api.TaskCheckRunStatusPatch{
								ID:        &taskCheckRun.ID,
								UpdaterId: api.SYSTEM_BOT_ID,
								Status:    api.TaskCheckRunFailed,
								Comment:   err.Error(),
							}
							_, err = s.server.TaskCheckRunService.PatchTaskCheckRunStatus(context.Background(), taskCheckRunStatusPatch)
							if err != nil {
								s.l.Error("Failed to mark task check run as FAILED",
									zap.Int("id", taskCheckRun.ID),
									zap.Int("task_id", taskCheckRun.TaskId),
									zap.String("type", string(taskCheckRun.Type)),
									zap.Error(err),
								)
							}
						}
					}(taskCheckRun)
				}
			}()

			time.Sleep(TASK_SCHEDULE_INTERVAL)
		}
	}()

	return nil
}

func (s *TaskCheckScheduler) Register(taskType string, executor TaskCheckExecutor) {
	if executor == nil {
		panic("scheduler: Register executor is nil for task type: " + taskType)
	}
	if _, dup := s.executors[taskType]; dup {
		panic("scheduler: Register called twice for task type: " + taskType)
	}
	s.executors[taskType] = executor
}

func (s *TaskCheckScheduler) ScheduleCheckIfNeeded(ctx context.Context, task *api.Task, creatorId int, skipIfAlreadyDone bool) (*api.Task, error) {
	if task.Type == api.TaskDatabaseSchemaUpdate {
		taskPayload := &api.TaskDatabaseSchemaUpdatePayload{}
		if err := json.Unmarshal([]byte(task.Payload), taskPayload); err != nil {
			return nil, fmt.Errorf("invalid database schema update payload: %w", err)
		}

		databaseFind := &api.DatabaseFind{
			ID: task.DatabaseId,
		}
		database, err := s.server.ComposeDatabaseByFind(context.Background(), databaseFind)
		if err != nil {
			return nil, err
		}

		payload, err := json.Marshal(api.TaskCheckDatabaseStatementAdvisePayload{
			Statement: taskPayload.Statement,
			DbType:    database.Instance.Engine,
			Charset:   database.CharacterSet,
			Collation: database.Collation,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to marshal activity after changing the task status: %v, err: %w", task.Name, err)
		}
		_, err = s.server.TaskCheckRunService.CreateTaskCheckRunIfNeeded(ctx, &api.TaskCheckRunCreate{
			CreatorId:         creatorId,
			TaskId:            task.ID,
			Type:              api.TaskCheckDatabaseStatementFakeAdvise,
			Payload:           string(payload),
			SkipIfAlreadyDone: skipIfAlreadyDone,
		})
		if err != nil {
			return nil, err
		}

		_, err = s.server.TaskCheckRunService.CreateTaskCheckRunIfNeeded(ctx, &api.TaskCheckRunCreate{
			CreatorId:         creatorId,
			TaskId:            task.ID,
			Type:              api.TaskCheckDatabaseStatementSyntax,
			Payload:           string(payload),
			SkipIfAlreadyDone: skipIfAlreadyDone,
		})
		if err != nil {
			return nil, err
		}

		taskCheckRunFind := &api.TaskCheckRunFind{
			TaskId: &task.ID,
		}
		task.TaskCheckRunList, err = s.server.TaskCheckRunService.FindTaskCheckRunList(ctx, taskCheckRunFind)
		if err != nil {
			return nil, err
		}

		return task, err
	}
	return task, nil
}
