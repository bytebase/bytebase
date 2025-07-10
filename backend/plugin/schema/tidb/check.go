package tidb

import (
	"fmt"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	tidbparser "github.com/bytebase/bytebase/backend/plugin/parser/tidb"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

func init() {
	schema.RegisterCheckColumnType(storepb.Engine_TIDB, checkColumnType)
}

func checkColumnType(tp string) bool {
	_, err := tidbparser.ParseTiDB(fmt.Sprintf("CREATE TABLE t (a %s NOT NULL)", tp), "", "")
	return err == nil
}
