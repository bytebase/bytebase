package store

import (
	"context"
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
)

// AccessGrantMessage is the message for access_grant.
type AccessGrantMessage struct {
	ProjectID  string
	Creator    string
	Status     storepb.AccessGrant_Status
	ExpireTime time.Time
	Payload    *storepb.AccessGrantPayload
	// Reason is used as the issue description during creation.
	Reason string
	// Output only.
	ID        string
	IssueUID  int
	CreatedAt time.Time
	UpdatedAt time.Time
}

// FindAccessGrantMessage is the message for finding access grants.
type FindAccessGrantMessage struct {
	ID        *string
	ProjectID *string
	Creator   *string
	IssueUID  *int
	Limit     *int
	Offset    *int

	FilterQ *qb.Query
}

// UpdateAccessGrantMessage is the message for updating an access grant.
type UpdateAccessGrantMessage struct {
	Status *storepb.AccessGrant_Status
}

// CreateAccessGrant creates a new access grant and its associated issue in a transaction.
func (s *Store) CreateAccessGrant(ctx context.Context, create *AccessGrantMessage) (*AccessGrantMessage, error) {
	payload, err := protojson.Marshal(create.Payload)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal payload")
	}

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	// Step 1: Insert access grant.
	insertGrantQ := qb.Q().Space(`
		INSERT INTO access_grant (
			project,
			creator,
			status,
			expire_time,
			payload
		) VALUES (
			?, ?, ?, ?, ?
		) RETURNING id, created_at, updated_at
	`, create.ProjectID, create.Creator, create.Status.String(), create.ExpireTime, payload)

	query, args, err := insertGrantQ.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}
	if err := tx.QueryRowContext(ctx, query, args...).Scan(&create.ID, &create.CreatedAt, &create.UpdatedAt); err != nil {
		return nil, errors.Wrapf(err, "failed to insert access grant")
	}

	// Step 2: Create the associated issue.
	issuePayload, err := protojson.Marshal(&storepb.Issue{
		Approval: &storepb.IssuePayloadApproval{
			ApprovalFindingDone: false,
			ApprovalTemplate:    nil,
			Approvers:           nil,
		},
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal issue payload")
	}
	title := fmt.Sprintf("JIT access request by %s", create.Creator)
	tsVector := getTSVector(fmt.Sprintf("%s %s", title, create.Reason))
	insertIssueQ := qb.Q().Space(`
		INSERT INTO issue (
			creator,
			project,
			name,
			status,
			type,
			description,
			payload,
			ts_vector
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING id
	`, create.Creator, create.ProjectID, title, storepb.Issue_OPEN.String(), storepb.Issue_ACCESS_GRANT.String(), create.Reason, issuePayload, tsVector)

	query, args, err = insertIssueQ.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build issue sql")
	}
	if err := tx.QueryRowContext(ctx, query, args...).Scan(&create.IssueUID); err != nil {
		return nil, errors.Wrapf(err, "failed to insert issue")
	}

	// Step 3: Update access grant payload with the issue ID.
	create.Payload.IssueId = int64(create.IssueUID)
	updatedPayload, err := protojson.Marshal(create.Payload)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal updated payload")
	}
	updateQ := qb.Q().Space(`UPDATE access_grant SET payload = ? WHERE id = ?`, updatedPayload, create.ID)
	query, args, err = updateQ.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build update sql")
	}
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return nil, errors.Wrapf(err, "failed to update access grant payload")
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit transaction")
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
	if v := find.IssueUID; v != nil {
		q.And("(payload->>'issueId')::bigint = ?", *v)
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
		var statusString string
		if err := rows.Scan(
			&grant.ID,
			&grant.ProjectID,
			&grant.Creator,
			&statusString,
			&grant.ExpireTime,
			&payload,
			&grant.CreatedAt,
			&grant.UpdatedAt,
		); err != nil {
			return nil, errors.Wrapf(err, "failed to scan access grant")
		}
		statusValue, ok := storepb.AccessGrant_Status_value[statusString]
		if !ok {
			return nil, errors.Errorf("invalid access grant status %q", statusString)
		}
		grant.Status = storepb.AccessGrant_Status(statusValue)
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
		set.Comma("status = ?", v.String())
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
	var statusString string
	if err := s.GetDB().QueryRowContext(ctx, query, args...).Scan(
		&grant.ID,
		&grant.ProjectID,
		&grant.Creator,
		&statusString,
		&grant.ExpireTime,
		&payload,
		&grant.CreatedAt,
		&grant.UpdatedAt,
	); err != nil {
		return nil, errors.Wrapf(err, "failed to update access grant")
	}
	statusValue, ok := storepb.AccessGrant_Status_value[statusString]
	if !ok {
		return nil, errors.Errorf("invalid access grant status %q", statusString)
	}
	grant.Status = storepb.AccessGrant_Status(statusValue)
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
				case "name":
					nameStr, ok := value.(string)
					if !ok {
						return nil, errors.Errorf("name value must be a string")
					}
					_, accessGrantID, err := common.GetProjectIDAccessGrantID(nameStr)
					if err != nil {
						return nil, errors.Wrapf(err, "invalid access grant name %q", nameStr)
					}
					return qb.Q().Space("access_grant.id = ?", accessGrantID), nil
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
				case "query":
					queryStr, ok := value.(string)
					if !ok {
						return nil, errors.Errorf("query value must be a string")
					}
					return qb.Q().Space("access_grant.payload->>'query' = ?", queryStr), nil
				case "issue":
					issueStr, ok := value.(string)
					if !ok {
						return nil, errors.Errorf("issue value must be a string")
					}
					_, issueUID, err := common.GetProjectIDIssueUID(issueStr)
					if err != nil {
						return nil, errors.Wrapf(err, "invalid issue name %q", issueStr)
					}
					return qb.Q().Space("(access_grant.payload->>'issueId')::bigint = ?", issueUID), nil
				default:
					return nil, errors.Errorf("unsupported variable %q", variable)
				}
			case celoverloads.Matches:
				call := expr.AsCall()
				target := call.Target()
				if target.Kind() != celast.IdentKind {
					return nil, errors.Errorf("contains target must be an identifier")
				}
				variable := target.AsIdent()
				if variable != "query" {
					return nil, errors.Errorf("contains is not supported on field %q", variable)
				}
				args := call.Args()
				if len(args) != 1 || args[0].Kind() != celast.LiteralKind {
					return nil, errors.Errorf("contains requires a single string literal argument")
				}
				value, ok := args[0].AsLiteral().Value().(string)
				if !ok {
					return nil, errors.Errorf("contains argument must be a string")
				}
				return qb.Q().Space("access_grant.payload->>'query' ILIKE ?", "%"+value+"%"), nil
			case celoperators.In:
				variable, value := getVariableAndValueFromExpr(expr)
				if variable != "status" {
					return nil, errors.Errorf("unsupported variable %q for \"in\" operator", variable)
				}
				rawList, ok := value.([]any)
				if !ok {
					return nil, errors.Errorf("invalid list value %q for %v", value, variable)
				}
				if len(rawList) == 0 {
					return nil, errors.Errorf("empty list value for filter %v", variable)
				}
				var statuses []string
				for _, raw := range rawList {
					s, ok := raw.(string)
					if !ok {
						return nil, errors.Errorf("status value must be a string")
					}
					statuses = append(statuses, s)
				}
				return qb.Q().Space("access_grant.status = ANY(?)", statuses), nil
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
