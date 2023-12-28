package pg

import (
	"fmt"

	pgrawparser "github.com/bytebase/bytebase/backend/plugin/parser/sql/engine/pg"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	schema.RegisterCheckColumnType(storepb.Engine_POSTGRES, checkColumnType)
}

func checkColumnType(tp string) bool {
	_, err := pgrawparser.Parse(pgrawparser.ParseContext{}, fmt.Sprintf("CREATE TABLE t (a %s NOT NULL)", tp))
	return err == nil
}
