package store

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/lib/pq"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
)

// issueRaw is the store model for an Issue.
// Fields have exactly the same meanings as Issue.
type issueRaw struct {
	ID int

	// Standard fields
	CreatorID int
	CreatedTs int64
	UpdaterID int
	UpdatedTs int64

	// Related fields
	ProjectID  int
	PipelineID int

	// Domain specific fields
	Name                  string
	Status                api.IssueStatus
	Type                  api.IssueType
	Description           string
	AssigneeID            int
	AssigneeNeedAttention bool
	Payload               string
}

// toIssue creates an instance of Issue based on the issueRaw.
// This is intended to be called when we need to compose an Issue relationship.
func (raw *issueRaw) toIssue() *api.Issue {
	return &api.Issue{
		ID: raw.ID,

		// Standard fields
		CreatorID: raw.CreatorID,
		CreatedTs: raw.CreatedTs,
		UpdaterID: raw.UpdaterID,
		UpdatedTs: raw.UpdatedTs,

		// Related fields
		ProjectID:  raw.ProjectID,
		PipelineID: raw.PipelineID,

		// Domain specific fields
		Name:                  raw.Name,
		Status:                raw.Status,
		Type:                  raw.Type,
		Description:           raw.Description,
		AssigneeID:            raw.AssigneeID,
		AssigneeNeedAttention: raw.AssigneeNeedAttention,
		Payload:               raw.Payload,
	}
}

// GetIssueByID gets an instance of Issue.
func (s *Store) GetIssueByID(ctx context.Context, id int) (*api.Issue, error) {
	find := &api.IssueFind{ID: &id}
	issueRaw, err := s.getIssueRaw(ctx, find)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get Issue with ID %d", id)
	}
	if issueRaw == nil {
		return nil, nil
	}
	issue, err := s.composeIssue(ctx, issueRaw)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compose Issue with issueRaw[%+v]", issueRaw)
	}
	return issue, nil
}

// FindIssue finds a list of issues.
func (s *Store) FindIssue(ctx context.Context, find *api.IssueFind) ([]*api.Issue, error) {
	issueRawList, err := s.findIssueRaw(ctx, find)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find Issue list with IssueFind[%+v]", find)
	}
	var issueList []*api.Issue
	for _, raw := range issueRawList {
		issue, err := s.composeIssue(ctx, raw)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to compose Issue with issueRaw[%+v]", raw)
		}
		issueList = append(issueList, issue)
	}

	return issueList, nil
}

// FindIssueStripped finds a list of issues in stripped format.
// We do not load the pipeline in order to reduce the size of the response payload and the complexity of composing the issue list.
func (s *Store) FindIssueStripped(ctx context.Context, find *api.IssueFind) ([]*api.Issue, error) {
	issueRawList, err := s.findIssueRaw(ctx, find)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find Issue list with IssueFind[%+v]", find)
	}
	var issueList []*api.Issue
	for _, raw := range issueRawList {
		issue, err := s.composeIssueStripped(ctx, raw)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to compose Issue with issueRaw[%+v]", raw)
		}
		// If no specified project, filter out issues belonging to archived project
		if issue == nil || issue.Project == nil || issue.Project.RowStatus == api.Archived {
			continue
		}
		issueList = append(issueList, issue)
	}

	return issueList, nil
}

// CreateIssueValidateOnly creates an issue for validation purpose
// Do NOT write to the database.
func (s *Store) CreateIssueValidateOnly(ctx context.Context, pipelineCreate *api.PipelineCreate, create *IssueMessage, creatorID int) (*api.Issue, error) {
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
		PipelineID:  pipeline.ID,
		Pipeline:    pipeline,
	}

	if err := s.composeIssueValidateOnly(ctx, issue); err != nil {
		return nil, err
	}

	return issue, nil
}

// createPipelineValidateOnly creates a pipeline for validation purpose
// Do NOT write to the database.
func (s *Store) createPipelineValidateOnly(ctx context.Context, create *api.PipelineCreate) (*api.Pipeline, error) {
	creator, err := s.GetPrincipalByID(ctx, create.CreatorID)
	if err != nil {
		return nil, err
	}
	// We cannot emit ID or use default zero by following https://google.aip.dev/163, otherwise
	// jsonapi resource relationships will collide different resources into the same bucket.
	id := 0
	ts := time.Now().Unix()
	pipeline := &api.Pipeline{
		ID:        id,
		Name:      create.Name,
		Status:    api.PipelineOpen,
		CreatorID: create.CreatorID,
		Creator:   creator,
		CreatedTs: ts,
		UpdaterID: create.CreatorID,
		Updater:   creator,
		UpdatedTs: ts,
	}
	for _, sc := range create.StageList {
		id++
		env, err := s.GetEnvironmentByID(ctx, sc.EnvironmentID)
		if err != nil {
			return nil, err
		}
		stage := &api.Stage{
			ID:            id,
			Name:          sc.Name,
			CreatorID:     create.CreatorID,
			Creator:       creator,
			CreatedTs:     ts,
			UpdaterID:     create.CreatorID,
			Updater:       creator,
			UpdatedTs:     ts,
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
				CreatorID:         create.CreatorID,
				Creator:           creator,
				CreatedTs:         ts,
				UpdaterID:         create.CreatorID,
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

func (s *Store) composeIssueValidateOnly(ctx context.Context, issue *api.Issue) error {
	creator, err := s.GetPrincipalByID(ctx, issue.CreatorID)
	if err != nil {
		return err
	}
	issue.Creator = creator

	updater, err := s.GetPrincipalByID(ctx, issue.UpdaterID)
	if err != nil {
		return err
	}
	issue.Updater = updater

	assignee, err := s.GetPrincipalByID(ctx, issue.AssigneeID)
	if err != nil {
		return err
	}
	issue.Assignee = assignee

	issueSubscriberFind := &api.IssueSubscriberFind{
		IssueID: &issue.ID,
	}
	issueSubscriberList, err := s.FindIssueSubscriber(ctx, issueSubscriberFind)
	if err != nil {
		return err
	}
	for _, issueSub := range issueSubscriberList {
		issue.SubscriberList = append(issue.SubscriberList, issueSub.Subscriber)
	}

	project, err := s.GetProjectByID(ctx, issue.ProjectID)
	if err != nil {
		return err
	}
	issue.Project = project
	return nil
}

// Note: MUST keep in sync with composeIssueValidateOnly.
func (s *Store) composeIssue(ctx context.Context, raw *issueRaw) (*api.Issue, error) {
	issue := raw.toIssue()

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

	issueSubscriberFind := &api.IssueSubscriberFind{
		IssueID: &issue.ID,
	}
	issueSubscriberList, err := s.FindIssueSubscriber(ctx, issueSubscriberFind)
	if err != nil {
		return nil, err
	}
	for _, issueSub := range issueSubscriberList {
		issue.SubscriberList = append(issue.SubscriberList, issueSub.Subscriber)
	}

	project, err := s.GetProjectByID(ctx, issue.ProjectID)
	if err != nil {
		return nil, err
	}
	issue.Project = project

	pipeline, err := s.GetPipelineByID(ctx, issue.PipelineID)
	if err != nil {
		return nil, err
	}
	issue.Pipeline = pipeline

	return issue, nil
}

// composeIssueStripped is a stripped version of compose issue only used in listing issues
// for reducing the cost and payload of composing a full issue.
func (s *Store) composeIssueStripped(ctx context.Context, raw *issueRaw) (*api.Issue, error) {
	issue := raw.toIssue()

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

	// TODO(d): add subscriber caching.
	issueSubscriberFind := &api.IssueSubscriberFind{
		IssueID: &issue.ID,
	}
	issueSubscriberList, err := s.FindIssueSubscriber(ctx, issueSubscriberFind)
	if err != nil {
		return nil, err
	}
	for _, issueSub := range issueSubscriberList {
		issue.SubscriberList = append(issue.SubscriberList, issueSub.Subscriber)
	}

	project, err := s.GetProjectByID(ctx, issue.ProjectID)
	if err != nil {
		return nil, err
	}
	issue.Project = project

	// Creating a stripped pipeline.
	find := &api.PipelineFind{ID: &issue.PipelineID}
	pipelineRaw, err := s.getPipelineRaw(ctx, find)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get Pipeline with ID %d", issue.PipelineID)
	}
	if pipelineRaw == nil {
		return nil, nil
	}
	pipeline := pipelineRaw.toPipeline()

	stageRawList, err := s.findStageRaw(ctx, &api.StageFind{PipelineID: &issue.PipelineID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find Stage list with StageFind[%+v]", find)
	}
	var stageList []*api.Stage
	for _, raw := range stageRawList {
		stage := raw.toStage()
		env, err := s.GetEnvironmentByID(ctx, stage.EnvironmentID)
		if err != nil {
			return nil, err
		}
		stage.Environment = env
		taskFind := &api.TaskFind{
			PipelineID: &stage.PipelineID,
			StageID:    &stage.ID,
		}
		taskRawList, err := s.findTaskRaw(ctx, taskFind)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to find task list with TaskFind[%+v]", taskFind)
		}
		for _, taskRaw := range taskRawList {
			stage.TaskList = append(stage.TaskList, taskRaw.toTask())
		}
		stageList = append(stageList, stage)
	}
	pipeline.StageList = stageList

	issue.Pipeline = pipeline

	return issue, nil
}

// findIssueRaw retrieves a list of issues based on find.
func (s *Store) findIssueRaw(ctx context.Context, find *api.IssueFind) ([]*issueRaw, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := s.findIssueImpl(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	if err == nil {
		for _, issue := range list {
			if err := s.cache.UpsertCache(issueCacheNamespace, issue.ID, issue); err != nil {
				return nil, err
			}
		}
	}

	return list, nil
}

// getIssueRaw retrieves a single issue based on find.
// Returns ECONFLICT if finding more than 1 matching records.
func (s *Store) getIssueRaw(ctx context.Context, find *api.IssueFind) (*issueRaw, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := s.findIssueImpl(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	if len(list) == 0 {
		return nil, nil
	} else if len(list) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: errors.Errorf("found %d issues with filter %+v, expect 1", len(list), find)}
	}
	if err := s.cache.UpsertCache(issueCacheNamespace, list[0].ID, list[0]); err != nil {
		return nil, err
	}
	return list[0], nil
}

func (*Store) findIssueImpl(ctx context.Context, tx *Tx, find *api.IssueFind) ([]*issueRaw, error) {
	// Build WHERE clause.
	where, args := []string{"TRUE"}, []interface{}{}
	if v := find.ID; v != nil {
		where, args = append(where, fmt.Sprintf("id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.PipelineID; v != nil {
		where, args = append(where, fmt.Sprintf("pipeline_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.ProjectID; v != nil {
		where, args = append(where, fmt.Sprintf("project_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.PrincipalID; v != nil {
		if find.CreatorID != nil || find.AssigneeID != nil || find.SubscriberID != nil {
			return nil, &common.Error{Code: common.Invalid, Err: errors.New("principal_id cannot be used with creator_id, assignee_id, or subscriber_id")}
		}
		where = append(where, fmt.Sprintf("(creator_id = $%d OR assignee_id = $%d OR EXISTS (SELECT 1 FROM issue_subscriber WHERE issue_id = issue.id AND subscriber_id = $%d))", len(args)+1, len(args)+2, len(args)+3))
		args = append(args, *v)
		args = append(args, *v)
		args = append(args, *v)
	}
	if v := find.CreatorID; v != nil {
		where, args = append(where, fmt.Sprintf("creator_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.AssigneeID; v != nil {
		where, args = append(where, fmt.Sprintf("assignee_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.AssigneeNeedAttention; v != nil {
		where, args = append(where, fmt.Sprintf("assignee_need_attention = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.SubscriberID; v != nil {
		where, args = append(where, fmt.Sprintf("EXISTS (SELECT 1 FROM issue_subscriber WHERE issue_id = issue.id AND subscriber_id = $%d)", len(args)+1)), append(args, *v)
	}
	if v := find.SinceID; v != nil {
		where, args = append(where, fmt.Sprintf("id <= $%d", len(args)+1)), append(args, *v)
	}

	if len(find.StatusList) != 0 {
		list := []string{}
		for _, status := range find.StatusList {
			list = append(list, fmt.Sprintf("$%d", len(args)+1))
			args = append(args, status)
		}
		where = append(where, fmt.Sprintf("status IN (%s)", strings.Join(list, ",")))
	}

	query := `
		SELECT
			id,
			creator_id,
			created_ts,
			updater_id,
			updated_ts,
			project_id,
			pipeline_id,
			name,
			status,
			type,
			description,
			assignee_id,
			assignee_need_attention,
			payload
		FROM issue
		WHERE ` + strings.Join(where, " AND ")
	query += " ORDER BY id DESC"
	if v := find.Limit; v != nil {
		query += fmt.Sprintf(" LIMIT %d", *v)
	}

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()

	// Iterate over result set and deserialize rows into issuerRawList.
	var issuerRawList []*issueRaw
	for rows.Next() {
		var issueRaw issueRaw
		if err := rows.Scan(
			&issueRaw.ID,
			&issueRaw.CreatorID,
			&issueRaw.CreatedTs,
			&issueRaw.UpdaterID,
			&issueRaw.UpdatedTs,
			&issueRaw.ProjectID,
			&issueRaw.PipelineID,
			&issueRaw.Name,
			&issueRaw.Status,
			&issueRaw.Type,
			&issueRaw.Description,
			&issueRaw.AssigneeID,
			&issueRaw.AssigneeNeedAttention,
			&issueRaw.Payload,
		); err != nil {
			return nil, FormatError(err)
		}

		issuerRawList = append(issuerRawList, &issueRaw)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return issuerRawList, nil
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
	PipelineUID   int

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
}

// FindIssueMessage is the message to find issues.
type FindIssueMessage struct {
	UID        *int
	ProjectUID *int
	PipelineID *int
	// Find issues where principalID is either creator, assignee or subscriber.
	PrincipalID *int
	// To support pagination, we add into creator, assignee and subscriber.
	// Only principleID or one of the following three fields can be set.
	CreatorID             *int
	AssigneeID            *int
	SubscriberID          *int
	AssigneeNeedAttention *bool

	StatusList []api.IssueStatus
	// If specified, only find issues whose ID is smaller that SinceID.
	SinceID *int
	// If specified, then it will only fetch "Limit" most recently updated issues
	Limit *int
}

// GetIssueV2 gets issue by issue UID.
func (s *Store) GetIssueV2(ctx context.Context, find *FindIssueMessage) (*IssueMessage, error) {
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

	return issue, nil
}

// CreateIssueV2 creates a new issue.
func (s *Store) CreateIssueV2(ctx context.Context, create *IssueMessage, creatorID int) (*IssueMessage, error) {
	if create.Payload == "" {
		create.Payload = "{}"
	}
	creator, err := s.GetUserByID(ctx, creatorID)
	if err != nil {
		return nil, err
	}

	query := `
		INSERT INTO issue (
			creator_id,
			updater_id,
			project_id,
			pipeline_id,
			name,
			status,
			type,
			description,
			assignee_id,
			assignee_need_attention,
			payload
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_ts, updated_ts
	`

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	if err := tx.QueryRowContext(ctx, query,
		creatorID,
		creatorID,
		create.Project.UID,
		create.PipelineUID,
		create.Title,
		api.IssueOpen,
		create.Type,
		create.Description,
		create.Assignee.ID,
		create.NeedAttention,
		create.Payload,
	).Scan(
		&create.UID,
		&create.createdTs,
		&create.updatedTs,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return nil, FormatError(err)
	}
	create.CreatedTime = time.Unix(create.createdTs, 0)
	create.UpdatedTime = time.Unix(create.updatedTs, 0)
	create.Creator = creator
	create.Updater = creator

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return create, nil
}

// UpdateIssueV2 updates an issue.
func (s *Store) UpdateIssueV2(ctx context.Context, uid int, patch *UpdateIssueMessage, updaterID int) (*IssueMessage, error) {
	set, args := []string{"updater_id = $1"}, []interface{}{updaterID}
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
	args = append(args, uid)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, fmt.Sprintf(`
		UPDATE issue
		SET `+strings.Join(set, ", ")+`
		WHERE id = $%d`, len(args)),
		args...,
	); err != nil {
		return nil, FormatError(err)
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return s.GetIssueV2(ctx, &FindIssueMessage{UID: &uid})
}

// ListIssueV2 returns the list of issues by find query.
func (s *Store) ListIssueV2(ctx context.Context, find *FindIssueMessage) ([]*IssueMessage, error) {
	where, args := []string{"TRUE"}, []interface{}{}
	if v := find.UID; v != nil {
		where, args = append(where, fmt.Sprintf("issue.id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.PipelineID; v != nil {
		where, args = append(where, fmt.Sprintf("issue.pipeline_id = $%d", len(args)+1)), append(args, *v)
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
	if v := find.AssigneeNeedAttention; v != nil {
		where, args = append(where, fmt.Sprintf("issue.assignee_need_attention = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.SubscriberID; v != nil {
		where, args = append(where, fmt.Sprintf("EXISTS (SELECT 1 FROM issue_subscriber WHERE issue_subscriber.issue_id = issue.id AND issue_subscriber.subscriber_id = $%d)", len(args)+1)), append(args, *v)
	}
	if v := find.SinceID; v != nil {
		where, args = append(where, fmt.Sprintf("issue.id <= $%d", len(args)+1)), append(args, *v)
	}

	if len(find.StatusList) != 0 {
		var list []string
		for _, status := range find.StatusList {
			list = append(list, fmt.Sprintf("$%d", len(args)+1))
			args = append(args, status)
		}
		where = append(where, fmt.Sprintf("issue.status IN (%s)", strings.Join(list, ", ")))
	}
	var issues []*IssueMessage
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, fmt.Sprintf(`
		SELECT
			issue.id,
			issue.creator_id,
			issue.created_ts,
			issue.updater_id,
			issue.updated_ts,
			issue.project_id,
			issue.pipeline_id,
			issue.name,
			issue.status,
			issue.type,
			issue.description,
			issue.assignee_id,
			issue.assignee_need_attention,
			issue.payload,
			ARRAY_AGG (
				issue_subscriber.subscriber_id
			) subscribers
		FROM issue
		LEFT JOIN issue_subscriber ON issue.id = issue_subscriber.issue_id
		WHERE %s
		GROUP BY issue.id`, strings.Join(where, " AND ")),
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()
	for rows.Next() {
		var issue IssueMessage
		var subscribers []sql.NullInt32
		if err := rows.Scan(
			&issue.UID,
			&issue.creatorUID,
			&issue.createdTs,
			&issue.updaterUID,
			&issue.updatedTs,
			&issue.projectUID,
			&issue.PipelineUID,
			&issue.Title,
			&issue.Status,
			&issue.Type,
			&issue.Description,
			&issue.assigneeUID,
			&issue.NeedAttention,
			&issue.Payload,
			pq.Array(&subscribers),
		); err != nil {
			return nil, FormatError(err)
		}
		for _, subscriber := range subscribers {
			if !subscriber.Valid {
				continue
			}
			issue.subscriberUIDs = append(issue.subscriberUIDs, int(subscriber.Int32))
		}
		issues = append(issues, &issue)
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
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
	}
	return issues, nil
}
