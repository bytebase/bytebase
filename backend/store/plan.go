package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// PlanMessage is the message for plan.
type PlanMessage struct {
	ProjectID   string
	PipelineUID *int
	Name        string
	Description string
	Config      *storepb.PlanConfig
	// output only
	UID        int64
	CreatorUID int
	CreatedTs  int64
	UpdaterUID int
	UpdatedTs  int64
}

// FindPlanMessage is the message to find a plan.
type FindPlanMessage struct {
	UID        *int64
	ProjectID  *string
	PipelineID *int

	Limit  *int
	Offset *int
}

// UpdatePlanMessage is the message to update a plan.
type UpdatePlanMessage struct {
	UID         int64
	PipelineUID *int
	Config      *storepb.PlanConfig
	UpdaterID   int
}

// CreatePlan creates a new plan.
func (s *Store) CreatePlan(ctx context.Context, plan *PlanMessage, creatorUID int) (*PlanMessage, error) {
	query := `
		INSERT INTO plan (
			creator_id,
			updater_id,
			project_id,
			pipeline_id,
			name,
			description,
			config
		) VALUES (
			$1,
			$2,
			(SELECT project.id FROM project WHERE project.resource_id = $3),
			$4,
			$5,
			$6,
			$7
		) RETURNING id, created_ts
	`

	config, err := protojson.Marshal(plan.Config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal plan config")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	var id, createdTs int64
	if err := tx.QueryRowContext(ctx, query,
		creatorUID,
		creatorUID,
		plan.ProjectID,
		plan.PipelineUID,
		plan.Name,
		plan.Description,
		config,
	).Scan(&id, &createdTs); err != nil {
		return nil, errors.Wrap(err, "failed to insert plan")
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrap(err, "failed to commit transaction")
	}

	plan.UID = id
	plan.CreatorUID = creatorUID
	plan.CreatedTs = createdTs
	plan.UpdaterUID = creatorUID
	plan.UpdatedTs = createdTs
	return plan, nil
}

// GetPlan gets a plan.
func (s *Store) GetPlan(ctx context.Context, uid int64) (*PlanMessage, error) {
	plans, err := s.ListPlans(ctx, &FindPlanMessage{UID: &uid})
	if err != nil {
		return nil, errors.Wrap(err, "failed to list plans")
	}
	if len(plans) == 0 {
		return nil, nil
	}
	if len(plans) > 1 {
		return nil, errors.Errorf("found multiple plans with UID %d", uid)
	}
	return plans[0], nil
}

// ListPlans retrieves a list of plans.
func (s *Store) ListPlans(ctx context.Context, find *FindPlanMessage) ([]*PlanMessage, error) {
	where, args := []string{"TRUE"}, []any{}
	if v := find.UID; v != nil {
		where, args = append(where, fmt.Sprintf("plan.id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.ProjectID; v != nil {
		where, args = append(where, fmt.Sprintf("project.resource_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.PipelineID; v != nil {
		where, args = append(where, fmt.Sprintf("plan.pipeline_id = $%d", len(args)+1)), append(args, *v)
	}
	query := fmt.Sprintf(`
		SELECT
			plan.id,	
			plan.creator_id,
			plan.created_ts,
			plan.updater_id,
			plan.updated_ts,
			project.resource_id,
			plan.pipeline_id,
			plan.name,
			plan.description,
			plan.config
		FROM plan
		LEFT JOIN project on plan.project_id = project.id
		WHERE %s
		ORDER BY id ASC
	`, strings.Join(where, " AND "))
	if v := find.Limit; v != nil {
		query += fmt.Sprintf(" LIMIT %d", *v)
	}
	if v := find.Offset; v != nil {
		query += fmt.Sprintf(" OFFSET %d", *v)
	}

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to select plans")
	}
	defer rows.Close()

	var plans []*PlanMessage
	for rows.Next() {
		plan := PlanMessage{
			Config: &storepb.PlanConfig{},
		}
		var config []byte
		if err := rows.Scan(
			&plan.UID,
			&plan.CreatorUID,
			&plan.CreatedTs,
			&plan.UpdaterUID,
			&plan.UpdatedTs,
			&plan.ProjectID,
			&plan.PipelineUID,
			&plan.Name,
			&plan.Description,
			&config,
		); err != nil {
			return nil, errors.Wrap(err, "failed to scan plan")
		}
		if err := protojson.Unmarshal(config, plan.Config); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal plan config")
		}
		plans = append(plans, &plan)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "failed to iterate plans")
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrap(err, "failed to commit transaction")
	}

	return plans, nil
}

// UpdatePlan updates an existing plan.
func (s *Store) UpdatePlan(ctx context.Context, patch *UpdatePlanMessage) error {
	set, args := []string{"updater_id = $1"}, []any{patch.UpdaterID}
	if v := patch.Config; v != nil {
		config, err := protojson.Marshal(v)
		if err != nil {
			return errors.Wrapf(err, "failed to marshal plan config")
		}
		set, args = append(set, fmt.Sprintf("config = $%d", len(args)+1)), append(args, config)
	}
	if v := patch.PipelineUID; v != nil {
		set, args = append(set, fmt.Sprintf("pipeline_id = $%d", len(args)+1)), append(args, v)
	}

	args = append(args, patch.UID)
	query := fmt.Sprintf(`
		UPDATE plan
		SET `+strings.Join(set, ", ")+`
		WHERE id = $%d
	`, len(args))

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrapf(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return errors.Wrapf(err, "failed to update plan")
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrapf(err, "failed to commit transaction")
	}

	return nil
}
