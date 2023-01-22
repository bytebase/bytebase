package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	api "github.com/bytebase/bytebase/backend/legacyapi"
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

// CreateStage creates an list of Stages.
func (s *Store) CreateStage(ctx context.Context, creates []*api.StageCreate) ([]*api.Stage, error) {
	stageRaws, err := s.createStageRaw(ctx, creates)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create Stage")
	}
	var stages []*api.Stage
	for _, stageRaw := range stageRaws {
		stage, err := s.composeStage(ctx, stageRaw)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to compose Stage with stageRaw[%+v]", stageRaw)
		}
		stages = append(stages, stage)
	}
	return stages, nil
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

// createStageRaw creates a list of stages.
func (s *Store) createStageRaw(ctx context.Context, creates []*api.StageCreate) ([]*stageRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	stages, err := s.createStageImpl(ctx, tx, creates)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return stages, nil
}

// findStageRaw retrieves a list of stages based on find.
func (s *Store) findStageRaw(ctx context.Context, find *api.StageFind) ([]*stageRaw, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
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
func (*Store) createStageImpl(ctx context.Context, tx *Tx, creates []*api.StageCreate) ([]*stageRaw, error) {
	var valueStr []string
	var values []interface{}
	for i, create := range creates {
		values = append(values,
			create.CreatorID,
			create.CreatorID,
			create.PipelineID,
			create.EnvironmentID,
			create.Name,
		)
		const count = 5
		valueStr = append(valueStr, fmt.Sprintf("($%d,$%d,$%d,$%d,$%d)", i*count+1, i*count+2, i*count+3, i*count+4, i*count+5))
	}

	query := fmt.Sprintf(`
    WITH inserted AS (
	  	INSERT INTO stage (
	  		creator_id,
	  		updater_id,
	  		pipeline_id,
	  		environment_id,
	  		name
	  	) VALUES %s
	  	RETURNING id, creator_id, created_ts, updater_id, updated_ts, pipeline_id, environment_id, name
    ) SELECT * FROM inserted ORDER BY id ASC
    `, strings.Join(valueStr, ","))
	rows, err := tx.QueryContext(ctx, query, values...)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()

	var stageRaws []*stageRaw
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
		stageRaws = append(stageRaws, &stageRaw)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return stageRaws, nil
}

func (*Store) findStageImpl(ctx context.Context, tx *Tx, find *api.StageFind) ([]*stageRaw, error) {
	// Build WHERE clause.
	where, args := []string{"TRUE"}, []interface{}{}
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
