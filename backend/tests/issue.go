package tests

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/jsonapi"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common/log"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	v1 "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// createIssue creates an issue.
func (ctl *controller) createIssue(issueCreate api.IssueCreate) (*api.Issue, error) {
	buf := new(bytes.Buffer)
	if err := jsonapi.MarshalPayload(buf, &issueCreate); err != nil {
		return nil, errors.Wrap(err, "failed to marshal issue create")
	}

	body, err := ctl.post("/issue", buf)
	if err != nil {
		return nil, err
	}

	issue := new(api.Issue)
	if err = jsonapi.UnmarshalPayload(body, issue); err != nil {
		return nil, errors.Wrap(err, "fail to unmarshal post issue response")
	}
	return issue, nil
}

// getIssue gets the issue with given ID.
func (ctl *controller) getIssue(id int) (*api.Issue, error) {
	body, err := ctl.get(fmt.Sprintf("/issue/%d", id), nil)
	if err != nil {
		return nil, err
	}

	issue := new(api.Issue)
	if err = jsonapi.UnmarshalPayload(body, issue); err != nil {
		return nil, errors.Wrap(err, "fail to unmarshal get issue response")
	}
	return issue, nil
}

// getIssue gets the issue with given ID.
func (ctl *controller) getIssues(projectID *int, statusList ...api.IssueStatus) ([]*api.Issue, error) {
	var ret []*api.Issue
	// call getOnePageIssuesWithToken until no more issues.
	token := ""
	for {
		issues, nextToken, err := ctl.getOnePageIssuesWithToken(projectID, statusList, token)
		if err != nil {
			return nil, err
		}
		if len(issues) == 0 {
			break
		}
		ret = append(ret, issues...)
		token = nextToken
	}
	return ret, nil
}

func (ctl *controller) getOnePageIssuesWithToken(projectID *int, statusList []api.IssueStatus, token string) ([]*api.Issue, string, error) {
	params := make(map[string]string)
	if projectID != nil {
		params["project"] = fmt.Sprintf("%d", *projectID)
	}
	if len(statusList) > 0 {
		var sl []string
		for _, status := range statusList {
			sl = append(sl, string(status))
		}
		params["status"] = strings.Join(sl, ",")
	}
	if token != "" {
		params["token"] = token
	}
	body, err := ctl.get("/issue", params)
	if err != nil {
		return nil, "", err
	}
	issueResp := new(api.IssueResponse)
	err = jsonapi.UnmarshalPayload(body, issueResp)
	if err != nil {
		return nil, "", errors.Wrap(err, "fail to unmarshal get issue response")
	}
	return issueResp.Issues, issueResp.NextToken, nil
}

func (ctl *controller) patchIssue(uid int, issuePatch api.IssuePatch) (*api.Issue, error) {
	buf := new(bytes.Buffer)
	if err := jsonapi.MarshalPayload(buf, &issuePatch); err != nil {
		return nil, errors.Wrap(err, "failed to marshal issue patch")
	}

	body, err := ctl.patch(fmt.Sprintf("/issue/%d", uid), buf)
	if err != nil {
		return nil, err
	}

	issue := new(api.Issue)
	if err = jsonapi.UnmarshalPayload(body, issue); err != nil {
		return nil, errors.Wrap(err, "fail to unmarshal patch issue patch response")
	}
	return issue, nil
}

// patchIssue patches the issue with given ID.
func (ctl *controller) patchIssueStatus(issueStatusPatch api.IssueStatusPatch) (*api.Issue, error) {
	buf := new(bytes.Buffer)
	if err := jsonapi.MarshalPayload(buf, &issueStatusPatch); err != nil {
		return nil, errors.Wrap(err, "failed to marshal issue status patch")
	}

	body, err := ctl.patch(fmt.Sprintf("/issue/%d/status", issueStatusPatch.ID), buf)
	if err != nil {
		return nil, err
	}

	issue := new(api.Issue)
	if err = jsonapi.UnmarshalPayload(body, issue); err != nil {
		return nil, errors.Wrap(err, "fail to unmarshal patch issue status patch response")
	}
	return issue, nil
}

// patchTask patches a task.
func (ctl *controller) patchTask(taskPatch api.TaskPatch, pipelineID int, taskID int) (*api.Task, error) {
	buf := new(bytes.Buffer)
	if err := jsonapi.MarshalPayload(buf, &taskPatch); err != nil {
		return nil, errors.Wrap(err, "failed to marshal taskPatch")
	}

	body, err := ctl.patch(fmt.Sprintf("/pipeline/%d/task/%d", pipelineID, taskID), buf)
	if err != nil {
		return nil, err
	}

	task := new(api.Task)
	if err = jsonapi.UnmarshalPayload(body, task); err != nil {
		return nil, errors.Wrap(err, "fail to unmarshal patchTask response")
	}
	return task, nil
}

// patchTaskStatus patches the status of a task in the pipeline stage.
func (ctl *controller) patchTaskStatus(taskStatusPatch api.TaskStatusPatch, pipelineID int, taskID int) (*api.Task, error) {
	buf := new(bytes.Buffer)
	if err := jsonapi.MarshalPayload(buf, &taskStatusPatch); err != nil {
		return nil, errors.Wrap(err, "failed to marshal patchTaskStatus")
	}

	body, err := ctl.patch(fmt.Sprintf("/pipeline/%d/task/%d/status", pipelineID, taskID), buf)
	if err != nil {
		return nil, err
	}

	task := new(api.Task)
	if err = jsonapi.UnmarshalPayload(body, task); err != nil {
		return nil, errors.Wrap(err, "fail to unmarshal patchTaskStatus response")
	}
	return task, nil
}

// patchStageAllTaskStatus patches the status of all tasks in the pipeline stage.
func (ctl *controller) patchStageAllTaskStatus(stageAllTaskStatusPatch api.StageAllTaskStatusPatch, pipelineID int) error {
	buf := new(bytes.Buffer)
	if err := jsonapi.MarshalPayload(buf, &stageAllTaskStatusPatch); err != nil {
		return errors.Wrap(err, "failed to marshal StageAllTaskStatusPatch")
	}

	_, err := ctl.patch(fmt.Sprintf("/pipeline/%d/stage/%d/status", pipelineID, stageAllTaskStatusPatch.ID), buf)
	return err
}

// approveIssueNext approves the next pending approval task.
func (ctl *controller) approveIssueNext(issue *api.Issue) error {
	for _, stage := range issue.Pipeline.StageList {
		for _, task := range stage.TaskList {
			if task.Status == api.TaskPendingApproval {
				if _, err := ctl.patchTaskStatus(
					api.TaskStatusPatch{
						Status: api.TaskPending,
					},
					issue.Pipeline.ID, task.ID); err != nil {
					return errors.Wrapf(err, "failed to patch task status for task %d", task.ID)
				}
				return nil
			}
		}
	}
	return nil
}

// approveIssueTasksWithStageApproval approves all pending approval tasks in the next stage.
func (ctl *controller) approveIssueTasksWithStageApproval(issue *api.Issue) error {
	stageID := 0
	for _, stage := range issue.Pipeline.StageList {
		for _, task := range stage.TaskList {
			if task.Status == api.TaskPendingApproval {
				stageID = stage.ID
				break
			}
		}
		if stageID != 0 {
			break
		}
	}
	if stageID != 0 {
		if err := ctl.patchStageAllTaskStatus(
			api.StageAllTaskStatusPatch{
				ID:     stageID,
				Status: api.TaskPending,
			},
			issue.Pipeline.ID,
		); err != nil {
			return errors.Wrapf(err, "failed to patch task status for stage %d", stageID)
		}
	}
	return nil
}

// getNextTaskStatus gets the next task status that needs to be handle.
func getNextTaskStatus(issue *api.Issue) (api.TaskStatus, error) {
	for _, stage := range issue.Pipeline.StageList {
		for _, task := range stage.TaskList {
			if task.Status == api.TaskDone {
				continue
			}
			if task.Status == api.TaskFailed {
				var runs []string
				for _, run := range task.TaskRunList {
					runs = append(runs, fmt.Sprintf("%+v", run))
				}
				return api.TaskFailed, errors.Errorf("pipeline task %v failed runs: %v", task.ID, strings.Join(runs, ", "))
			}
			return task.Status, nil
		}
	}
	return api.TaskDone, nil
}

// waitIssueNextTaskWithTaskApproval waits for next task in pipeline to finish and approves it when necessary.
func (ctl *controller) waitIssueNextTaskWithTaskApproval(ctx context.Context, id int) (api.TaskStatus, error) {
	return ctl.waitIssuePipelineTaskImpl(ctx, id, ctl.approveIssueNext, true)
}

// waitIssuePipeline waits for pipeline to finish and approves tasks when necessary.
func (ctl *controller) waitIssuePipeline(ctx context.Context, id int) (api.TaskStatus, error) {
	return ctl.waitIssuePipelineTaskImpl(ctx, id, ctl.approveIssueNext, false)
}

// waitIssuePipelineWithStageApproval waits for pipeline to finish and approves tasks when necessary.
func (ctl *controller) waitIssuePipelineWithStageApproval(ctx context.Context, id int) (api.TaskStatus, error) {
	return ctl.waitIssuePipelineTaskImpl(ctx, id, ctl.approveIssueTasksWithStageApproval, false)
}

// waitIssuePipelineWithNoApproval waits for pipeline to finish and do nothing when approvals are needed.
func (ctl *controller) waitIssuePipelineWithNoApproval(ctx context.Context, id int) (api.TaskStatus, error) {
	noop := func(*api.Issue) error {
		return nil
	}
	return ctl.waitIssuePipelineTaskImpl(ctx, id, noop, false)
}

// waitIssuePipelineImpl waits for the tasks in pipeline to finish and approves tasks when necessary.
func (ctl *controller) waitIssuePipelineTaskImpl(ctx context.Context, id int, approveFunc func(legacyIssue *api.Issue) error, approveOnce bool) (api.TaskStatus, error) {
	// Sleep for 1 second between issues so that we don't get migration version conflict because we are using second-level timestamp for the version string. We choose sleep because it mimics the user's behavior.
	time.Sleep(1 * time.Second)

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	approved := false

	log.Debug("Waiting for issue pipeline tasks.")
	prevStatus := "UNKNOWN"
	for range ticker.C {
		legacyIssue, err := ctl.getIssue(id)
		if err != nil {
			return api.TaskFailed, err
		}

		issue, err := ctl.issueServiceClient.GetIssue(ctx, &v1.GetIssueRequest{
			Name: fmt.Sprintf("projects/%d/issues/%d", legacyIssue.ProjectID, legacyIssue.ID),
		})
		if err != nil {
			return api.TaskFailed, err
		}
		if !issue.ApprovalFindingDone {
			continue
		}

		status, err := getNextTaskStatus(legacyIssue)
		if err != nil {
			return status, err
		}
		if string(status) != prevStatus {
			log.Debug(fmt.Sprintf("Status changed: %s -> %s.", prevStatus, status))
			prevStatus = string(status)
		}
		switch status {
		case api.TaskPendingApproval:
			if approveOnce && approved {
				return api.TaskDone, nil
			}
			if err := approveFunc(legacyIssue); err != nil {
				if strings.Contains(err.Error(), "The task has not passed all the checks yet") {
					continue
				}
				// "invalid task status transition" error means this task has been approved.
				// we continue and get its status in the next round.
				if strings.Contains(err.Error(), "invalid task status transition") {
					continue
				}
				return api.TaskFailed, err
			}
			approved = true
		case api.TaskFailed, api.TaskDone, api.TaskCanceled:
			return status, err
		case api.TaskPending, api.TaskRunning:
			approved = true
			// keep waiting
		}
	}
	return api.TaskDone, nil
}
