package store

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-ego/gse"
	"github.com/jackc/pgtype"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
)

// GetIssueByID gets an instance of Issue.
func (s *Store) GetIssueByID(ctx context.Context, id int) (*api.Issue, error) {
	issue, err := s.GetIssueV2(ctx, &FindIssueMessage{UID: &id})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get Issue with ID %d", id)
	}
	if issue == nil {
		return nil, nil
	}
	composedIssue, err := s.composeIssue(ctx, issue)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compose issue %d", id)
	}
	return composedIssue, nil
}

// FindIssueStripped finds a list of issues in stripped format.
// We do not load the pipeline in order to reduce the size of the response payload and the complexity of composing the issue list.
func (s *Store) FindIssueStripped(ctx context.Context, find *FindIssueMessage) ([]*api.Issue, error) {
	issues, err := s.ListIssueV2(ctx, find)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find Issue list with IssueFind[%+v]", find)
	}
	var composedIssues []*api.Issue
	for _, issue := range issues {
		composedIssue, err := s.composeIssueStripped(ctx, issue)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to compose issue %d", issue.UID)
		}
		// If no specified project, filter out issues belonging to archived project
		if composedIssue == nil || composedIssue.Project == nil || composedIssue.Project.RowStatus == api.Archived {
			continue
		}
		composedIssues = append(composedIssues, composedIssue)
	}

	return composedIssues, nil
}

// CreateIssueValidateOnly creates an issue for validation purpose
// Do NOT write to the database.
func (s *Store) CreateIssueValidateOnly(ctx context.Context, pipelineCreate *PipelineMessage, create *IssueMessage, creatorID int) (*api.Issue, error) {
	pipeline, err := s.createPipelineValidateOnly(ctx, pipelineCreate)
	if err != nil {
		return nil, err
	}
	issue := &api.Issue{
		CreatorID:   creatorID,
		CreatedTs:   time.Now().Unix(),
		UpdaterID:   creatorID,
		UpdatedTs:   time.Now().Unix(),
		ProjectID:   create.Project.UID,
		Name:        create.Title,
		Status:      api.IssueOpen,
		Type:        create.Type,
		Description: create.Description,
		AssigneeID:  create.Assignee.ID,
		PipelineID:  &pipeline.ID,
		Pipeline:    pipeline,
	}

	creator, err := s.GetPrincipalByID(ctx, issue.CreatorID)
	if err != nil {
		return nil, err
	}
	issue.Creator = creator

	updater, err := s.GetPrincipalByID(ctx, issue.UpdaterID)
	if err != nil {
		return nil, err
	}
	issue.Updater = updater

	assignee, err := s.GetPrincipalByID(ctx, issue.AssigneeID)
	if err != nil {
		return nil, err
	}
	issue.Assignee = assignee

	project, err := s.GetProjectByID(ctx, issue.ProjectID)
	if err != nil {
		return nil, err
	}
	issue.Project = project

	return issue, nil
}

// createPipelineValidateOnly creates a pipeline for validation purpose
// Do NOT write to the database.
func (s *Store) createPipelineValidateOnly(ctx context.Context, create *PipelineMessage) (*api.Pipeline, error) {
	creator, err := s.GetPrincipalByID(ctx, api.SystemBotID)
	if err != nil {
		return nil, err
	}
	// We cannot emit ID or use default zero by following https://google.aip.dev/163, otherwise
	// jsonapi resource relationships will collide different resources into the same bucket.
	id := 0
	ts := time.Now().Unix()
	pipeline := &api.Pipeline{
		ID:   id,
		Name: create.Name,
	}
	for _, sc := range create.Stages {
		id++
		env, err := s.GetEnvironmentByID(ctx, sc.EnvironmentID)
		if err != nil {
			return nil, err
		}
		stage := &api.Stage{
			ID:            id,
			Name:          sc.Name,
			PipelineID:    sc.PipelineID,
			EnvironmentID: sc.EnvironmentID,
			Environment:   env,
		}
		// We don't know IDs before inserting, so we use array index instead.
		// indexBlockedByIndex[indexA] holds indices of the tasks that block taskList[indexA]
		indexBlockedByIndex := make(map[int][]int)
		for _, indexDAG := range sc.TaskIndexDAGList {
			indexBlockedByIndex[indexDAG.ToIndex] = append(indexBlockedByIndex[indexDAG.ToIndex], indexDAG.FromIndex)
		}
		idOffset := id + 1
		// The ID of sc.TaskList[index].ID equals index + idOffset.
		for index, tc := range sc.TaskList {
			id++
			var blockedBy []string
			for _, blockedByIndex := range indexBlockedByIndex[index] {
				// Convert array index to ID.
				blockedBy = append(blockedBy, strconv.Itoa(blockedByIndex+idOffset))
			}
			task := &api.Task{
				ID:                id,
				Name:              tc.Name,
				Status:            tc.Status,
				CreatorID:         api.SystemBotID,
				Creator:           creator,
				CreatedTs:         ts,
				UpdaterID:         api.SystemBotID,
				Updater:           creator,
				UpdatedTs:         ts,
				Type:              tc.Type,
				Payload:           tc.Payload,
				EarliestAllowedTs: tc.EarliestAllowedTs,
				PipelineID:        pipeline.ID,
				StageID:           stage.ID,
				InstanceID:        tc.InstanceID,
				DatabaseID:        tc.DatabaseID,
				BlockedBy:         blockedBy,
				Statement:         tc.Statement,
			}
			instance, err := s.GetInstanceByID(ctx, task.InstanceID)
			if err != nil {
				return nil, err
			}
			if instance == nil {
				return nil, errors.Errorf("instance not found with ID %v", task.InstanceID)
			}
			task.Instance = instance
			if task.DatabaseID != nil {
				database, err := s.GetDatabase(ctx, &api.DatabaseFind{ID: task.DatabaseID})
				if err != nil {
					return nil, err
				}
				if database == nil {
					return nil, errors.Errorf("database not found with ID %v", task.DatabaseID)
				}
				task.Database = database
			}

			stage.TaskList = append(stage.TaskList, task)
		}
		pipeline.StageList = append(pipeline.StageList, stage)
	}

	return pipeline, nil
}

//
// private functions
//

// Note: MUST keep in sync with composeIssueValidateOnly.
func (s *Store) composeIssue(ctx context.Context, issue *IssueMessage) (*api.Issue, error) {
	composedIssue := &api.Issue{
		ID:                    issue.UID,
		CreatorID:             issue.Creator.ID,
		CreatedTs:             issue.CreatedTime.Unix(),
		UpdaterID:             issue.Updater.ID,
		UpdatedTs:             issue.UpdatedTime.Unix(),
		ProjectID:             issue.Project.UID,
		PipelineID:            issue.PipelineUID,
		Name:                  issue.Title,
		Status:                issue.Status,
		Type:                  issue.Type,
		Description:           issue.Description,
		AssigneeID:            issue.Assignee.ID,
		AssigneeNeedAttention: issue.NeedAttention,
		Payload:               issue.Payload,
	}

	creator, err := s.GetPrincipalByID(ctx, issue.Creator.ID)
	if err != nil {
		return nil, err
	}
	composedIssue.Creator = creator
	updater, err := s.GetPrincipalByID(ctx, issue.Updater.ID)
	if err != nil {
		return nil, err
	}
	composedIssue.Updater = updater
	assignee, err := s.GetPrincipalByID(ctx, issue.Assignee.ID)
	if err != nil {
		return nil, err
	}
	composedIssue.Assignee = assignee

	for _, subscriber := range issue.Subscribers {
		composedSubscriber, err := s.GetPrincipalByID(ctx, subscriber.ID)
		if err != nil {
			return nil, err
		}
		composedIssue.SubscriberList = append(composedIssue.SubscriberList, composedSubscriber)
	}

	composedProject, err := s.GetProjectByID(ctx, issue.Project.UID)
	if err != nil {
		return nil, err
	}
	composedIssue.Project = composedProject

	if issue.PipelineUID != nil {
		pipeline, err := s.GetPipelineV2ByID(ctx, *issue.PipelineUID)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get pipeline with ID %d", *issue.PipelineUID)
		}
		if pipeline == nil {
			return nil, nil
		}
		composedPipeline, err := s.composePipeline(ctx, pipeline)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to compose pipeline")
		}
		composedIssue.Pipeline = composedPipeline
	}

	return composedIssue, nil
}

// composeIssueStripped is a stripped version of compose issue only used in listing issues
// for reducing the cost and payload of composing a full issue.
func (s *Store) composeIssueStripped(ctx context.Context, issue *IssueMessage) (*api.Issue, error) {
	composedIssue := &api.Issue{
		ID:                    issue.UID,
		CreatorID:             issue.Creator.ID,
		CreatedTs:             issue.CreatedTime.Unix(),
		UpdaterID:             issue.Updater.ID,
		UpdatedTs:             issue.UpdatedTime.Unix(),
		ProjectID:             issue.Project.UID,
		PipelineID:            issue.PipelineUID,
		Name:                  issue.Title,
		Status:                issue.Status,
		Type:                  issue.Type,
		Description:           issue.Description,
		AssigneeID:            issue.Assignee.ID,
		AssigneeNeedAttention: issue.NeedAttention,
		Payload:               issue.Payload,
	}

	creator, err := s.GetPrincipalByID(ctx, issue.Creator.ID)
	if err != nil {
		return nil, err
	}
	composedIssue.Creator = creator
	updater, err := s.GetPrincipalByID(ctx, issue.Updater.ID)
	if err != nil {
		return nil, err
	}
	composedIssue.Updater = updater
	assignee, err := s.GetPrincipalByID(ctx, issue.Assignee.ID)
	if err != nil {
		return nil, err
	}
	composedIssue.Assignee = assignee

	for _, subscriber := range issue.Subscribers {
		composedSubscriber, err := s.GetPrincipalByID(ctx, subscriber.ID)
		if err != nil {
			return nil, err
		}
		composedIssue.SubscriberList = append(composedIssue.SubscriberList, composedSubscriber)
	}

	composedProject, err := s.GetProjectByID(ctx, issue.Project.UID)
	if err != nil {
		return nil, err
	}
	composedIssue.Project = composedProject

	// Creating a stripped pipeline.
	if issue.PipelineUID != nil {
		pipeline, err := s.GetPipelineV2ByID(ctx, *issue.PipelineUID)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get Pipeline with ID %d", issue.PipelineUID)
		}
		if pipeline == nil {
			return nil, nil
		}
		composedPipeline, err := s.composeSimplePipeline(ctx, pipeline)
		if err != nil {
			return nil, err
		}
		composedIssue.Pipeline = composedPipeline
	}

	return composedIssue, nil
}

func (s *Store) composePipeline(ctx context.Context, pipeline *PipelineMessage) (*api.Pipeline, error) {
	composedPipeline := &api.Pipeline{
		ID:   pipeline.ID,
		Name: pipeline.Name,
	}

	tasks, err := s.ListTasks(ctx, &api.TaskFind{PipelineID: &pipeline.ID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find Task list for pipeline %v", pipeline.ID)
	}
	taskRuns, err := s.ListTaskRun(ctx, &TaskRunFind{PipelineID: &pipeline.ID})
	if err != nil {
		return nil, err
	}
	taskCheckRuns, err := s.ListTaskCheckRuns(ctx, &TaskCheckRunFind{PipelineID: &pipeline.ID})
	if err != nil {
		return nil, err
	}
	var composedTasks []*api.Task
	for _, task := range tasks {
		composedTask := task.toTask()
		creator, err := s.GetPrincipalByID(ctx, task.CreatorID)
		if err != nil {
			return nil, err
		}
		composedTask.Creator = creator
		updater, err := s.GetPrincipalByID(ctx, task.UpdaterID)
		if err != nil {
			return nil, err
		}
		composedTask.Updater = updater

		for _, taskRun := range taskRuns {
			if taskRun.TaskUID == task.ID {
				composedTaskRun := taskRun.toTaskRun()
				creator, err := s.GetPrincipalByID(ctx, composedTaskRun.CreatorID)
				if err != nil {
					return nil, err
				}
				composedTaskRun.Creator = creator
				updater, err := s.GetPrincipalByID(ctx, composedTaskRun.UpdaterID)
				if err != nil {
					return nil, err
				}
				composedTaskRun.Updater = updater
				composedTask.TaskRunList = append(composedTask.TaskRunList, composedTaskRun)
			}
		}
		for _, taskCheckRun := range taskCheckRuns {
			if taskCheckRun.TaskID == task.ID {
				composedTaskCheckRun := taskCheckRun.toTaskCheckRun()
				creator, err := s.GetPrincipalByID(ctx, taskCheckRun.CreatorID)
				if err != nil {
					return nil, err
				}
				composedTaskCheckRun.Creator = creator
				updater, err := s.GetPrincipalByID(ctx, taskCheckRun.UpdaterID)
				if err != nil {
					return nil, err
				}
				composedTaskCheckRun.Updater = updater
				composedTaskCheckRun.CreatedTs = taskCheckRun.CreatedTs
				composedTaskCheckRun.UpdatedTs = taskCheckRun.UpdatedTs
				composedTask.TaskCheckRunList = append(composedTask.TaskCheckRunList, composedTaskCheckRun)
			}
		}

		instance, err := s.GetInstanceByID(ctx, task.InstanceID)
		if err != nil {
			return nil, err
		}
		if instance == nil {
			return nil, errors.Errorf("instance not found with ID %v", task.InstanceID)
		}
		composedTask.Instance = instance
		if task.DatabaseID != nil {
			database, err := s.GetDatabase(ctx, &api.DatabaseFind{ID: task.DatabaseID})
			if err != nil {
				return nil, err
			}
			if database == nil {
				return nil, errors.Errorf("database not found with ID %v", task.DatabaseID)
			}
			composedTask.Database = database
		}

		composedTasks = append(composedTasks, composedTask)
	}

	stages, err := s.ListStageV2(ctx, pipeline.ID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find stages for pipeline %d", pipeline.ID)
	}
	var composedStages []*api.Stage
	for _, stage := range stages {
		environment, err := s.GetEnvironmentByID(ctx, stage.EnvironmentID)
		if err != nil {
			return nil, err
		}
		composedStage := &api.Stage{
			ID:            stage.ID,
			PipelineID:    stage.PipelineID,
			EnvironmentID: stage.EnvironmentID,
			Environment:   environment,
			Name:          stage.Name,
		}
		for _, composedTask := range composedTasks {
			if composedTask.StageID == stage.ID {
				composedStage.TaskList = append(composedStage.TaskList, composedTask)
			}
		}

		composedStages = append(composedStages, composedStage)
	}
	composedPipeline.StageList = composedStages

	return composedPipeline, nil
}

func (s *Store) composeSimplePipeline(ctx context.Context, pipeline *PipelineMessage) (*api.Pipeline, error) {
	tasks, err := s.ListTasks(ctx, &api.TaskFind{PipelineID: &pipeline.ID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find tasks for pipeline %d", pipeline.ID)
	}

	stages, err := s.ListStageV2(ctx, pipeline.ID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find stage list")
	}
	var composedStages []*api.Stage
	for _, stage := range stages {
		environment, err := s.GetEnvironmentByID(ctx, stage.EnvironmentID)
		if err != nil {
			return nil, err
		}
		composedStage := &api.Stage{
			ID:            stage.ID,
			Name:          stage.Name,
			EnvironmentID: stage.EnvironmentID,
			Environment:   environment,
			PipelineID:    stage.PipelineID,
		}

		for _, task := range tasks {
			if task.StageID == stage.ID {
				composedStage.TaskList = append(composedStage.TaskList, task.toTask())
			}
		}
		composedStages = append(composedStages, composedStage)
	}

	return &api.Pipeline{
		ID:        pipeline.ID,
		Name:      pipeline.Name,
		StageList: composedStages,
	}, nil
}

// IssueMessage is the mssage for issues.
type IssueMessage struct {
	Project       *ProjectMessage
	Title         string
	Status        api.IssueStatus
	Type          api.IssueType
	Description   string
	Assignee      *UserMessage
	NeedAttention bool
	Payload       string
	Subscribers   []*UserMessage
	PipelineUID   *int
	PlanUID       *int64

	// The following fields are output only and not used for create().
	UID         int
	Creator     *UserMessage
	CreatedTime time.Time
	Updater     *UserMessage
	UpdatedTime time.Time

	// Internal fields.
	projectUID     int
	assigneeUID    int
	subscriberUIDs []int
	creatorUID     int
	createdTs      int64
	updaterUID     int
	updatedTs      int64
}

// UpdateIssueMessage is the mssage for updating an issue.
type UpdateIssueMessage struct {
	Title         *string
	Status        *api.IssueStatus
	Description   *string
	Assignee      *UserMessage
	NeedAttention *bool
	Payload       *string
	Subscribers   *[]*UserMessage

	PipelineUID *int
}

// FindIssueMessage is the message to find issues.
type FindIssueMessage struct {
	UID        *int
	ProjectUID *int
	PlanUID    *int64
	PipelineID *int
	// Find issues where principalID is either creator, assignee or subscriber.
	PrincipalID *int
	// To support pagination, we add into creator, assignee and subscriber.
	// Only principleID or one of the following three fields can be set.
	CreatorID     *int
	AssigneeID    *int
	SubscriberID  *int
	NeedAttention *bool

	StatusList []api.IssueStatus
	// If specified, only find issues whose ID is smaller that SinceID.
	SinceID *int
	// If specified, then it will only fetch "Limit" most recently updated issues
	Limit *int

	Stripped bool

	Query *string
}

// GetIssueV2 gets issue by issue UID.
func (s *Store) GetIssueV2(ctx context.Context, find *FindIssueMessage) (*IssueMessage, error) {
	if find.UID != nil {
		if issue, ok := s.issueCache.Load(*find.UID); ok {
			return issue.(*IssueMessage), nil
		}
	}
	if find.PipelineID != nil {
		if issue, ok := s.issueByPipelineCache.Load(*find.PipelineID); ok {
			return issue.(*IssueMessage), nil
		}
	}

	issues, err := s.ListIssueV2(ctx, find)
	if err != nil {
		return nil, err
	}
	if len(issues) == 0 {
		return nil, nil
	}
	if len(issues) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: errors.Errorf("found %d issues with find %#v, expect 1", len(issues), find)}
	}
	issue := issues[0]

	s.issueCache.Store(issue.UID, issue)
	s.issueByPipelineCache.Store(issue.PipelineUID, issue)
	return issue, nil
}

// CreateIssueV2 creates a new issue.
func (s *Store) CreateIssueV2(ctx context.Context, create *IssueMessage, creatorID int) (*IssueMessage, error) {
	create.Status = api.IssueOpen
	if create.Payload == "" {
		create.Payload = "{}"
	}
	creator, err := s.GetUserByID(ctx, creatorID)
	if err != nil {
		return nil, err
	}

	var seg gse.Segmenter
	seg.LoadDict()
	tsVector := getTsVector(&seg, fmt.Sprintf("%s %s", create.Title, create.Description))

	query := `
		INSERT INTO issue (
			creator_id,
			updater_id,
			project_id,
			pipeline_id,
			plan_id,
			name,
			status,
			type,
			description,
			assignee_id,
			assignee_need_attention,
			payload,
			ts_vector
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13::tsvector)
		RETURNING id, created_ts, updated_ts
	`

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if err := tx.QueryRowContext(ctx, query,
		creatorID,
		creatorID,
		create.Project.UID,
		create.PipelineUID,
		create.PlanUID,
		create.Title,
		create.Status,
		create.Type,
		create.Description,
		create.Assignee.ID,
		create.NeedAttention,
		create.Payload,
		tsVector,
	).Scan(
		&create.UID,
		&create.createdTs,
		&create.updatedTs,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return nil, err
	}
	create.CreatedTime = time.Unix(create.createdTs, 0)
	create.UpdatedTime = time.Unix(create.updatedTs, 0)
	create.Creator = creator
	create.Updater = creator

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	s.issueCache.Store(create.UID, create)
	s.issueByPipelineCache.Store(create.PipelineUID, create)
	return create, nil
}

// UpdateIssueV2 updates an issue.
func (s *Store) UpdateIssueV2(ctx context.Context, uid int, patch *UpdateIssueMessage, updaterID int) (*IssueMessage, error) {
	oldIssue, err := s.GetIssueV2(ctx, &FindIssueMessage{UID: &uid})
	if err != nil {
		return nil, err
	}

	set, args := []string{"updater_id = $1"}, []any{updaterID}

	if v := patch.PipelineUID; v != nil {
		set, args = append(set, fmt.Sprintf("pipeline_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Title; v != nil {
		set, args = append(set, fmt.Sprintf("name = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Status; v != nil {
		set, args = append(set, fmt.Sprintf("status = $%d", len(args)+1)), append(args, api.IssueStatus(*v))
	}
	if v := patch.Description; v != nil {
		set, args = append(set, fmt.Sprintf("description = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Assignee; v != nil {
		set, args = append(set, fmt.Sprintf("assignee_id = $%d", len(args)+1)), append(args, v.ID)
	}
	if v := patch.NeedAttention; v != nil {
		set, args = append(set, fmt.Sprintf("assignee_need_attention = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Payload; v != nil {
		set, args = append(set, fmt.Sprintf("payload = $%d", len(args)+1)), append(args, *v)
	}
	if patch.Title != nil || patch.Description != nil {
		title := oldIssue.Title
		if patch.Title != nil {
			title = *patch.Title
		}
		description := oldIssue.Description
		if patch.Description != nil {
			description = *patch.Description
		}

		var seg gse.Segmenter
		seg.LoadDict()

		tsVector := getTsVector(&seg, fmt.Sprintf("%s %s", title, description))
		set = append(set, fmt.Sprintf("ts_vector = $%d", len(args)+1))
		args = append(args, tsVector)
	}

	args = append(args, uid)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, fmt.Sprintf(`
		UPDATE issue
		SET `+strings.Join(set, ", ")+`
		WHERE id = $%d`, len(args)),
		args...,
	); err != nil {
		return nil, err
	}

	if patch.Subscribers != nil {
		if err := setSubscribers(ctx, tx, uid, *patch.Subscribers); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	// Invalid the cache and read the value again.
	s.issueCache.Delete(uid)
	s.issueByPipelineCache.Delete(oldIssue.PipelineUID)
	return s.GetIssueV2(ctx, &FindIssueMessage{UID: &uid})
}

func setSubscribers(ctx context.Context, tx *Tx, issueUID int, subscribers []*UserMessage) error {
	subscriberIDs := make(map[int]bool)
	for _, subscriber := range subscribers {
		subscriberIDs[subscriber.ID] = true
	}

	oldSubscriberIDs := make(map[int]bool)
	rows, err := tx.QueryContext(ctx, `
		SELECT
			subscriber_id
		FROM issue_subscriber
		WHERE issue_id = $1`,
		issueUID,
	)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var subscriberID int
		if err := rows.Scan(
			&subscriberID,
		); err != nil {
			return err
		}

		oldSubscriberIDs[subscriberID] = true
	}
	if err := rows.Err(); err != nil {
		return err
	}

	var adds, deletes []int
	for v := range oldSubscriberIDs {
		if _, ok := subscriberIDs[v]; !ok {
			deletes = append(deletes, v)
		}
	}
	for v := range subscriberIDs {
		if _, ok := oldSubscriberIDs[v]; !ok {
			adds = append(adds, v)
		}
	}
	if len(adds) > 0 {
		var tokens []string
		var args []any
		for i, v := range adds {
			tokens = append(tokens, fmt.Sprintf("($%d, $%d)", 2*i+1, 2*i+2))
			args = append(args, issueUID, v)
		}
		query := fmt.Sprintf(`INSERT INTO issue_subscriber (issue_id, subscriber_id) VALUES %s`, strings.Join(tokens, ", "))
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return err
		}
	}
	if len(deletes) > 0 {
		var tokens []string
		var args []any
		args = append(args, issueUID)
		for i, v := range deletes {
			tokens = append(tokens, fmt.Sprintf("$%d", i+2))
			args = append(args, v)
		}
		query := fmt.Sprintf(`DELETE FROM issue_subscriber WHERE issue_id = $1 AND subscriber_id IN (%s)`, strings.Join(tokens, ", "))
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return err
		}
	}
	return nil
}

// ListIssueV2 returns the list of issues by find query.
func (s *Store) ListIssueV2(ctx context.Context, find *FindIssueMessage) ([]*IssueMessage, error) {
	orderByClause := "ORDER BY issue.id DESC"
	from := "issue"
	where, args := []string{"TRUE"}, []any{}
	if v := find.UID; v != nil {
		where, args = append(where, fmt.Sprintf("issue.id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.PipelineID; v != nil {
		where, args = append(where, fmt.Sprintf("issue.pipeline_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.PlanUID; v != nil {
		where, args = append(where, fmt.Sprintf("issue.plan_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.ProjectUID; v != nil {
		where, args = append(where, fmt.Sprintf("issue.project_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.PrincipalID; v != nil {
		if find.CreatorID != nil || find.AssigneeID != nil || find.SubscriberID != nil {
			return nil, &common.Error{Code: common.Invalid, Err: errors.New("principal_id cannot be used with creator_id, assignee_id, or subscriber_id")}
		}
		where = append(where, fmt.Sprintf("(issue.creator_id = $%d OR issue.assignee_id = $%d OR EXISTS (SELECT 1 FROM issue_subscriber WHERE issue_subscriber.issue_id = issue.id AND issue_subscriber.subscriber_id = $%d))", len(args)+1, len(args)+2, len(args)+3))
		args = append(args, *v)
		args = append(args, *v)
		args = append(args, *v)
	}
	if v := find.CreatorID; v != nil {
		where, args = append(where, fmt.Sprintf("issue.creator_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.AssigneeID; v != nil {
		where, args = append(where, fmt.Sprintf("issue.assignee_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.NeedAttention; v != nil {
		where, args = append(where, fmt.Sprintf("issue.assignee_need_attention = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.SubscriberID; v != nil {
		where, args = append(where, fmt.Sprintf("EXISTS (SELECT 1 FROM issue_subscriber WHERE issue_subscriber.issue_id = issue.id AND issue_subscriber.subscriber_id = $%d)", len(args)+1)), append(args, *v)
	}
	if v := find.SinceID; v != nil {
		where, args = append(where, fmt.Sprintf("issue.id <= $%d", len(args)+1)), append(args, *v)
	}
	if v := find.Query; v != nil {
		var seg gse.Segmenter
		seg.LoadDict()
		tsQuery := getTsQuery(&seg, *v)
		from += fmt.Sprintf(`, CAST($%d AS tsquery) AS query`, len(args)+1)
		args = append(args, tsQuery)
		where = append(where, "issue.ts_vector @@ query")
		orderByClause = "ORDER BY ts_rank(issue.ts_vector, query) DESC, issue.id DESC"
	}
	if len(find.StatusList) != 0 {
		var list []string
		for _, status := range find.StatusList {
			list = append(list, fmt.Sprintf("$%d", len(args)+1))
			args = append(args, status)
		}
		where = append(where, fmt.Sprintf("issue.status IN (%s)", strings.Join(list, ", ")))
	}
	limitClause := ""
	if v := find.Limit; v != nil {
		limitClause = fmt.Sprintf(" LIMIT %d", *v)
	}

	var issues []*IssueMessage
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	query := fmt.Sprintf(`
	SELECT
		issue.id,
		issue.creator_id,
		issue.created_ts,
		issue.updater_id,
		issue.updated_ts,
		issue.project_id,
		issue.pipeline_id,
		issue.plan_id,
		issue.name,
		issue.status,
		issue.type,
		issue.description,
		issue.assignee_id,
		issue.assignee_need_attention,
		issue.payload,
		(SELECT ARRAY_AGG (issue_subscriber.subscriber_id) FROM issue_subscriber WHERE issue_subscriber.issue_id = issue.id) subscribers
	FROM %s
	WHERE %s
	%s
	%s`, from, strings.Join(where, " AND "), orderByClause, limitClause)

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var issue IssueMessage
		var pipelineUID sql.NullInt32
		var subscriberUIDs pgtype.Int4Array
		if err := rows.Scan(
			&issue.UID,
			&issue.creatorUID,
			&issue.createdTs,
			&issue.updaterUID,
			&issue.updatedTs,
			&issue.projectUID,
			&pipelineUID,
			&issue.PlanUID,
			&issue.Title,
			&issue.Status,
			&issue.Type,
			&issue.Description,
			&issue.assigneeUID,
			&issue.NeedAttention,
			&issue.Payload,
			&subscriberUIDs,
		); err != nil {
			return nil, err
		}
		if err := subscriberUIDs.AssignTo(&issue.subscriberUIDs); err != nil {
			return nil, err
		}
		if pipelineUID.Valid {
			v := int(pipelineUID.Int32)
			issue.PipelineUID = &v
		}
		issues = append(issues, &issue)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	// Populate from internal fields.
	for _, issue := range issues {
		project, err := s.GetProjectV2(ctx, &FindProjectMessage{UID: &issue.projectUID})
		if err != nil {
			return nil, err
		}
		issue.Project = project
		assignee, err := s.GetUserByID(ctx, issue.assigneeUID)
		if err != nil {
			return nil, err
		}
		issue.Assignee = assignee
		creator, err := s.GetUserByID(ctx, issue.creatorUID)
		if err != nil {
			return nil, err
		}
		issue.Creator = creator
		updater, err := s.GetUserByID(ctx, issue.updaterUID)
		if err != nil {
			return nil, err
		}
		issue.Updater = updater
		for _, subscriberUID := range issue.subscriberUIDs {
			subscriber, err := s.GetUserByID(ctx, subscriberUID)
			if err != nil {
				return nil, err
			}
			issue.Subscribers = append(issue.Subscribers, subscriber)
		}
		issue.CreatedTime = time.Unix(issue.createdTs, 0)
		issue.UpdatedTime = time.Unix(issue.updatedTs, 0)

		s.issueCache.Store(issue.UID, issue)
		s.issueByPipelineCache.Store(issue.PipelineUID, issue)
	}

	return issues, nil
}

// BatchUpdateIssueStatuses updates the status of multiple issues.
func (s *Store) BatchUpdateIssueStatuses(ctx context.Context, issueUIDs []int, status api.IssueStatus, updaterID int) error {
	var ids []string
	for _, id := range issueUIDs {
		ids = append(ids, fmt.Sprintf("%d", id))
	}
	query := fmt.Sprintf(`
		UPDATE issue
		SET status = $1, updater_id = $2
		WHERE id IN (%s)
		RETURNING id, pipeline_id;
	`, strings.Join(ids, ","))

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrapf(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, query, status, updaterID)
	if err != nil {
		return errors.Wrapf(err, "failed to query")
	}
	defer rows.Close()

	var issueIDs []int
	var pipelineIDs []int
	for rows.Next() {
		var issueID int
		var pipelineID sql.NullInt32
		if err := rows.Scan(&issueID, &pipelineID); err != nil {
			return errors.Wrapf(err, "failed to scan")
		}
		issueIDs = append(issueIDs, issueID)
		if pipelineID.Valid {
			pipelineIDs = append(pipelineIDs, int(pipelineID.Int32))
		}
	}
	if err := rows.Err(); err != nil {
		return errors.Wrapf(err, "failed to scan")
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrapf(err, "failed to commit")
	}

	for _, issueID := range issueIDs {
		s.issueCache.Delete(issueID)
	}
	for _, pipelineID := range pipelineIDs {
		s.issueByPipelineCache.Delete(pipelineID)
	}

	return nil
}

func getTsVector(seg *gse.Segmenter, text string) string {
	parts := seg.CutTrim(text)
	var tsVector strings.Builder
	for i, part := range parts {
		if i != 0 {
			_, _ = tsVector.WriteString(" ")
		}
		_, _ = tsVector.WriteString(fmt.Sprintf("%s:%d", part, i+1))
	}
	return tsVector.String()
}

func getTsQuery(seg *gse.Segmenter, text string) string {
	parts := seg.Trim(seg.CutSearch(text))
	var tsQuery strings.Builder
	for i, part := range parts {
		if i != 0 {
			_, _ = tsQuery.WriteString("|")
		}
		_, _ = tsQuery.WriteString(fmt.Sprintf("%s:*", part))
	}
	if tsQuery.Len() == 0 {
		return text
	}
	return tsQuery.String()
}
