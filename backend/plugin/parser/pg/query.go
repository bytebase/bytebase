package pg

import (
	"encoding/json"
	"sort"

	pgquery "github.com/pganalyze/pg_query_go/v4"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func ExtractPostgresResourceList(currentDatabase string, currentSchema string, sql string) ([]base.SchemaResource, error) {
	jsonText, err := pgquery.ParseToJSON(sql)
	if err != nil {
		return nil, err
	}

	var jsonData map[string]any

	if err := json.Unmarshal([]byte(jsonText), &jsonData); err != nil {
		return nil, err
	}

	resourceMap := make(map[string]base.SchemaResource)
	list := extractRangeVarFromJSON(currentDatabase, currentSchema, jsonData)
	for _, resource := range list {
		resourceMap[resource.String()] = resource
	}
	list = []base.SchemaResource{}
	for _, resource := range resourceMap {
		list = append(list, resource)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].String() < list[j].String()
	})
	return list, nil
}

func extractRangeVarFromJSON(currentDatabase string, currentSchema string, jsonData map[string]any) []base.SchemaResource {
	var result []base.SchemaResource
	if jsonData["RangeVar"] != nil {
		resource := base.SchemaResource{
			Database: currentDatabase,
			Schema:   currentSchema,
		}
		rangeVar := jsonData["RangeVar"].(map[string]any)
		if rangeVar["schemaname"] != nil {
			resource.Schema = rangeVar["schemaname"].(string)
		}
		if rangeVar["relname"] != nil {
			resource.Table = rangeVar["relname"].(string)
		}
		result = append(result, resource)
	}

	for _, value := range jsonData {
		switch v := value.(type) {
		case map[string]any:
			result = append(result, extractRangeVarFromJSON(currentDatabase, currentSchema, v)...)
		case []any:
			for _, item := range v {
				if m, ok := item.(map[string]any); ok {
					result = append(result, extractRangeVarFromJSON(currentDatabase, currentSchema, m)...)
				}
			}
		}
	}

	return result
}
