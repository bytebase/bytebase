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

// taskRunRaw is the store model for an TaskRun.
// Fields have exactly the same meanings as TaskRun.
type taskRunRaw struct {
	ID int

	// Standard fields
	CreatorID int
	CreatedTs int64
	UpdaterID int
	UpdatedTs int64

	// Related fields
	TaskID int

	// Domain specific fields
	Name    string
	Status  api.TaskRunStatus
	Type    api.TaskType
	Code    common.Code
	Comment string
	Result  string
	Payload string
}

// toTaskRun creates an instance of TaskRun based on the taskRunRaw.
// This is intended to be called when we need to compose an TaskRun relationship.
func (raw *taskRunRaw) toTaskRun() *api.TaskRun {
	return &api.TaskRun{
		ID: raw.ID,

		// Standard fields
		CreatorID: raw.CreatorID,
		CreatedTs: raw.CreatedTs,
		UpdaterID: raw.UpdaterID,
		UpdatedTs: raw.UpdatedTs,

		// Related fields
		TaskID: raw.TaskID,

		// Domain specific fields
		Name:    raw.Name,
		Status:  raw.Status,
		Type:    raw.Type,
		Code:    raw.Code,
		Comment: raw.Comment,
		Result:  raw.Result,
		Payload: raw.Payload,
	}
}

// createTaskRunImpl creates a new taskRun.
func (*Store) createTaskRunImpl(ctx context.Context, tx *Tx, create *api.TaskRunCreate) (*taskRunRaw, error) {
	if create.Payload == "" {
		create.Payload = "{}"
	}
	query := `
		INSERT INTO task_run (
			creator_id,
			updater_id,
			task_id,
			name,
			status,
			type,
			payload
		)
		VALUES ($1, $2, $3, $4, 'RUNNING', $5, $6)
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, task_id, name, status, type, code, comment, result, payload
	`
	var taskRunRaw taskRunRaw
	if err := tx.QueryRowContext(ctx, query,
		create.CreatorID,
		create.CreatorID,
		create.TaskID,
		create.Name,
		create.Type,
		create.Payload,
	).Scan(
		&taskRunRaw.ID,
		&taskRunRaw.CreatorID,
		&taskRunRaw.CreatedTs,
		&taskRunRaw.UpdaterID,
		&taskRunRaw.UpdatedTs,
		&taskRunRaw.TaskID,
		&taskRunRaw.Name,
		&taskRunRaw.Status,
		&taskRunRaw.Type,
		&taskRunRaw.Code,
		&taskRunRaw.Comment,
		&taskRunRaw.Result,
		&taskRunRaw.Payload,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return nil, FormatError(err)
	}
	return &taskRunRaw, nil
}

// getTaskRunRawTx retrieves a single taskRun based on find.
// Returns ECONFLICT if finding more than 1 matching records.
func (s *Store) getTaskRunRawTx(ctx context.Context, tx *Tx, find *api.TaskRunFind) (*taskRunRaw, error) {
	taskRunRawList, err := s.findTaskRunImpl(ctx, tx, find)
	if err != nil {
		return nil, err
	}
	if len(taskRunRawList) == 0 {
		return nil, nil
	} else if len(taskRunRawList) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: errors.Errorf("found %d task runs with filter %+v, expect 1", len(taskRunRawList), find)}
	}
	return taskRunRawList[0], nil
}

// patchTaskRunStatusImpl updates a taskRun status. Returns the new state of the taskRun after update.
func (*Store) patchTaskRunStatusImpl(ctx context.Context, tx *Tx, patch *api.TaskRunStatusPatch) (*taskRunRaw, error) {
	// Build UPDATE clause.
	set, args := []string{"updater_id = $1"}, []interface{}{patch.UpdaterID}
	set, args = append(set, "status = $2"), append(args, patch.Status)
	if v := patch.Code; v != nil {
		set, args = append(set, fmt.Sprintf("code = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Comment; v != nil {
		set, args = append(set, fmt.Sprintf("comment = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Result; v != nil {
		result := "{}"
		if *v != "" {
			result = *v
		}
		set, args = append(set, fmt.Sprintf("result = $%d", len(args)+1)), append(args, result)
	}

	// Build WHERE clause.
	where := []string{"1 = 1"}
	if v := patch.ID; v != nil {
		where, args = append(where, fmt.Sprintf("id = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.TaskID; v != nil {
		where, args = append(where, fmt.Sprintf("task_id = $%d", len(args)+1)), append(args, *v)
	}

	var taskRunRaw taskRunRaw
	if err := tx.QueryRowContext(ctx, `
		UPDATE task_run
		SET `+strings.Join(set, ", ")+`
		WHERE `+strings.Join(where, " AND ")+`
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, task_id, name, status, type, code, comment, result, payload
	`,
		args...,
	).Scan(
		&taskRunRaw.ID,
		&taskRunRaw.CreatorID,
		&taskRunRaw.CreatedTs,
		&taskRunRaw.UpdaterID,
		&taskRunRaw.UpdatedTs,
		&taskRunRaw.TaskID,
		&taskRunRaw.Name,
		&taskRunRaw.Status,
		&taskRunRaw.Type,
		&taskRunRaw.Code,
		&taskRunRaw.Comment,
		&taskRunRaw.Result,
		&taskRunRaw.Payload,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, &common.Error{Code: common.NotFound, Err: errors.Errorf("project ID not found: %d", patch.ID)}
		}
		return nil, FormatError(err)
	}
	return &taskRunRaw, nil
}

func (s *Store) listTaskRun(ctx context.Context, find *api.TaskRunFind) ([]*taskRunRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := s.findTaskRunImpl(ctx, tx, find)
	if err != nil {
		return nil, FormatError(err)
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return list, nil
}

func (*Store) findTaskRunImpl(ctx context.Context, tx *Tx, find *api.TaskRunFind) ([]*taskRunRaw, error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.ID; v != nil {
		where, args = append(where, fmt.Sprintf("id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.TaskID; v != nil {
		where, args = append(where, fmt.Sprintf("task_id = $%d", len(args)+1)), append(args, *v)
	}

	if v := find.StatusList; v != nil {
		list := []string{}
		for _, status := range *v {
			list = append(list, fmt.Sprintf("$%d", len(args)+1))
			args = append(args, status)
		}
		where = append(where, fmt.Sprintf("status in (%s)", strings.Join(list, ",")))
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT
			id,
			creator_id,
			created_ts,
			updater_id,
			updated_ts,
			task_id,
			name,
			status,
			type,
			code,
			comment,
			result,
			payload
		FROM task_run
		WHERE `+strings.Join(where, " AND "),
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()

	// Iterate over result set and deserialize rows into taskRunRawList.
	var taskRunRawList []*taskRunRaw
	for rows.Next() {
		var taskRunRaw taskRunRaw
		if err := rows.Scan(
			&taskRunRaw.ID,
			&taskRunRaw.CreatorID,
			&taskRunRaw.CreatedTs,
			&taskRunRaw.UpdaterID,
			&taskRunRaw.UpdatedTs,
			&taskRunRaw.TaskID,
			&taskRunRaw.Name,
			&taskRunRaw.Status,
			&taskRunRaw.Type,
			&taskRunRaw.Code,
			&taskRunRaw.Comment,
			&taskRunRaw.Result,
			&taskRunRaw.Payload,
		); err != nil {
			return nil, FormatError(err)
		}

		taskRunRawList = append(taskRunRawList, &taskRunRaw)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return taskRunRawList, nil
}
