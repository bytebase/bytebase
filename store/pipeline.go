package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/common/log"
)

// pipelineRaw is the store model for an Pipeline.
// Fields have exactly the same meanings as Pipeline.
type pipelineRaw struct {
	ID int

	// Standard fields
	CreatorID int
	CreatedTs int64
	UpdaterID int
	UpdatedTs int64

	// Domain specific fields
	Name   string
	Status api.PipelineStatus
}

// toPipeline creates an instance of Pipeline based on the pipelineRaw.
// This is intended to be called when we need to compose an Pipeline relationship.
func (raw *pipelineRaw) toPipeline() *api.Pipeline {
	return &api.Pipeline{
		ID: raw.ID,

		// Standard fields
		CreatorID: raw.CreatorID,
		CreatedTs: raw.CreatedTs,
		UpdaterID: raw.UpdaterID,
		UpdatedTs: raw.UpdatedTs,

		// Domain specific fields
		Name:   raw.Name,
		Status: raw.Status,
	}
}

// CreatePipeline creates an instance of Pipeline.
func (s *Store) CreatePipeline(ctx context.Context, create *api.PipelineCreate) (*api.Pipeline, error) {
	pipelineRaw, err := s.createPipelineRaw(ctx, create)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create Pipeline with PipelineCreate[%+v]", create)
	}
	pipeline, err := s.composePipeline(ctx, pipelineRaw)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compose Pipeline with pipelineRaw[%+v]", pipelineRaw)
	}
	return pipeline, nil
}

// GetPipelineByID gets an instance of Pipeline.
func (s *Store) GetPipelineByID(ctx context.Context, id int) (*api.Pipeline, error) {
	find := &api.PipelineFind{ID: &id}
	pipelineRaw, err := s.getPipelineRaw(ctx, find)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get Pipeline with ID %d", id)
	}
	if pipelineRaw == nil {
		return nil, nil
	}
	pipeline, err := s.composePipeline(ctx, pipelineRaw)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compose Pipeline with pipelineRaw[%+v]", pipelineRaw)
	}
	return pipeline, nil
}

// FindPipeline finds a list of Pipeline instances.
func (s *Store) FindPipeline(ctx context.Context, find *api.PipelineFind, returnOnErr bool) ([]*api.Pipeline, error) {
	pipelineRawList, err := s.findPipelineRaw(ctx, find)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find Pipeline list with PipelineFind[%+v]", find)
	}
	var pipelineList []*api.Pipeline
	for _, raw := range pipelineRawList {
		pipeline, err := s.composePipeline(ctx, raw)
		if err != nil {
			if returnOnErr {
				return nil, errors.Wrapf(err, "failed to compose Pipeline with pipelineRaw[%+v]", raw)
			}
			log.Error("failed to compose pipeline",
				zap.Any("pipelineRaw", raw),
				zap.Error(err),
			)
			continue
		}
		pipelineList = append(pipelineList, pipeline)
	}
	return pipelineList, nil
}

// PatchPipeline patches an instance of Pipeline.
func (s *Store) PatchPipeline(ctx context.Context, patch *api.PipelinePatch) (*api.Pipeline, error) {
	pipelineRaw, err := s.patchPipelineRaw(ctx, patch)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to patch Pipeline with PipelinePatch[%+v]", patch)
	}
	pipeline, err := s.composePipeline(ctx, pipelineRaw)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compose Pipeline with pipelineRaw[%+v]", pipelineRaw)
	}
	return pipeline, nil
}

//
// private function
//

// Note: MUST keep in sync with composePipelineValidateOnly.
func (s *Store) composePipeline(ctx context.Context, raw *pipelineRaw) (*api.Pipeline, error) {
	pipeline := raw.toPipeline()

	creator, err := s.GetPrincipalByID(ctx, pipeline.CreatorID)
	if err != nil {
		return nil, err
	}
	pipeline.Creator = creator

	updater, err := s.GetPrincipalByID(ctx, pipeline.UpdaterID)
	if err != nil {
		return nil, err
	}
	pipeline.Updater = updater

	stageList, err := s.FindStage(ctx, &api.StageFind{PipelineID: &pipeline.ID})
	if err != nil {
		return nil, err
	}
	pipeline.StageList = stageList

	return pipeline, nil
}

// createPipelineRaw creates a new pipeline.
func (s *Store) createPipelineRaw(ctx context.Context, create *api.PipelineCreate) (*pipelineRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	pipeline, err := s.createPipelineImpl(ctx, tx, create)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	if err := s.cache.UpsertCache(pipelineCacheNamespace, pipeline.ID, pipeline); err != nil {
		return nil, err
	}

	return pipeline, nil
}

// findPipelineRaw retrieves a list of pipelines based on find.
func (s *Store) findPipelineRaw(ctx context.Context, find *api.PipelineFind) ([]*pipelineRaw, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := s.findPipelineImpl(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	if err == nil {
		for _, pipeline := range list {
			if err := s.cache.UpsertCache(pipelineCacheNamespace, pipeline.ID, pipeline); err != nil {
				return nil, err
			}
		}
	}

	return list, nil
}

// getPipelineRaw retrieves a single pipeline based on find.
// Returns ECONFLICT if finding more than 1 matching records.
func (s *Store) getPipelineRaw(ctx context.Context, find *api.PipelineFind) (*pipelineRaw, error) {
	if find.ID != nil {
		pipelineRaw := &pipelineRaw{}
		has, err := s.cache.FindCache(pipelineCacheNamespace, *find.ID, pipelineRaw)
		if err != nil {
			return nil, err
		}
		if has {
			return pipelineRaw, nil
		}
	}

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	pipelineRawList, err := s.findPipelineImpl(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	if len(pipelineRawList) == 0 {
		return nil, nil
	} else if len(pipelineRawList) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: errors.Errorf("found %d pipelines with filter %+v, expect 1", len(pipelineRawList), find)}
	}
	if err := s.cache.UpsertCache(pipelineCacheNamespace, pipelineRawList[0].ID, pipelineRawList[0]); err != nil {
		return nil, err
	}
	return pipelineRawList[0], nil
}

// patchPipelineRaw updates an existing pipeline by ID.
// Returns ENOTFOUND if pipeline does not exist.
func (s *Store) patchPipelineRaw(ctx context.Context, patch *api.PipelinePatch) (*pipelineRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	pipelineRaw, err := s.patchPipelineImpl(ctx, tx, patch)
	if err != nil {
		return nil, FormatError(err)
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	if err := s.cache.UpsertCache(pipelineCacheNamespace, pipelineRaw.ID, pipelineRaw); err != nil {
		return nil, err
	}

	return pipelineRaw, nil
}

// createPipelineImpl creates a new pipeline.
func (*Store) createPipelineImpl(ctx context.Context, tx *Tx, create *api.PipelineCreate) (*pipelineRaw, error) {
	query := `
		INSERT INTO pipeline (
			creator_id,
			updater_id,
			name,
			status
		)
		VALUES ($1, $2, $3, 'OPEN')
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, name, status
	`
	var pipelineRaw pipelineRaw
	if err := tx.QueryRowContext(ctx, query,
		create.CreatorID,
		create.CreatorID,
		create.Name,
	).Scan(
		&pipelineRaw.ID,
		&pipelineRaw.CreatorID,
		&pipelineRaw.CreatedTs,
		&pipelineRaw.UpdaterID,
		&pipelineRaw.UpdatedTs,
		&pipelineRaw.Name,
		&pipelineRaw.Status,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return nil, FormatError(err)
	}
	return &pipelineRaw, nil
}

func (*Store) findPipelineImpl(ctx context.Context, tx *Tx, find *api.PipelineFind) ([]*pipelineRaw, error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.ID; v != nil {
		where, args = append(where, fmt.Sprintf("id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.Status; v != nil {
		where, args = append(where, fmt.Sprintf("status = $%d", len(args)+1)), append(args, *v)
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT
			id,
			creator_id,
			created_ts,
			updater_id,
			updated_ts,
			name,
			status
		FROM pipeline
		WHERE `+strings.Join(where, " AND "),
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()

	// Iterate over result set and deserialize rows into pipelineRawList.
	var pipelineRawList []*pipelineRaw
	for rows.Next() {
		var pipelineRaw pipelineRaw
		if err := rows.Scan(
			&pipelineRaw.ID,
			&pipelineRaw.CreatorID,
			&pipelineRaw.CreatedTs,
			&pipelineRaw.UpdaterID,
			&pipelineRaw.UpdatedTs,
			&pipelineRaw.Name,
			&pipelineRaw.Status,
		); err != nil {
			return nil, FormatError(err)
		}

		pipelineRawList = append(pipelineRawList, &pipelineRaw)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return pipelineRawList, nil
}

// patchPipelineImpl updates a pipeline by ID. Returns the new state of the pipeline after update.
func (*Store) patchPipelineImpl(ctx context.Context, tx *Tx, patch *api.PipelinePatch) (*pipelineRaw, error) {
	// Build UPDATE clause.
	set, args := []string{"updater_id = $1"}, []interface{}{patch.UpdaterID}
	if v := patch.Status; v != nil {
		set, args = append(set, fmt.Sprintf("status = $%d", len(args)+1)), append(args, api.PipelineStatus(*v))
	}

	args = append(args, patch.ID)

	var pipelineRaw pipelineRaw
	// Execute update query with RETURNING.
	if err := tx.QueryRowContext(ctx, fmt.Sprintf(`
		UPDATE pipeline
		SET `+strings.Join(set, ", ")+`
		WHERE id = $%d
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, name, status
	`, len(args)),
		args...,
	).Scan(
		&pipelineRaw.ID,
		&pipelineRaw.CreatorID,
		&pipelineRaw.CreatedTs,
		&pipelineRaw.UpdaterID,
		&pipelineRaw.UpdatedTs,
		&pipelineRaw.Name,
		&pipelineRaw.Status,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, &common.Error{Code: common.NotFound, Err: errors.Errorf("pipeline ID not found: %d", patch.ID)}
		}
		return nil, FormatError(err)
	}
	return &pipelineRaw, nil
}
