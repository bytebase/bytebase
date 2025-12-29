package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/cel-go/cel"
	celast "github.com/google/cel-go/common/ast"
	celoperators "github.com/google/cel-go/common/operators"
	celoverloads "github.com/google/cel-go/common/overloads"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/qb"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// PlanMessage is the message for plan.
type PlanMessage struct {
	ProjectID   string
	Name        string
	Description string
	Config      *storepb.PlanConfig
	// output only
	UID       int64
	Creator   string
	CreatedAt time.Time
	UpdatedAt time.Time
	Deleted   bool
}

// FindPlanMessage is the message to find a plan.
type FindPlanMessage struct {
	UID        *int64
	ProjectID  *string
	ProjectIDs *[]string

	HasRollout *bool

	Limit  *int
	Offset *int

	FilterQ *qb.Query
}

// UpdatePlanMessage is the message to update a plan.
type UpdatePlanMessage struct {
	UID         int64
	Name        *string
	Description *string
	// Config replaces the entire plan config.
	// Callers should clone the existing config and modify only the fields they want to change.
	// Example: config := proto.CloneOf(plan.Config); config.HasRollout = true; patch.Config = config
	Config  *storepb.PlanConfig
	Deleted *bool
}

// CreatePlan creates a new plan.
func (s *Store) CreatePlan(ctx context.Context, plan *PlanMessage, creator string) (*PlanMessage, error) {
	config, err := protojson.Marshal(plan.Config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal plan config")
	}

	q := qb.Q().Space(`
		INSERT INTO plan (
			creator,
			project,
			name,
			description,
			config
		) VALUES (
			?, ?, ?, ?, ?
		) RETURNING id, created_at, updated_at
	`, creator, plan.ProjectID, plan.Name, plan.Description, config)

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
	plan.Creator = creator
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
	q := qb.Q().Space(`
		SELECT
			plan.id,
			plan.creator,
			plan.created_at,
			plan.updated_at,
			plan.project,
			plan.name,
			plan.description,
			plan.config,
			plan.deleted
		FROM plan
		LEFT JOIN issue on plan.id = issue.plan_id
		WHERE TRUE
	`)

	if filterQ := find.FilterQ; filterQ != nil {
		q.And("?", filterQ)
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
	if v := find.HasRollout; v != nil {
		if *v {
			q.And("plan.config->>'hasRollout' = ?", "true")
		} else {
			q.And("(plan.config->>'hasRollout' IS NULL OR plan.config->>'hasRollout' = ?)", "false")
		}
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
			&plan.Creator,
			&plan.CreatedAt,
			&plan.UpdatedAt,
			&plan.ProjectID,
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

// UpdatePlan updates an existing plan and returns the updated plan.
func (s *Store) UpdatePlan(ctx context.Context, patch *UpdatePlanMessage) (*PlanMessage, error) {
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
	if v := patch.Config; v != nil {
		config, err := protojson.Marshal(v)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to marshal plan config")
		}
		set = append(set, "config = ?")
		args = append(args, config)
	}

	args = append(args, patch.UID)
	q := qb.Q().Space(fmt.Sprintf("UPDATE plan SET %s WHERE id = ? RETURNING id, creator, created_at, updated_at, project, name, description, config, deleted", strings.Join(set, ", ")), args...)

	query, finalArgs, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	plan := PlanMessage{
		Config: &storepb.PlanConfig{},
	}
	var config []byte
	if err := tx.QueryRowContext(ctx, query, finalArgs...).Scan(
		&plan.UID,
		&plan.Creator,
		&plan.CreatedAt,
		&plan.UpdatedAt,
		&plan.ProjectID,
		&plan.Name,
		&plan.Description,
		&config,
		&plan.Deleted,
	); err != nil {
		return nil, errors.Wrapf(err, "failed to update plan")
	}
	if err := common.ProtojsonUnmarshaler.Unmarshal(config, plan.Config); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal plan config")
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit transaction")
	}

	return &plan, nil
}

// GetListPlanFilter parses a CEL filter expression into a query builder query for listing plans.
func GetListPlanFilter(filter string) (*qb.Query, error) {
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
					return qb.Q().Space("plan.creator = ?", creatorEmail), nil
				case "has_rollout":
					hasRollout, ok := value.(bool)
					if !ok {
						return nil, errors.Errorf(`"has_rollout" should be bool`)
					}
					if !hasRollout {
						return qb.Q().Space("(plan.config->>'hasRollout' IS NULL OR plan.config->>'hasRollout' = ?)", "false"), nil
					}
					return qb.Q().Space("plan.config->>'hasRollout' = ?", "true"), nil
				case "has_issue":
					hasIssue, ok := value.(bool)
					if !ok {
						return nil, errors.Errorf(`"has_issue" should be bool`)
					}
					if !hasIssue {
						return qb.Q().Space("issue.id IS NULL"), nil
					}
					return qb.Q().Space("issue.id IS NOT NULL"), nil
				case "title":
					return qb.Q().Space("plan.name = ?", value), nil
				case "spec_type":
					specType, ok := value.(string)
					if !ok {
						return nil, errors.Errorf("spec_type value must be a string")
					}
					switch specType {
					case "create_database_config":
						return qb.Q().Space("EXISTS (SELECT 1 FROM jsonb_array_elements(plan.config->'specs') AS spec WHERE spec->>'createDatabaseConfig' IS NOT NULL)"), nil
					case "change_database_config":
						return qb.Q().Space("EXISTS (SELECT 1 FROM jsonb_array_elements(plan.config->'specs') AS spec WHERE spec->>'changeDatabaseConfig' IS NOT NULL)"), nil
					case "export_data_config":
						return qb.Q().Space("EXISTS (SELECT 1 FROM jsonb_array_elements(plan.config->'specs') AS spec WHERE spec->>'exportDataConfig' IS NOT NULL)"), nil
					default:
						return nil, errors.Errorf("invalid spec_type value: %s, must be one of: create_database_config, change_database_config, export_data_config", specType)
					}
				case "state":
					stateStr, ok := value.(string)
					if !ok {
						return nil, errors.Errorf("state value must be string, got %T", value)
					}
					v1State, ok := v1pb.State_value[stateStr]
					if !ok {
						if v, exists := v1pb.State_value[strings.TrimPrefix(stateStr, "STATE_")]; exists {
							v1State = v
							ok = true
						}
					}
					if !ok {
						return nil, errors.Errorf("invalid state filter %q", value)
					}
					return qb.Q().Space("plan.deleted = ?", v1pb.State(v1State) == v1pb.State_DELETED), nil
				default:
					return nil, errors.Errorf("unsupported variable %q", variable)
				}
			case celoperators.GreaterEquals, celoperators.LessEquals:
				variable, rawValue := getVariableAndValueFromExpr(expr)
				value, ok := rawValue.(string)
				if !ok {
					return nil, errors.Errorf("expect string, got %T, hint: filter literals should be string", rawValue)
				}
				if variable != "create_time" {
					return nil, errors.Errorf(`">=" and "<=" are only supported for "create_time"`)
				}
				t, err := time.Parse(time.RFC3339, value)
				if err != nil {
					return nil, errors.Errorf("failed to parse time %v, error: %v", value, err)
				}
				if functionName == celoperators.GreaterEquals {
					return qb.Q().Space("plan.created_at >= ?", t), nil
				}
				return qb.Q().Space("plan.created_at <= ?", t), nil
			case celoverloads.Matches:
				variable := expr.AsCall().Target().AsIdent()
				args := expr.AsCall().Args()
				if len(args) != 1 {
					return nil, errors.Errorf(`invalid args for %q`, variable)
				}
				value := args[0].AsLiteral().Value()
				strValue, ok := value.(string)
				if !ok {
					return nil, errors.Errorf("expect string, got %T, hint: filter literals should be string", value)
				}
				strValue = strings.ToLower(strValue)

				switch variable {
				case "title":
					return qb.Q().Space("LOWER(plan.name) LIKE ?", "%"+strValue+"%"), nil
				default:
					return nil, errors.Errorf(`only "title" supports %q operator, but found %q`, celoverloads.Matches, variable)
				}
			default:
				return nil, errors.Errorf("unsupported function %v", functionName)
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
