package pg

import (
	"encoding/json"
	"log/slog"
	"regexp"
	"sort"
	"strings"

	pgquery "github.com/pganalyze/pg_query_go/v4"
	pgparser "github.com/pganalyze/pg_query_go/v4/parser"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/parser/sql/ast"
	pgrawparser "github.com/bytebase/bytebase/backend/plugin/parser/sql/engine/pg"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	base.RegisterQueryValidator(storepb.Engine_POSTGRES, validateQuery)
	base.RegisterQueryValidator(storepb.Engine_REDSHIFT, validateQuery)
	base.RegisterQueryValidator(storepb.Engine_RISINGWAVE, validateQuery)
	base.RegisterExtractResourceListFunc(storepb.Engine_POSTGRES, ExtractResourceList)
	base.RegisterExtractResourceListFunc(storepb.Engine_REDSHIFT, ExtractResourceList)
	base.RegisterExtractResourceListFunc(storepb.Engine_RISINGWAVE, ExtractResourceList)
}

// validateQuery validates the SQL statement for SQL editor.
// Consider that the tokenizer cannot handle the dollar-sign($), so that we use pg_query_go to parse the statement.
// For EXPLAIN and normal SELECT statements, we can directly use regexp to check.
// For CTE, we need to parse the statement to JSON and check the JSON keys.
func validateQuery(statement string) (bool, error) {
	stmtList, err := pgrawparser.Parse(pgrawparser.ParseContext{}, statement)
	if err != nil {
		return false, convertToSyntaxError(statement, err)
	}
	for _, stmt := range stmtList {
		switch stmt.(type) {
		case *ast.SelectStmt, *ast.ExplainStmt:
		default:
			return false, nil
		}
	}

	// TODO(d): figure out whether this is still needed.
	jsonText, err := pgquery.ParseToJSON(statement)
	if err != nil {
		slog.Debug("Failed to parse statement to JSON", slog.String("statement", statement), log.BBError(err))
		return false, err
	}

	formattedStr := strings.ToUpper(strings.TrimSpace(statement))
	if isSelect, _ := regexp.MatchString(`^SELECT\s+?`, formattedStr); isSelect {
		return true, nil
	}

	if isSelect, _ := regexp.MatchString(`^SELECT\*\s+?`, formattedStr); isSelect {
		return true, nil
	}

	if isExplain, _ := regexp.MatchString(`^EXPLAIN\s+?`, formattedStr); isExplain {
		if isExplainAnalyze, _ := regexp.MatchString(`^EXPLAIN\s+ANALYZE\s+?`, formattedStr); isExplainAnalyze {
			return false, nil
		}
		return true, nil
	}

	cteRegex := regexp.MustCompile(`^WITH\s+?`)
	if matchResult := cteRegex.MatchString(formattedStr); matchResult {
		var jsonData map[string]any

		if err := json.Unmarshal([]byte(jsonText), &jsonData); err != nil {
			slog.Debug("Failed to unmarshal JSON", slog.String("jsonText", jsonText), log.BBError(err))
			return false, err
		}

		dmlKeyList := []string{"InsertStmt", "UpdateStmt", "DeleteStmt"}

		return !keyExistsInJSONData(jsonData, dmlKeyList), nil
	}

	return false, nil
}

func keyExistsInJSONData(jsonData map[string]any, keyList []string) bool {
	for _, key := range keyList {
		if _, ok := jsonData[key]; ok {
			return true
		}
	}

	for _, value := range jsonData {
		switch v := value.(type) {
		case map[string]any:
			if keyExistsInJSONData(v, keyList) {
				return true
			}
		case []any:
			for _, item := range v {
				if m, ok := item.(map[string]any); ok {
					if keyExistsInJSONData(m, keyList) {
						return true
					}
				}
			}
		}
	}

	return false
}

func ExtractResourceList(currentDatabase string, currentSchema string, sql string) ([]base.SchemaResource, error) {
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

func convertToSyntaxError(statement string, err error) *base.SyntaxError {
	if pgErr, ok := err.(*pgparser.Error); ok {
		line, column := getLineAndColumn(statement, pgErr.Cursorpos)
		return &base.SyntaxError{
			Line:    line,
			Column:  column,
			Message: pgErr.Message,
		}
	}

	return &base.SyntaxError{
		Line:    1,
		Column:  0,
		Message: err.Error(),
	}
}

func getLineAndColumn(statement string, pos int) (int, int) {
	var line, column int
	for i := 0; i < pos; i++ {
		if statement[i] == '\n' {
			line++
			column = 0
		} else {
			column++
		}
	}

	return line + 1, column
}
