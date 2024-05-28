package mysql

import (
	"fmt"

	parser "github.com/bytebase/mysql-parser"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	base.RegisterGenerateRestoreSQL(storepb.Engine_MYSQL, GenerateRestoreSQL)
}

func GenerateRestoreSQL(statement string, backupDatabase string, backupTable string, originalDatabase string, originalTable string) (string, error) {
	return "", nil
}

type generator struct {
	*parser.BaseMySQLParserListener

	backupDatabase   string
	backupTable      string
	originalDatabase string
	originalTable    string
	result           string
	err              error
}

func (g *generator) EnterDeleteStatement(ctx *parser.DeleteStatementContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	g.result = fmt.Sprintf("INSERT INTO %s.%s SELECT * FROM %s.%s;", g.originalDatabase, g.originalTable, g.backupDatabase, g.backupTable)
}

func (g *generator) EnterUpdateStatement(ctx *parser.UpdateStatementContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

}
