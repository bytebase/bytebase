package store

import (
	"time"

	"github.com/google/cel-go/cel"
	celast "github.com/google/cel-go/common/ast"
	celoperators "github.com/google/cel-go/common/operators"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common/qb"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// GetListRolloutFilter parses a CEL filter expression into a query builder query for listing rollouts.
func GetListRolloutFilter(filter string) (*qb.Query, error) {
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
			case celoperators.In:
				variable, value := getVariableAndValueFromExpr(expr)
				switch variable {
				case "task_type":
					rawList, ok := value.([]any)
					if !ok {
						return nil, errors.Errorf("invalid list value %q for %v", value, variable)
					}
					if len(rawList) == 0 {
						return nil, errors.Errorf("empty list value for filter %v", variable)
					}
					var taskTypes []string
					for _, raw := range rawList {
						taskType, ok := raw.(string)
						if !ok {
							return nil, errors.Errorf("task_type value must be a string")
						}
						if _, ok := v1pb.Task_Type_value[taskType]; !ok {
							return nil, errors.Errorf("invalid task_type value: %s", taskType)
						}
						v1TaskType := v1pb.Task_Type(v1pb.Task_Type_value[taskType])
						storeTaskType := convertV1ToStoreTaskType(v1TaskType)
						taskTypes = append(taskTypes, storeTaskType.String())
					}
					return qb.Q().Space("EXISTS (SELECT 1 FROM task WHERE task.plan_id = plan.id AND task.type = ANY(?))", taskTypes), nil
				default:
					return nil, errors.Errorf("unsupported variable %q", variable)
				}
			case celoperators.GreaterEquals, celoperators.LessEquals:
				variable, rawValue := getVariableAndValueFromExpr(expr)
				value, ok := rawValue.(string)
				if !ok {
					return nil, errors.Errorf("expect string, got %T, hint: filter literals should be string", rawValue)
				}
				if variable != "update_time" {
					return nil, errors.Errorf(`">=" and "<=" are only supported for "update_time"`)
				}
				t, err := time.Parse(time.RFC3339, value)
				if err != nil {
					return nil, errors.Errorf("failed to parse time %v, error: %v", value, err)
				}
				updatedAtSubquery := `COALESCE((SELECT MAX(task_run.updated_at) FROM task JOIN task_run ON task_run.task_id = task.id WHERE task.plan_id = plan.id), plan.created_at)`
				if functionName == celoperators.GreaterEquals {
					return qb.Q().Space(updatedAtSubquery+" >= ?", t), nil
				}
				return qb.Q().Space(updatedAtSubquery+" <= ?", t), nil
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

func convertV1ToStoreTaskType(taskType v1pb.Task_Type) storepb.Task_Type {
	switch taskType {
	case v1pb.Task_DATABASE_CREATE:
		return storepb.Task_DATABASE_CREATE
	case v1pb.Task_DATABASE_MIGRATE:
		return storepb.Task_DATABASE_MIGRATE
	case v1pb.Task_DATABASE_EXPORT:
		return storepb.Task_DATABASE_EXPORT
	case v1pb.Task_TYPE_UNSPECIFIED, v1pb.Task_GENERAL:
		return storepb.Task_TASK_TYPE_UNSPECIFIED
	default:
		return storepb.Task_TASK_TYPE_UNSPECIFIED
	}
}
