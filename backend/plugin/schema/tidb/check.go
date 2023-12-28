package tidb

import (
	"fmt"

	tidbparser "github.com/bytebase/bytebase/backend/plugin/parser/tidb"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	schema.RegisterCheckColumnType(storepb.Engine_POSTGRES, checkColumnType)
}

func checkColumnType(tp string) bool {
	_, err := tidbparser.ParseTiDB(fmt.Sprintf("CREATE TABLE t (a %s NOT NULL)", tp), "", "")
	return err == nil
}
