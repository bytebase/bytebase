package common

import storepb "github.com/bytebase/bytebase/backend/generated-go/store"

// EnvironmentOrderMap returns the workspace display order keyed by environment ID.
func EnvironmentOrderMap(environments []*storepb.EnvironmentSetting_Environment) map[string]int {
	orderMap := make(map[string]int)
	for i, environment := range environments {
		orderMap[environment.Id] = i
	}
	return orderMap
}
