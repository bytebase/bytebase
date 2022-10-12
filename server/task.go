package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/common/log"
)

var (
	applicableTaskStatusTransition = map[api.TaskStatus][]api.TaskStatus{
		api.TaskPendingApproval: {api.TaskPending},
		// TODO(p0ny): support cancel pending task.
		api.TaskPending:  {api.TaskRunning, api.TaskPendingApproval},
		api.TaskRunning:  {api.TaskDone, api.TaskFailed, api.TaskCanceled},
		api.TaskDone:     {},
		api.TaskFailed:   {api.TaskRunning, api.TaskPendingApproval},
		api.TaskCanceled: {api.TaskPendingApproval},
	}
	taskCancellationImplemented = map[api.TaskType]bool{
		api.TaskDatabaseSchemaUpdateGhostSync: true,
	}
)

func isTaskStatusTransitionAllowed(fromStatus, toStatus api.TaskStatus) bool {
	for _, allowedStatus := range applicableTaskStatusTransition[fromStatus] {
		if allowedStatus == toStatus {
			return true
		}
	}
	return false
}

func (s *Server) canUpdateTaskStatement(ctx context.Context, task *api.Task) *echo.HTTPError {
	// Allow frontend to change the SQL statement of
	// 1. a PendingApproval task which hasn't started yet
	// 2. a Failed task which can be retried
	// 3. a Pending task which can't be scheduled because of failed task checks, task dependency or earliest allowed time
	if task.Status != api.TaskPendingApproval && task.Status != api.TaskFailed && task.Status != api.TaskPending {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("cannot update task in %q state", task.Status))
	}
	if task.Status == api.TaskPending {
		ok, err := s.TaskScheduler.canSchedule(ctx, task)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to check whether the task can be scheduled").SetInternal(err)
		}
		if ok {
			return echo.NewHTTPError(http.StatusBadRequest, "cannot update the PENDING task because it can be running at any time")
		}
	}
	return nil
}

func (s *Server) registerTaskRoutes(g *echo.Group) {
	g.PATCH("/pipeline/:pipelineID/task/all", func(c echo.Context) error {
		ctx := c.Request().Context()
		pipelineID, err := strconv.Atoi(c.Param("pipelineID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Pipeline ID is not a number: %s", c.Param("pipelineID"))).SetInternal(err)
		}

		taskPatch := &api.TaskPatch{
			UpdaterID: c.Get(getPrincipalIDContextKey()).(int),
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, taskPatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed update task request").SetInternal(err)
		}

		if taskPatch.EarliestAllowedTs != nil && !s.feature(api.FeatureTaskScheduleTime) {
			return echo.NewHTTPError(http.StatusForbidden, api.FeatureTaskScheduleTime.AccessErrorMessage())
		}

		issue, err := s.store.GetIssueByPipelineID(ctx, pipelineID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch issue with pipeline ID: %d", pipelineID)).SetInternal(err)
		}
		if issue == nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Issue not found with pipelineID: %d", pipelineID))
		}

		if taskPatch.Statement != nil {
			// check if all tasks can update statement
			for _, stage := range issue.Pipeline.StageList {
				for _, task := range stage.TaskList {
					if httpErr := s.canUpdateTaskStatement(ctx, task); httpErr != nil {
						return httpErr
					}
				}
			}
		}

		var taskPatchedList []*api.Task
		for _, stage := range issue.Pipeline.StageList {
			for _, task := range stage.TaskList {
				taskPatch := *taskPatch
				taskPatch.ID = task.ID
				taskPatched, httpErr := s.patchTask(ctx, task, &taskPatch, issue)
				if httpErr != nil {
					return httpErr
				}
				taskPatchedList = append(taskPatchedList, taskPatched)
			}
		}
		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, taskPatchedList); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal update pipeline %q tasks response", issue.Pipeline.Name)).SetInternal(err)
		}
		return nil
	})

	g.PATCH("/pipeline/:pipelineID/task/:taskID", func(c echo.Context) error {
		ctx := c.Request().Context()
		taskID, err := strconv.Atoi(c.Param("taskID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Task ID is not a number: %s", c.Param("taskID"))).SetInternal(err)
		}

		taskPatch := &api.TaskPatch{
			ID:        taskID,
			UpdaterID: c.Get(getPrincipalIDContextKey()).(int),
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, taskPatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed update task request").SetInternal(err)
		}

		if taskPatch.EarliestAllowedTs != nil && !s.feature(api.FeatureTaskScheduleTime) {
			return echo.NewHTTPError(http.StatusForbidden, api.FeatureTaskScheduleTime.AccessErrorMessage())
		}

		task, err := s.store.GetTaskByID(ctx, taskID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update task").SetInternal(err)
		}
		if task == nil {
			return echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("Task ID not found: %d", taskID))
		}

		issue, err := s.store.GetIssueByPipelineID(ctx, task.PipelineID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch issue with pipeline ID: %d", task.PipelineID)).SetInternal(err)
		}
		if issue == nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Issue not found with pipelineID: %d", task.PipelineID))
		}

		if taskPatch.Statement != nil {
			// Tenant mode project don't allow updating SQL statement for a single task.
			project, err := s.store.GetProjectByID(ctx, issue.ProjectID)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch project with ID: %d", issue.ProjectID)).SetInternal(err)
			}
			if project.TenantMode == api.TenantModeTenant && task.Type == api.TaskDatabaseSchemaUpdate {
				return echo.NewHTTPError(http.StatusBadRequest, "cannot update SQL statement of a single task for projects in tenant mode")
			}
		}

		taskPatched, httpErr := s.patchTask(ctx, task, taskPatch, issue)
		if httpErr != nil {
			return httpErr
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, taskPatched); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal update task \"%v\" status response", taskPatched.Name)).SetInternal(err)
		}
		return nil
	})

	g.PATCH("/pipeline/:pipelineID/task/:taskID/status", func(c echo.Context) error {
		ctx := c.Request().Context()
		taskID, err := strconv.Atoi(c.Param("taskID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Task ID is not a number: %s", c.Param("taskID"))).SetInternal(err)
		}

		currentPrincipalID := c.Get(getPrincipalIDContextKey()).(int)
		taskStatusPatch := &api.TaskStatusPatch{
			ID:        taskID,
			UpdaterID: currentPrincipalID,
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, taskStatusPatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed update task status request").SetInternal(err)
		}

		task, err := s.store.GetTaskByID(ctx, taskID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update task status").SetInternal(err)
		}
		if task == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Task not found with ID %d", taskID))
		}

		ok, err := s.canPrincipalChangeTaskStatus(ctx, currentPrincipalID, task, taskStatusPatch.Status)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to validate if the principal can change task status").SetInternal(err)
		}
		if !ok {
			return echo.NewHTTPError(http.StatusUnauthorized, "Not allowed to change task status")
		}

		taskPatched, err := s.patchTaskStatus(ctx, task, taskStatusPatch)
		if err != nil {
			if common.ErrorCode(err) == common.Invalid {
				return echo.NewHTTPError(http.StatusBadRequest, common.ErrorMessage(err))
			}
			if common.ErrorCode(err) == common.NotImplemented {
				return echo.NewHTTPError(http.StatusNotImplemented, common.ErrorMessage(err))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to update task \"%v\" status", task.Name)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, taskPatched); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal update task \"%v\" status response", taskPatched.Name)).SetInternal(err)
		}
		return nil
	})

	g.POST("/pipeline/:pipelineID/task/:taskID/check", func(c echo.Context) error {
		ctx := c.Request().Context()
		taskID, err := strconv.Atoi(c.Param("taskID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Task ID is not a number: %s", c.Param("taskID"))).SetInternal(err)
		}

		task, err := s.store.GetTaskByID(ctx, taskID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update task status").SetInternal(err)
		}
		if task == nil {
			return echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("Task not found with ID %d", taskID))
		}

		taskUpdated, err := s.TaskCheckScheduler.ScheduleCheckIfNeeded(ctx, task, c.Get(getPrincipalIDContextKey()).(int))
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to run task check \"%v\"", task.Name)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, taskUpdated); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal update task \"%v\" status response", taskUpdated.Name)).SetInternal(err)
		}
		return nil
	})
}

func (s *Server) patchTask(ctx context.Context, task *api.Task, taskPatch *api.TaskPatch, issue *api.Issue) (*api.Task, *echo.HTTPError) {
	var oldStatement string
	var newStatement string
	if taskPatch.Statement != nil {
		if httpErr := s.canUpdateTaskStatement(ctx, task); httpErr != nil {
			return nil, httpErr
		}
		newStatement = *taskPatch.Statement

		switch task.Type {
		case api.TaskDatabaseSchemaUpdate:
			payload := &api.TaskDatabaseSchemaUpdatePayload{}
			if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
				return nil, echo.NewHTTPError(http.StatusBadRequest, "Malformed database schema update payload").SetInternal(err)
			}
			oldStatement = payload.Statement
			payload.Statement = *taskPatch.Statement
			// 1. For VCS workflows, patchTask only happens when we modify the same file.
			// 	  In that case, we want to use the same schema version parsed from the file name.
			//    The task executor will force retry using the new SQL statement.
			// 2. We should update the schema version if we've updated the SQL in the UI workflow, otherwise we will
			//    get migration history version conflict if the previous task has been attempted.
			if issue.Project.WorkflowType == api.UIWorkflow {
				payload.SchemaVersion = common.DefaultMigrationVersion()
			}
			bytes, err := json.Marshal(payload)
			if err != nil {
				return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to construct updated task payload").SetInternal(err)
			}
			payloadStr := string(bytes)
			taskPatch.Payload = &payloadStr
		case api.TaskDatabaseDataUpdate:
			payload := &api.TaskDatabaseDataUpdatePayload{}
			if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
				return nil, echo.NewHTTPError(http.StatusBadRequest, "Malformed database data update payload").SetInternal(err)
			}
			oldStatement = payload.Statement
			payload.Statement = *taskPatch.Statement
			// 1. For VCS workflows, patchTask only happens when we modify the same file.
			// 	  In that case, we want to use the same schema version parsed from the file name.
			//    The task executor will force retry using the new SQL statement.
			// 2. We should update the schema version if we've updated the SQL in the UI workflow, otherwise we will
			//    get migration history version conflict if the previous task has been attempted.
			if issue.Project.WorkflowType == api.UIWorkflow {
				payload.SchemaVersion = common.DefaultMigrationVersion()
			}
			bytes, err := json.Marshal(payload)
			if err != nil {
				return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to construct updated task payload").SetInternal(err)
			}
			payloadStr := string(bytes)
			taskPatch.Payload = &payloadStr
		case api.TaskDatabaseCreate:
			payload := &api.TaskDatabaseCreatePayload{}
			if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
				return nil, echo.NewHTTPError(http.StatusBadRequest, "Malformed database create payload").SetInternal(err)
			}
			oldStatement = payload.Statement
			payload.Statement = *taskPatch.Statement
			// We should update the schema version if we've updated the SQL, otherwise we will
			// get migration history version conflict if the previous task has been attempted.
			payload.SchemaVersion = common.DefaultMigrationVersion()
			bytes, err := json.Marshal(payload)
			if err != nil {
				return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to construct updated task payload").SetInternal(err)
			}
			payloadStr := string(bytes)
			taskPatch.Payload = &payloadStr
		case api.TaskDatabaseSchemaUpdateGhostSync:
			payload := &api.TaskDatabaseSchemaUpdateGhostSyncPayload{}
			if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
				return nil, echo.NewHTTPError(http.StatusBadRequest, "Malformed database data update payload").SetInternal(err)
			}
			oldStatement = payload.Statement
			payload.Statement = *taskPatch.Statement
			// We should update the schema version if we've updated the SQL, otherwise we will
			// get migration history version conflict if the previous task has been attempted.
			payload.SchemaVersion = common.DefaultMigrationVersion()
			bytes, err := json.Marshal(payload)
			if err != nil {
				return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to construct updated task payload").SetInternal(err)
			}
			payloadStr := string(bytes)
			taskPatch.Payload = &payloadStr
		}
	}

	taskPatched, err := s.store.PatchTask(ctx, taskPatch)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to update task \"%v\"", task.Name)).SetInternal(err)
	}

	// create an activity and trigger task check for statement update
	if taskPatched.Type == api.TaskDatabaseSchemaUpdate || taskPatched.Type == api.TaskDatabaseDataUpdate || taskPatched.Type == api.TaskDatabaseSchemaUpdateGhostSync {
		if oldStatement != newStatement {
			if issue == nil {
				err := errors.Errorf("issue not found with pipeline ID %v", task.PipelineID)
				return nil, echo.NewHTTPError(http.StatusNotFound, err).SetInternal(err)
			}

			// create a task statement update activity
			payload, err := json.Marshal(api.ActivityPipelineTaskStatementUpdatePayload{
				TaskID:       taskPatched.ID,
				OldStatement: oldStatement,
				NewStatement: newStatement,
				TaskName:     task.Name,
				IssueName:    issue.Name,
			})
			if err != nil {
				return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to create activity after updating task statement: %v", taskPatched.Name).SetInternal(err)
			}

			if _, err = s.ActivityManager.CreateActivity(ctx, &api.ActivityCreate{
				CreatorID:   taskPatched.CreatorID,
				ContainerID: taskPatched.PipelineID,
				Type:        api.ActivityPipelineTaskStatementUpdate,
				Payload:     string(payload),
				Level:       api.ActivityInfo,
			}, &ActivityMeta{
				issue: issue,
			}); err != nil {
				return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to create activity after updating task statement: %v", taskPatched.Name)).SetInternal(err)
			}

			// updated statement, dismiss stale approvals and transfer the status to PendingApproval.
			if taskPatched.Status != api.TaskPendingApproval {
				t, err := s.patchTaskStatus(ctx, taskPatched, &api.TaskStatusPatch{
					ID:        taskPatch.ID,
					UpdaterID: taskPatch.UpdaterID,
					Status:    api.TaskPendingApproval,
				})
				if err != nil {
					return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to change task status to PendingApproval after updating task: %v", taskPatched.Name)).SetInternal(err)
				}
				taskPatched = t
			}

			if taskPatched.Type == api.TaskDatabaseSchemaUpdateGhostSync {
				_, err = s.store.CreateTaskCheckRunIfNeeded(ctx, &api.TaskCheckRunCreate{
					CreatorID: taskPatched.CreatorID,
					TaskID:    task.ID,
					Type:      api.TaskCheckGhostSync,
				})
				if err != nil {
					// It's OK if we failed to trigger a check, just emit an error log
					log.Error("Failed to trigger gh-ost dry run after changing the task statement",
						zap.Int("task_id", task.ID),
						zap.String("task_name", task.Name),
						zap.Error(err),
					)
				}
			}

			if api.IsSyntaxCheckSupported(task.Database.Instance.Engine, s.profile.Mode) {
				payload, err := json.Marshal(api.TaskCheckDatabaseStatementAdvisePayload{
					Statement: *taskPatch.Statement,
					DbType:    task.Database.Instance.Engine,
					Charset:   taskPatched.Database.CharacterSet,
					Collation: taskPatched.Database.Collation,
				})
				if err != nil {
					return nil, echo.NewHTTPError(http.StatusInternalServerError, errors.Wrapf(err, "failed to marshal statement advise payload: %v", task.Name))
				}
				_, err = s.store.CreateTaskCheckRunIfNeeded(ctx, &api.TaskCheckRunCreate{
					CreatorID: api.SystemBotID,
					TaskID:    task.ID,
					Type:      api.TaskCheckDatabaseStatementSyntax,
					Payload:   string(payload),
				})
				if err != nil {
					// It's OK if we failed to trigger a check, just emit an error log
					log.Error("Failed to trigger syntax check after changing the task statement",
						zap.Int("task_id", task.ID),
						zap.String("task_name", task.Name),
						zap.Error(err),
					)
				}
			}

			if api.IsSQLReviewSupported(task.Database.Instance.Engine, s.profile.Mode) {
				if err := s.triggerDatabaseStatementAdviseTask(ctx, *taskPatch.Statement, taskPatched); err != nil {
					return nil, echo.NewHTTPError(http.StatusInternalServerError, errors.Wrap(err, "failed to trigger database statement advise task")).SetInternal(err)
				}
			}

			if api.IsStatementTypeCheckSupported(task.Instance.Engine) {
				payload, err := json.Marshal(api.TaskCheckDatabaseStatementTypePayload{
					Statement: *taskPatch.Statement,
					DbType:    task.Instance.Engine,
					Charset:   task.Database.CharacterSet,
					Collation: task.Database.Collation,
				})
				if err != nil {
					return nil, echo.NewHTTPError(http.StatusInternalServerError, errors.Wrapf(err, "failed to marshal check statement type payload: %v", task.Name))
				}
				if _, err := s.store.CreateTaskCheckRunIfNeeded(ctx, &api.TaskCheckRunCreate{
					CreatorID: api.SystemBotID,
					TaskID:    task.ID,
					Type:      api.TaskCheckDatabaseStatementType,
					Payload:   string(payload),
				}); err != nil {
					// It's OK if we failed to trigger a check, just emit an error log
					log.Error("Failed to trigger statement type check after changing the task statement",
						zap.Int("task_id", task.ID),
						zap.String("task_name", task.Name),
						zap.Error(err),
					)
				}
			}
		}
	}

	// earliest allowed time update.
	// - create an activity.
	// - dismiss stale approval.
	if taskPatched.EarliestAllowedTs != task.EarliestAllowedTs {
		// create an activity
		if issue == nil {
			err := errors.Errorf("issue not found with pipeline ID %v", task.PipelineID)
			return nil, echo.NewHTTPError(http.StatusNotFound, err.Error()).SetInternal(err)
		}

		payload, err := json.Marshal(api.ActivityPipelineTaskEarliestAllowedTimeUpdatePayload{
			TaskID:               taskPatched.ID,
			OldEarliestAllowedTs: task.EarliestAllowedTs,
			NewEarliestAllowedTs: taskPatched.EarliestAllowedTs,
			TaskName:             task.Name,
			IssueName:            issue.Name,
		})
		if err != nil {
			return nil, echo.NewHTTPError(http.StatusInternalServerError, errors.Wrapf(err, "failed to marshal earliest allowed time activity payload: %v", task.Name))
		}
		activityCreate := &api.ActivityCreate{
			CreatorID:   taskPatched.CreatorID,
			ContainerID: taskPatched.PipelineID,
			Type:        api.ActivityPipelineTaskEarliestAllowedTimeUpdate,
			Payload:     string(payload),
			Level:       api.ActivityInfo,
		}
		_, err = s.ActivityManager.CreateActivity(ctx, activityCreate, &ActivityMeta{
			issue: issue,
		})
		if err != nil {
			return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to create activity after updating task earliest allowed time: %v", taskPatched.Name)).SetInternal(err)
		}

		// updated earliest allowed time, dismiss stale approvals and transfer the status to PendingApproval.
		if taskPatched.Status != api.TaskPendingApproval {
			t, err := s.patchTaskStatus(ctx, taskPatched, &api.TaskStatusPatch{
				ID:        taskPatch.ID,
				UpdaterID: taskPatch.UpdaterID,
				Status:    api.TaskPendingApproval,
			})
			if err != nil {
				return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to change task status to PendingApproval after updating task: %v", taskPatched.Name)).SetInternal(err)
			}
			taskPatched = t
		}
	}
	return taskPatched, nil
}

// canPrincipalBeAssignee checks if a principal could be the assignee of an issue, judging by the principal role and the environment policy.
func (s *Server) canPrincipalBeAssignee(ctx context.Context, principalID int, environmentID int, projectID int, issueType api.IssueType) (bool, error) {
	policy, err := s.store.GetPipelineApprovalPolicy(ctx, environmentID)
	if err != nil {
		return false, err
	}
	var groupValue *api.AssigneeGroupValue
	for i, group := range policy.AssigneeGroupList {
		if group.IssueType == issueType {
			groupValue = &policy.AssigneeGroupList[i].Value
			break
		}
	}
	if groupValue == nil {
		// no value is set, fallback to default.
		// the assignee group is the workspace owner and DBA.
		principal, err := s.store.GetPrincipalByID(ctx, principalID)
		if err != nil {
			return false, common.Wrapf(err, common.Internal, "failed to get principal by ID %d", principalID)
		}
		if principal == nil {
			return false, common.Errorf(common.NotFound, "principal not found by ID %d", principalID)
		}
		if principal.Role == api.Owner || principal.Role == api.DBA {
			return true, nil
		}
	} else if *groupValue == api.AssigneeGroupValueProjectOwner {
		// the assignee group is the project owner.
		member, err := s.store.GetProjectMember(ctx, &api.ProjectMemberFind{
			ProjectID:   &projectID,
			PrincipalID: &principalID,
		})
		if err != nil {
			return false, common.Wrapf(err, common.Internal, "failed to get project member by projectID %d, principalID %d", projectID, principalID)
		}
		if member == nil {
			return false, common.Errorf(common.NotFound, "project member not found by projectID %d, principalID %d", projectID, principalID)
		}
		if member.Role == string(api.Owner) {
			return true, nil
		}
	}
	return false, nil
}

// canPrincipalChangeTaskStatus validates if the principal has the privilege to update task status, judging from the principal role and the environment policy.
func (s *Server) canPrincipalChangeTaskStatus(ctx context.Context, principalID int, task *api.Task, toStatus api.TaskStatus) (bool, error) {
	// The creator can cancel task.
	if toStatus == api.TaskCanceled {
		if principalID == task.CreatorID {
			return true, nil
		}
	}
	// the workspace owner and DBA roles can always change task status.
	principal, err := s.store.GetPrincipalByID(ctx, principalID)
	if err != nil {
		return false, common.Wrapf(err, common.Internal, "failed to get principal by ID %d", principalID)
	}
	if principal == nil {
		return false, common.Errorf(common.NotFound, "principal not found by ID %d", principalID)
	}
	if principal.Role == api.Owner || principal.Role == api.DBA {
		return true, nil
	}

	issue, err := s.store.GetIssueByPipelineID(ctx, task.PipelineID)
	if err != nil {
		return false, common.Wrapf(err, common.Internal, "failed to find issue")
	}
	if issue == nil {
		return false, common.Errorf(common.NotFound, "issue not found by pipeline ID: %d", task.PipelineID)
	}
	groupValue, err := s.getGroupValueForTask(ctx, issue, task)
	if err != nil {
		return false, common.Wrapf(err, common.Internal, "failed to get assignee group value for taskID %d", task.ID)
	}
	if groupValue == nil {
		return false, nil
	}
	// as the policy says, the project owner has the privilege to change task status.
	if *groupValue == api.AssigneeGroupValueProjectOwner {
		member, err := s.store.GetProjectMember(ctx, &api.ProjectMemberFind{
			ProjectID:   &issue.ProjectID,
			PrincipalID: &principalID,
		})
		if err != nil {
			return false, common.Wrapf(err, common.Internal, "failed to get project member by projectID %d, principalID %d", issue.ProjectID, principalID)
		}
		if member != nil && member.Role == string(api.Owner) {
			return true, nil
		}
	}
	return false, nil
}

func (s *Server) getGroupValueForTask(ctx context.Context, issue *api.Issue, task *api.Task) (*api.AssigneeGroupValue, error) {
	environmentID := api.UnknownID
	for _, stage := range issue.Pipeline.StageList {
		if stage.ID == task.StageID {
			environmentID = stage.EnvironmentID
			break
		}
	}
	if environmentID == api.UnknownID {
		return nil, common.Errorf(common.NotFound, "failed to find environmentID by task.StageID %d", task.StageID)
	}

	policy, err := s.store.GetPipelineApprovalPolicy(ctx, environmentID)
	if err != nil {
		return nil, common.Wrapf(err, common.Internal, "failed to get pipeline approval policy by environmentID %d", environmentID)
	}

	for _, assigneeGroup := range policy.AssigneeGroupList {
		if assigneeGroup.IssueType == issue.Type {
			return &assigneeGroup.Value, nil
		}
	}
	return nil, nil
}

func (s *Server) patchTaskStatus(ctx context.Context, task *api.Task, taskStatusPatch *api.TaskStatusPatch) (_ *api.Task, err error) {
	defer func() {
		if err != nil {
			log.Error("Failed to change task status.",
				zap.Int("id", task.ID),
				zap.String("name", task.Name),
				zap.String("old_status", string(task.Status)),
				zap.String("new_status", string(taskStatusPatch.Status)),
				zap.Error(err))
		}
	}()

	if !isTaskStatusTransitionAllowed(task.Status, taskStatusPatch.Status) {
		return nil, &common.Error{
			Code: common.Invalid,
			Err:  errors.Errorf("invalid task status transition from %v to %v. Applicable transition(s) %v", task.Status, taskStatusPatch.Status, applicableTaskStatusTransition[task.Status]),
		}
	}

	if taskStatusPatch.Status == api.TaskCanceled {
		if !taskCancellationImplemented[task.Type] {
			return nil, common.Errorf(common.NotImplemented, "Canceling task type %s is not supported", task.Type)
		}
		s.TaskScheduler.runningExecutorsMutex.Lock()
		cancel, ok := s.TaskScheduler.runningExecutorsCancel[task.ID]
		s.TaskScheduler.runningExecutorsMutex.Unlock()
		if !ok {
			return nil, errors.New("Failed to cancel task")
		}
		cancel()
	}

	taskPatched, err := s.store.PatchTaskStatus(ctx, taskStatusPatch)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to change task %v(%v) status", task.ID, task.Name)
	}

	// Most tasks belong to a pipeline which in turns belongs to an issue. The followup code
	// behaves differently depending on whether the task is wrapped in an issue.
	// TODO(tianzhou): Refactor the followup code into chained onTaskStatusChange hook.
	issue, err := s.store.GetIssueByPipelineID(ctx, task.PipelineID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to fetch containing issue after changing the task status: %v", task.Name)
	}
	// Not all pipelines belong to an issue, so it's OK if issue is not found.
	if issue == nil {
		log.Debug("Pipeline has no linking issue",
			zap.Int("pipelineID", task.PipelineID),
			zap.String("task", task.Name))
	}

	// Create an activity
	issueName := ""
	if issue != nil {
		issueName = issue.Name
	}
	payload, err := json.Marshal(api.ActivityPipelineTaskStatusUpdatePayload{
		TaskID:    task.ID,
		OldStatus: task.Status,
		NewStatus: taskPatched.Status,
		IssueName: issueName,
		TaskName:  task.Name,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal activity after changing the task status: %v", task.Name)
	}

	level := api.ActivityInfo
	if taskPatched.Status == api.TaskFailed {
		level = api.ActivityError
	}
	activityCreate := &api.ActivityCreate{
		CreatorID:   taskStatusPatch.UpdaterID,
		ContainerID: task.PipelineID,
		Type:        api.ActivityPipelineTaskStatusUpdate,
		Level:       level,
		Payload:     string(payload),
	}
	if taskStatusPatch.Comment != nil {
		activityCreate.Comment = *taskStatusPatch.Comment
	}

	activityMeta := ActivityMeta{}
	if issue != nil {
		activityMeta.issue = issue
	}
	_, err = s.ActivityManager.CreateActivity(ctx, activityCreate, &activityMeta)
	if err != nil {
		return nil, err
	}

	// If create database, schema update and gh-ost cutover task completes, we sync the corresponding instance schema immediately.
	if (taskPatched.Type == api.TaskDatabaseCreate || taskPatched.Type == api.TaskDatabaseSchemaUpdate || taskPatched.Type == api.TaskDatabaseSchemaUpdateGhostCutover) && taskPatched.Status == api.TaskDone {
		instance, err := s.store.GetInstanceByID(ctx, task.InstanceID)
		if err != nil {
			return nil, errors.Wrap(err, "failed to sync instance schema after completing task")
		}
		if err := s.syncDatabaseSchema(ctx, instance, taskPatched.Database.Name); err != nil {
			log.Error("failed to sync database schema",
				zap.String("instance", instance.Name),
				zap.String("databaseName", taskPatched.Database.Name),
			)
		}
	}

	// If every task in the pipeline completes, and the assignee is system bot:
	// Case 1: If the task is associated with an issue, then we mark the issue (including the pipeline) as DONE.
	// Case 2: If the task is NOT associated with an issue, then we mark the pipeline as DONE.
	if taskPatched.Status == "DONE" && (issue == nil || issue.AssigneeID == api.SystemBotID) {
		pipeline, err := s.store.GetPipelineByID(ctx, taskPatched.PipelineID)
		if err != nil {
			return nil, errors.Errorf("failed to fetch pipeline/issue as DONE after completing task %v", taskPatched.Name)
		}
		if pipeline == nil {
			return nil, errors.Errorf("pipeline not found for ID %v", taskPatched.PipelineID)
		}
		if areAllTasksDone(pipeline) {
			if issue == nil {
				// System-generated tasks such as backup tasks don't have corresponding issues.
				status := api.PipelineDone
				pipelinePatch := &api.PipelinePatch{
					ID:        pipeline.ID,
					UpdaterID: taskStatusPatch.UpdaterID,
					Status:    &status,
				}
				if _, err := s.store.PatchPipeline(ctx, pipelinePatch); err != nil {
					return nil, errors.Wrapf(err, "failed to mark pipeline %v as DONE after completing task %v", pipeline.Name, taskPatched.Name)
				}
			} else {
				issue.Pipeline = pipeline
				_, err := s.changeIssueStatus(ctx, issue, api.IssueDone, taskStatusPatch.UpdaterID, "")
				if err != nil {
					return nil, errors.Wrapf(err, "failed to mark issue %v as DONE after completing task %v", issue.Name, taskPatched.Name)
				}
			}
		}
	}

	return taskPatched, nil
}

func (s *Server) triggerDatabaseStatementAdviseTask(ctx context.Context, statement string, task *api.Task) error {
	policyID, err := s.store.GetSQLReviewPolicyIDByEnvID(ctx, task.Instance.EnvironmentID)
	if err != nil {
		// It's OK if we failed to find the SQL review policy, just emit an error log
		log.Error("Failed to found SQL review policy id for task",
			zap.Int("task_id", task.ID),
			zap.String("task_name", task.Name),
			zap.Int("environment_id", task.Instance.EnvironmentID),
			zap.Error(err),
		)
		return nil
	}

	payload, err := json.Marshal(api.TaskCheckDatabaseStatementAdvisePayload{
		Statement: statement,
		DbType:    task.Database.Instance.Engine,
		Charset:   task.Database.CharacterSet,
		Collation: task.Database.Collation,
		PolicyID:  policyID,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to marshal statement advise payload: %v", task.Name)
	}

	if _, err := s.store.CreateTaskCheckRunIfNeeded(ctx, &api.TaskCheckRunCreate{
		CreatorID: api.SystemBotID,
		TaskID:    task.ID,
		Type:      api.TaskCheckDatabaseStatementAdvise,
		Payload:   string(payload),
	}); err != nil {
		// It's OK if we failed to trigger a check, just emit an error log
		log.Error("Failed to trigger statement advise task after changing task statement",
			zap.Int("task_id", task.ID),
			zap.String("task_name", task.Name),
			zap.Error(err),
		)
	}

	return nil
}

func (s *Server) getDefaultAssigneeID(ctx context.Context, environmentID int, projectID int, issueType api.IssueType) (int, error) {
	policy, err := s.store.GetPipelineApprovalPolicy(ctx, environmentID)
	if err != nil {
		return api.UnknownID, errors.Wrapf(err, "failed to GetPipelineApprovalPolicy for environmentID %d", environmentID)
	}
	if policy.Value == api.PipelineApprovalValueManualNever {
		// use SystemBot for auto approval tasks.
		return api.SystemBotID, nil
	}

	var groupValue *api.AssigneeGroupValue
	for i, group := range policy.AssigneeGroupList {
		if group.IssueType == issueType {
			groupValue = &policy.AssigneeGroupList[i].Value
			break
		}
	}
	if groupValue == nil {
		member, err := s.getAnyWorkspaceOwnerOrDBA(ctx)
		if err != nil {
			return api.UnknownID, errors.Wrap(err, "failed to get a workspace owner or DBA")
		}
		return member.PrincipalID, nil
	} else if *groupValue == api.AssigneeGroupValueProjectOwner {
		projectMember, err := s.getAnyProjectOwner(ctx, projectID)
		if err != nil {
			return api.UnknownID, errors.Wrap(err, "failed to get a project owner")
		}
		return projectMember.PrincipalID, nil
	}
	// never reached
	return api.UnknownID, errors.New("invalid assigneeGroupValue")
}

func areAllTasksDone(pipeline *api.Pipeline) bool {
	for _, stage := range pipeline.StageList {
		for _, task := range stage.TaskList {
			if task.Status != api.TaskDone {
				return false
			}
		}
	}
	return true
}
