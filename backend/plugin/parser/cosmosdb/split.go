package cosmosdb

import (
	"strings"

	omnicosmosdb "github.com/bytebase/omni/cosmosdb"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterSplitterFunc(storepb.Engine_COSMOSDB, SplitSQL)
}

// SplitSQL splits the given SQL statement into individual statements.
// CosmosDB only supports single SELECT statements.
func SplitSQL(statement string) ([]base.Statement, error) {
	if strings.TrimSpace(statement) == "" {
		return nil, nil
	}

	stmts, err := omnicosmosdb.Parse(statement)
	if err != nil {
		return nil, err
	}
	if len(stmts) == 0 {
		return nil, nil
	}

	var result []base.Statement
	for _, stmt := range stmts {
		startPos := byteOffsetToRunePosition(statement, stmt.ByteStart)
		endPos := byteOffsetToRunePosition(statement, stmt.ByteEnd)

		result = append(result, base.Statement{
			Text: stmt.Text,
			Range: &storepb.Range{
				Start: int32(stmt.ByteStart),
				End:   int32(stmt.ByteEnd),
			},
			Start: startPos,
			End:   endPos,
			Empty: stmt.Empty(),
		})
	}
	return result, nil
}
