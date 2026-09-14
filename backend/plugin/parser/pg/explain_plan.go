package pg

import (
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
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
	roots, err := parseExplainJSON(plan)
	if err != nil {
		return 0, err
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
	for _, root := range roots {
		if err := walk(root); err != nil {
			return 0, err
		}
	}
	if modifyTableCount == 0 {
		return 0, errors.New("the plan has no ModifyTable node")
	}
	return common.RoundRows(total), nil
}

// GetEstimatedInsertedRowsFromExplainJSON returns the planner's estimate of the rows a top-level
// INSERT adds in `EXPLAIN (FORMAT JSON)` output, without the rows its data-modifying CTEs change.
func GetEstimatedInsertedRowsFromExplainJSON(plan string) (int64, error) {
	roots, err := parseExplainJSON(plan)
	if err != nil {
		return 0, err
	}
	if len(roots) != 1 || roots[0].NodeType != "ModifyTable" {
		return 0, errors.New("the plan root is not a ModifyTable node")
	}
	rows, err := getModifiedRows(roots[0])
	if err != nil {
		return 0, err
	}
	return common.RoundRows(rows), nil
}

// parseExplainJSON returns the root plan node of each statement in `EXPLAIN (FORMAT JSON)` output.
func parseExplainJSON(plan string) ([]*explainPlanNode, error) {
	var statements []struct {
		Plan *explainPlanNode `json:"Plan"`
	}
	if err := json.Unmarshal([]byte(plan), &statements); err != nil {
		return nil, errors.Wrap(err, "failed to parse the EXPLAIN (FORMAT JSON) output")
	}
	var roots []*explainPlanNode
	for _, statement := range statements {
		if statement.Plan != nil {
			roots = append(roots, statement.Plan)
		}
	}
	return roots, nil
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
