package mongodb

import (
	"github.com/bytebase/omni/mongo/analysis"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// getQueryType parses a MongoDB shell statement and returns its query type.
// Parse errors, empty input, and multi-statement scripts fall back to base.DML.
func getQueryType(statement string) base.QueryType {
	qt, _ := analysis.ClassifyStatement(statement)
	return base.OmniQueryType(qt)
}
