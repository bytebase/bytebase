package server

import (
	"encoding/json"
	"sort"

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
// The matrix will include the stage even if the stage has no database.
func getDatabaseMatrixFromDeploymentSchedule(schedule *api.DeploymentSchedule, baseDatabaseName, dbNameTemplate string, databaseList []*api.Database) ([][]*api.Database, error) {
	var matrix [][]*api.Database

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
			return nil, err
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
			labels := idToLabels[database.ID]

			if dbNameTemplate != "" {
				// The tenant database should match the database name if the template is not empty.
				name, err := formatDatabaseName(baseDatabaseName, dbNameTemplate, labels)
				if err != nil {
					continue
				}
				if database.Name != name {
					continue
				}
			}

			// Skip if the database is already in a stage.
			if _, ok := idsSeen[database.ID]; ok {
				continue
			}

			if isMatchExpressions(labels, deployment.Spec.Selector.MatchExpressions) {
				matchedDatabaseList = append(matchedDatabaseList, database.ID)
				idsSeen[database.ID] = true
			}
		}

		var databaseList []*api.Database
		for _, id := range matchedDatabaseList {
			databaseList = append(databaseList, databaseMap[id])
		}
		// sort databases in stage based on IDs.
		if len(databaseList) > 0 {
			sort.Slice(databaseList, func(i, j int) bool {
				return databaseList[i].ID < databaseList[j].ID
			})
		}

		matrix = append(matrix, databaseList)
	}

	return matrix, nil
}

// formatDatabaseName will return the full database name given the dbNameTemplate, base database name, and labels.
func formatDatabaseName(baseDatabaseName, dbNameTemplate string, labels map[string]string) (string, error) {
	if dbNameTemplate == "" {
		return baseDatabaseName, nil
	}
	tokens := make(map[string]string)
	tokens[api.DBNameToken] = baseDatabaseName
	for k, v := range labels {
		switch k {
		case api.LocationLabelKey:
			tokens[api.LocationToken] = v
		case api.TenantLabelKey:
			tokens[api.TenantToken] = v
		}
	}
	return api.FormatTemplate(dbNameTemplate, tokens)
}
