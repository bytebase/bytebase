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
	stmts, err := chanalysis.Split(statement)
	if err != nil {
		return nil, err
	}
	return toBaseStatements(statement, stmts), nil
}

func hiveSplitSQL(statement string) ([]base.Statement, error) {
	stmts, err := hiveanalysis.Split(statement)
	if err != nil {
		return nil, err
	}
	return toBaseStatements(statement, stmts), nil
}

func databricksSplitSQL(statement string) ([]base.Statement, error) {
	stmts, err := dbxanalysis.Split(statement)
	if err != nil {
		return nil, err
	}
	return toBaseStatements(statement, stmts), nil
}

func toBaseStatements(script string, stmts []omnianalysis.Statement) []base.Statement {
	if len(stmts) == 0 {
		return nil
	}
	list := make([]base.Statement, 0, len(stmts))
	for _, s := range stmts {
		startLine, startColumn := base.CalculateLineAndColumn(script, s.Start)
		endLine, endColumn := base.CalculateLineAndColumn(script, s.End)
		list = append(list, base.Statement{
			Text: s.Text,
			Start: &storepb.Position{
				Line:   int32(startLine + 1),   // 1-based
				Column: int32(startColumn + 1), // 1-based per proto spec
			},
			End: &storepb.Position{
				Line:   int32(endLine + 1),   // 1-based
				Column: int32(endColumn + 1), // 1-based per proto spec
			},
			Range: &storepb.Range{
				Start: int32(s.Start),
				End:   int32(s.End),
			},
			Empty: s.Empty,
		})
	}
	return list
}
