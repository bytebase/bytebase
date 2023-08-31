// Package taskcheck is a runner for task checks.
package taskcheck

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/state"
	enterpriseAPI "github.com/bytebase/bytebase/backend/enterprise/api"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store"
)

const (
	taskCheckSchedulerInterval = time.Duration(1) * time.Second
)

// NewScheduler creates a task check scheduler.
func NewScheduler(store *store.Store, licenseService enterpriseAPI.LicenseService, stateCfg *state.State) *Scheduler {
	return &Scheduler{
		store:          store,
		licenseService: licenseService,
		stateCfg:       stateCfg,
		executors:      make(map[api.TaskCheckType]Executor),
	}
}

// Scheduler is the task check scheduler.
type Scheduler struct {
	store          *store.Store
	licenseService enterpriseAPI.LicenseService
	stateCfg       *state.State
	executors      map[api.TaskCheckType]Executor
}

// Run will run the task check scheduler once.
func (s *Scheduler) Run(ctx context.Context, wg *sync.WaitGroup) {
	ticker := time.NewTicker(taskCheckSchedulerInterval)
	defer ticker.Stop()
	defer wg.Done()
	log.Debug(fmt.Sprintf("Task check scheduler started and will run every %v", taskCheckSchedulerInterval))
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
				taskCheckRunFind := &store.TaskCheckRunFind{
					StatusList: &taskCheckRunStatusList,
				}
				taskCheckRuns, err := s.store.ListTaskCheckRuns(ctx, taskCheckRunFind)
				if err != nil {
					log.Error("Failed to retrieve running tasks", zap.Error(err))
					return
				}
				for _, taskCheckRun := range taskCheckRuns {
					executor, ok := s.executors[taskCheckRun.Type]
					if !ok {
						log.Error("Skip running task check run with unknown type",
							zap.Int("id", taskCheckRun.ID),
							zap.Int("task_id", taskCheckRun.TaskID),
							zap.String("type", string(taskCheckRun.Type)),
						)
						continue
					}

					if _, ok := s.stateCfg.RunningTaskChecks.Load(taskCheckRun.ID); ok {
						continue
					}

					task, err := s.store.GetTaskV2ByID(ctx, taskCheckRun.TaskID)
					if err != nil {
						log.Error("Failed to get task for task check run",
							zap.Int("task_check_run_id", taskCheckRun.ID),
							zap.Int("task_id", taskCheckRun.TaskID),
							zap.String("type", string(taskCheckRun.Type)),
							zap.Error(err),
						)
						taskCheckRunStatusPatch := &store.TaskCheckRunStatusPatch{
							ID:        &taskCheckRun.ID,
							UpdaterID: api.SystemBotID,
							Status:    api.TaskCheckRunFailed,
							Code:      common.Internal,
						}
						if err := s.store.PatchTaskCheckRunStatus(ctx, taskCheckRunStatusPatch); err != nil {
							log.Error("Failed to mark task check run as FAILED",
								zap.Int("id", taskCheckRun.ID),
								zap.Int("task_id", taskCheckRun.TaskID),
								zap.String("type", string(taskCheckRun.Type)),
								zap.Error(err),
							)
						}
						continue
					}

					s.stateCfg.Lock()
					if s.stateCfg.InstanceOutstandingConnections[task.InstanceID] >= state.InstanceMaximumConnectionNumber {
						s.stateCfg.Unlock()
						continue
					}
					s.stateCfg.InstanceOutstandingConnections[task.InstanceID]++
					s.stateCfg.Unlock()

					s.stateCfg.RunningTaskChecks.Store(taskCheckRun.ID, true)
					go func(taskCheckRun *store.TaskCheckRunMessage, task *store.TaskMessage) {
						defer func() {
							s.stateCfg.RunningTaskChecks.Delete(taskCheckRun.ID)
							s.stateCfg.Lock()
							s.stateCfg.InstanceOutstandingConnections[task.InstanceID]--
							s.stateCfg.Unlock()
						}()
						checkResultList, err := runExecutorOnce(ctx, executor, taskCheckRun, task)

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

							taskCheckRunStatusPatch := &store.TaskCheckRunStatusPatch{
								ID:        &taskCheckRun.ID,
								UpdaterID: api.SystemBotID,
								Status:    api.TaskCheckRunDone,
								Code:      common.Ok,
								Result:    string(bytes),
							}
							if err := s.store.PatchTaskCheckRunStatus(ctx, taskCheckRunStatusPatch); err != nil {
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

							taskCheckRunStatusPatch := &store.TaskCheckRunStatusPatch{
								ID:        &taskCheckRun.ID,
								UpdaterID: api.SystemBotID,
								Status:    api.TaskCheckRunFailed,
								Code:      common.ErrorCode(err),
								Result:    string(bytes),
							}
							if err := s.store.PatchTaskCheckRunStatus(ctx, taskCheckRunStatusPatch); err != nil {
								log.Error("Failed to mark task check run as FAILED",
									zap.Int("id", taskCheckRun.ID),
									zap.Int("task_id", taskCheckRun.TaskID),
									zap.String("type", string(taskCheckRun.Type)),
									zap.Error(err),
								)
							}
						}
					}(taskCheckRun, task)
				}
			}()
		case <-ctx.Done(): // if cancel() execute
			return
		}
	}
}

// Register will register the task check executor.
func (s *Scheduler) Register(taskType api.TaskCheckType, executor Executor) {
	if executor == nil {
		panic("scheduler: Register executor is nil for task type: " + taskType)
	}
	if _, dup := s.executors[taskType]; dup {
		panic("scheduler: Register called twice for task type: " + taskType)
	}
	s.executors[taskType] = executor
}

func (s *Scheduler) getTaskCheck(ctx context.Context, task *store.TaskMessage, creatorID int) ([]*store.TaskCheckRunMessage, error) {
	var createList []*store.TaskCheckRunMessage

	create, err := s.getPITRTaskCheck(task, creatorID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to schedule backup/PITR task check")
	}
	if create != nil {
		createList = append(createList, create...)
	}

	if task.Type != api.TaskDatabaseSchemaUpdate && task.Type != api.TaskDatabaseSchemaUpdateSDL && task.Type != api.TaskDatabaseDataUpdate && task.Type != api.TaskDatabaseSchemaUpdateGhostSync {
		return createList, nil
	}

	create, err = s.getGeneralTaskCheck(task, creatorID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to schedule general task check")
	}
	if create != nil {
		createList = append(createList, create...)
	}

	create, err = s.getGhostTaskCheck(task, creatorID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to schedule gh-ost task check")
	}
	if create != nil {
		createList = append(createList, create...)
	}

	database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{UID: task.DatabaseID})
	if err != nil {
		return nil, err
	}
	if database == nil {
		return nil, errors.Errorf("database ID not found %v", task.DatabaseID)
	}
	dbSchema, err := s.store.GetDBSchema(ctx, database.UID)
	if err != nil {
		return nil, err
	}
	if dbSchema == nil {
		return nil, errors.Errorf("database schema not found %v", task.DatabaseID)
	}
	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &database.InstanceID})
	if err != nil {
		return nil, err
	}
	if instance == nil {
		return nil, errors.Errorf("instance %q not found", database.InstanceID)
	}

	create, err = s.getSQLReviewTaskCheck(task, instance)
	if err != nil {
		return nil, errors.Wrap(err, "failed to schedule SQL review task check")
	}
	if create != nil {
		createList = append(createList, create...)
	}

	create, err = getStmtTypeTaskCheck(task, instance)
	if err != nil {
		return nil, errors.Wrap(err, "failed to schedule statement type task check")
	}
	if create != nil {
		createList = append(createList, create...)
	}

	create, err = getStatementTypeReportTaskCheck(task, instance, creatorID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to schedule statement type report task check")
	}
	if create != nil {
		createList = append(createList, create...)
	}

	create, err = getStatementAffectedRowsReportTaskCheck(task, instance, creatorID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to schedule statement affected rows report task check")
	}
	if create != nil {
		createList = append(createList, create...)
	}

	return createList, nil
}

// ScheduleCheck schedules various task checks depending on the task type.
func (s *Scheduler) ScheduleCheck(ctx context.Context, task *store.TaskMessage, creatorID int) error {
	createList, err := s.getTaskCheck(ctx, task, creatorID)
	if err != nil {
		return errors.Wrap(err, "failed to getTaskCheck")
	}
	return s.store.CreateTaskCheckRun(ctx, createList...)
}

func getStmtTypeTaskCheck(task *store.TaskMessage, instance *store.InstanceMessage) ([]*store.TaskCheckRunMessage, error) {
	if !api.IsStatementTypeCheckSupported(instance.Engine) {
		return nil, nil
	}
	return []*store.TaskCheckRunMessage{
		{
			CreatorID: api.SystemBotID,
			TaskID:    task.ID,
			Type:      api.TaskCheckDatabaseStatementType,
		},
	}, nil
}

func (*Scheduler) getSQLReviewTaskCheck(task *store.TaskMessage, instance *store.InstanceMessage) ([]*store.TaskCheckRunMessage, error) {
	if !api.IsSQLReviewSupported(instance.Engine) {
		return nil, nil
	}
	return []*store.TaskCheckRunMessage{
		{
			CreatorID: api.SystemBotID,
			TaskID:    task.ID,
			Type:      api.TaskCheckDatabaseStatementAdvise,
		},
	}, nil
}

func (*Scheduler) getGeneralTaskCheck(task *store.TaskMessage, creatorID int) ([]*store.TaskCheckRunMessage, error) {
	return []*store.TaskCheckRunMessage{
		{
			CreatorID: creatorID,
			TaskID:    task.ID,
			Type:      api.TaskCheckDatabaseConnect,
		},
	}, nil
}

func (*Scheduler) getGhostTaskCheck(task *store.TaskMessage, creatorID int) ([]*store.TaskCheckRunMessage, error) {
	if task.Type != api.TaskDatabaseSchemaUpdateGhostSync {
		return nil, nil
	}
	return []*store.TaskCheckRunMessage{
		{
			CreatorID: creatorID,
			TaskID:    task.ID,
			Type:      api.TaskCheckGhostSync,
		},
	}, nil
}

func (*Scheduler) getPITRTaskCheck(task *store.TaskMessage, creatorID int) ([]*store.TaskCheckRunMessage, error) {
	if task.Type != api.TaskDatabaseRestorePITRRestore {
		return nil, nil
	}
	return []*store.TaskCheckRunMessage{
		{
			CreatorID: creatorID,
			TaskID:    task.ID,
			Type:      api.TaskCheckPITRMySQL,
		},
	}, nil
}

func getStatementTypeReportTaskCheck(task *store.TaskMessage, instance *store.InstanceMessage, creatorID int) ([]*store.TaskCheckRunMessage, error) {
	if !api.IsTaskCheckReportSupported(instance.Engine) {
		return nil, nil
	}
	if !api.IsTaskCheckReportNeededForTaskType(task.Type) {
		return nil, nil
	}
	return []*store.TaskCheckRunMessage{
		{
			CreatorID: creatorID,
			TaskID:    task.ID,
			Type:      api.TaskCheckDatabaseStatementTypeReport,
		},
	}, nil
}

func getStatementAffectedRowsReportTaskCheck(task *store.TaskMessage, instance *store.InstanceMessage, creatorID int) ([]*store.TaskCheckRunMessage, error) {
	if !api.IsTaskCheckReportSupported(instance.Engine) {
		return nil, nil
	}
	if !api.IsTaskCheckReportNeededForTaskType(task.Type) {
		return nil, nil
	}

	return []*store.TaskCheckRunMessage{
		{
			CreatorID: creatorID,
			TaskID:    task.ID,
			Type:      api.TaskCheckDatabaseStatementAffectedRowsReport,
		},
	}, nil
}

// SchedulePipelineTaskCheck schedules the task checks for a pipeline.
func (s *Scheduler) SchedulePipelineTaskCheck(ctx context.Context, pipelineID int) error {
	var createList []*store.TaskCheckRunMessage
	tasks, err := s.store.ListTasks(ctx, &api.TaskFind{PipelineID: &pipelineID})
	if err != nil {
		return err
	}
	for _, task := range tasks {
		create, err := s.getTaskCheck(ctx, task, api.SystemBotID)
		if err != nil {
			return errors.Wrapf(err, "failed to get task check for task %d", task.ID)
		}
		createList = append(createList, create...)
	}
	return s.store.CreateTaskCheckRun(ctx, createList...)
}

// SchedulePipelineTaskCheckReport schedules the task check reports for a pipeline.
func (s *Scheduler) SchedulePipelineTaskCheckReport(ctx context.Context, pipelineID int) error {
	var createList []*store.TaskCheckRunMessage
	tasks, err := s.store.ListTasks(ctx, &api.TaskFind{PipelineID: &pipelineID})
	if err != nil {
		return err
	}
	for _, task := range tasks {
		instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: &task.InstanceID})
		if err != nil {
			return err
		}
		if instance == nil {
			return errors.Errorf("instance %q not found", task.InstanceID)
		}

		create, err := getStatementTypeReportTaskCheck(task, instance, api.SystemBotID)
		if err != nil {
			return errors.Wrap(err, "failed to schedule statement type report task check")
		}
		if create != nil {
			createList = append(createList, create...)
		}

		create, err = getStatementAffectedRowsReportTaskCheck(task, instance, api.SystemBotID)
		if err != nil {
			return errors.Wrap(err, "failed to schedule statement affected rows report task check")
		}
		if create != nil {
			createList = append(createList, create...)
		}
	}
	return s.store.CreateTaskCheckRun(ctx, createList...)
}
