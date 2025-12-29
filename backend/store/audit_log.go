package store

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/cel-go/cel"
	celast "github.com/google/cel-go/common/ast"
	celoperators "github.com/google/cel-go/common/operators"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/qb"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

type AuditLog struct {
	ID        int
	CreatedAt time.Time
	Payload   *storepb.AuditLog
}

type AuditLogFind struct {
	Project     *string
	FilterQ     *qb.Query
	Limit       *int
	Offset      *int
	OrderByKeys []*OrderByKey
}

func (s *Store) CreateAuditLog(ctx context.Context, payload *storepb.AuditLog) error {
	p, err := protojson.Marshal(payload)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal payload")
	}

	q := qb.Q().Space("INSERT INTO audit_log (payload) VALUES (?)", p)
	sql, args, err := q.ToSQL()
	if err != nil {
		return errors.Wrapf(err, "failed to build sql")
	}

	if _, err := s.GetDB().ExecContext(ctx, sql, args...); err != nil {
		return errors.Wrapf(err, "failed to create audit log")
	}
	return nil
}

func (s *Store) SearchAuditLogs(ctx context.Context, find *AuditLogFind) ([]*AuditLog, error) {
	q := qb.Q().Space(`
		SELECT
			id,
			created_at,
			payload
		FROM audit_log
		WHERE TRUE
	`)

	if filterQ := find.FilterQ; filterQ != nil {
		q.And("?", filterQ)
	}
	if v := find.Project; v != nil {
		q.And("payload->>'parent' = ?", *v)
	}

	if len(find.OrderByKeys) > 0 {
		orderBy := []string{}
		for _, v := range find.OrderByKeys {
			orderBy = append(orderBy, fmt.Sprintf("%s %s", v.Key, v.SortOrder.String()))
		}
		q.Space(fmt.Sprintf("ORDER BY %s", strings.Join(orderBy, ", ")))
	} else {
		q.Space("ORDER BY id DESC")
	}
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

	rows, err := s.GetDB().QueryContext(ctx, sql, args...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to query context")
	}
	defer rows.Close()

	var logs []*AuditLog
	for rows.Next() {
		l := &AuditLog{
			Payload: &storepb.AuditLog{},
		}
		var payload []byte

		if err := rows.Scan(
			&l.ID,
			&l.CreatedAt,
			&payload,
		); err != nil {
			return nil, errors.Wrapf(err, "failed to scan")
		}

		if err := common.ProtojsonUnmarshaler.Unmarshal(payload, l.Payload); err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal payload")
		}

		logs = append(logs, l)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrapf(err, "failed to query")
	}

	return logs, nil
}

func GetSearchAuditLogsFilter(filter string) (*qb.Query, error) {
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
				variable, rawValue := getVariableAndValueFromExpr(expr)
				value, ok := rawValue.(string)
				if !ok {
					return nil, errors.Errorf("expect string, got %T, hint: filter literals should be string", rawValue)
				}
				switch variable {
				case "resource", "method", "user", "severity":
				default:
					return nil, errors.Errorf("unknown variable %s", variable)
				}
				return qb.Q().Space(fmt.Sprintf("payload->>'%s' = ?", variable), value), nil

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
					return qb.Q().Space("created_at >= ?", t), nil
				}
				return qb.Q().Space("created_at <= ?", t), nil

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

// ApplyRetentionFilter merges retention-based filtering with user-provided filters.
func ApplyRetentionFilter(userFilterQ *qb.Query, cutoff *time.Time) *qb.Query {
	if cutoff == nil {
		return userFilterQ
	}

	retentionQ := qb.Q().Space("created_at >= ?", *cutoff)
	if userFilterQ == nil {
		return qb.Q().Space("(?)", retentionQ)
	}

	// Combine with existing filter using AND
	q := qb.Q()
	q.Space("?", userFilterQ)
	q.And("?", retentionQ)
	return qb.Q().Space("(?)", q)
}

func GetAuditLogOrders(orderBy string) ([]*OrderByKey, error) {
	keys, err := parseOrderBy(orderBy)
	if err != nil {
		return nil, err
	}

	if len(keys) == 0 {
		return nil, nil
	}
	if len(keys) > 1 || keys[0].Key != "create_time" {
		return nil, errors.Errorf(`only support order by "create_time"`)
	}

	return []*OrderByKey{
		{
			Key:       "created_at",
			SortOrder: keys[0].SortOrder,
		},
	}, nil
}
