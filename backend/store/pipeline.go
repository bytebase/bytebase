package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase"
	"github.com/bytebase/bytebase/api"
)

var (
	_ api.PipelineService = (*PipelineService)(nil)
)

// PipelineService represents a service for managing pipeline.
type PipelineService struct {
	l  *bytebase.Logger
	db *DB
}

// NewPipelineService returns a new instance of PipelineService.
func NewPipelineService(logger *bytebase.Logger, db *DB) *PipelineService {
	return &PipelineService{l: logger, db: db}
}

// CreatePipeline creates a new pipeline.
func (s *PipelineService) CreatePipeline(ctx context.Context, create *api.PipelineCreate) (*api.Pipeline, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	pipeline, err := s.createPipeline(ctx, tx, create)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return pipeline, nil
}

// FindPipeline retrieves a single pipeline based on find.
// Returns ENOTFOUND if no matching record.
// Returns the first matching one and prints a warning if finding more than 1 matching records.
func (s *PipelineService) FindPipeline(ctx context.Context, find *api.PipelineFind) (*api.Pipeline, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := s.findPipelineList(ctx, tx, find)
	if err != nil {
		return nil, err
	} else if len(list) == 0 {
		return nil, &bytebase.Error{Code: bytebase.ENOTFOUND, Message: fmt.Sprintf("pipeline not found: %v", find)}
	} else if len(list) > 1 {
		s.l.Warnf("found mulitple pipelines: %d, expect 1", len(list))
	}
	return list[0], nil
}

// PatchPipeline updates an existing pipeline by ID.
// Returns ENOTFOUND if pipeline does not exist.
func (s *PipelineService) PatchPipeline(ctx context.Context, patch *api.PipelinePatch) (*api.Pipeline, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	pipeline, err := s.patchPipeline(ctx, tx, patch)
	if err != nil {
		return nil, FormatError(err)
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return pipeline, nil
}

// createPipeline creates a new pipeline.
func (s *PipelineService) createPipeline(ctx context.Context, tx *Tx, create *api.PipelineCreate) (*api.Pipeline, error) {
	row, err := tx.QueryContext(ctx, `
		INSERT INTO pipeline (
			creator_id,
			updater_id,
			workspace_id,
			name,
			`+"`status`"+`	
		)
		VALUES (?, ?, ?, 'OPEN')
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, workspace_id, name, `+"`status`"+`
	`,
		create.CreatorId,
		create.CreatorId,
		create.WorkspaceId,
		create.Name,
	)

	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	row.Next()
	var pipeline api.Pipeline
	if err := row.Scan(
		&pipeline.ID,
		&pipeline.CreatorId,
		&pipeline.CreatedTs,
		&pipeline.UpdaterId,
		&pipeline.UpdatedTs,
		&pipeline.WorkspaceId,
		&pipeline.Name,
		&pipeline.Status,
	); err != nil {
		return nil, FormatError(err)
	}

	return &pipeline, nil
}

func (s *PipelineService) findPipelineList(ctx context.Context, tx *Tx, find *api.PipelineFind) (_ []*api.Pipeline, err error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.ID; v != nil {
		where, args = append(where, "id = ?"), append(args, *v)
	}
	if v := find.WorkspaceId; v != nil {
		where, args = append(where, "workspace_id = ?"), append(args, *v)
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT 
		    id,
		    creator_id,
		    created_ts,
		    updater_id,
		    updated_ts,
			workspace_id,
		    name,
		    `+"`status`"+`
		FROM pipeline
		WHERE `+strings.Join(where, " AND "),
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()

	// Iterate over result set and deserialize rows into list.
	list := make([]*api.Pipeline, 0)
	for rows.Next() {
		var pipeline api.Pipeline
		if err := rows.Scan(
			&pipeline.ID,
			&pipeline.CreatorId,
			&pipeline.CreatedTs,
			&pipeline.UpdaterId,
			&pipeline.UpdatedTs,
			&pipeline.WorkspaceId,
			&pipeline.Name,
			&pipeline.Status,
		); err != nil {
			return nil, FormatError(err)
		}

		list = append(list, &pipeline)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return list, nil
}

// patchPipeline updates a pipeline by ID. Returns the new state of the pipeline after update.
func (s *PipelineService) patchPipeline(ctx context.Context, tx *Tx, patch *api.PipelinePatch) (*api.Pipeline, error) {
	// Build UPDATE clause.
	set, args := []string{"updater_id = ?"}, []interface{}{patch.UpdaterId}
	if v := patch.Status; v != nil {
		set, args = append(set, "status = ?"), append(args, api.PipelineStatus(*v))
	}

	args = append(args, patch.ID)

	// Execute update query with RETURNING.
	row, err := tx.QueryContext(ctx, `
		UPDATE pipeline
		SET `+strings.Join(set, ", ")+`
		WHERE id = ?
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, workspace_id, name, `+"`status`"+`
	`,
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	if row.Next() {
		var pipeline api.Pipeline
		if err := row.Scan(
			&pipeline.ID,
			&pipeline.CreatorId,
			&pipeline.CreatedTs,
			&pipeline.UpdaterId,
			&pipeline.UpdatedTs,
			&pipeline.WorkspaceId,
			&pipeline.Name,
			&pipeline.Status,
		); err != nil {
			return nil, FormatError(err)
		}
		return &pipeline, nil
	}

	return nil, &bytebase.Error{Code: bytebase.ENOTFOUND, Message: fmt.Sprintf("pipeline ID not found: %d", patch.ID)}
}
