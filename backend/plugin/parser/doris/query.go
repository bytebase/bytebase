package doris

import (
	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/doris-parser"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	base.RegisterQueryValidator(storepb.Engine_STARROCKS, validateQuery)
	base.RegisterQueryValidator(storepb.Engine_DORIS, validateQuery)
}

func validateQuery(statement string) (bool, bool, error) {
	// TODO: support other readonly statements like SHOW TABLES, SHOW CREATE TABLE, etc.
	result, err := ParseDorisSQL(statement)
	if err != nil {
		return false, false, err
	}
	l := &queryValidateListener{
		valid: true,
	}
	antlr.ParseTreeWalkerDefault.Walk(l, result.Tree)
	if !l.valid {
		return false, false, nil
	}
	return true, true, nil
}

type queryValidateListener struct {
	*parser.BaseDorisSQLListener

	valid bool
}

func (l *queryValidateListener) EnterSingleStatement(ctx *parser.SingleStatementContext) {
	if !l.valid {
		return
	}

	if ctx.Statement() == nil {
		return
	}

	if ctx.Statement().QueryStatement() == nil {
		l.valid = false
	}
}
