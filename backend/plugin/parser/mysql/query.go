package mysql

import (
	"github.com/bytebase/omni/mysql/analysis"
	mysqlparser "github.com/bytebase/omni/mysql/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterQueryValidator(storepb.Engine_MYSQL, validateQuery)
}

// validateQuery validates the SQL statement for SQL editor.
func validateQuery(statement string) (bool, bool, error) {
	stmts, err := mysqlparser.Parse(statement)
	if err != nil {
		return false, false, convertError(err, base.Statement{Text: statement})
	}
	readOnly, returnsData := analysis.ValidateQuery(stmts.Items)
	return readOnly, returnsData, nil
}
