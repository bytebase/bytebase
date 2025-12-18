package store

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/google/cel-go/cel"
	celast "github.com/google/cel-go/common/ast"
	celoperators "github.com/google/cel-go/common/operators"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/qb"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// PipelineMessage is the message for pipelines.
type PipelineMessage struct {
	ProjectID string
	Tasks     []*TaskMessage
	// Output only.
	ID        int
	Creator   string
	CreatedAt time.Time
	// The UpdatedAt is a latest time of task/taskRun updates.
	// If there are no tasks, it will be the same as CreatedAt.
	UpdatedAt time.Time
	IssueID   *int
}

// PipelineFind is the API message for finding pipelines.
type PipelineFind struct {
	ID        *int
	ProjectID *string

	Limit   *int
	Offset  *int
	FilterQ *qb.Query
}

// CreatePipelineAIO creates a pipeline with tasks all in one.
func (s *Store) CreatePipelineAIO(ctx context.Context, planUID int64, pipeline *PipelineMessage, creator string) (createdPipelineUID int, err error) {
	tx, err := s.GetDB().BeginTx(ctx, nil)
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
		createdPipeline, err := s.createPipeline(ctx, tx, pipeline, creator)
		if err != nil {
			return 0, errors.Wrapf(err, "failed to create pipeline")
		}
		createdPipelineUID = createdPipeline.ID

		// update pipeline uid of associated issue and plan
		if invalidateCacheF, err = s.updatePipelineUIDOfPlan(ctx, tx, planUID, createdPipelineUID); err != nil {
			return 0, errors.Wrapf(err, "failed to update associated plan or issue")
		}
	} else {
		createdPipelineUID = *pipelineUIDMaybe
	}

	// Check existing tasks to avoid duplicates
	existingTasks, err := s.listTasksTx(ctx, tx, &TaskFind{
		PipelineID: &createdPipelineUID,
	})
	if err != nil {
		return 0, errors.Wrapf(err, "failed to list existing tasks")
	}

	type taskKey struct {
		instance string
		database string
		sheet    int
	}

	createdTasks := map[taskKey]struct{}{}
	for _, task := range existingTasks {
		k := taskKey{
			instance: task.InstanceID,
			sheet:    int(task.Payload.GetSheetId()),
		}
		if task.DatabaseName != nil {
			k.database = *task.DatabaseName
		}
		createdTasks[k] = struct{}{}
	}

	var taskCreateList []*TaskMessage

	for _, taskCreate := range pipeline.Tasks {
		k := taskKey{
			instance: taskCreate.InstanceID,
			sheet:    int(taskCreate.Payload.GetSheetId()),
		}
		if taskCreate.DatabaseName != nil {
			k.database = *taskCreate.DatabaseName
		}

		if _, ok := createdTasks[k]; ok {
			continue
		}
		taskCreate.PipelineID = createdPipelineUID
		taskCreateList = append(taskCreateList, taskCreate)
	}

	if len(taskCreateList) > 0 {
		if _, err := s.createTasks(ctx, tx, taskCreateList...); err != nil {
			return 0, errors.Wrap(err, "failed to create tasks")
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, errors.Wrapf(err, "failed to commit tx")
	}
	invalidateCacheF()

	return createdPipelineUID, nil
}

// returns func() to invalidate cache.
func (*Store) updatePipelineUIDOfPlan(ctx context.Context, txn *sql.Tx, planUID int64, pipelineUID int) (func(), error) {
	q := qb.Q().Space(`
		UPDATE plan
		SET pipeline_id = ?
		WHERE id = ?
	`, pipelineUID, planUID)
	querySQL, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}
	if _, err := txn.ExecContext(ctx, querySQL, args...); err != nil {
		return nil, errors.Wrapf(err, "failed to update plan pipeline_id")
	}

	return func() {
		// TODO: need to remove planCache once we add planCache
	}, nil
}

func lockPlanAndGetPipelineUID(ctx context.Context, txn *sql.Tx, planUID int64) (*int, error) {
	q := qb.Q().Space("SELECT pipeline_id FROM plan WHERE id = ? FOR UPDATE", planUID)
	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}
	var uid sql.NullInt32
	if err := txn.QueryRowContext(ctx, query, args...).Scan(&uid); err != nil {
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

func (*Store) createPipeline(ctx context.Context, txn *sql.Tx, create *PipelineMessage, creator string) (*PipelineMessage, error) {
	q := qb.Q().Space(`
		INSERT INTO pipeline (
			project,
			creator
		)
		VALUES (
			?,
			?
		)
		RETURNING id, created_at
	`, create.ProjectID, creator)
	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}
	pipeline := &PipelineMessage{
		ProjectID: create.ProjectID,
		Creator:   creator,
	}
	if err := txn.QueryRowContext(ctx, query, args...).Scan(
		&pipeline.ID,
		&pipeline.CreatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return nil, errors.Wrapf(err, "failed to insert")
	}
	// Initialize UpdatedAt with CreatedAt for new pipelines
	pipeline.UpdatedAt = pipeline.CreatedAt

	return pipeline, nil
}

// GetPipeline gets the pipeline.
func (s *Store) GetPipeline(ctx context.Context, find *PipelineFind) (*PipelineMessage, error) {
	pipelines, err := s.ListPipelines(ctx, find)
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

// GetPipelineByID gets the pipeline by ID.
func (s *Store) GetPipelineByID(ctx context.Context, id int) (*PipelineMessage, error) {
	return s.GetPipeline(ctx, &PipelineFind{ID: &id})
}

// ListPipelines lists pipelines.
func (s *Store) ListPipelines(ctx context.Context, find *PipelineFind) ([]*PipelineMessage, error) {
	q := qb.Q().Space(`
		SELECT
			pipeline.id,
			pipeline.creator,
			pipeline.created_at,
			pipeline.project,
			issue.id,
			COALESCE(
				(
					SELECT MAX(task_run.updated_at)
					FROM task
					JOIN task_run ON task_run.task_id = task.id
					WHERE task.pipeline_id = pipeline.id
				),
				pipeline.created_at
			) AS updated_at
		FROM pipeline
		LEFT JOIN plan ON plan.pipeline_id = pipeline.id
		LEFT JOIN issue ON issue.plan_id = plan.id
		WHERE TRUE
	`)

	if filterQ := find.FilterQ; filterQ != nil {
		q.And("?", filterQ)
	}
	if v := find.ID; v != nil {
		q.And("pipeline.id = ?", *v)
	}
	if v := find.ProjectID; v != nil {
		q.And("pipeline.project = ?", *v)
	}

	q.Space("ORDER BY pipeline.id DESC")
	if v := find.Limit; v != nil {
		q.Space("LIMIT ?", *v)
	}
	if v := find.Offset; v != nil {
		q.Space("OFFSET ?", *v)
	}

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	tx, err := s.GetDB().BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
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
			&pipeline.Creator,
			&pipeline.CreatedAt,
			&pipeline.ProjectID,
			&pipeline.IssueID,
			&pipeline.UpdatedAt,
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
	return pipelines, nil
}

// GetListRolloutFilter parses a CEL filter expression into a query builder query for listing rollouts.
func (s *Store) GetListRolloutFilter(_ context.Context, filter string) (*qb.Query, error) { // nolint:revive
	if filter == "" {
		return nil, nil
	}

	e, err := cel.NewEnv()
	if err != nil {
		return nil, errors.Errorf("failed to create cel env")
	}
	ast, iss := e.Parse(filter)
	if iss != nil {
		return nil, errors.Errorf("failed to parse filter %v, error: %v", filter, iss.String())
	}

	var getFilter func(expr celast.Expr) (*qb.Query, error)

	getFilter = func(expr celast.Expr) (*qb.Query, error) {
		q := qb.Q()
		switch expr.Kind() {
		case celast.CallKind:
			functionName := expr.AsCall().FunctionName()
			switch functionName {
			case celoperators.LogicalAnd:
				for _, arg := range expr.AsCall().Args() {
					qq, err := getFilter(arg)
					if err != nil {
						return nil, err
					}
					q.And("?", qq)
				}
				return qb.Q().Space("(?)", q), nil
			case celoperators.Equals:
				variable, value := getVariableAndValueFromExpr(expr)
				switch variable {
				case "creator":
					creatorEmail := strings.TrimPrefix(value.(string), "users/")
					if creatorEmail == "" {
						return nil, errors.New("invalid empty creator identifier")
					}
					return qb.Q().Space("pipeline.creator = ?", creatorEmail), nil
				case "task_type":
					taskType, ok := value.(string)
					if !ok {
						return nil, errors.Errorf("task_type value must be a string")
					}
					if _, ok := v1pb.Task_Type_value[taskType]; !ok {
						return nil, errors.Errorf("invalid task_type value: %s", taskType)
					}
					v1TaskType := v1pb.Task_Type(v1pb.Task_Type_value[taskType])
					storeTaskType := convertV1ToStoreTaskType(v1TaskType)
					return qb.Q().Space("EXISTS (SELECT 1 FROM task WHERE task.pipeline_id = pipeline.id AND task.type = ?)", storeTaskType.String()), nil
				default:
					return nil, errors.Errorf("unsupported variable %q", variable)
				}
			case celoperators.In:
				variable, value := getVariableAndValueFromExpr(expr)
				switch variable {
				case "task_type":
					rawList, ok := value.([]any)
					if !ok {
						return nil, errors.Errorf("invalid list value %q for %v", value, variable)
					}
					if len(rawList) == 0 {
						return nil, errors.Errorf("empty list value for filter %v", variable)
					}
					var taskTypes []string
					for _, raw := range rawList {
						taskType, ok := raw.(string)
						if !ok {
							return nil, errors.Errorf("task_type value must be a string")
						}
						if _, ok := v1pb.Task_Type_value[taskType]; !ok {
							return nil, errors.Errorf("invalid task_type value: %s", taskType)
						}
						v1TaskType := v1pb.Task_Type(v1pb.Task_Type_value[taskType])
						storeTaskType := convertV1ToStoreTaskType(v1TaskType)
						taskTypes = append(taskTypes, storeTaskType.String())
					}
					return qb.Q().Space("EXISTS (SELECT 1 FROM task WHERE task.pipeline_id = pipeline.id AND task.type = ANY(?))", taskTypes), nil
				default:
					return nil, errors.Errorf("unsupported variable %q", variable)
				}
			case celoperators.GreaterEquals, celoperators.LessEquals:
				variable, rawValue := getVariableAndValueFromExpr(expr)
				value, ok := rawValue.(string)
				if !ok {
					return nil, errors.Errorf("expect string, got %T, hint: filter literals should be string", rawValue)
				}
				if variable != "update_time" {
					return nil, errors.Errorf(`">=" and "<=" are only supported for "update_time"`)
				}
				t, err := time.Parse(time.RFC3339, value)
				if err != nil {
					return nil, errors.Errorf("failed to parse time %v, error: %v", value, err)
				}
				if functionName == celoperators.GreaterEquals {
					return qb.Q().Space("updated_at >= ?", t), nil
				}
				return qb.Q().Space("updated_at <= ?", t), nil
			default:
				return nil, errors.Errorf("unexpected function %v", functionName)
			}
		default:
			return nil, errors.Errorf("unexpected expr kind %v", expr.Kind())
		}
	}

	q, err := getFilter(ast.NativeRep().Expr())
	if err != nil {
		return nil, err
	}
	return qb.Q().Space("(?)", q), nil
}

func convertV1ToStoreTaskType(taskType v1pb.Task_Type) storepb.Task_Type {
	switch taskType {
	case v1pb.Task_DATABASE_CREATE:
		return storepb.Task_DATABASE_CREATE
	case v1pb.Task_DATABASE_MIGRATE:
		return storepb.Task_DATABASE_MIGRATE
	case v1pb.Task_DATABASE_SDL:
		return storepb.Task_DATABASE_SDL
	case v1pb.Task_DATABASE_EXPORT:
		return storepb.Task_DATABASE_EXPORT
	case v1pb.Task_TYPE_UNSPECIFIED, v1pb.Task_GENERAL:
		return storepb.Task_TASK_TYPE_UNSPECIFIED
	default:
		return storepb.Task_TASK_TYPE_UNSPECIFIED
	}
}
