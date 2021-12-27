package server

import "github.com/bytebase/bytebase/api"

// checkMatchExpression checks whether a databases matches the query.
// keyValue stores the database's label key and value.
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
	for _, expression := range expressionList {
		if !isMatchExpression(labels, expression) {
			return false
		}
	}
	return true
}

// generatePipelineCreateFromDeploymentSchedule generates a pipeline, based on deployment schedule. It creates each stage and corresponding task list filled with database id. The caller is responsible to fill in other fields.
func generatePipelineCreateFromDeploymentSchedule(schedule *api.DeploymentSchedule, labelList []*api.DatabaseLabel) *api.PipelineCreate {
	create := &api.PipelineCreate{StageList: make([]api.StageCreate, 0, len(schedule.Deployments))}

	// idToLabels maps databaseID -> label.Key -> label.Value
	idToLabels := make(map[int]map[string]string)
	for _, label := range labelList {
		_, ok := idToLabels[label.DatabaseID]
		if !ok {
			idToLabels[label.DatabaseID] = make(map[string]string)
		}
		idToLabels[label.DatabaseID][label.Key] = label.Value
	}

	// idInStage stores database id which is already in a stage.
	idInStage := make(map[int]bool)

	// For each stage, we loop over all databases to see if it is a match.
	for _, deployment := range schedule.Deployments {

		// For each stage, we will get a list of matched databases.
		databaseList := make([]int, 0)

		// Loop over all databases.
		for databaseID, labels := range idToLabels {
			// Skip if database is already in a stage.
			if idInStage[databaseID] {
				continue
			}
			if isMatchExpressions(labels, deployment.Spec.Selector.MatchExpressions) {
				databaseList = append(databaseList, databaseID)
			}
		}

		taskList := make([]api.TaskCreate, 0, len(databaseList))

		for _, id := range databaseList {
			databaseID := id
			taskList = append(taskList, api.TaskCreate{DatabaseID: &databaseID})
			idInStage[id] = true
		}

		stage := api.StageCreate{TaskList: taskList}
		create.StageList = append(create.StageList, stage)
	}

	return create
}
