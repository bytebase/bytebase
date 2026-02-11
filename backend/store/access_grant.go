package store

import (
	"context"
	"strings"
	"time"

	"github.com/google/cel-go/cel"
	celast "github.com/google/cel-go/common/ast"
	celoperators "github.com/google/cel-go/common/operators"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/qb"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// AccessGrantMessage is the message for access_grant.
type AccessGrantMessage struct {
	ProjectID  string
	Creator    string
	Status     string
	ExpireTime time.Time
	Payload    *storepb.AccessGrantPayload
	// Output only.
	ID        string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// FindAccessGrantMessage is the message for finding access grants.
type FindAccessGrantMessage struct {
	ID        *string
	ProjectID *string
	Creator   *string
	Limit     *int
	Offset    *int

	FilterQ *qb.Query
}

// UpdateAccessGrantMessage is the message for updating an access grant.
type UpdateAccessGrantMessage struct {
	Status *string
}

// CreateAccessGrant creates a new access grant.
func (s *Store) CreateAccessGrant(ctx context.Context, create *AccessGrantMessage) (*AccessGrantMessage, error) {
	payload, err := protojson.Marshal(create.Payload)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal payload")
	}

	create.ID = uuid.NewString()

	q := qb.Q().Space(`
		INSERT INTO access_grant (
			id,
			project,
			creator,
			status,
			expire_time,
			payload
		) VALUES (
			?, ?, ?, ?, ?, ?
		) RETURNING created_at, updated_at
	`, create.ID, create.ProjectID, create.Creator, create.Status, create.ExpireTime, payload)

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	if err := s.GetDB().QueryRowContext(ctx, query, args...).Scan(&create.CreatedAt, &create.UpdatedAt); err != nil {
		return nil, errors.Wrapf(err, "failed to insert access grant")
	}

	return create, nil
}

// GetAccessGrant gets an access grant.
func (s *Store) GetAccessGrant(ctx context.Context, find *FindAccessGrantMessage) (*AccessGrantMessage, error) {
	grants, err := s.ListAccessGrants(ctx, find)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list access grants")
	}
	if len(grants) == 0 {
		return nil, nil
	}
	if len(grants) > 1 {
		return nil, errors.Errorf("expect to find one access grant, found %d", len(grants))
	}
	return grants[0], nil
}

// ListAccessGrants retrieves a list of access grants.
func (s *Store) ListAccessGrants(ctx context.Context, find *FindAccessGrantMessage) ([]*AccessGrantMessage, error) {
	q := qb.Q().Space(`
		SELECT
			id,
			project,
			creator,
			status,
			expire_time,
			payload,
			created_at,
			updated_at
		FROM access_grant
		WHERE TRUE
	`)

	if filterQ := find.FilterQ; filterQ != nil {
		q.And("?", filterQ)
	}
	if v := find.ID; v != nil {
		q.And("id = ?", *v)
	}
	if v := find.ProjectID; v != nil {
		q.And("project = ?", *v)
	}
	if v := find.Creator; v != nil {
		q.And("creator = ?", *v)
	}

	q.Space("ORDER BY created_at DESC")
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

	rows, err := s.GetDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to select access grants")
	}
	defer rows.Close()

	var grants []*AccessGrantMessage
	for rows.Next() {
		grant := AccessGrantMessage{
			Payload: &storepb.AccessGrantPayload{},
		}
		var payload []byte
		if err := rows.Scan(
			&grant.ID,
			&grant.ProjectID,
			&grant.Creator,
			&grant.Status,
			&grant.ExpireTime,
			&payload,
			&grant.CreatedAt,
			&grant.UpdatedAt,
		); err != nil {
			return nil, errors.Wrapf(err, "failed to scan access grant")
		}
		if err := common.ProtojsonUnmarshaler.Unmarshal(payload, grant.Payload); err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal payload")
		}
		grants = append(grants, &grant)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrapf(err, "failed to iterate access grants")
	}

	return grants, nil
}

// UpdateAccessGrant updates an existing access grant.
func (s *Store) UpdateAccessGrant(ctx context.Context, id string, update *UpdateAccessGrantMessage) (*AccessGrantMessage, error) {
	set := qb.Q()

	if v := update.Status; v != nil {
		set.Comma("status = ?", *v)
	}
	if set.Len() == 0 {
		return nil, errors.New("no update field provided")
	}
	set.Comma("updated_at = ?", time.Now())

	q := qb.Q().Space("UPDATE access_grant SET ?", set).Space(`WHERE id = ?
		RETURNING id, project, creator, status, expire_time, payload, created_at, updated_at`, id)

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	grant := AccessGrantMessage{
		Payload: &storepb.AccessGrantPayload{},
	}
	var payload []byte
	if err := s.GetDB().QueryRowContext(ctx, query, args...).Scan(
		&grant.ID,
		&grant.ProjectID,
		&grant.Creator,
		&grant.Status,
		&grant.ExpireTime,
		&payload,
		&grant.CreatedAt,
		&grant.UpdatedAt,
	); err != nil {
		return nil, errors.Wrapf(err, "failed to update access grant")
	}
	if err := common.ProtojsonUnmarshaler.Unmarshal(payload, grant.Payload); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal payload")
	}

	return &grant, nil
}

// GetListAccessGrantFilter parses a CEL filter expression into a query builder query for listing access grants.
func GetListAccessGrantFilter(filter string) (*qb.Query, error) {
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
					creatorStr, ok := value.(string)
					if !ok {
						return nil, errors.Errorf("creator value must be a string")
					}
					if !strings.HasPrefix(creatorStr, "users/") {
						return nil, errors.Errorf("creator must have format \"users/{email}\", got %q", creatorStr)
					}
					creatorEmail := strings.TrimPrefix(creatorStr, "users/")
					return qb.Q().Space("access_grant.creator = ?", creatorEmail), nil
				case "status":
					statusStr, ok := value.(string)
					if !ok {
						return nil, errors.Errorf("status value must be a string")
					}
					return qb.Q().Space("access_grant.status = ?", statusStr), nil
				case "issue":
					return nil, errors.Errorf("filtering by issue is not supported yet")
				default:
					return nil, errors.Errorf("unsupported variable %q", variable)
				}
			case celoperators.GreaterEquals, celoperators.LessEquals, celoperators.Greater, celoperators.Less:
				variable, rawValue := getVariableAndValueFromExpr(expr)
				value, ok := rawValue.(string)
				if !ok {
					return nil, errors.Errorf("expect string, got %T, hint: filter literals should be string", rawValue)
				}
				var column string
				switch variable {
				case "expire_time":
					column = "access_grant.expire_time"
				case "create_time":
					column = "access_grant.created_at"
				default:
					return nil, errors.Errorf("unsupported variable %q for comparison operator", variable)
				}
				t, err := time.Parse(time.RFC3339, value)
				if err != nil {
					return nil, errors.Errorf("failed to parse time %v, error: %v", value, err)
				}
				switch functionName {
				case celoperators.GreaterEquals:
					return qb.Q().Space(column+" >= ?", t), nil
				case celoperators.LessEquals:
					return qb.Q().Space(column+" <= ?", t), nil
				case celoperators.Greater:
					return qb.Q().Space(column+" > ?", t), nil
				default:
					return qb.Q().Space(column+" < ?", t), nil
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
