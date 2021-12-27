package server

import "github.com/bytebase/bytebase/api"

// checkMatchExpression checks whether a databases matches the query.
// keyValue stores the database's label key and value.
func checkMatchExpression(keyValue map[string]string, expression *api.LabelSelectorRequirement) bool {
	switch expression.Operator {
	case api.InOperatorType:
		value, ok := keyValue[expression.Key]
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
		_, ok := keyValue[expression.Key]
		return ok
	default:
		return false
	}
}

func checkMatchExpressions(keyValue map[string]string, expressionList []*api.LabelSelectorRequirement) bool {
	for _, expression := range expressionList {
		if !checkMatchExpression(keyValue, expression) {
			return false
		}
	}
	return true
}

// generatePipelineCreateFromDeploymentSchedule generates a pipeline, based on deployment schedule. It creates each stage and corresponding task list filled with database id. The caller is responsible to fill in other fields.
func generatePipelineCreateFromDeploymentSchedule(schedule *api.DeploymentSchedule, labelList []*api.DatabaseLabel) *api.PipelineCreate {
	create := &api.PipelineCreate{StageList: make([]api.StageCreate, len(schedule.Deployments))}

	databaseKeyValue := make(map[int]map[string]string)
	for _, label := range labelList {
		keyValue, ok := databaseKeyValue[label.DatabaseID]
		if !ok {
			databaseKeyValue[label.DatabaseID] = make(map[string]string)
			keyValue = databaseKeyValue[label.DatabaseID]
		}
		keyValue[label.Key] = label.Value
	}

	// For each stage, we loop over all databases to see if it is a match.
	// Databases matching the query in a stage should exclude all databases from previous stages. So we loop reversely and remove already matched databases in each stage.
	for i := len(schedule.Deployments) - 1; i >= 0; i-- {

		// For each stage, we will get a list of matched databases.
		databaseList := make([]int, 0)

		// Loop over all databases.
		for databaseID, keyValue := range databaseKeyValue {
			if checkMatchExpressions(keyValue, schedule.Deployments[i].Spec.Selector.MatchExpressions) {
				databaseList = append(databaseList, databaseID)
			}
		}

		create.StageList[i].TaskList = make([]api.TaskCreate, len(databaseList))

		for j := range databaseList {
			create.StageList[i].TaskList[j].DatabaseID = &databaseList[j]
			// Delete so that it won't show up in previous stages.
			delete(databaseKeyValue, databaseList[j])
		}
	}

	return create
}
