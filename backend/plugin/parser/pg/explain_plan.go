package pg

import (
	"encoding/json"
	"math"

	"github.com/pkg/errors"
)

type explainPlanNode struct {
	NodeType           string            `json:"Node Type"`
	ParentRelationship string            `json:"Parent Relationship"`
	PlanRows           *float64          `json:"Plan Rows"`
	Plans              []explainPlanNode `json:"Plans"`
}

// GetEstimatedAffectedRowsFromExplainJSON returns the planner's estimate of the rows modified
// by the INSERT, UPDATE, DELETE, and MERGE operations in `EXPLAIN (FORMAT JSON)` output. It
// sums every ModifyTable node, including those of data-modifying CTEs, and returns an error
// when the plan has none or one cannot be interpreted.
func GetEstimatedAffectedRowsFromExplainJSON(plan string) (int64, error) {
	var statements []struct {
		Plan *explainPlanNode `json:"Plan"`
	}
	if err := json.Unmarshal([]byte(plan), &statements); err != nil {
		return 0, errors.Wrap(err, "failed to parse the EXPLAIN (FORMAT JSON) output")
	}

	var total float64
	modifyTableCount := 0
	var walk func(node *explainPlanNode) error
	walk = func(node *explainPlanNode) error {
		if node.NodeType == "ModifyTable" {
			rows, err := getModifiedRows(node)
			if err != nil {
				return err
			}
			total += rows
			modifyTableCount++
		}
		for i := range node.Plans {
			if err := walk(&node.Plans[i]); err != nil {
				return err
			}
		}
		return nil
	}
	for _, statement := range statements {
		if statement.Plan == nil {
			continue
		}
		if err := walk(statement.Plan); err != nil {
			return 0, err
		}
	}
	if modifyTableCount == 0 {
		return 0, errors.New("the plan has no ModifyTable node")
	}
	return int64(math.Round(total)), nil
}

// getModifiedRows reads the estimate from the subplans feeding a ModifyTable node. PostgreSQL 14
// and later label its only subplan "Outer"; earlier versions label one "Member" per target table.
func getModifiedRows(modifyTable *explainPlanNode) (float64, error) {
	var rows float64
	found := false
	for _, child := range modifyTable.Plans {
		switch child.ParentRelationship {
		case "Outer", "Member":
			if child.PlanRows == nil {
				return 0, errors.Errorf("the %s node feeding ModifyTable has no Plan Rows", child.NodeType)
			}
			rows += *child.PlanRows
			found = true
		default:
		}
	}
	if !found {
		return 0, errors.New("the ModifyTable node has no subplan feeding it")
	}
	return rows, nil
}
