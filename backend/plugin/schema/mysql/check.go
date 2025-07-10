package mysql

import (
	"fmt"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

func init() {
	schema.RegisterCheckColumnType(storepb.Engine_MYSQL, checkColumnType)
}

func checkColumnType(tp string) bool {
	_, err := mysqlparser.ParseMySQL(fmt.Sprintf("CREATE TABLE t (a %s NOT NULL)", tp))
	return err == nil
}
