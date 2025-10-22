package store

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/common/qb"
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
	Name        *string
	Description *string
	Specs       *[]*storepb.PlanConfig_Spec
	Deployment  **storepb.PlanConfig_Deployment
	Deleted     *bool
}

// CreatePlan creates a new plan.
func (s *Store) CreatePlan(ctx context.Context, plan *PlanMessage, creatorUID int) (*PlanMessage, error) {
	config, err := protojson.Marshal(plan.Config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal plan config")
	}

	q := qb.Q().Space(`
		INSERT INTO plan (
			creator_id,
			project,
			pipeline_id,
			name,
			description,
			config
		) VALUES (
			?, ?, ?, ?, ?, ?
		) RETURNING id, created_at, updated_at
	`, creatorUID, plan.ProjectID, plan.PipelineUID, plan.Name, plan.Description, config)

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrap(err, "failed to build sql")
	}

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	var id int64
	if err := tx.QueryRowContext(ctx, query, args...).Scan(&id, &plan.CreatedAt, &plan.UpdatedAt); err != nil {
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
		slog.Error("expect to find one plan, found multiple",
			slog.Int("count", len(plans)),
			slog.Any("uid", find.UID),
			slog.Any("project_id", find.ProjectID),
			slog.Any("project_ids", find.ProjectIDs),
			slog.Any("pipeline_id", find.PipelineID),
			log.BBStack("stack"),
		)
		return nil, errors.Errorf("expect to find one plan, found %d", len(plans))
	}
	return plans[0], nil
}

// ListPlans retrieves a list of plans.
func (s *Store) ListPlans(ctx context.Context, find *FindPlanMessage) ([]*PlanMessage, error) {
	q := qb.Q().Space(`
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
		WHERE TRUE
	`)

	if filter := find.Filter; filter != nil {
		// Convert $1, $2, etc. to ? for qb
		q.And(ConvertDollarPlaceholders(filter.Where), filter.Args...)
	}
	if v := find.UID; v != nil {
		q.And("plan.id = ?", *v)
	}
	if v := find.ProjectID; v != nil {
		q.And("plan.project = ?", *v)
	}
	if v := find.ProjectIDs; v != nil {
		if len(*v) == 0 {
			q.And("FALSE")
		} else {
			q.And("plan.project = ANY(?)", *v)
		}
	}
	if v := find.PipelineID; v != nil {
		q.And("plan.pipeline_id = ?", *v)
	}

	q.Space("ORDER BY id DESC")
	if v := find.Limit; v != nil {
		q.Space("LIMIT ?", *v)
	}
	if v := find.Offset; v != nil {
		q.Space("OFFSET ?", *v)
	}

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrap(err, "failed to build sql")
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
	set := []string{"updated_at = ?"}
	args := []any{time.Now()}

	if v := patch.Name; v != nil {
		set = append(set, "name = ?")
		args = append(args, *v)
	}
	if v := patch.Description; v != nil {
		set = append(set, "description = ?")
		args = append(args, *v)
	}
	if v := patch.Deleted; v != nil {
		set = append(set, "deleted = ?")
		args = append(args, *v)
	}

	var payloadSets []string
	if v := patch.Specs; v != nil {
		config, err := protojson.Marshal(&storepb.PlanConfig{
			Specs: *v,
		})
		if err != nil {
			return errors.Wrapf(err, "failed to marshal plan config")
		}
		payloadSets = append(payloadSets, "jsonb_build_object('specs', (?)::JSONB->'specs')")
		args = append(args, config)
	}
	if v := patch.Deployment; v != nil {
		p, err := protojson.Marshal(*v)
		if err != nil {
			return errors.Wrapf(err, "failed to marshal deployment")
		}
		payloadSets = append(payloadSets, "jsonb_build_object('deployment', (?)::JSONB)")
		args = append(args, p)
	}
	if len(payloadSets) > 0 {
		set = append(set, fmt.Sprintf("config = config || %s", strings.Join(payloadSets, " || ")))
	}

	args = append(args, patch.UID)
	q := qb.Q().Space(fmt.Sprintf("UPDATE plan SET %s WHERE id = ?", strings.Join(set, ", ")), args...)

	query, finalArgs, err := q.ToSQL()
	if err != nil {
		return errors.Wrapf(err, "failed to build sql")
	}

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrapf(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, query, finalArgs...); err != nil {
		return errors.Wrapf(err, "failed to update plan")
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrapf(err, "failed to commit transaction")
	}

	return nil
}
