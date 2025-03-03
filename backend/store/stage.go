package store

import (
	"context"
	"database/sql"

	"github.com/pkg/errors"
)

// StageMessage is the message for stage.
type StageMessage struct {
	Name        string
	Environment string
	PipelineID  int
	TaskList    []*TaskMessage

	// empty for legacy stages
	DeploymentID string

	// Output only.
	ID     int
	Active bool
}

func (*Store) createStages(ctx context.Context, tx *Tx, stagesCreate []*StageMessage, pipelineUID int) ([]*StageMessage, error) {
	if len(stagesCreate) == 0 {
		return nil, nil
	}
	var environments []string
	var names []string
	var deploymentIDs []string
	for _, create := range stagesCreate {
		environments = append(environments, create.Environment)
		names = append(names, create.Name)
		deploymentIDs = append(deploymentIDs, create.DeploymentID)
	}

	query := `
		INSERT INTO stage (
			pipeline_id,
			environment,
			name,
			deployment_id
		) SELECT
			$1,
			unnest(CAST($2 AS TEXT[])) AS environment,
			unnest(CAST($3 AS TEXT[])),
			unnest(CAST($4 AS TEXT[])) AS deployment_id
		RETURNING id
    `
	rows, err := tx.QueryContext(ctx, query, pipelineUID, environments, names, deploymentIDs)
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

func (*Store) listStages(ctx context.Context, tx *Tx, pipelineUID int) ([]*StageMessage, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT
			stage.id,
			stage.pipeline_id,
			stage.environment,
			stage.deployment_id,
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
			&stage.DeploymentID,
			&stage.Name,
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
