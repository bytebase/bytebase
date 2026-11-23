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
	return base.BuildStatements(statement, stmts, func(s omnicosmosdb.Statement) (int, int, bool) {
		return s.ByteStart, s.ByteEnd, s.Empty()
	}, false), nil
}
