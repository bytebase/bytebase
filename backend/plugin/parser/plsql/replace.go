package plsql

import (
	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/plsql-parser"
)

type EraseContext struct {
	eraseSchemaName     bool
	eraseIndexName      bool
	eraseConstraintName bool
}

func EraseString(ctx EraseContext, rule antlr.ParserRuleContext, tokens antlr.TokenStream) string {
	listener := &eraseListener{
		ctx:      ctx,
		rewriter: *antlr.NewTokenStreamRewriter(tokens),
	}
	antlr.ParseTreeWalkerDefault.Walk(listener, rule)
	return listener.rewriter.GetText(antlr.DefaultProgramName, antlr.Interval{
		Start: rule.GetStart().GetTokenIndex(),
		Stop:  rule.GetStop().GetTokenIndex(),
	})
}

type eraseListener struct {
	*parser.BasePlSqlParserListener

	ctx      EraseContext
	rewriter antlr.TokenStreamRewriter
}

func (l *eraseListener) EnterTableview_name(ctx *parser.Tableview_nameContext) {
	if l.ctx.eraseSchemaName && ctx.Id_expression() != nil {
		l.rewriter.DeleteDefault(
			ctx.Identifier().GetStart().GetTokenIndex(),
			ctx.Id_expression().GetStart().GetTokenIndex()-1,
		)
	}
}

func (l *eraseListener) EnterSchema_name(ctx *parser.Schema_nameContext) {
	if l.ctx.eraseSchemaName {
		l.rewriter.DeleteDefault(
			ctx.Identifier().GetStart().GetTokenIndex(),
			ctx.Identifier().GetStop().GetTokenIndex(),
		)
	}
}

func (l *eraseListener) EnterCreate_index(ctx *parser.Create_indexContext) {
	if l.ctx.eraseIndexName && ctx.Index_name() != nil {
		l.rewriter.DeleteDefault(
			ctx.Index_name().GetStart().GetTokenIndex(),
			ctx.Index_name().GetStop().GetTokenIndex(),
		)
	}
}

func (l *eraseListener) EnterConstraint_name(ctx *parser.Constraint_nameContext) {
	if l.ctx.eraseConstraintName {
		l.rewriter.DeleteDefault(
			ctx.Identifier().GetStart().GetTokenIndex(),
			ctx.Identifier().GetStop().GetTokenIndex(),
		)
	}
}
