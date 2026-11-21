package standard

import (
	chanalysis "github.com/bytebase/omni/clickhouse/analysis"
	dbxanalysis "github.com/bytebase/omni/databricks/analysis"
	hiveanalysis "github.com/bytebase/omni/hive/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterExplainStatementFunc(storepb.Engine_CLICKHOUSE, clickhouseExplainStatement)
	base.RegisterExplainStatementFunc(storepb.Engine_DATABRICKS, databricksExplainStatement)
	base.RegisterExplainStatementFunc(storepb.Engine_HIVE, hiveExplainStatement)
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

func clickhouseExplainStatement(statement string, format base.ExplainFormat) (string, error) {
	return chanalysis.ExplainStatement(statement, explainFormatName(format)), nil
}
