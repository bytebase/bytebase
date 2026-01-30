package mongodb

import (
	"strings"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterSplitterFunc(storepb.Engine_MONGODB, SplitSQL)
}

// SplitSQL splits the input into multiple MongoDB statements.
// Unlike SQL-based engines that split on semicolons, MongoDB statements can be
// separated by newlines without semicolons, so we use the parser to identify
// statement boundaries.
func SplitSQL(statement string) ([]base.Statement, error) {
	stmts, err := ParseMongoShell(statement)
	if err != nil {
		return nil, err
	}

	if len(stmts) == 0 {
		if len(strings.TrimSpace(statement)) == 0 {
			return nil, nil
		}
		return []base.Statement{{Text: statement}}, nil
	}

	var result []base.Statement
	for _, ps := range stmts {
		result = append(result, ps.Statement)
	}
	return result, nil
}
