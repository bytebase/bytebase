package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"go.uber.org/zap"
)

var (
	_ api.PipelineService = (*PipelineService)(nil)
)

// PipelineService represents a service for managing pipeline.
type PipelineService struct {
	l  *zap.Logger
	db *DB

	cache api.CacheService
}

// NewPipelineService returns a new instance of PipelineService.
func NewPipelineService(logger *zap.Logger, db *DB, cache api.CacheService) *PipelineService {
	return &PipelineService{l: logger, db: db, cache: cache}
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

	if err := s.cache.UpsertCache(api.PipelineCache, pipeline.ID, pipeline); err != nil {
		return nil, err
	}

	return pipeline, nil
}

// FindPipelineList retrieves a list of pipelines based on find.
func (s *PipelineService) FindPipelineList(ctx context.Context, find *api.PipelineFind) ([]*api.Pipeline, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := s.findPipelineList(ctx, tx, find)
	if err != nil {
		return []*api.Pipeline{}, err
	}

	if err == nil {
		for _, pipeline := range list {
			if err := s.cache.UpsertCache(api.PipelineCache, pipeline.ID, pipeline); err != nil {
				return nil, err
			}
		}
	}

	return list, nil
}

// FindPipeline retrieves a single pipeline based on find.
// Returns ECONFLICT if finding more than 1 matching records.
func (s *PipelineService) FindPipeline(ctx context.Context, find *api.PipelineFind) (*api.Pipeline, error) {
	if find.ID != nil {
		pipeline := &api.Pipeline{}
		has, err := s.cache.FindCache(api.PipelineCache, *find.ID, pipeline)
		if err != nil {
			return nil, err
		}
		if has {
			return pipeline, nil
		}
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := s.findPipelineList(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	if len(list) == 0 {
		return nil, nil
	} else if len(list) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: fmt.Errorf("found %d pipelines with filter %+v, expect 1", len(list), find)}
	}
	if err := s.cache.UpsertCache(api.PipelineCache, list[0].ID, list[0]); err != nil {
		return nil, err
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

	if err := s.cache.UpsertCache(api.PipelineCache, pipeline.ID, pipeline); err != nil {
		return nil, err
	}

	return pipeline, nil
}

// createPipeline creates a new pipeline.
func (s *PipelineService) createPipeline(ctx context.Context, tx *Tx, create *api.PipelineCreate) (*api.Pipeline, error) {
	row, err := tx.QueryContext(ctx, `
		INSERT INTO pipeline (
			creator_id,
			updater_id,
			name,
			`+"`status`"+`
		)
		VALUES (?, ?, ?, 'OPEN')
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, name, `+"`status`"+`
	`,
		create.CreatorID,
		create.CreatorID,
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
		&pipeline.CreatorID,
		&pipeline.CreatedTs,
		&pipeline.UpdaterID,
		&pipeline.UpdatedTs,
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
	if v := find.Status; v != nil {
		where, args = append(where, "`status` = ?"), append(args, *v)
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT
		    id,
		    creator_id,
		    created_ts,
		    updater_id,
		    updated_ts,
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
			&pipeline.CreatorID,
			&pipeline.CreatedTs,
			&pipeline.UpdaterID,
			&pipeline.UpdatedTs,
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
	set, args := []string{"updater_id = ?"}, []interface{}{patch.UpdaterID}
	if v := patch.Status; v != nil {
		set, args = append(set, "status = ?"), append(args, api.PipelineStatus(*v))
	}

	args = append(args, patch.ID)

	// Execute update query with RETURNING.
	row, err := tx.QueryContext(ctx, `
		UPDATE pipeline
		SET `+strings.Join(set, ", ")+`
		WHERE id = ?
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, name, `+"`status`"+`
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
			&pipeline.CreatorID,
			&pipeline.CreatedTs,
			&pipeline.UpdaterID,
			&pipeline.UpdatedTs,
			&pipeline.Name,
			&pipeline.Status,
		); err != nil {
			return nil, FormatError(err)
		}
		return &pipeline, nil
	}

	return nil, &common.Error{Code: common.NotFound, Err: fmt.Errorf("pipeline ID not found: %d", patch.ID)}
}
