package store

import (
	"context"
	"database/sql"

	"github.com/pkg/errors"
)

// StageMessage is the message for stage.
type StageMessage struct {
	Environment string
	PipelineID  int
	TaskList    []*TaskMessage

	// Output only.
	ID     int
	Active bool
}

func (*Store) createStages(ctx context.Context, txn *sql.Tx, stagesCreate []*StageMessage, pipelineUID int) ([]*StageMessage, error) {
	if len(stagesCreate) == 0 {
		return nil, nil
	}
	var environments []string
	for _, create := range stagesCreate {
		environments = append(environments, create.Environment)
	}

	query := `
		INSERT INTO stage (
			pipeline_id,
			environment
		) SELECT
			$1,
			unnest(CAST($2 AS TEXT[])) AS environment
		RETURNING id
    `
	rows, err := txn.QueryContext(ctx, query, pipelineUID, environments)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	i := 0
	for rows.Next() {
		stage := stagesCreate[i]
		if err := rows.Scan(
			&stage.ID,
		); err != nil {
			return nil, err
		}
		i++
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return stagesCreate, nil
}

func (*Store) listStages(ctx context.Context, txn *sql.Tx, pipelineUID int) ([]*StageMessage, error) {
	rows, err := txn.QueryContext(ctx, `
		SELECT
			stage.id,
			stage.pipeline_id,
			stage.environment,
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
		WHERE pipeline_id = $1 ORDER BY id ASC`,
		pipelineUID,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to query rows")
	}
	defer rows.Close()

	var stages []*StageMessage
	for rows.Next() {
		var stage StageMessage
		if err := rows.Scan(
			&stage.ID,
			&stage.PipelineID,
			&stage.Environment,
			&stage.Active,
		); err != nil {
			return nil, errors.Wrapf(err, "failed to scan")
		}

		stages = append(stages, &stage)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrapf(err, "rows err")
	}
	return stages, nil
}

// ListStageV2 finds a list of stages based on find.
func (s *Store) ListStageV2(ctx context.Context, pipelineUID int) ([]*StageMessage, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to begin tx")
	}
	defer tx.Rollback()

	stages, err := s.listStages(ctx, tx, pipelineUID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list stages")
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit tx")
	}
	return stages, nil
}
