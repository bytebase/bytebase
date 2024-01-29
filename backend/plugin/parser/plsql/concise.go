package plsql

import (
	"strings"

	"github.com/antlr4-go/antlr/v4"
	plsql "github.com/bytebase/plsql-parser"
)

func GetConciseSchema(schema string) (string, error) {
	node, _, err := ParsePLSQL(schema)
	if err != nil {
		return "", err
	}

	listener := &ConciseListener{
		buf: &strings.Builder{},
	}
	antlr.ParseTreeWalkerDefault.Walk(listener, node)
	return listener.buf.String(), listener.err
}

type ConciseListener struct {
	*plsql.BasePlSqlParserListener

	buf *strings.Builder
	err error
}

func (l *ConciseListener) EnterUnit_statement(ctx *plsql.Unit_statementContext) {
	switch {
	case ctx.Create_table() != nil:
		if _, err := l.buf.WriteString(EraseString(EraseContext{
			eraseSchemaName:     true,
			normalizeIndexName:  true,
			eraseConstraintName: true,
			eraseStoreOption:    true,
		}, ctx.Create_table(), ctx.GetParser().GetTokenStream())); err != nil {
			l.err = err
			return
		}
		if _, err := l.buf.WriteString("\n\n"); err != nil {
			l.err = err
			return
		}
	case ctx.Create_index() != nil:
		if _, err := l.buf.WriteString(EraseString(EraseContext{
			eraseSchemaName:     true,
			normalizeIndexName:  true,
			eraseConstraintName: true,
			eraseStoreOption:    true,
		}, ctx.Create_index(), ctx.GetParser().GetTokenStream())); err != nil {
			l.err = err
			return
		}
		if _, err := l.buf.WriteString("\n\n"); err != nil {
			l.err = err
			return
		}
	}
}
