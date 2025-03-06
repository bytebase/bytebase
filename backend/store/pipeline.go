package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
)

// PipelineMessage is the message for pipelines.
type PipelineMessage struct {
	ProjectID string
	Name      string
	Stages    []*StageMessage
	// Output only.
	ID         int
	CreatorUID int
	CreatedAt  time.Time
	IssueID    *int
}

// PipelineFind is the API message for finding pipelines.
type PipelineFind struct {
	ID        *int
	ProjectID *string

	Limit  *int
	Offset *int
}

// targetStage == nil means deploy all stages.
// targetStage == "" means deploy no stages.
func (s *Store) CreatePipelineAIO(ctx context.Context, planUID int64, pipeline *PipelineMessage, creatorUID int) (createdPipelineUID int, err error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to begin tx")
	}
	defer tx.Rollback()

	pipelineUIDMaybe, err := lockPlanAndGetPipelineUID(ctx, tx, planUID)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to SELECT plan FOR UPDATE")
	}
	invalidateCacheF := func() {}
	if pipelineUIDMaybe == nil {
		createdPipeline, err := s.createPipeline(ctx, tx, pipeline, creatorUID)
		if err != nil {
			return 0, errors.Wrapf(err, "failed to create pipeline")
		}
		createdPipelineUID = createdPipeline.ID

		// update pipeline uid of associated issue and plan
		if invalidateCacheF, err = s.updatePipelineUIDOfIssueAndPlan(ctx, tx, planUID, createdPipelineUID); err != nil {
			return 0, errors.Wrapf(err, "failed to update associated plan or issue")
		}
	} else {
		createdPipelineUID = *pipelineUIDMaybe
	}

	stages, err := s.listStages(ctx, tx, createdPipelineUID)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to list stages")
	}
	oldCreatedStages := map[string]*StageMessage{}
	for _, stage := range stages {
		oldCreatedStages[stage.DeploymentID] = stage
	}

	var stagesToCreate []*StageMessage
	for _, stage := range pipeline.Stages {
		if createdStage, ok := oldCreatedStages[stage.DeploymentID]; ok {
			// The stage was created, but we could have tasks to create.
			tasks, err := s.listTasksTx(ctx, tx, &TaskFind{
				PipelineID: &createdPipelineUID,
				StageID:    &createdStage.ID,
			})
			if err != nil {
				return 0, errors.Wrapf(err, "failed to list tasks in a created stage")
			}

			type taskKey struct {
				instance string
				database string
				sheet    int
			}

			createdTasks := map[taskKey]struct{}{}
			for _, task := range tasks {
				var payloadSheetUID struct {
					SheetUID int `json:"sheetId"`
				}
				if err := json.Unmarshal([]byte(task.Payload), &payloadSheetUID); err != nil {
					return 0, errors.Wrapf(err, "failed to unmarshal task payload")
				}
				k := taskKey{
					instance: task.InstanceID,
					sheet:    payloadSheetUID.SheetUID,
				}
				if task.DatabaseName != nil {
					k.database = *task.DatabaseName
				}
				createdTasks[k] = struct{}{}
			}

			var taskCreateList []*TaskMessage

			for _, taskCreate := range stage.TaskList {
				var payloadSheetUID struct {
					SheetUID int `json:"sheetId"`
				}
				if err := json.Unmarshal([]byte(taskCreate.Payload), &payloadSheetUID); err != nil {
					return 0, errors.Wrapf(err, "failed to unmarshal task payload")
				}
				k := taskKey{
					instance: taskCreate.InstanceID,
					sheet:    payloadSheetUID.SheetUID,
				}
				if taskCreate.DatabaseName != nil {
					k.database = *taskCreate.DatabaseName
				}

				if _, ok := createdTasks[k]; ok {
					continue
				}
				taskCreate.PipelineID = createdPipelineUID
				taskCreate.StageID = createdStage.ID
				taskCreateList = append(taskCreateList, taskCreate)
			}

			if len(taskCreateList) > 0 {
				if _, err := s.createTasks(ctx, tx, taskCreateList...); err != nil {
					return 0, errors.Wrap(err, "failed to create tasks")
				}
			}
		} else {
			// Create the stage and the tasks.
			stagesToCreate = append(stagesToCreate, stage)
		}
	}

	createdStages, err := s.createStages(ctx, tx, stagesToCreate, createdPipelineUID)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to create stages")
	}

	for _, stage := range createdStages {
		var taskCreateList []*TaskMessage
		for _, taskCreate := range stage.TaskList {
			c := taskCreate
			c.PipelineID = createdPipelineUID
			c.StageID = stage.ID
			taskCreateList = append(taskCreateList, c)
		}
		if len(taskCreateList) > 0 {
			if _, err := s.createTasks(ctx, tx, taskCreateList...); err != nil {
				return 0, errors.Wrap(err, "failed to create tasks")
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, errors.Wrapf(err, "failed to commit tx")
	}
	invalidateCacheF()

	return createdPipelineUID, nil
}

// returns func() to invalidate cache.
func (s *Store) updatePipelineUIDOfIssueAndPlan(ctx context.Context, tx *Tx, planUID int64, pipelineUID int) (func(), error) {
	if _, err := tx.ExecContext(ctx, `
		UPDATE plan
		SET pipeline_id = $1
		WHERE id = $2
	`, pipelineUID, planUID); err != nil {
		return nil, errors.Wrapf(err, "failed to update plan pipeline_id")
	}
	var issueUID int
	if err := tx.QueryRowContext(ctx, `
		UPDATE issue
		SET pipeline_id = $1
		WHERE plan_id = $2
		RETURNING id
	`, pipelineUID, planUID).Scan(&issueUID); err != nil {
		if err != sql.ErrNoRows {
			return nil, errors.Wrapf(err, "failed to update issue pipeline_id")
		}
	}
	return func() {
		// TODO: need to remove planCache once we add planCache
		if issueUID != 0 {
			s.issueCache.Remove(issueUID)
		}
	}, nil
}

func lockPlanAndGetPipelineUID(ctx context.Context, tx *Tx, planUID int64) (*int, error) {
	query := `
		SELECT pipeline_id FROM plan WHERE id = $1 FOR UPDATE
	`
	var uid sql.NullInt32
	if err := tx.QueryRowContext(ctx, query, planUID).Scan(&uid); err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.Errorf("plan %d not found", planUID)
		}
		return nil, errors.Wrapf(err, "failed to get pipeline uid")
	}

	if uid.Valid {
		uidInt := int(uid.Int32)
		return &uidInt, nil
	}
	return nil, nil
}

func (*Store) createPipeline(ctx context.Context, tx *Tx, create *PipelineMessage, creatorUID int) (*PipelineMessage, error) {
	query := `
		INSERT INTO pipeline (
			project,
			creator_id,
			name
		)
		VALUES (
			$1,
			$2,
			$3
		)
		RETURNING id, created_at
	`
	pipeline := &PipelineMessage{
		ProjectID:  create.ProjectID,
		CreatorUID: creatorUID,
		Name:       create.Name,
	}
	if err := tx.QueryRowContext(ctx, query,
		create.ProjectID,
		creatorUID,
		create.Name,
	).Scan(
		&pipeline.ID,
		&pipeline.CreatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return nil, errors.Wrapf(err, "failed to insert")
	}

	return pipeline, nil
}

// GetPipelineV2ByID gets the pipeline by ID.
func (s *Store) GetPipelineV2ByID(ctx context.Context, id int) (*PipelineMessage, error) {
	if v, ok := s.pipelineCache.Get(id); ok {
		return v, nil
	}
	pipelines, err := s.ListPipelineV2(ctx, &PipelineFind{ID: &id})
	if err != nil {
		return nil, err
	}

	if len(pipelines) == 0 {
		return nil, nil
	} else if len(pipelines) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: errors.Errorf("found %d pipelines, expect 1", len(pipelines))}
	}
	pipeline := pipelines[0]
	return pipeline, nil
}

// ListPipelineV2 lists pipelines.
func (s *Store) ListPipelineV2(ctx context.Context, find *PipelineFind) ([]*PipelineMessage, error) {
	where, args := []string{"TRUE"}, []any{}
	if v := find.ID; v != nil {
		where, args = append(where, fmt.Sprintf("pipeline.id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.ProjectID; v != nil {
		where, args = append(where, fmt.Sprintf("pipeline.project = $%d", len(args)+1)), append(args, *v)
	}
	query := fmt.Sprintf(`
		SELECT
			pipeline.id,
			pipeline.creator_id,
			pipeline.created_at,
			pipeline.project,
			pipeline.name,
			issue.id
		FROM pipeline
		LEFT JOIN issue ON pipeline.id = issue.pipeline_id
		WHERE %s
		ORDER BY pipeline.id DESC`, strings.Join(where, " AND "))
	if v := find.Limit; v != nil {
		query += fmt.Sprintf(" LIMIT %d", *v)
	}
	if v := find.Offset; v != nil {
		query += fmt.Sprintf(" OFFSET %d", *v)
	}

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pipelines []*PipelineMessage
	for rows.Next() {
		var pipeline PipelineMessage
		if err := rows.Scan(
			&pipeline.ID,
			&pipeline.CreatorUID,
			&pipeline.CreatedAt,
			&pipeline.ProjectID,
			&pipeline.Name,
			&pipeline.IssueID,
		); err != nil {
			return nil, err
		}
		pipelines = append(pipelines, &pipeline)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	for _, pipeline := range pipelines {
		s.pipelineCache.Add(pipeline.ID, pipeline)
	}
	return pipelines, nil
}
