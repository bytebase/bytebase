package store

import (
	"context"
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

type QueryHistoryType string

const (
	// QueryHistoryTypeQuery is the type for query.
	QueryHistoryTypeQuery QueryHistoryType = "QUERY"
	// QueryHistoryTypeExport is the type for export.
	QueryHistoryTypeExport QueryHistoryType = "EXPORT"
)

// QueryHistoryMessage is the API message for query history.
type QueryHistoryMessage struct {
	// Output only fields
	UID       int
	CreatedAt time.Time

	// Related fields
	Creator string
	// ProjectID is the project resource id.
	ProjectID string

	// Domain specific fields
	Type      QueryHistoryType
	Statement string
	Payload   *storepb.QueryHistoryPayload
	// Database is the database resource name, like instances/{instance}/databases/{database}
	Database string
}

// FindQueryHistoryMessage is the API message for finding query histories.
type FindQueryHistoryMessage struct {
	Creator   *string
	ProjectID *string
	// Instance is the instance resource name like instances/{instance}.
	Instance *string
	// Database is database resource name like instances/{instance}/databases/{database}.
	Database *string
	Type     *QueryHistoryType

	Limit   *int
	Offset  *int
	FilterQ *qb.Query
}

// CreateQueryHistory creates the query history.
func (s *Store) CreateQueryHistory(ctx context.Context, create *QueryHistoryMessage) (*QueryHistoryMessage, error) {
	payload, err := protojson.Marshal(create.Payload)
	if err != nil {
		return nil, err
	}

	q := qb.Q().Space(`
		INSERT INTO query_history (
			creator,
			project_id,
			database,
			statement,
			type,
			payload
		)
		VALUES (?, ?, ?, ?, ?, ?)
		RETURNING
			id,
			created_at
	`,
		create.Creator,
		create.ProjectID,
		create.Database,
		create.Statement,
		create.Type,
		payload,
	)

	sql, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if err := tx.QueryRowContext(ctx, sql, args...).Scan(
		&create.UID,
		&create.CreatedAt,
	); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit transaction")
	}

	return create, nil
}

// ListQueryHistories lists the query history.
func (s *Store) ListQueryHistories(ctx context.Context, find *FindQueryHistoryMessage) ([]*QueryHistoryMessage, error) {
	q := qb.Q().Space(`
		SELECT
			query_history.id,
			query_history.creator,
			query_history.created_at,
			query_history.project_id,
			query_history.database,
			query_history.statement,
			query_history.type,
			query_history.payload
		FROM query_history
		WHERE TRUE
	`)

	if filterQ := find.FilterQ; filterQ != nil {
		q.And("?", filterQ)
	}
	if v := find.Creator; v != nil {
		q.And("query_history.creator = ?", *v)
	}
	if v := find.ProjectID; v != nil {
		q.And("query_history.project_id = ?", *v)
	}
	if v := find.Instance; v != nil {
		q.And("query_history.database LIKE ?", *v)
	} else if v := find.Database; v != nil {
		q.And("query_history.database = ?", *v)
	}
	if v := find.Type; v != nil {
		q.And("query_history.type = ?", *v)
	}

	q.Space("ORDER BY id DESC")
	if v := find.Limit; v != nil {
		q.Space("LIMIT ?", *v)
	}
	if v := find.Offset; v != nil {
		q.Space("OFFSET ?", *v)
	}

	sql, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var queryHistories []*QueryHistoryMessage
	rows, err := tx.QueryContext(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		queryHistory := &QueryHistoryMessage{}
		var payloadStr string
		if err := rows.Scan(
			&queryHistory.UID,
			&queryHistory.Creator,
			&queryHistory.CreatedAt,
			&queryHistory.ProjectID,
			&queryHistory.Database,
			&queryHistory.Statement,
			&queryHistory.Type,
			&payloadStr,
		); err != nil {
			return nil, err
		}

		var payload storepb.QueryHistoryPayload
		if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(payloadStr), &payload); err != nil {
			return nil, err
		}
		queryHistory.Payload = &payload

		queryHistories = append(queryHistories, queryHistory)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return queryHistories, nil
}

func GetListQueryHistoryFilter(filter string) (*qb.Query, error) {
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

	parseToSQL := func(variable, value any) (*qb.Query, error) {
		switch variable {
		case "project":
			projectID, err := common.GetProjectID(value.(string))
			if err != nil {
				return nil, errors.Errorf("invalid project filter %q", value)
			}
			return qb.Q().Space("query_history.project_id = ?", projectID), nil
		case "database":
			return qb.Q().Space("query_history.database = ?", value.(string)), nil
		case "instance":
			return qb.Q().Space("query_history.database LIKE ?", value.(string)), nil
		case "type":
			historyType := QueryHistoryType(value.(string))
			return qb.Q().Space("query_history.type = ?", historyType), nil
		case "statement":
			return qb.Q().Space("query_history.statement LIKE ?", value), nil
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
				if variable != "statement" {
					return nil, errors.Errorf(`only "statement" support %q operator, but found %q`, celoverloads.Matches, variable)
				}
				strValue, ok := value.(string)
				if !ok {
					return nil, errors.Errorf("expect string, got %T, hint: filter literals should be string", value)
				}
				return parseToSQL("statement", "%"+strValue+"%")
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
