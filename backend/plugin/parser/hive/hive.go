// Package hive registers Hive's parser functions, backed by omni's hive packages.
package hive

import (
	omnianalysis "github.com/bytebase/omni/hive/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.Register(base.Engine{
		Split:         SplitSQL,
		ValidateQuery: omnianalysis.ValidateQuery,
		Explain:       explainStatement,
	}, storepb.Engine_HIVE)
}

// SplitSQL splits the given SQL statement into multiple SQL statements.
func SplitSQL(statement string) ([]base.Statement, error) {
	stmts, err := omnianalysis.Split(statement)
	if err != nil {
		return nil, err
	}
	return base.OmniStatements(statement, stmts), nil
}

func explainStatement(statement string, format base.ExplainFormat) (string, error) {
	formatName := ""
	if format == base.ExplainFormatJSON {
		formatName = "JSON"
	}
	return omnianalysis.ExplainStatement(statement, formatName), nil
}
