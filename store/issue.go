package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"go.uber.org/zap"
)

var (
	_ api.IssueService = (*IssueService)(nil)
)

// IssueService represents a service for managing issue.
type IssueService struct {
	l  *zap.Logger
	db *DB

	cache api.CacheService
}

// NewIssueService returns a new instance of IssueService.
func NewIssueService(logger *zap.Logger, db *DB, cache api.CacheService) *IssueService {
	return &IssueService{l: logger, db: db, cache: cache}
}

// CreateIssue creates a new issue.
func (s *IssueService) CreateIssue(ctx context.Context, create *api.IssueCreate) (*api.Issue, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Tx.Rollback()
	defer tx.PTx.Rollback()

	issue, err := s.pgCreateIssue(ctx, tx.PTx, create)
	if err != nil {
		return nil, err
	}
	if _, err := s.createIssue(ctx, tx.Tx, create); err != nil {
		return nil, err
	}

	if err := tx.Tx.Commit(); err != nil {
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

// FindIssueList retrieves a list of issues based on find.
func (s *IssueService) FindIssueList(ctx context.Context, find *api.IssueFind) ([]*api.Issue, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Tx.Rollback()
	defer tx.PTx.Rollback()

	list, err := s.findIssueList(ctx, tx.PTx, find)
	if err != nil {
		return []*api.Issue{}, err
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

// FindIssue retrieves a single issue based on find.
// Returns ECONFLICT if finding more than 1 matching records.
func (s *IssueService) FindIssue(ctx context.Context, find *api.IssueFind) (*api.Issue, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Tx.Rollback()
	defer tx.PTx.Rollback()

	list, err := s.findIssueList(ctx, tx.PTx, find)
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

// PatchIssue updates an existing issue by ID.
// Returns ENOTFOUND if issue does not exist.
func (s *IssueService) PatchIssue(ctx context.Context, patch *api.IssuePatch) (*api.Issue, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Tx.Rollback()
	defer tx.PTx.Rollback()

	issue, err := s.pgPatchIssue(ctx, tx.PTx, patch)
	if err != nil {
		return nil, FormatError(err)
	}
	if _, err := s.patchIssue(ctx, tx.Tx, patch); err != nil {
		return nil, FormatError(err)
	}

	if err := tx.Tx.Commit(); err != nil {
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

// createIssue creates a new issue.
func (s *IssueService) createIssue(ctx context.Context, tx *sql.Tx, create *api.IssueCreate) (*api.Issue, error) {
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
		VALUES (?, ?, ?, ?, ?, 'OPEN', ?, ?, ?, ?)
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
	var issue api.Issue
	if err := row.Scan(
		&issue.ID,
		&issue.CreatorID,
		&issue.CreatedTs,
		&issue.UpdaterID,
		&issue.UpdatedTs,
		&issue.ProjectID,
		&issue.PipelineID,
		&issue.Name,
		&issue.Status,
		&issue.Type,
		&issue.Description,
		&issue.AssigneeID,
		&issue.Payload,
	); err != nil {
		return nil, FormatError(err)
	}

	return &issue, nil
}

// pgCreateIssue creates a new issue.
func (s *IssueService) pgCreateIssue(ctx context.Context, tx *sql.Tx, create *api.IssueCreate) (*api.Issue, error) {
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
	var issue api.Issue
	if err := row.Scan(
		&issue.ID,
		&issue.CreatorID,
		&issue.CreatedTs,
		&issue.UpdaterID,
		&issue.UpdatedTs,
		&issue.ProjectID,
		&issue.PipelineID,
		&issue.Name,
		&issue.Status,
		&issue.Type,
		&issue.Description,
		&issue.AssigneeID,
		&issue.Payload,
	); err != nil {
		return nil, FormatError(err)
	}

	return &issue, nil
}

func (s *IssueService) findIssueList(ctx context.Context, tx *sql.Tx, find *api.IssueFind) (_ []*api.Issue, err error) {
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

	// Iterate over result set and deserialize rows into list.
	list := make([]*api.Issue, 0)
	for rows.Next() {
		var issue api.Issue
		if err := rows.Scan(
			&issue.ID,
			&issue.CreatorID,
			&issue.CreatedTs,
			&issue.UpdaterID,
			&issue.UpdatedTs,
			&issue.ProjectID,
			&issue.PipelineID,
			&issue.Name,
			&issue.Status,
			&issue.Type,
			&issue.Description,
			&issue.AssigneeID,
			&issue.Payload,
		); err != nil {
			return nil, FormatError(err)
		}

		list = append(list, &issue)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return list, nil
}

// patchIssue updates a issue by ID. Returns the new state of the issue after update.
func (s *IssueService) patchIssue(ctx context.Context, tx *sql.Tx, patch *api.IssuePatch) (*api.Issue, error) {
	// Build UPDATE clause.
	set, args := []string{"updater_id = ?"}, []interface{}{patch.UpdaterID}
	if v := patch.Name; v != nil {
		set, args = append(set, "name = ?"), append(args, *v)
	}
	if v := patch.Status; v != nil {
		set, args = append(set, "status = ?"), append(args, api.IssueStatus(*v))
	}
	if v := patch.Description; v != nil {
		set, args = append(set, "description = ?"), append(args, *v)
	}
	if v := patch.AssigneeID; v != nil {
		set, args = append(set, "assignee_id = ?"), append(args, *v)
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
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, project_id, pipeline_id, name, status, type, description, assignee_id, payload
	`,
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	if row.Next() {
		var issue api.Issue
		if err := row.Scan(
			&issue.ID,
			&issue.CreatorID,
			&issue.CreatedTs,
			&issue.UpdaterID,
			&issue.UpdatedTs,
			&issue.ProjectID,
			&issue.PipelineID,
			&issue.Name,
			&issue.Status,
			&issue.Type,
			&issue.Description,
			&issue.AssigneeID,
			&issue.Payload,
		); err != nil {
			return nil, FormatError(err)
		}

		return &issue, nil
	}

	return nil, &common.Error{Code: common.NotFound, Err: fmt.Errorf("unable to find issue ID to update: %d", patch.ID)}
}

// pgPatchIssue updates a issue by ID. Returns the new state of the issue after update.
func (s *IssueService) pgPatchIssue(ctx context.Context, tx *sql.Tx, patch *api.IssuePatch) (*api.Issue, error) {
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
		var issue api.Issue
		if err := row.Scan(
			&issue.ID,
			&issue.CreatorID,
			&issue.CreatedTs,
			&issue.UpdaterID,
			&issue.UpdatedTs,
			&issue.ProjectID,
			&issue.PipelineID,
			&issue.Name,
			&issue.Status,
			&issue.Type,
			&issue.Description,
			&issue.AssigneeID,
			&issue.Payload,
		); err != nil {
			return nil, FormatError(err)
		}

		return &issue, nil
	}

	return nil, &common.Error{Code: common.NotFound, Err: fmt.Errorf("unable to find issue ID to update: %d", patch.ID)}
}
