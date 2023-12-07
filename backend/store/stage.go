package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// StageMessage is the message for stage.
type StageMessage struct {
	Name          string
	EnvironmentID int
	PipelineID    int
	TaskList      []*TaskMessage

	// Output only.
	ID     int
	Active bool

	// TODO(d): this is used to create the tasks.
	TaskIndexDAGList []TaskIndexDAG
}

// TaskIndexDAG describes task dependency relationship using array index to represent task.
// It is needed because we don't know task id before insertion, so we describe the dependency
// using the in-memory representation, i.e, the array index.
type TaskIndexDAG struct {
	FromIndex int
	ToIndex   int
}

// CreateStageV2 creates a list of stages.
func (s *Store) CreateStageV2(ctx context.Context, stagesCreate []*StageMessage, creatorID int) ([]*StageMessage, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var valueStr []string
	var values []any
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
		return nil, err
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
			return nil, err
		}
		stages = append(stages, &stage)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return stages, nil
}

// ListStageV2 finds a list of stages based on find.
func (s *Store) ListStageV2(ctx context.Context, pipelineUID int) ([]*StageMessage, error) {
	where, args := []string{"TRUE"}, []any{}
	where, args = append(where, fmt.Sprintf("pipeline_id = $%d", len(args)+1)), append(args, pipelineUID)

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, fmt.Sprintf(`
		SELECT
			stage.id,
			stage.pipeline_id,
			stage.environment_id,
			stage.name,
			(
				SELECT EXISTS (
					SELECT 1 FROM task
					LEFT JOIN LATERAL (
						SELECT COALESCE(
							(SELECT
								task_run.status
							FROM task_run
							WHERE task_run.task_id = task.id
							ORDER BY task_run.id DESC
							LIMIT 1
							), 'NOT_STARTED'
						) AS status
					) AS latest_task_run ON TRUE
					WHERE task.pipeline_id = stage.pipeline_id
					AND task.stage_id <= stage.id
					AND NOT (
						COALESCE((task.payload->>'skipped')::BOOLEAN, FALSE) IS TRUE
						OR latest_task_run.status = 'DONE'
					)
				)
			) AS active
		FROM stage
		WHERE %s ORDER BY id ASC`, strings.Join(where, " AND ")),
		args...,
	)
	if err != nil {
		return nil, err
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
			&stage.Active,
		); err != nil {
			return nil, err
		}

		stages = append(stages, &stage)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return stages, nil
}
