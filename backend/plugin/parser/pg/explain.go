package pg

import (
	"github.com/bytebase/omni/pg/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterExplainStatementFunc(storepb.Engine_POSTGRES, explainStatement)
	// CockroachDB parses through the PostgreSQL grammar here as it does for the
	// read-only gate (see query.go).
	base.RegisterExplainStatementFunc(storepb.Engine_COCKROACHDB, cockroachExplainStatement)
}

// explainStatement implements base.ExplainStatement for PostgreSQL.
func explainStatement(statement string, format base.ExplainFormat) (string, error) {
	var formatName string
	switch format {
	case base.ExplainFormatJSON:
		formatName = "json"
	case base.ExplainFormatXML:
		formatName = "xml"
	case base.ExplainFormatYAML:
		formatName = "yaml"
	default:
	}
	explain, err := analysis.ExplainStatement(statement, formatName)
	return base.ExplainStatementResult(explain, err, nil)
}

// cockroachExplainStatement implements base.ExplainStatement for CockroachDB.
func cockroachExplainStatement(statement string, format base.ExplainFormat) (string, error) {
	explain, err := analysis.CockroachExplainStatement(statement, format == base.ExplainFormatJSON)
	return base.ExplainStatementResult(explain, err, nil)
}

// DescribeExplain returns the output format of an EXPLAIN statement and
// whether it executes the statement it plans.
func DescribeExplain(statement string) (format string, executed bool, ok bool) {
	return analysis.DescribeExplain(statement)
}
