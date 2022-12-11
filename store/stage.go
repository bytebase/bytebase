package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
)

// stageRaw is the store model for an Stage.
// Fields have exactly the same meanings as Stage.
type stageRaw struct {
	ID int

	// Standard fields
	CreatorID int
	CreatedTs int64
	UpdaterID int
	UpdatedTs int64

	// Related fields
	PipelineID    int
	EnvironmentID int

	// Domain specific fields
	Name string
}

// toStage creates an instance of Stage based on the stageRaw.
// This is intended to be called when we need to compose an Stage relationship.
func (raw *stageRaw) toStage() *api.Stage {
	return &api.Stage{
		ID: raw.ID,

		// Standard fields
		CreatorID: raw.CreatorID,
		CreatedTs: raw.CreatedTs,
		UpdaterID: raw.UpdaterID,
		UpdatedTs: raw.UpdatedTs,

		// Related fields
		PipelineID:    raw.PipelineID,
		EnvironmentID: raw.EnvironmentID,

		// Domain specific fields
		Name: raw.Name,
	}
}

// CreateStage creates an instance of Stage.
func (s *Store) CreateStage(ctx context.Context, create *api.StageCreate) (*api.Stage, error) {
	stageRaw, err := s.createStageRaw(ctx, create)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create Stage with StageCreate[%+v]", create)
	}
	stage, err := s.composeStage(ctx, stageRaw)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compose Stage with stageRaw[%+v]", stageRaw)
	}
	return stage, nil
}

// FindStage finds a list of Stage instances.
func (s *Store) FindStage(ctx context.Context, find *api.StageFind) ([]*api.Stage, error) {
	stageRawList, err := s.findStageRaw(ctx, find)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find Stage list with StageFind[%+v]", find)
	}
	var stageList []*api.Stage
	for _, raw := range stageRawList {
		stage, err := s.composeStage(ctx, raw)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to compose Stage with stageRaw[%+v]", raw)
		}
		stageList = append(stageList, stage)
	}
	return stageList, nil
}

//
// private functions
//

// Note: MUST keep in sync with composeStageValidateOnly.
func (s *Store) composeStage(ctx context.Context, raw *stageRaw) (*api.Stage, error) {
	stage := raw.toStage()

	creator, err := s.GetPrincipalByID(ctx, stage.CreatorID)
	if err != nil {
		return nil, err
	}
	stage.Creator = creator

	updater, err := s.GetPrincipalByID(ctx, stage.UpdaterID)
	if err != nil {
		return nil, err
	}
	stage.Updater = updater

	env, err := s.GetEnvironmentByID(ctx, stage.EnvironmentID)
	if err != nil {
		return nil, err
	}
	stage.Environment = env

	taskFind := &api.TaskFind{
		PipelineID: &stage.PipelineID,
		StageID:    &stage.ID,
	}
	taskList, err := s.FindTask(ctx, taskFind, true)
	if err != nil {
		return nil, err
	}
	stage.TaskList = taskList

	return stage, nil
}

// createStageRaw creates a new stage.
func (s *Store) createStageRaw(ctx context.Context, create *api.StageCreate) (*stageRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	stage, err := s.createStageImpl(ctx, tx, create)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return stage, nil
}

// findStageRaw retrieves a list of stages based on find.
func (s *Store) findStageRaw(ctx context.Context, find *api.StageFind) ([]*stageRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	stageRawList, err := s.findStageImpl(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	return stageRawList, nil
}

// createStageImpl creates a new stage.
func (*Store) createStageImpl(ctx context.Context, tx *Tx, create *api.StageCreate) (*stageRaw, error) {
	query := `
		INSERT INTO stage (
			creator_id,
			updater_id,
			pipeline_id,
			environment_id,
			name
		)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, pipeline_id, environment_id, name` + `
	`
	var stageRaw stageRaw
	if err := tx.QueryRowContext(ctx, query,
		create.CreatorID,
		create.CreatorID,
		create.PipelineID,
		create.EnvironmentID,
		create.Name,
	).Scan(
		&stageRaw.ID,
		&stageRaw.CreatorID,
		&stageRaw.CreatedTs,
		&stageRaw.UpdaterID,
		&stageRaw.UpdatedTs,
		&stageRaw.PipelineID,
		&stageRaw.EnvironmentID,
		&stageRaw.Name,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return nil, FormatError(err)
	}
	return &stageRaw, nil
}

func (*Store) findStageImpl(ctx context.Context, tx *Tx, find *api.StageFind) ([]*stageRaw, error) {
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
	var stageRawList []*stageRaw
	for rows.Next() {
		var stageRaw stageRaw
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
