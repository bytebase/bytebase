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

	"github.com/bytebase/bytebase/backend/api"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	enterpriseAPI "github.com/bytebase/bytebase/backend/enterprise/api"
	"github.com/bytebase/bytebase/backend/server/component/state"
	"github.com/bytebase/bytebase/backend/server/utils"
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

					if _, ok := s.stateCfg.RunningTaskChecks.Load(taskCheckRun.ID); ok {
						continue
					}

					task, err := s.store.GetTaskByID(ctx, taskCheckRun.TaskID)
					if err != nil {
						log.Error("Failed to get task for task check run",
							zap.Int("task_check_run_id", taskCheckRun.ID),
							zap.Int("task_id", taskCheckRun.TaskID),
							zap.String("type", string(taskCheckRun.Type)),
							zap.Error(err),
						)
						taskCheckRunStatusPatch := &api.TaskCheckRunStatusPatch{
							ID:        &taskCheckRun.ID,
							UpdaterID: api.SystemBotID,
							Status:    api.TaskCheckRunFailed,
							Code:      common.Internal,
						}
						if _, err := s.store.PatchTaskCheckRunStatus(ctx, taskCheckRunStatusPatch); err != nil {
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
					go func(taskCheckRun *api.TaskCheckRun, task *api.Task) {
						defer func() {
							s.stateCfg.RunningTaskChecks.Delete(taskCheckRun.ID)
							s.stateCfg.Lock()
							s.stateCfg.InstanceOutstandingConnections[task.InstanceID]--
							s.stateCfg.Unlock()
						}()
						checkResultList, err := executor.Run(ctx, taskCheckRun, task)

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
							if _, err := s.store.PatchTaskCheckRunStatus(ctx, taskCheckRunStatusPatch); err != nil {
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
							if _, err := s.store.PatchTaskCheckRunStatus(ctx, taskCheckRunStatusPatch); err != nil {
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

func (s *Scheduler) getTaskCheck(ctx context.Context, task *api.Task, creatorID int) ([]*api.TaskCheckRunCreate, error) {
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

	statement, err := utils.GetTaskStatement(task)
	if err != nil {
		return nil, err
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
	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{EnvironmentID: &database.EnvironmentID, ResourceID: &database.InstanceID})
	if err != nil {
		return nil, err
	}
	if instance == nil {
		return nil, errors.Errorf("instance %q not found", database.InstanceID)
	}

	create, err = getSyntaxCheckTaskCheck(task, instance, dbSchema, statement)
	if err != nil {
		return nil, errors.Wrap(err, "failed to schedule syntax check task check")
	}
	createList = append(createList, create...)

	create, err = s.getSQLReviewTaskCheck(ctx, task, instance, dbSchema, statement)
	if err != nil {
		return nil, errors.Wrap(err, "failed to schedule SQL review task check")
	}
	createList = append(createList, create...)

	create, err = getStmtTypeTaskCheck(task, instance, dbSchema, statement)
	if err != nil {
		return nil, errors.Wrap(err, "failed to schedule statement type task check")
	}
	createList = append(createList, create...)

	return createList, nil
}

// ScheduleCheck schedules variouse task checks depending on the task type.
func (s *Scheduler) ScheduleCheck(ctx context.Context, task *api.Task, creatorID int) (*api.Task, error) {
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

func getStmtTypeTaskCheck(task *api.Task, instance *store.InstanceMessage, dbSchema *store.DBSchema, statement string) ([]*api.TaskCheckRunCreate, error) {
	if !api.IsStatementTypeCheckSupported(instance.Engine) {
		return nil, nil
	}
	payload, err := json.Marshal(api.TaskCheckDatabaseStatementTypePayload{
		Statement: statement,
		DbType:    instance.Engine,
		Charset:   dbSchema.Metadata.CharacterSet,
		Collation: dbSchema.Metadata.Collation,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal statement type payload: %v", task.Name)
	}
	return []*api.TaskCheckRunCreate{
		{
			CreatorID: api.SystemBotID,
			TaskID:    task.ID,
			Type:      api.TaskCheckDatabaseStatementType,
			Payload:   string(payload),
		},
	}, nil
}

func (s *Scheduler) getSQLReviewTaskCheck(ctx context.Context, task *api.Task, instance *store.InstanceMessage, dbSchema *store.DBSchema, statement string) ([]*api.TaskCheckRunCreate, error) {
	if !api.IsSQLReviewSupported(instance.Engine) {
		return nil, nil
	}
	policyID, err := s.store.GetSQLReviewPolicyIDByEnvID(ctx, task.Instance.EnvironmentID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get SQL review policy ID for task: %v, in environment: %v", task.Name, task.Instance.EnvironmentID)
	}
	payload, err := json.Marshal(api.TaskCheckDatabaseStatementAdvisePayload{
		Statement: statement,
		DbType:    instance.Engine,
		Charset:   dbSchema.Metadata.CharacterSet,
		Collation: dbSchema.Metadata.Collation,
		PolicyID:  policyID,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal statement advise payload: %v", task.Name)
	}
	return []*api.TaskCheckRunCreate{
		{
			CreatorID: api.SystemBotID,
			TaskID:    task.ID,
			Type:      api.TaskCheckDatabaseStatementAdvise,
			Payload:   string(payload),
		},
	}, nil
}

func getSyntaxCheckTaskCheck(task *api.Task, instance *store.InstanceMessage, dbSchema *store.DBSchema, statement string) ([]*api.TaskCheckRunCreate, error) {
	if !api.IsSyntaxCheckSupported(instance.Engine) {
		return nil, nil
	}
	payload, err := json.Marshal(api.TaskCheckDatabaseStatementAdvisePayload{
		Statement: statement,
		DbType:    instance.Engine,
		Charset:   dbSchema.Metadata.CharacterSet,
		Collation: dbSchema.Metadata.Collation,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal statement advise payload: %v", task.Name)
	}
	return []*api.TaskCheckRunCreate{
		{
			CreatorID: api.SystemBotID,
			TaskID:    task.ID,
			Type:      api.TaskCheckDatabaseStatementSyntax,
			Payload:   string(payload),
		},
	}, nil
}

func (*Scheduler) getGeneralTaskCheck(_ context.Context, task *api.Task, creatorID int) ([]*api.TaskCheckRunCreate, error) {
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

func (*Scheduler) getGhostTaskCheck(_ context.Context, task *api.Task, creatorID int) ([]*api.TaskCheckRunCreate, error) {
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

func (*Scheduler) getPITRTaskCheck(_ context.Context, task *api.Task, creatorID int) ([]*api.TaskCheckRunCreate, error) {
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

func (s *Scheduler) getLGTMTaskCheck(ctx context.Context, task *api.Task, creatorID int) ([]*api.TaskCheckRunCreate, error) {
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

// SchedulePipelineTaskCheck schedules the task checks for a pipeline.
func (s *Scheduler) SchedulePipelineTaskCheck(ctx context.Context, pipeline *api.Pipeline) error {
	var createList []*api.TaskCheckRunCreate
	for _, stage := range pipeline.StageList {
		for _, task := range stage.TaskList {
			create, err := s.getTaskCheck(ctx, task, api.SystemBotID)
			if err != nil {
				return errors.Wrapf(err, "failed to get task check for task %d", task.ID)
			}
			createList = append(createList, create...)
		}
	}
	if _, err := s.store.BatchCreateTaskCheckRun(ctx, createList); err != nil {
		return errors.Wrap(err, "failed to batch insert TaskCheckRunCreate")
	}
	return nil
}
