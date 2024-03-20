package oracle

import (
	"github.com/bytebase/bytebase/backend/plugin/parser/plsql"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	schema.RegisterCheckColumnType(storepb.Engine_ORACLE, checkColumnType)
}

func checkColumnType(tp string) bool {
	_, _, err := plsql.ParsePLSQL("CREATE TABLE t (a " + tp + " NOT NULL)")
	return err == nil
}
