package server

import (
	"encoding/json"

	"github.com/bytebase/bytebase/api"
)

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
func generatePipelineCreateFromDeploymentSchedule(schedule *api.DeploymentSchedule, databaseList []*api.Database) *api.PipelineCreate {
	create := &api.PipelineCreate{}

	// idToLabels maps databaseID -> label.Key -> label.Value
	idToLabels := make(map[int]map[string]string)
	for _, database := range databaseList {
		if _, ok := idToLabels[database.ID]; !ok {
			idToLabels[database.ID] = make(map[string]string)
		}
		var labelList []*api.DatabaseLabel
		json.Unmarshal([]byte(database.Labels), &labelList)
		for _, label := range labelList {
			idToLabels[database.ID][label.Key] = label.Value
		}
	}

	// idsSeen records database id which is already in a stage.
	idsSeen := make(map[int]bool)

	// For each stage, we loop over all databases to see if it is a match.
	for _, deployment := range schedule.Deployments {

		// For each stage, we will get a list of matched databases.
		var matchedDatabaseList []int

		// Loop over databaseList instead of idToLabels to get determinant results.
		for _, database := range databaseList {
			databaseID := database.ID
			labels := idToLabels[databaseID]

			// Skip if the database is already in a stage.
			if idsSeen[databaseID] {
				continue
			}
			if isMatchExpressions(labels, deployment.Spec.Selector.MatchExpressions) {
				matchedDatabaseList = append(matchedDatabaseList, databaseID)
			}
		}

		stageCreate := api.StageCreate{}

		for _, id := range matchedDatabaseList {
			databaseID := id
			stageCreate.TaskList = append(stageCreate.TaskList, api.TaskCreate{DatabaseID: &databaseID})
			idsSeen[id] = true
		}

		create.StageList = append(create.StageList, stageCreate)
	}

	return create
}
