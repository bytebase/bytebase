package pg

import (
	"encoding/json"
	"log/slog"
	"regexp"
	"strings"

	pgquery "github.com/pganalyze/pg_query_go/v6"
	pgparser "github.com/pganalyze/pg_query_go/v6/parser"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/parser/sql/ast"
	pgrawparser "github.com/bytebase/bytebase/backend/plugin/parser/sql/engine/pg"
)

func init() {
	base.RegisterQueryValidator(storepb.Engine_POSTGRES, validateQuery)
	base.RegisterQueryValidator(storepb.Engine_REDSHIFT, validateQuery)
	base.RegisterQueryValidator(storepb.Engine_COCKROACHDB, validateQuery)
}

// validateQuery validates the SQL statement for SQL editor.
// Consider that the tokenizer cannot handle the dollar-sign($), so that we use pg_query_go to parse the statement.
// For EXPLAIN and normal SELECT statements, we can directly use regexp to check.
// For CTE, we need to parse the statement to JSON and check the JSON keys.
func validateQuery(statement string) (bool, bool, error) {
	stmtList, err := pgrawparser.Parse(pgrawparser.ParseContext{}, statement)
	if err != nil {
		return false, false, convertToSyntaxError(statement, err)
	}

	explainAnalyze := false
	hasExecute := false
	for _, stmt := range stmtList {
		switch stmt := stmt.(type) {
		case *ast.SelectStmt, *ast.VariableShowStmt:
		case *ast.VariableSetStmt:
			hasExecute = true
		case *ast.ExplainStmt:
			if stmt.Analyze {
				// We only support analyze select, because analyze will actually execute the statement.
				if _, ok := stmt.Statement.(*ast.SelectStmt); !ok {
					return false, false, nil
				}
				explainAnalyze = true
			}
		default:
			return false, false, nil
		}
	}

	// TODO(d): figure out whether this is still needed.
	jsonText, err := pgquery.ParseToJSON(statement)
	if err != nil {
		slog.Debug("Failed to parse statement to JSON", slog.String("statement", statement), log.BBError(err))
		return false, false, err
	}

	formattedStr := strings.ToUpper(strings.TrimSpace(statement))

	cteRegex := regexp.MustCompile(`^WITH\s+?`)
	if matchResult := cteRegex.MatchString(formattedStr); matchResult || explainAnalyze {
		var jsonData map[string]any

		if err := json.Unmarshal([]byte(jsonText), &jsonData); err != nil {
			slog.Debug("Failed to unmarshal JSON", slog.String("jsonText", jsonText), log.BBError(err))
			return false, false, err
		}

		dmlKeyList := []string{"InsertStmt", "UpdateStmt", "DeleteStmt"}

		ok := !keyExistsInJSONData(jsonData, dmlKeyList)
		return ok, ok, nil
	}

	return true, !hasExecute, nil
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

func convertToSyntaxError(statement string, err error) *base.SyntaxError {
	if pgErr, ok := err.(*pgparser.Error); ok {
		position := common.ConvertPGParserErrorCursorPosToPosition(pgErr.Cursorpos, statement)
		return &base.SyntaxError{
			Position: position,
			Message:  pgErr.Message,
		}
	}

	return &base.SyntaxError{
		Position: &storepb.Position{
			Line:   0,
			Column: 0,
		},
		Message: err.Error(),
	}
}
