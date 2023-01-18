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
		return nil, errors.Wrapf(err, "failed to create TaskDAG with TaskDAGCreate[%+v]", create)
	}
	taskDAG := taskDAGRaw.toTaskDAG()
	return taskDAG, nil
}

// FindTaskDAGList finds a TaskDAG list by ToTaskID.
func (s *Store) FindTaskDAGList(ctx context.Context, find *api.TaskDAGFind) ([]*api.TaskDAG, error) {
	taskDAGRawList, err := s.findTaskDAGRawList(ctx, find)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find TaskDAG with TaskDAGFind[%+v]", find)
	}
	var taskDAGList []*api.TaskDAG
	for _, taskDAGRaw := range taskDAGRawList {
		taskDAGList = append(taskDAGList, taskDAGRaw.toTaskDAG())
	}
	return taskDAGList, nil
}

// GetTaskDAGByToTaskID gets a single TaskDAG by ToTaskID.
func (s *Store) GetTaskDAGByToTaskID(ctx context.Context, id int) (*api.TaskDAG, error) {
	taskDAGList, err := s.FindTaskDAGList(ctx, &api.TaskDAGFind{ToTaskID: &id})
	if err != nil {
		return nil, err
	}
	if len(taskDAGList) != 1 {
		return nil, &common.Error{Code: common.Conflict, Err: errors.Errorf("found %d tasks with ToTaskID %v, expect 1", len(taskDAGList), id)}
	}
	return taskDAGList[0], nil
}

func (s *Store) createTaskDAGRaw(ctx context.Context, create *api.TaskDAGCreate) (*taskDAGRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	taskDAG, err := createTaskDAGImpl(ctx, tx, create)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}
	return taskDAG, nil
}

func (s *Store) findTaskDAGRawList(ctx context.Context, find *api.TaskDAGFind) ([]*taskDAGRaw, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := findTaskDAGRawListImpl(ctx, tx, find)
	if err != nil {
		return nil, err
	}
	return list, nil
}

func createTaskDAGImpl(ctx context.Context, tx *Tx, create *api.TaskDAGCreate) (*taskDAGRaw, error) {
	query := `
		INSERT INTO task_dag (
			from_task_id,
			to_task_id,
			payload
		)
		VALUES ($1, $2, $3)
		RETURNING id, created_ts, updated_ts, from_task_id, to_task_id, payload
	`
	var taskDAGRaw taskDAGRaw
	if err := tx.QueryRowContext(ctx, query,
		create.FromTaskID,
		create.ToTaskID,
		create.Payload,
	).Scan(
		&taskDAGRaw.ID,
		&taskDAGRaw.CreatedTs,
		&taskDAGRaw.UpdatedTs,
		&taskDAGRaw.FromTaskID,
		&taskDAGRaw.ToTaskID,
		&taskDAGRaw.Payload,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return nil, FormatError(err)
	}
	return &taskDAGRaw, nil
}

func findTaskDAGRawListImpl(ctx context.Context, tx *Tx, find *api.TaskDAGFind) ([]*taskDAGRaw, error) {
	where, args := []string{"TRUE"}, []interface{}{}
	if v := find.FromTaskID; v != nil {
		where, args = append(where, fmt.Sprintf("from_task_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.ToTaskID; v != nil {
		where, args = append(where, fmt.Sprintf("to_task_id = $%d", len(args)+1)), append(args, *v)
	}
	rows, err := tx.QueryContext(ctx, `
		SELECT
			id,
			created_ts,
			updated_ts,
			from_task_id,
			to_task_id,
			payload
		FROM task_dag
		WHERE `+strings.Join(where, " AND "),
		args...,
	)
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
