package tidb

import (
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	// omni is the sole TiDB statement parser (parseTiDBStatements in omni.go).
	base.RegisterParseStatementsFunc(storepb.Engine_TIDB, parseTiDBStatements)
	base.RegisterGetStatementTypes(storepb.Engine_TIDB, GetStatementTypes)
}
