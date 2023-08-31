package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// TaskDAGMessage is the message for task dags.
type TaskDAGMessage struct {
	FromTaskID int
	ToTaskID   int
}

// TaskDAGFind is the API message to find TaskDAG.
type TaskDAGFind struct {
	StageID    *int
	PipelineID *int
}

// CreateTaskDAGV2 creates a task DAG.
func (s *Store) CreateTaskDAGV2(ctx context.Context, create *TaskDAGMessage) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
		INSERT INTO task_dag (
			from_task_id,
			to_task_id,
			payload
		)
		VALUES ($1, $2, $3)
		RETURNING from_task_id, to_task_id
	`
	var taskDAG TaskDAGMessage
	if err := tx.QueryRowContext(ctx, query,
		create.FromTaskID,
		create.ToTaskID,
		"{}", /* payload */
	).Scan(
		&taskDAG.FromTaskID,
		&taskDAG.ToTaskID,
	); err != nil {
		return err
	}

	return tx.Commit()
}

// ListTaskDags lists task dags.
func (s *Store) ListTaskDags(ctx context.Context, find *TaskDAGFind) ([]*TaskDAGMessage, error) {
	joinClause := ""
	where, args := []string{"TRUE"}, []any{}
	if v := find.StageID; v != nil {
		where, args = append(where, fmt.Sprintf("task.stage_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.PipelineID; v != nil {
		where, args = append(where, fmt.Sprintf("task.pipeline_id = $%d", len(args)+1)), append(args, *v)
	}
	if find.StageID != nil || find.PipelineID != nil {
		// FROM and TO tasks are from the same pipeline and same stage.
		joinClause = "JOIN task ON task.id = task_dag.from_task_id"
	}

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, fmt.Sprintf(`
		SELECT
			task_dag.from_task_id,
			task_dag.to_task_id
		FROM task_dag
		%s
		WHERE %s`, joinClause, strings.Join(where, " AND ")),
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var taskDAGs []*TaskDAGMessage
	for rows.Next() {
		var taskDAG TaskDAGMessage
		if err := rows.Scan(
			&taskDAG.FromTaskID,
			&taskDAG.ToTaskID,
		); err != nil {
			return nil, err
		}
		taskDAGs = append(taskDAGs, &taskDAG)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return taskDAGs, nil
}
