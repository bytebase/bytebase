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

// getDatabaseMatrixFromDeploymentSchedule gets a pipeline based on deployment schedule.
// The returned matrix doesn't include deployment with no matched database.
func getDatabaseMatrixFromDeploymentSchedule(schedule *api.DeploymentSchedule, name string, databaseList []*api.Database) ([]*api.Deployment, [][]*api.Database, error) {
	var pipeline [][]*api.Database
	var deployments []*api.Deployment

	// idToLabels maps databaseID -> label.Key -> label.Value
	idToLabels := make(map[int]map[string]string)
	databaseMap := make(map[int]*api.Database)
	for _, database := range databaseList {
		databaseMap[database.ID] = database
		if _, ok := idToLabels[database.ID]; !ok {
			idToLabels[database.ID] = make(map[string]string)
		}
		var labelList []*api.DatabaseLabel
		if err := json.Unmarshal([]byte(database.Labels), &labelList); err != nil {
			return nil, nil, err
		}
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
			// The tenant database should match the database name.
			if database.Name != name {
				continue
			}
			// Skip if the database is already in a stage.
			if _, ok := idsSeen[database.ID]; ok {
				continue
			}

			labels := idToLabels[database.ID]
			if isMatchExpressions(labels, deployment.Spec.Selector.MatchExpressions) {
				matchedDatabaseList = append(matchedDatabaseList, database.ID)
				idsSeen[database.ID] = true
			}
		}

		var stage []*api.Database
		for _, id := range matchedDatabaseList {
			stage = append(stage, databaseMap[id])
		}

		if len(stage) > 0 {
			pipeline = append(pipeline, stage)
			deployments = append(deployments, deployment)
		}
	}

	return deployments, pipeline, nil
}
