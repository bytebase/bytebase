package store

import (
	"context"
	"database/sql"
	"strings"

	"github.com/google/cel-go/cel"
	celast "github.com/google/cel-go/common/ast"
	celoperators "github.com/google/cel-go/common/operators"
	celoverloads "github.com/google/cel-go/common/overloads"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/qb"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// FindGroupMessage is the message for finding groups.
type FindGroupMessage struct {
	ID        *string
	Email     *string
	ProjectID *string
	FilterQ   *qb.Query

	Limit  *int
	Offset *int
}

// UpdateGroupMessage is the message to update a group.
type UpdateGroupMessage struct {
	// identifier
	ID string

	// payload
	Email       *string
	Title       *string
	Description *string
	Payload     *storepb.GroupPayload
}

// GroupMessage is the message for a group.
type GroupMessage struct {
	ID          string
	Email       string
	Title       string
	Description string
	Payload     *storepb.GroupPayload
}

// GetGroup gets a group.
func (s *Store) GetGroup(ctx context.Context, find *FindGroupMessage) (*GroupMessage, error) {
	if find.Email != nil {
		if v, ok := s.groupCache.Get(*find.Email); ok && s.enableCache {
			return v, nil
		}
	}

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	groups, err := s.ListGroups(ctx, find)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	if len(groups) == 0 {
		return nil, nil
	} else if len(groups) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: errors.Errorf("found %d groups with filter %+v, expect 1", len(groups), find)}
	}
	return groups[0], nil
}

// ListGroups list all groups.
func (s *Store) ListGroups(ctx context.Context, find *FindGroupMessage) ([]*GroupMessage, error) {
	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	with := qb.Q()
	from := qb.Q().Space("user_group")
	where := qb.Q().Space("TRUE")

	// Build CTE for project filtering if needed
	if v := find.ProjectID; v != nil {
		with.Space(`WITH all_members AS (
			SELECT
				jsonb_array_elements_text(jsonb_array_elements(policy.payload->'bindings')->'members') AS member,
				jsonb_array_elements(policy.payload->'bindings')->>'role' AS role
			FROM policy
			WHERE ((resource_type = ? AND resource = ?) OR resource_type = ?) AND type = ?
		),
		project_members AS (
			SELECT ARRAY_AGG(member) AS members FROM all_members WHERE role NOT LIKE 'roles/workspace%'
		)`, storepb.Policy_PROJECT.String(), "projects/"+*v, storepb.Policy_WORKSPACE.String(), storepb.Policy_IAM.String())
		from.Space(`INNER JOIN project_members ON (CONCAT('groups/', user_group.email) = ANY(project_members.members) OR ? = ANY(project_members.members))`, common.AllUsers)
	}

	if filterQ := find.FilterQ; filterQ != nil {
		where.And("?", filterQ)
	}
	if v := find.ID; v != nil {
		where.And("id = ?", *v)
	}
	if v := find.Email; v != nil {
		where.And("email = ?", *v)
	}

	q := qb.Q()
	if with.Len() > 0 {
		q.Space("?", with)
	}
	q.Space(`
		SELECT
			user_group.id,
			user_group.email,
			user_group.name,
			user_group.description,
			user_group.payload
		FROM ?
		WHERE ?
		ORDER BY email
	`, from, where)

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

	var groups []*GroupMessage
	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var group GroupMessage
		var email sql.NullString
		var payload []byte
		if err := rows.Scan(
			&group.ID,
			&email,
			&group.Title,
			&group.Description,
			&payload,
		); err != nil {
			return nil, err
		}
		if email.Valid {
			group.Email = email.String
		}
		groupPayload := storepb.GroupPayload{}
		if err := common.ProtojsonUnmarshaler.Unmarshal(payload, &groupPayload); err != nil {
			return nil, err
		}
		group.Payload = &groupPayload
		groups = append(groups, &group)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	for _, group := range groups {
		s.groupCache.Add(group.Email, group)
	}
	return groups, nil
}

// CreateGroup creates a group.
func (s *Store) CreateGroup(ctx context.Context, create *GroupMessage) (*GroupMessage, error) {
	if create.Payload == nil {
		create.Payload = &storepb.GroupPayload{}
	}

	payloadBytes, err := protojson.Marshal(create.Payload)
	if err != nil {
		return nil, err
	}

	var email *string
	if create.Email != "" {
		email = &create.Email
	}

	q := qb.Q().Space(`
		INSERT INTO user_group (
			id,
			email,
			name,
			description,
			payload
		) VALUES (COALESCE(NULLIF(?, ''), gen_random_uuid()::text), ?, ?, ?, ?)
		RETURNING id
	`, create.ID, email, create.Title, create.Description, payloadBytes)

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin tx")
	}
	defer tx.Rollback()

	if err := tx.QueryRowContext(ctx, query, args...).Scan(&create.ID); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrap(err, "failed to commit")
	}

	if create.Email != "" {
		s.groupCache.Add(create.Email, create)
	}
	return create, nil
}

// UpdateGroup updates a group.
func (s *Store) UpdateGroup(ctx context.Context, patch *UpdateGroupMessage) (*GroupMessage, error) {
	set := qb.Q()
	if v := patch.Email; v != nil {
		set.Comma("email = ?", *v)
	}
	if v := patch.Title; v != nil {
		set.Comma("name = ?", *v)
	}
	if v := patch.Description; v != nil {
		set.Comma("description = ?", *v)
	}
	if v := patch.Payload; v != nil {
		payload, err := protojson.Marshal(v)
		if err != nil {
			return nil, err
		}
		set.Comma("payload = ?", payload)
	}

	q := qb.Q().Space(`
		UPDATE user_group
		SET ?
		WHERE id = ?
		RETURNING
			id,
			email,
			name,
			description,
			payload
	`, set, patch.ID)

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	var group GroupMessage
	var payload []byte
	var email sql.NullString

	if err := tx.QueryRowContext(ctx, query, args...).Scan(
		&group.ID,
		&email,
		&group.Title,
		&group.Description,
		&payload,
	); err != nil {
		return nil, err
	}

	groupPayload := storepb.GroupPayload{}
	if err := common.ProtojsonUnmarshaler.Unmarshal(payload, &groupPayload); err != nil {
		return nil, err
	}
	group.Payload = &groupPayload
	if email.Valid {
		group.Email = email.String
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrap(err, "failed to commit transaction")
	}

	if group.Email != "" {
		s.groupCache.Add(group.Email, &group)
	}
	return &group, nil
}

// DeleteGroup deletes a group.
func (s *Store) DeleteGroup(ctx context.Context, id string) error {
	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	q := qb.Q().Space("DELETE FROM user_group WHERE id = ? RETURNING email", id)
	query, args, err := q.ToSQL()
	if err != nil {
		return errors.Wrapf(err, "failed to build sql")
	}

	var email sql.NullString
	if err := tx.QueryRowContext(ctx, query, args...).Scan(&email); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "failed to commit transaction")
	}

	if email.Valid && email.String != "" {
		s.groupCache.Remove(email.String)
	}
	return nil
}

func GetListGroupFilter(find *FindGroupMessage, filter string) (*qb.Query, error) {
	if filter == "" {
		return nil, nil
	}

	e, err := cel.NewEnv()
	if err != nil {
		return nil, errors.New("failed to create cel env")
	}
	ast, iss := e.Parse(filter)
	if iss != nil {
		return nil, errors.Errorf("failed to parse filter %v, error: %v", filter, iss.String())
	}

	var getFilter func(expr celast.Expr) (*qb.Query, error)

	parseToSQL := func(variable, value any) (*qb.Query, error) {
		switch variable {
		case "title":
			return qb.Q().Space("name = ?", value.(string)), nil
		case "email":
			return qb.Q().Space("email = ?", value.(string)), nil
		case "project":
			projectID, err := common.GetProjectID(value.(string))
			if err != nil {
				return nil, errors.Errorf("invalid project filter %q", value)
			}
			find.ProjectID = &projectID
			return qb.Q().Space("TRUE"), nil
		default:
			return nil, errors.Errorf("unsupport variable %q", variable)
		}
	}

	getFilter = func(expr celast.Expr) (*qb.Query, error) {
		q := qb.Q()
		switch expr.Kind() {
		case celast.CallKind:
			functionName := expr.AsCall().FunctionName()
			switch functionName {
			case celoperators.LogicalOr:
				for _, arg := range expr.AsCall().Args() {
					qq, err := getFilter(arg)
					if err != nil {
						return nil, err
					}
					q.Or("?", qq)
				}
				return qb.Q().Space("(?)", q), nil
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
				return parseToSQL(variable, value)
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
				if strValue == "" {
					return nil, errors.Errorf(`empty value for %q`, variable)
				}

				switch variable {
				case "title":
					return qb.Q().Space("LOWER(name) LIKE ?", "%"+strings.ToLower(strValue)+"%"), nil
				case "email":
					return qb.Q().Space("LOWER(email) LIKE ?", "%"+strings.ToLower(strValue)+"%"), nil
				default:
					return nil, errors.Errorf("unsupport variable %q", variable)
				}
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
