package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
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
	CreatedAt  time.Time
	UpdatedAt  time.Time
	Deleted    bool
}

// FindPlanMessage is the message to find a plan.
type FindPlanMessage struct {
	UID        *int64
	ProjectID  *string
	ProjectIDs *[]string
	PipelineID *int

	Limit  *int
	Offset *int

	Filter *ListResourceFilter
}

// UpdatePlanMessage is the message to update a plan.
type UpdatePlanMessage struct {
	UID         int64
	PipelineUID *int
	Name        *string
	Description *string
	Specs       *[]*storepb.PlanConfig_Spec
	Deployment  **storepb.PlanConfig_Deployment
	Deleted     *bool
}

// CreatePlan creates a new plan.
func (s *Store) CreatePlan(ctx context.Context, plan *PlanMessage, creatorUID int) (*PlanMessage, error) {
	query := `
		INSERT INTO plan (
			creator_id,
			project,
			pipeline_id,
			name,
			description,
			config
		) VALUES (
			$1,
			$2,
			$3,
			$4,
			$5,
			$6
		) RETURNING id, created_at, updated_at
	`

	config, err := protojson.Marshal(plan.Config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal plan config")
	}
	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	var id int64
	if err := tx.QueryRowContext(ctx, query,
		creatorUID,
		plan.ProjectID,
		plan.PipelineUID,
		plan.Name,
		plan.Description,
		config,
	).Scan(&id, &plan.CreatedAt, &plan.UpdatedAt); err != nil {
		return nil, errors.Wrap(err, "failed to insert plan")
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrap(err, "failed to commit transaction")
	}

	plan.UID = id
	plan.CreatorUID = creatorUID
	return plan, nil
}

// GetPlan gets a plan.
func (s *Store) GetPlan(ctx context.Context, find *FindPlanMessage) (*PlanMessage, error) {
	plans, err := s.ListPlans(ctx, find)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list plans")
	}
	if len(plans) == 0 {
		return nil, nil
	}
	if len(plans) > 1 {
		return nil, errors.Errorf("expect to find one plan, found %d", len(plans))
	}
	return plans[0], nil
}

// ListPlans retrieves a list of plans.
func (s *Store) ListPlans(ctx context.Context, find *FindPlanMessage) ([]*PlanMessage, error) {
	where, args := []string{"TRUE"}, []any{}
	if filter := find.Filter; filter != nil {
		where = append(where, filter.Where)
		args = append(args, filter.Args...)
	}
	if v := find.UID; v != nil {
		where, args = append(where, fmt.Sprintf("plan.id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.ProjectID; v != nil {
		where, args = append(where, fmt.Sprintf("plan.project = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.ProjectIDs; v != nil {
		if len(*v) == 0 {
			where = append(where, "FALSE")
		} else {
			where, args = append(where, fmt.Sprintf("plan.project = ANY($%d)", len(args)+1)), append(args, *v)
		}
	}
	if v := find.PipelineID; v != nil {
		where, args = append(where, fmt.Sprintf("plan.pipeline_id = $%d", len(args)+1)), append(args, *v)
	}

	query := fmt.Sprintf(`
		SELECT
			plan.id,
			plan.creator_id,
			plan.created_at,
			plan.updated_at,
			plan.project,
			plan.pipeline_id,
			plan.name,
			plan.description,
			plan.config,
			plan.deleted
		FROM plan
		LEFT JOIN issue on plan.id = issue.plan_id
		WHERE %s
		ORDER BY id DESC
	`, strings.Join(where, " AND "))
	if v := find.Limit; v != nil {
		query += fmt.Sprintf(" LIMIT %d", *v)
	}
	if v := find.Offset; v != nil {
		query += fmt.Sprintf(" OFFSET %d", *v)
	}

	tx, err := s.GetDB().BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
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
			&plan.CreatedAt,
			&plan.UpdatedAt,
			&plan.ProjectID,
			&plan.PipelineUID,
			&plan.Name,
			&plan.Description,
			&config,
			&plan.Deleted,
		); err != nil {
			return nil, errors.Wrap(err, "failed to scan plan")
		}
		if err := common.ProtojsonUnmarshaler.Unmarshal(config, plan.Config); err != nil {
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
	set, args := []string{"updated_at = $1"}, []any{time.Now()}
	if v := patch.Name; v != nil {
		set, args = append(set, fmt.Sprintf("name = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Description; v != nil {
		set, args = append(set, fmt.Sprintf("description = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Deleted; v != nil {
		set, args = append(set, fmt.Sprintf("deleted = $%d", len(args)+1)), append(args, *v)
	}
	{
		var payloadSets []string
		if v := patch.Specs; v != nil {
			config, err := protojson.Marshal(&storepb.PlanConfig{
				Specs: *v,
			})
			if err != nil {
				return errors.Wrapf(err, "failed to marshal plan config")
			}
			payloadSets = append(payloadSets, fmt.Sprintf("jsonb_build_object('specs', ($%d)::JSONB->'specs')", len(args)+1))
			args = append(args, config)
		}
		if v := patch.Deployment; v != nil {
			p, err := protojson.Marshal(*v)
			if err != nil {
				return errors.Wrapf(err, "failed to marshal deployment")
			}
			payloadSets = append(payloadSets, fmt.Sprintf("jsonb_build_object('deployment', ($%d)::JSONB)", len(args)+1))
			args = append(args, p)
		}
		if len(payloadSets) > 0 {
			set = append(set, fmt.Sprintf("config = config || %s", strings.Join(payloadSets, " || ")))
		}
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

	tx, err := s.GetDB().BeginTx(ctx, nil)
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
