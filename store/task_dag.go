package store

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
)

type taskDAGRaw struct {
	ID int

	// Standard fields
	CreatedTs int64
	UpdatedTs int64

	// Domain Specific fields
	FromTaskID int
	ToTaskID   int
	Payload    string
}

func (raw *taskDAGRaw) toTaskDAG() *api.TaskDAG {
	return &api.TaskDAG{
		ID:         raw.ID,
		CreatedTs:  raw.CreatedTs,
		UpdatedTs:  raw.UpdatedTs,
		FromTaskID: raw.FromTaskID,
		ToTaskID:   raw.ToTaskID,
		Payload:    raw.Payload,
	}
}

// CreateTaskDAG creates TaskDAG.
func (s *Store) CreateTaskDAG(ctx context.Context, create *api.TaskDAGCreate) (*api.TaskDAG, error) {
	taskDAGRaw, err := s.createTaskDAGRaw(ctx, create)
	if err != nil {
		return nil, fmt.Errorf("failed to create TaskDAG with TaskDAGCreate[%+v], error[%w]", create, err)
	}
	taskDAG := taskDAGRaw.toTaskDAG()
	return taskDAG, nil
}

// FindTaskDAGList finds TaskDAG list by FromTaskID.
func (s *Store) FindTaskDAGList(ctx context.Context, find *api.TaskDAGFind) ([]*api.TaskDAG, error) {
	// TODO(xz): remove this release guard once the gh-ost feature is ready to release.
	if s.db.mode != common.ReleaseModeDev {
		return nil, nil
	}
	taskDAGRawList, err := s.findTaskDAGRawList(ctx, find)
	if err != nil {
		return nil, fmt.Errorf("failed to find TaskDAG with TaskDAGFind[%+v], error[%w]", find, err)
	}
	var taskDAGList []*api.TaskDAG
	for _, taskDAGRaw := range taskDAGRawList {
		taskDAGList = append(taskDAGList, taskDAGRaw.toTaskDAG())
	}
	return taskDAGList, nil
}

func (s *Store) createTaskDAGRaw(ctx context.Context, create *api.TaskDAGCreate) (*taskDAGRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	taskDAG, err := createTaskDAGImpl(ctx, tx.PTx, create)
	if err != nil {
		return nil, err
	}
	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}
	return taskDAG, nil
}

func (s *Store) findTaskDAGRawList(ctx context.Context, find *api.TaskDAGFind) ([]*taskDAGRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	list, err := findTaskDAGRawListImpl(ctx, tx.PTx, find)
	if err != nil {
		return nil, err
	}
	return list, nil
}

func createTaskDAGImpl(ctx context.Context, tx *sql.Tx, create *api.TaskDAGCreate) (*taskDAGRaw, error) {
	row, err := tx.QueryContext(ctx, `
		INSERT INTO task_dag (
			from_task_id,
			to_task_id,
			payload
		)
		VALUES ($1, $2, $3)
		RETURNING id, created_ts, updated_ts, from_task_id, to_task_id, payload
	`,
		create.FromTaskID,
		create.ToTaskID,
		create.Payload,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	row.Next()
	var taskDAGRaw taskDAGRaw
	if err := row.Scan(
		&taskDAGRaw.ID,
		&taskDAGRaw.CreatedTs,
		&taskDAGRaw.UpdatedTs,
		&taskDAGRaw.FromTaskID,
		&taskDAGRaw.ToTaskID,
		&taskDAGRaw.Payload,
	); err != nil {
		return nil, FormatError(err)
	}

	return &taskDAGRaw, nil
}

func findTaskDAGRawListImpl(ctx context.Context, tx *sql.Tx, find *api.TaskDAGFind) ([]*taskDAGRaw, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT
			id,
			created_ts,
			updated_ts,
			from_task_id,
			to_task_id,
			payload
		FROM task_dag
		WHERE to_task_id = $1
	`, find.ToTaskID)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()

	var taskDAGRawList []*taskDAGRaw
	for rows.Next() {
		var taskDAGRaw taskDAGRaw
		if err := rows.Scan(
			&taskDAGRaw.ID,
			&taskDAGRaw.CreatedTs,
			&taskDAGRaw.UpdatedTs,
			&taskDAGRaw.FromTaskID,
			&taskDAGRaw.ToTaskID,
			&taskDAGRaw.Payload,
		); err != nil {
			return nil, FormatError(err)
		}

		taskDAGRawList = append(taskDAGRawList, &taskDAGRaw)
	}

	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return taskDAGRawList, nil
}
