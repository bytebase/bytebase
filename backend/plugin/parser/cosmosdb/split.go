package cosmosdb

import (
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	base.RegisterSplitterFunc(storepb.Engine_COSMOSDB, SplitSQL)
}

// SplitSQL splits the given SQL statement into multiple SQL statements.
func SplitSQL(statement string) ([]base.SingleSQL, error) {
	// TODO(zp): Does CosmosDB support multiple statements? And how we use the split function?
	return []base.SingleSQL{
		{
			Text: statement,
		},
	}, nil
}
