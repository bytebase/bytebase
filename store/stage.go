package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"go.uber.org/zap"
)

var (
	_ api.StageService = (*StageService)(nil)
)

// StageService represents a service for managing stage.
type StageService struct {
	l  *zap.Logger
	db *DB
}

// NewStageService returns a new instance of StageService.
func NewStageService(logger *zap.Logger, db *DB) *StageService {
	return &StageService{l: logger, db: db}
}

// CreateStage creates a new stage.
func (s *StageService) CreateStage(ctx context.Context, create *api.StageCreate) (*api.StageRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	stage, err := s.createStage(ctx, tx.PTx, create)
	if err != nil {
		return nil, err
	}

	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return stage, nil
}

// FindStageList retrieves a list of stages based on find.
func (s *StageService) FindStageList(ctx context.Context, find *api.StageFind) ([]*api.StageRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	stageRawList, err := s.findStageList(ctx, tx.PTx, find)
	if err != nil {
		return nil, err
	}

	return stageRawList, nil
}

// FindStage retrieves a single stage based on find.
// Returns ECONFLICT if finding more than 1 matching records.
func (s *StageService) FindStage(ctx context.Context, find *api.StageFind) (*api.StageRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	stageRawList, err := s.findStageList(ctx, tx.PTx, find)
	if err != nil {
		return nil, err
	}

	if len(stageRawList) == 0 {
		return nil, nil
	} else if len(stageRawList) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: fmt.Errorf("found %d stages with filter %+v, expect 1", len(stageRawList), find)}
	}
	return stageRawList[0], nil
}

// createStage creates a new stage.
func (s *StageService) createStage(ctx context.Context, tx *sql.Tx, create *api.StageCreate) (*api.StageRaw, error) {
	row, err := tx.QueryContext(ctx, `
		INSERT INTO stage (
			creator_id,
			updater_id,
			pipeline_id,
			environment_id,
			name
		)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, pipeline_id, environment_id, name`+`
	`,
		create.CreatorID,
		create.CreatorID,
		create.PipelineID,
		create.EnvironmentID,
		create.Name,
	)

	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	row.Next()
	var stageRaw api.StageRaw
	if err := row.Scan(
		&stageRaw.ID,
		&stageRaw.CreatorID,
		&stageRaw.CreatedTs,
		&stageRaw.UpdaterID,
		&stageRaw.UpdatedTs,
		&stageRaw.PipelineID,
		&stageRaw.EnvironmentID,
		&stageRaw.Name,
	); err != nil {
		return nil, FormatError(err)
	}

	return &stageRaw, nil
}

func (s *StageService) findStageList(ctx context.Context, tx *sql.Tx, find *api.StageFind) ([]*api.StageRaw, error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.ID; v != nil {
		where, args = append(where, fmt.Sprintf("id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.PipelineID; v != nil {
		where, args = append(where, fmt.Sprintf("pipeline_id = $%d", len(args)+1)), append(args, *v)
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT
			id,
			creator_id,
			created_ts,
			updater_id,
			updated_ts,
			pipeline_id,
			environment_id,
			name
		FROM stage
		WHERE `+strings.Join(where, " AND ")+` ORDER BY id ASC`,
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()

	// Iterate over result set and deserialize rows into stageRawList.
	var stageRawList []*api.StageRaw
	for rows.Next() {
		var stageRaw api.StageRaw
		if err := rows.Scan(
			&stageRaw.ID,
			&stageRaw.CreatorID,
			&stageRaw.CreatedTs,
			&stageRaw.UpdaterID,
			&stageRaw.UpdatedTs,
			&stageRaw.PipelineID,
			&stageRaw.EnvironmentID,
			&stageRaw.Name,
		); err != nil {
			return nil, FormatError(err)
		}

		stageRawList = append(stageRawList, &stageRaw)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return stageRawList, nil
}
