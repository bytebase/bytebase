package store

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/bytebase/bytebase"
	"github.com/bytebase/bytebase/api"
	"go.uber.org/zap"
)

var (
	_ api.IssueService = (*IssueService)(nil)
)

// IssueService represents a service for managing issue.
type IssueService struct {
	l  *zap.Logger
	db *DB
}

// NewIssueService returns a new instance of IssueService.
func NewIssueService(logger *zap.Logger, db *DB) *IssueService {
	return &IssueService{l: logger, db: db}
}

// CreateIssue creates a new issue.
func (s *IssueService) CreateIssue(ctx context.Context, create *api.IssueCreate) (*api.Issue, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	issue, err := s.createIssue(ctx, tx, create)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return issue, nil
}

// FindIssueList retrieves a list of issues based on find.
func (s *IssueService) FindIssueList(ctx context.Context, find *api.IssueFind) ([]*api.Issue, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := s.findIssueList(ctx, tx, find)
	if err != nil {
		return []*api.Issue{}, err
	}

	return list, nil
}

// FindIssue retrieves a single issue based on find.
// Returns ENOTFOUND if no matching record.
// Returns the first matching one and prints a warning if finding more than 1 matching records.
func (s *IssueService) FindIssue(ctx context.Context, find *api.IssueFind) (*api.Issue, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := s.findIssueList(ctx, tx, find)
	if err != nil {
		return nil, err
	} else if len(list) == 0 {
		return nil, &bytebase.Error{Code: bytebase.ENOTFOUND, Message: fmt.Sprintf("issue not found: %v", find)}
	} else if len(list) > 1 {
		s.l.Warn(fmt.Sprintf("found mulitple issues: %d, expect 1", len(list)))
	}
	return list[0], nil
}

// PatchIssue updates an existing issue by ID.
// Returns ENOTFOUND if issue does not exist.
func (s *IssueService) PatchIssue(ctx context.Context, patch *api.IssuePatch) (*api.Issue, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	issue, err := s.patchIssue(ctx, tx, patch)
	if err != nil {
		return nil, FormatError(err)
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return issue, nil
}

// createIssue creates a new issue.
func (s *IssueService) createIssue(ctx context.Context, tx *Tx, create *api.IssueCreate) (*api.Issue, error) {
	subscriberIdList := []string{}
	for _, item := range create.SubscriberIdList {
		subscriberIdList = append(subscriberIdList, strconv.Itoa(item))
	}
	row, err := tx.QueryContext(ctx, `
		INSERT INTO issue (
			creator_id,
			updater_id,
			project_id,
			pipeline_id,
			name,
			`+"`status`,"+`
			`+"`type`,"+`
			description,
			assignee_id,
			subscriber_id_list,
			payload
		)
		VALUES (?, ?, ?, ?, ?, 'OPEN', ?, ?, ?, ?, ?)
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, project_id, pipeline_id, name, `+"`status`, `type`, description, assignee_id, subscriber_id_list, payload"+`
	`,
		create.CreatorId,
		create.CreatorId,
		create.ProjectId,
		create.PipelineId,
		create.Name,
		create.Type,
		create.Description,
		create.AssigneeId,
		strings.Join(subscriberIdList, ","),
		create.Payload,
	)

	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	row.Next()
	var issue api.Issue
	var idList string
	if err := row.Scan(
		&issue.ID,
		&issue.CreatorId,
		&issue.CreatedTs,
		&issue.UpdaterId,
		&issue.UpdatedTs,
		&issue.ProjectId,
		&issue.PipelineId,
		&issue.Name,
		&issue.Status,
		&issue.Type,
		&issue.Description,
		&issue.AssigneeId,
		&idList,
		&issue.Payload,
	); err != nil {
		return nil, FormatError(err)
	}

	issue.SubscriberIdList = []int{}
	if idList != "" {
		for _, item := range strings.Split(idList, ",") {
			oneId, err := strconv.Atoi(item)
			if err != nil {
				s.l.Error("Issue contains invalid subscriber id",
					zap.Int("issue_id", issue.ID),
					zap.String("subscriber_id", item),
				)
			}
			issue.SubscriberIdList = append(issue.SubscriberIdList, oneId)
		}
	}
	return &issue, nil
}

func (s *IssueService) findIssueList(ctx context.Context, tx *Tx, find *api.IssueFind) (_ []*api.Issue, err error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.ID; v != nil {
		where, args = append(where, "id = ?"), append(args, *v)
	}
	if v := find.PipelineId; v != nil {
		where, args = append(where, "pipeline_id = ?"), append(args, *v)
	}
	if v := find.ProjectId; v != nil {
		where, args = append(where, "project_id = ?"), append(args, *v)
	}
	if v := find.PrincipalId; v != nil {
		where = append(where, "(creator_id = ? OR assignee_id = ? OR subscriber_id_list = ? OR subscriber_id_list LIKE ? OR subscriber_id_list LIKE ? OR subscriber_id_list LIKE ?)")
		args = append(args, *v)
		args = append(args, *v)
		// For subscriber_id_list search, there are 4 possible patterns
		// 1. There is just 1 id.
		args = append(args, *v)
		// 2. id is the first element
		args = append(args, fmt.Sprintf("%d,%%", *v))
		// 3. id is in the middle
		args = append(args, fmt.Sprintf("%%,%d,%%", *v))
		// 4. id is the last element
		args = append(args, fmt.Sprintf("%%,%d", *v))
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT 
		    id,
		    creator_id,
		    created_ts,
		    updater_id,
		    updated_ts,
			project_id,
			pipeline_id,
		    name,
			`+"`status`,"+`
			`+"`type`,"+`
			description,
			assignee_id,
			subscriber_id_list,
			payload
		FROM issue
		WHERE `+strings.Join(where, " AND "),
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()

	// Iterate over result set and deserialize rows into list.
	list := make([]*api.Issue, 0)
	for rows.Next() {
		var issue api.Issue
		var idList string
		if err := rows.Scan(
			&issue.ID,
			&issue.CreatorId,
			&issue.CreatedTs,
			&issue.UpdaterId,
			&issue.UpdatedTs,
			&issue.ProjectId,
			&issue.PipelineId,
			&issue.Name,
			&issue.Status,
			&issue.Type,
			&issue.Description,
			&issue.AssigneeId,
			&idList,
			&issue.Payload,
		); err != nil {
			return nil, FormatError(err)
		}

		issue.SubscriberIdList = []int{}
		if idList != "" {
			for _, item := range strings.Split(idList, ",") {
				oneId, err := strconv.Atoi(item)
				if err != nil {
					s.l.Error("Issue contains invalid subscriber id",
						zap.Int("issue_id", issue.ID),
						zap.String("subscriber_id", item),
					)
				}
				issue.SubscriberIdList = append(issue.SubscriberIdList, oneId)
			}
		}

		list = append(list, &issue)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return list, nil
}

// patchIssue updates a issue by ID. Returns the new state of the issue after update.
func (s *IssueService) patchIssue(ctx context.Context, tx *Tx, patch *api.IssuePatch) (*api.Issue, error) {
	// Build UPDATE clause.
	set, args := []string{"updater_id = ?"}, []interface{}{patch.UpdaterId}
	if v := patch.Name; v != nil {
		set, args = append(set, "name = ?"), append(args, *v)
	}
	if v := patch.Status; v != nil {
		set, args = append(set, "`status` = ?"), append(args, api.IssueStatus(*v))
	}
	if v := patch.Description; v != nil {
		set, args = append(set, "description = ?"), append(args, *v)
	}
	if v := patch.AssigneeId; v != nil {
		set, args = append(set, "assignee_id = ?"), append(args, *v)
	}
	if v := patch.SubscriberIdListStr; v != nil {
		set, args = append(set, "subscriber_id_list = ?"), append(args, *v)
	}
	if v := patch.Payload; v != nil {
		payload, err := json.Marshal(*patch.Payload)
		if err != nil {
			return nil, FormatError(err)
		}
		set, args = append(set, "`payload` = ?"), append(args, payload)
	}

	args = append(args, patch.ID)

	// Execute update query with RETURNING.
	row, err := tx.QueryContext(ctx, `
		UPDATE issue
		SET `+strings.Join(set, ", ")+`
		WHERE id = ?
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, project_id, pipeline_id, name, `+"`status`, `type`, description, assignee_id, subscriber_id_list, payload"+`
	`,
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	if row.Next() {
		var issue api.Issue
		var idList string
		if err := row.Scan(
			&issue.ID,
			&issue.CreatorId,
			&issue.CreatedTs,
			&issue.UpdaterId,
			&issue.UpdatedTs,
			&issue.ProjectId,
			&issue.PipelineId,
			&issue.Name,
			&issue.Status,
			&issue.Type,
			&issue.Description,
			&issue.AssigneeId,
			&idList,
			&issue.Payload,
		); err != nil {
			return nil, FormatError(err)
		}

		issue.SubscriberIdList = []int{}
		if idList != "" {
			for _, item := range strings.Split(idList, ",") {
				oneId, err := strconv.Atoi(item)
				if err != nil {
					s.l.Error("Issue contains invalid subscriber id",
						zap.Int("issue_id", issue.ID),
						zap.String("subscriber_id", item),
					)
				}
				issue.SubscriberIdList = append(issue.SubscriberIdList, oneId)
			}
		}

		return &issue, nil
	}

	return nil, &bytebase.Error{Code: bytebase.ENOTFOUND, Message: fmt.Sprintf("unable to find issue ID to update: %d", patch.ID)}
}
