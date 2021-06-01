package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase"
	"github.com/bytebase/bytebase/api"
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
func (s *StageService) CreateStage(ctx context.Context, create *api.StageCreate) (*api.Stage, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	stage, err := s.createStage(ctx, tx, create)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return stage, nil
}

// FindStageList retrieves a list of stages based on find.
func (s *StageService) FindStageList(ctx context.Context, find *api.StageFind) ([]*api.Stage, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := s.findStageList(ctx, tx, find)
	if err != nil {
		return []*api.Stage{}, err
	}

	return list, nil
}

// FindStage retrieves a single stage based on find.
// Returns ENOTFOUND if no matching record.
// Returns the first matching one and prints a warning if finding more than 1 matching records.
func (s *StageService) FindStage(ctx context.Context, find *api.StageFind) (*api.Stage, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := s.findStageList(ctx, tx, find)
	if err != nil {
		return nil, err
	} else if len(list) == 0 {
		return nil, &bytebase.Error{Code: bytebase.ENOTFOUND, Message: fmt.Sprintf("stage not found: %v", find)}
	} else if len(list) > 1 {
		s.l.Warn(fmt.Sprintf("found mulitple stages: %d, expect 1", len(list)))
	}
	return list[0], nil
}

// createStage creates a new stage.
func (s *StageService) createStage(ctx context.Context, tx *Tx, create *api.StageCreate) (*api.Stage, error) {
	row, err := tx.QueryContext(ctx, `
		INSERT INTO stage (
			creator_id,
			updater_id,
			pipeline_id,
			environment_id,
			name,
			`+"`type`"+`	
		)
		VALUES (?, ?, ?, ?, ?, ?)
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, pipeline_id, environment_id, name, `+"`type`"+`
	`,
		create.CreatorId,
		create.CreatorId,
		create.PipelineId,
		create.EnvironmentId,
		create.Name,
		create.Type,
	)

	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	row.Next()
	var stage api.Stage
	if err := row.Scan(
		&stage.ID,
		&stage.CreatorId,
		&stage.CreatedTs,
		&stage.UpdaterId,
		&stage.UpdatedTs,
		&stage.PipelineId,
		&stage.EnvironmentId,
		&stage.Name,
		&stage.Type,
	); err != nil {
		return nil, FormatError(err)
	}

	return &stage, nil
}

func (s *StageService) findStageList(ctx context.Context, tx *Tx, find *api.StageFind) (_ []*api.Stage, err error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.ID; v != nil {
		where, args = append(where, "id = ?"), append(args, *v)
	}
	if v := find.PipelineId; v != nil {
		where, args = append(where, "pipeline_id = ?"), append(args, *v)
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
		    name,
		    `+"`type`"+`
		FROM stage
		WHERE `+strings.Join(where, " AND "),
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()

	// Iterate over result set and deserialize rows into list.
	list := make([]*api.Stage, 0)
	for rows.Next() {
		var stage api.Stage
		if err := rows.Scan(
			&stage.ID,
			&stage.CreatorId,
			&stage.CreatedTs,
			&stage.UpdaterId,
			&stage.UpdatedTs,
			&stage.PipelineId,
			&stage.EnvironmentId,
			&stage.Name,
			&stage.Type,
		); err != nil {
			return nil, FormatError(err)
		}

		list = append(list, &stage)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return list, nil
}
