package standard

import (
	omnianalysis "github.com/bytebase/omni/analysis"
	dbxanalysis "github.com/bytebase/omni/databricks/analysis"
	hiveanalysis "github.com/bytebase/omni/hive/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.Register(base.Engine{
		Split:         hiveSplitSQL,
		ValidateQuery: hiveanalysis.ValidateQuery,
		Explain:       hiveExplainStatement,
	}, storepb.Engine_HIVE)

	base.Register(base.Engine{
		Split:   databricksSplitSQL,
		Explain: databricksExplainStatement,
	}, storepb.Engine_DATABRICKS)
}

func hiveSplitSQL(statement string) ([]base.Statement, error) {
	return splitWith(statement, hiveanalysis.Split)
}

func databricksSplitSQL(statement string) ([]base.Statement, error) {
	return splitWith(statement, dbxanalysis.Split)
}

func splitWith(statement string, split func(string) ([]omnianalysis.Statement, error)) ([]base.Statement, error) {
	stmts, err := split(statement)
	if err != nil {
		return nil, err
	}
	return base.OmniStatements(statement, stmts), nil
}

func explainFormatName(format base.ExplainFormat) string {
	if format == base.ExplainFormatJSON {
		return "JSON"
	}
	return ""
}

func hiveExplainStatement(statement string, format base.ExplainFormat) (string, error) {
	return hiveanalysis.ExplainStatement(statement, explainFormatName(format)), nil
}

func databricksExplainStatement(statement string, format base.ExplainFormat) (string, error) {
	return dbxanalysis.ExplainStatement(statement, explainFormatName(format)), nil
}
