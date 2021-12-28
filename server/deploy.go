package server

import "github.com/bytebase/bytebase/api"

// isMatchExpression checks whether a databases matches the query.
// labels is a mapping from database label key to value.
func isMatchExpression(labels map[string]string, expression *api.LabelSelectorRequirement) bool {
	switch expression.Operator {
	case api.InOperatorType:
		value, ok := labels[expression.Key]
		if !ok {
			return false
		}
		for _, exprValue := range expression.Values {
			if exprValue == value {
				return true
			}
		}
		return false
	case api.ExistsOperatorType:
		_, ok := labels[expression.Key]
		return ok
	default:
		return false
	}
}

func isMatchExpressions(labels map[string]string, expressionList []*api.LabelSelectorRequirement) bool {
	// Empty expression list matches no databases.
	if len(expressionList) == 0 {
		return false
	}
	// Expressions are ANDed.
	for _, expression := range expressionList {
		if !isMatchExpression(labels, expression) {
			return false
		}
	}
	return true
}

// generatePipelineCreateFromDeploymentSchedule generates a pipeline, based on deployment schedule. It creates each stage and corresponding task list filled with database id. The caller is responsible to fill in other fields.
func generatePipelineCreateFromDeploymentSchedule(schedule *api.DeploymentSchedule, labelList []*api.DatabaseLabel) *api.PipelineCreate {
	create := &api.PipelineCreate{}

	// idToLabels maps databaseID -> label.Key -> label.Value
	idToLabels := make(map[int]map[string]string)
	for _, label := range labelList {
		if _, ok := idToLabels[label.DatabaseID]; !ok {
			idToLabels[label.DatabaseID] = make(map[string]string)
		}
		idToLabels[label.DatabaseID][label.Key] = label.Value
	}

	// idsSeen records database id which is already in a stage.
	idsSeen := make(map[int]bool)

	// For each stage, we loop over all databases to see if it is a match.
	for _, deployment := range schedule.Deployments {

		// For each stage, we will get a list of matched databases.
		var databaseList []int

		// Loop over all databases.
		for databaseID, labels := range idToLabels {
			// Skip if database is already in a stage.
			if idsSeen[databaseID] {
				continue
			}
			if isMatchExpressions(labels, deployment.Spec.Selector.MatchExpressions) {
				databaseList = append(databaseList, databaseID)
			}
		}

		stageCreate := api.StageCreate{}

		for _, id := range databaseList {
			databaseID := id
			stageCreate.TaskList = append(stageCreate.TaskList, api.TaskCreate{DatabaseID: &databaseID})
			idsSeen[id] = true
		}

		create.StageList = append(create.StageList, stageCreate)
	}

	return create
}
