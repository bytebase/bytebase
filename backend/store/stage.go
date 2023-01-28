package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	api "github.com/bytebase/bytebase/backend/legacyapi"
)

// StageMessage is the message for stage.
type StageMessage struct {
	Name          string
	EnvironmentID int
	PipelineID    int
	// Output only.
	ID int
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

func (s *Store) composeStage(ctx context.Context, stage *StageMessage) (*api.Stage, error) {
	composedStage := &api.Stage{
		ID:            stage.ID,
		PipelineID:    stage.PipelineID,
		EnvironmentID: stage.EnvironmentID,
		Name:          stage.Name,
	}

	env, err := s.GetEnvironmentByID(ctx, stage.EnvironmentID)
	if err != nil {
		return nil, err
	}
	composedStage.Environment = env

	taskFind := &api.TaskFind{
		PipelineID: &stage.PipelineID,
		StageID:    &stage.ID,
	}
	taskList, err := s.FindTask(ctx, taskFind, true)
	if err != nil {
		return nil, err
	}
	composedStage.TaskList = taskList

	return composedStage, nil
}

// CreateStageV2 creates a list of stages.
func (s *Store) CreateStageV2(ctx context.Context, stagesCreate []*StageMessage, creatorID int) ([]*StageMessage, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	var valueStr []string
	var values []interface{}
	for i, create := range stagesCreate {
		values = append(values,
			creatorID,
			creatorID,
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
	  	RETURNING id, pipeline_id, environment_id, name
    ) SELECT * FROM inserted ORDER BY id ASC
    `, strings.Join(valueStr, ","))
	rows, err := tx.QueryContext(ctx, query, values...)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()

	var stages []*StageMessage
	for rows.Next() {
		var stage StageMessage
		if err := rows.Scan(
			&stage.ID,
			&stage.PipelineID,
			&stage.EnvironmentID,
			&stage.Name,
		); err != nil {
			return nil, FormatError(err)
		}
		stages = append(stages, &stage)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return stages, nil
}

// findStageRaw retrieves a list of stages based on find.
func (s *Store) findStageRaw(ctx context.Context, find *api.StageFind) ([]*StageMessage, error) {
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

func (*Store) findStageImpl(ctx context.Context, tx *Tx, find *api.StageFind) ([]*StageMessage, error) {
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
	var stageRawList []*StageMessage
	for rows.Next() {
		var stageRaw StageMessage
		if err := rows.Scan(
			&stageRaw.ID,
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
