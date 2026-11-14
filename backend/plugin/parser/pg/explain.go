package pg

import (
	"errors"

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
	return convertExplainError(analysis.ExplainStatement(statement, formatName))
}

// cockroachExplainStatement implements base.ExplainStatement for CockroachDB.
func cockroachExplainStatement(statement string, format base.ExplainFormat) (string, error) {
	return convertExplainError(analysis.CockroachExplainStatement(statement, format == base.ExplainFormatJSON))
}

func convertExplainError(explain string, err error) (string, error) {
	if errors.Is(err, analysis.ErrExplainAnalyzeExecutes) {
		return "", base.ErrExplainAnalyzeExecutes
	}
	return explain, err
}

// DescribeExplain returns the output format of an EXPLAIN statement and
// whether it executes the statement it plans.
func DescribeExplain(statement string) (format string, executed bool, ok bool) {
	return analysis.DescribeExplain(statement)
}
