package tidb

import (
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

func init() {
	base.RegisterGenerateRestoreSQL(storepb.Engine_TIDB, mysql.GenerateRestoreSQL)
}
