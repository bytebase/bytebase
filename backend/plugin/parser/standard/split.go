package standard

import (
	omnianalysis "github.com/bytebase/omni/analysis"
	chanalysis "github.com/bytebase/omni/clickhouse/analysis"
	dbxanalysis "github.com/bytebase/omni/databricks/analysis"
	hiveanalysis "github.com/bytebase/omni/hive/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterSplitterFunc(storepb.Engine_CLICKHOUSE, SplitSQL)
	base.RegisterSplitterFunc(storepb.Engine_HIVE, hiveSplitSQL)
	base.RegisterSplitterFunc(storepb.Engine_DATABRICKS, databricksSplitSQL)
}

// SplitSQL splits the given SQL statement into multiple SQL statements.
func SplitSQL(statement string) ([]base.Statement, error) {
	return splitWith(statement, chanalysis.Split)
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
