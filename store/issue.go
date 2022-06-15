package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/metric"
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
	Name        string
	Status      api.IssueStatus
	Type        api.IssueType
	Description string
	AssigneeID  int
	Payload     string
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
		Name:        raw.Name,
		Status:      raw.Status,
		Type:        raw.Type,
		Description: raw.Description,
		AssigneeID:  raw.AssigneeID,
		Payload:     raw.Payload,
	}
}

// CreateIssue creates an instance of Issue
func (s *Store) CreateIssue(ctx context.Context, create *api.IssueCreate) (*api.Issue, error) {
	issueRaw, err := s.createIssueRaw(ctx, create)
	if err != nil {
		return nil, fmt.Errorf("failed to create Issue with IssueCreate[%+v], error[%w]", create, err)
	}
	issue, err := s.composeIssue(ctx, issueRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to compose Issue with issueRaw[%+v], error[%w]", issueRaw, err)
	}
	return issue, nil
}

// GetIssueByID gets an instance of Issue
func (s *Store) GetIssueByID(ctx context.Context, id int) (*api.Issue, error) {
	find := &api.IssueFind{ID: &id}
	issueRaw, err := s.getIssueRaw(ctx, find)
	if err != nil {
		return nil, fmt.Errorf("failed to get Issue with ID[%d], error[%w]", id, err)
	}
	if issueRaw == nil {
		return nil, nil
	}
	issue, err := s.composeIssue(ctx, issueRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to compose Issue with issueRaw[%+v], error[%w]", issueRaw, err)
	}
	return issue, nil
}

// GetIssueByPipelineID gets an instance of Issue
func (s *Store) GetIssueByPipelineID(ctx context.Context, id int) (*api.Issue, error) {
	find := &api.IssueFind{PipelineID: &id}
	issueRaw, err := s.getIssueRaw(ctx, find)
	if err != nil {
		return nil, fmt.Errorf("failed to get Issue with PipelineID[%d], error[%w]", id, err)
	}
	if issueRaw == nil {
		return nil, nil
	}
	issue, err := s.composeIssue(ctx, issueRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to compose Issue with issueRaw[%+v], error[%w]", issueRaw, err)
	}
	return issue, nil
}

// FindIssue finds a list of Issue instances
func (s *Store) FindIssue(ctx context.Context, find *api.IssueFind) ([]*api.Issue, error) {
	issueRawList, err := s.findIssueRaw(ctx, find)
	if err != nil {
		return nil, fmt.Errorf("failed to find Issue list with IssueFind[%+v], error[%w]", find, err)
	}
	var issueList []*api.Issue
	for _, raw := range issueRawList {
		issue, err := s.composeIssue(ctx, raw)
		if err != nil {
			return nil, fmt.Errorf("failed to compose Issue with issueRaw[%+v], error[%w]", raw, err)
		}
		issueList = append(issueList, issue)
	}
	return issueList, nil
}

// PatchIssue patches an instance of Issue
func (s *Store) PatchIssue(ctx context.Context, patch *api.IssuePatch) (*api.Issue, error) {
	issueRaw, err := s.patchIssueRaw(ctx, patch)
	if err != nil {
		return nil, fmt.Errorf("failed to patch Issue with IssuePatch[%+v], error[%w]", patch, err)
	}
	issue, err := s.composeIssue(ctx, issueRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to compose Issue with issueRaw[%+v], error[%w]", issueRaw, err)
	}
	return issue, nil
}

// CountIssueGroupByTypeAndStatus counts the number of issue and group by type and status.
// Used by the metric collector.
func (s *Store) CountIssueGroupByTypeAndStatus(ctx context.Context) ([]*metric.IssueCountMetric, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	rows, err := tx.PTx.QueryContext(ctx, `
		SELECT type, status, COUNT(*)
		FROM issue
		WHERE (id <= 101 AND updater_id != 1) OR id > 101
		GROUP BY type, status`,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()

	var res []*metric.IssueCountMetric

	for rows.Next() {
		var metric metric.IssueCountMetric
		if err := rows.Scan(&metric.Type, &metric.Status, &metric.Count); err != nil {
			return nil, FormatError(err)
		}
		res = append(res, &metric)
	}

	return res, nil
}

// CreateIssueValidateOnly creates an issue for validation purpose
// Do NOT write to the database
func (s *Store) CreateIssueValidateOnly(ctx context.Context, pipeline *api.Pipeline, create *api.IssueCreate, creatorID int) (*api.Issue, error) {
	issue := &api.Issue{
		CreatorID:   creatorID,
		CreatedTs:   time.Now().Unix(),
		UpdaterID:   creatorID,
		UpdatedTs:   time.Now().Unix(),
		ProjectID:   create.ProjectID,
		Name:        create.Name,
		Status:      api.IssueOpen,
		Type:        create.Type,
		Description: create.Description,
		AssigneeID:  create.AssigneeID,
		PipelineID:  pipeline.ID,
		Pipeline:    pipeline,
	}

	if err := s.composeIssueValidateOnly(ctx, issue); err != nil {
		return nil, err
	}

	return issue, nil
}

// CreatePipelineValidateOnly creates a pipeline for validation purpose
// Do NOT write to the database
func (s *Store) CreatePipelineValidateOnly(ctx context.Context, create *api.PipelineCreate, creatorID int) (*api.Pipeline, error) {
	// We cannot emit ID or use default zero by following https://google.aip.dev/163, otherwise
	// jsonapi resource relationships will collide different resources into the same bucket.
	id := 0
	ts := time.Now().Unix()
	pipeline := &api.Pipeline{
		ID:        id,
		Name:      create.Name,
		Status:    api.PipelineOpen,
		CreatorID: creatorID,
		CreatedTs: ts,
		UpdaterID: creatorID,
		UpdatedTs: ts,
	}
	for _, sc := range create.StageList {
		id++
		stage := &api.Stage{
			ID:            id,
			Name:          sc.Name,
			CreatorID:     creatorID,
			CreatedTs:     ts,
			UpdaterID:     creatorID,
			UpdatedTs:     ts,
			PipelineID:    sc.PipelineID,
			EnvironmentID: sc.EnvironmentID,
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
			taskRaw := &taskRaw{
				ID:                id,
				Name:              tc.Name,
				Status:            tc.Status,
				CreatorID:         creatorID,
				CreatedTs:         ts,
				UpdaterID:         creatorID,
				UpdatedTs:         ts,
				Type:              tc.Type,
				Payload:           tc.Payload,
				EarliestAllowedTs: tc.EarliestAllowedTs,
				PipelineID:        pipeline.ID,
				StageID:           stage.ID,
				InstanceID:        tc.InstanceID,
				DatabaseID:        tc.DatabaseID,
			}
			task, err := s.composeTask(ctx, taskRaw)
			// We need to compose task.BlockedBy here because task and taskDAG are not inserted yet.
			task.BlockedBy = blockedBy
			if err != nil {
				return nil, err
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

	// Note: issue.Pipeline must be generated by CreatePipelineValidateOnly().
	if err := s.composePipelineValidateOnly(ctx, issue.Pipeline); err != nil {
		return err
	}

	return nil
}

// Note: MUST keep in sync with composeIssueValidateOnly
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

// createIssueRaw creates a new issue.
func (s *Store) createIssueRaw(ctx context.Context, create *api.IssueCreate) (*issueRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	issue, err := s.createIssueImpl(ctx, tx.PTx, create)
	if err != nil {
		return nil, err
	}

	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	if err := s.cache.UpsertCache(api.IssueCache, issue.ID, issue); err != nil {
		return nil, err
	}

	return issue, nil
}

// findIssueRaw retrieves a list of issues based on find.
func (s *Store) findIssueRaw(ctx context.Context, find *api.IssueFind) ([]*issueRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	list, err := s.findIssueImpl(ctx, tx.PTx, find)
	if err != nil {
		return nil, err
	}

	if err == nil {
		for _, issue := range list {
			if err := s.cache.UpsertCache(api.IssueCache, issue.ID, issue); err != nil {
				return nil, err
			}
		}
	}

	return list, nil
}

// getIssueRaw retrieves a single issue based on find.
// Returns ECONFLICT if finding more than 1 matching records.
func (s *Store) getIssueRaw(ctx context.Context, find *api.IssueFind) (*issueRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	list, err := s.findIssueImpl(ctx, tx.PTx, find)
	if err != nil {
		return nil, err
	}

	if len(list) == 0 {
		return nil, nil
	} else if len(list) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: fmt.Errorf("found %d issues with filter %+v, expect 1", len(list), find)}
	}
	if err := s.cache.UpsertCache(api.IssueCache, list[0].ID, list[0]); err != nil {
		return nil, err
	}
	return list[0], nil
}

// patchIssueRaw updates an existing issue by ID.
// Returns ENOTFOUND if issue does not exist.
func (s *Store) patchIssueRaw(ctx context.Context, patch *api.IssuePatch) (*issueRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	issue, err := s.patchIssueImpl(ctx, tx.PTx, patch)
	if err != nil {
		return nil, FormatError(err)
	}

	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	if err := s.cache.UpsertCache(api.IssueCache, issue.ID, issue); err != nil {
		return nil, err
	}

	return issue, nil
}

// createIssueImpl creates a new issue.
func (s *Store) createIssueImpl(ctx context.Context, tx *sql.Tx, create *api.IssueCreate) (*issueRaw, error) {
	if create.Payload == "" {
		create.Payload = "{}"
	}
	row, err := tx.QueryContext(ctx, `
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
			payload
		)
		VALUES ($1, $2, $3, $4, $5, 'OPEN', $6, $7, $8, $9)
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, project_id, pipeline_id, name, status, type, description, assignee_id, payload
	`,
		create.CreatorID,
		create.CreatorID,
		create.ProjectID,
		create.PipelineID,
		create.Name,
		create.Type,
		create.Description,
		create.AssigneeID,
		create.Payload,
	)

	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	row.Next()
	var issueRaw issueRaw
	if err := row.Scan(
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
		&issueRaw.Payload,
	); err != nil {
		return nil, FormatError(err)
	}

	return &issueRaw, nil
}

func (s *Store) findIssueImpl(ctx context.Context, tx *sql.Tx, find *api.IssueFind) ([]*issueRaw, error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
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
		where = append(where, fmt.Sprintf("(creator_id = $%d OR assignee_id = $%d OR EXISTS (SELECT 1 FROM issue_subscriber WHERE issue_id = issue.id AND subscriber_id = $%d))", len(args)+1, len(args)+2, len(args)+3))
		args = append(args, *v)
		args = append(args, *v)
		args = append(args, *v)
	}
	if v := find.StatusList; v != nil {
		list := []string{}
		for _, status := range *v {
			list = append(list, fmt.Sprintf("$%d", len(args)+1))
			args = append(args, status)
		}
		where = append(where, fmt.Sprintf("status in (%s)", strings.Join(list, ",")))
	}

	var query = `
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
			payload
		FROM issue
		WHERE ` + strings.Join(where, " AND ")
	if v := find.Limit; v != nil {
		query += fmt.Sprintf(" ORDER BY updated_ts DESC LIMIT %d", *v)
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

// patchIssueImpl updates a issue by ID. Returns the new state of the issue after update.
func (s *Store) patchIssueImpl(ctx context.Context, tx *sql.Tx, patch *api.IssuePatch) (*issueRaw, error) {
	// Build UPDATE clause.
	set, args := []string{"updater_id = $1"}, []interface{}{patch.UpdaterID}
	if v := patch.Name; v != nil {
		set, args = append(set, fmt.Sprintf("name = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Status; v != nil {
		set, args = append(set, fmt.Sprintf("status = $%d", len(args)+1)), append(args, api.IssueStatus(*v))
	}
	if v := patch.Description; v != nil {
		set, args = append(set, fmt.Sprintf("description = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.AssigneeID; v != nil {
		set, args = append(set, fmt.Sprintf("assignee_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Payload; v != nil {
		payload, err := json.Marshal(*patch.Payload)
		if err != nil {
			return nil, FormatError(err)
		}
		set, args = append(set, fmt.Sprintf("payload = $%d", len(args)+1)), append(args, payload)
	}

	args = append(args, patch.ID)

	// Execute update query with RETURNING.
	row, err := tx.QueryContext(ctx, fmt.Sprintf(`
		UPDATE issue
		SET `+strings.Join(set, ", ")+`
		WHERE id = $%d
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, project_id, pipeline_id, name, status, type, description, assignee_id, payload
	`, len(args)),
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	if row.Next() {
		var issueRaw issueRaw
		if err := row.Scan(
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
			&issueRaw.Payload,
		); err != nil {
			return nil, FormatError(err)
		}

		return &issueRaw, nil
	}

	return nil, &common.Error{Code: common.NotFound, Err: fmt.Errorf("unable to find issue ID to update: %d", patch.ID)}
}
