package store

import (
	"strings"

	"github.com/google/cel-go/cel"
	celast "github.com/google/cel-go/common/ast"
	celoperators "github.com/google/cel-go/common/operators"
	celoverloads "github.com/google/cel-go/common/overloads"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/qb"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// GetListUserFilterResult contains the filter query and optional project ID.
type GetListUserFilterResult struct {
	Query     *qb.Query
	ProjectID *string
}

// GetListUserFilter parses a CEL filter string and returns a query for filtering users.
func GetListUserFilter(filter string) (*GetListUserFilterResult, error) {
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
	var projectID *string

	convertToPrincipalType := func(v1UserType v1pb.UserType) (storepb.PrincipalType, error) {
		switch v1UserType {
		case v1pb.UserType_USER:
			return storepb.PrincipalType_END_USER, nil
		case v1pb.UserType_SYSTEM_BOT:
			return storepb.PrincipalType_SYSTEM_BOT, nil
		case v1pb.UserType_SERVICE_ACCOUNT:
			return storepb.PrincipalType_SERVICE_ACCOUNT, nil
		case v1pb.UserType_WORKLOAD_IDENTITY:
			return storepb.PrincipalType_WORKLOAD_IDENTITY, nil
		default:
			return storepb.PrincipalType_END_USER, errors.Errorf("invalid user type %s", v1UserType)
		}
	}

	parseToSQL := func(variable, value any) (*qb.Query, error) {
		switch variable {
		case "email":
			return qb.Q().Space("principal.email = ?", value.(string)), nil
		case "name":
			return qb.Q().Space("principal.name = ?", value.(string)), nil
		case "user_type":
			v1UserType, ok := v1pb.UserType_value[value.(string)]
			if !ok {
				return nil, errors.Errorf("invalid user type filter %q", value)
			}
			principalType, err := convertToPrincipalType(v1pb.UserType(v1UserType))
			if err != nil {
				return nil, errors.Errorf("failed to parse the user type %q with error: %v", v1UserType, err)
			}
			return qb.Q().Space("principal.type = ?", principalType.String()), nil
		case "state":
			stateStr, ok := value.(string)
			if !ok {
				return nil, errors.Errorf("state value must be string, got %T", value)
			}
			// Try with STATE_ prefix first (e.g., "STATE_ACTIVE", "STATE_DELETED")
			v1State, ok := v1pb.State_value[stateStr]
			if !ok {
				// If not found, try without STATE_ prefix (e.g., "ACTIVE", "DELETED")
				if v, exists := v1pb.State_value[strings.TrimPrefix(stateStr, "STATE_")]; exists {
					v1State = v
					ok = true
				}
			}
			if !ok {
				return nil, errors.Errorf("invalid state filter %q", value)
			}
			return qb.Q().Space("principal.deleted = ?", v1pb.State(v1State) == v1pb.State_DELETED), nil
		case "project":
			pid, err := common.GetProjectID(value.(string))
			if err != nil {
				return nil, errors.Errorf("invalid project filter %q", value)
			}
			projectID = &pid
			return qb.Q().Space("TRUE"), nil
		default:
			return nil, errors.Errorf("unsupport variable %q", variable)
		}
	}

	parseToUserTypeSQL := func(expr celast.Expr) (*qb.Query, error) {
		variable, value := getVariableAndValueFromExpr(expr)
		if variable != "user_type" {
			return nil, errors.Errorf(`only "user_type" support "user_type in [xx]"/"!(user_type in [xx])" operator`)
		}

		rawTypeList, ok := value.([]any)
		if !ok {
			return nil, errors.Errorf("invalid user_type value %q", value)
		}
		if len(rawTypeList) == 0 {
			return nil, errors.Errorf("empty user_type filter")
		}

		userTypeList := make([]any, 0, len(rawTypeList))
		for _, rawType := range rawTypeList {
			v1UserType, ok := v1pb.UserType_value[rawType.(string)]
			if !ok {
				return nil, errors.Errorf("invalid user type filter %q", rawType)
			}
			principalType, err := convertToPrincipalType(v1pb.UserType(v1UserType))
			if err != nil {
				return nil, errors.Errorf("failed to parse the user type %q with error: %v", v1UserType, err)
			}
			userTypeList = append(userTypeList, principalType.String())
		}

		return qb.Q().Space("principal.type = ANY(?)", userTypeList), nil
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
				if variable != "name" && variable != "email" {
					return nil, errors.Errorf(`only "name" and "email" support %q operator, but found %q`, celoverloads.Matches, variable)
				}
				strValue, ok := value.(string)
				if !ok {
					return nil, errors.Errorf("expect string, got %T, hint: filter literals should be string", value)
				}
				return qb.Q().Space("LOWER(principal."+variable+") LIKE ?", "%"+strings.ToLower(strValue)+"%"), nil
			case celoperators.In:
				return parseToUserTypeSQL(expr)
			case celoperators.LogicalNot:
				args := expr.AsCall().Args()
				if len(args) != 1 {
					return nil, errors.Errorf(`only support !(user_type in ["{type1}", "{type2}"]) format`)
				}
				qq, err := getFilter(args[0])
				if err != nil {
					return nil, err
				}
				return q.Space("(NOT (?))", qq), nil
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

	return &GetListUserFilterResult{
		Query:     qb.Q().Space("(?)", q),
		ProjectID: projectID,
	}, nil
}
