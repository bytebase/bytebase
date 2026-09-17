package util

import (
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/parser/standard"
)

// PrefixExplain is the explain of an engine that plans a statement prefixed
// with EXPLAIN, which returns a text plan.
func PrefixExplain(defaultFormat v1pb.QueryOption_ExplainFormat) db.Explain {
	return db.Explain{
		Formats:       []v1pb.QueryOption_ExplainFormat{v1pb.QueryOption_TEXT},
		DefaultFormat: defaultFormat,
		Statement:     standard.ExplainStatement,
	}
}
