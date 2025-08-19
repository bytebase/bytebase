package pg

import (
	"fmt"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

func init() {
	schema.RegisterCheckColumnType(storepb.Engine_POSTGRES, checkColumnType)
}

func checkColumnType(tp string) bool {
	_, err := pgparser.ParsePostgreSQL(fmt.Sprintf("CREATE TABLE t (a %s NOT NULL)", tp))
	return err == nil
}
