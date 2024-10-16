package pg

import (
	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/postgresql-parser"
)

const (
	backupSchemaName = "bbdataarchive"
)

func FilterBackupSchema(schema string) (string, error) {
	parseResult, err := ParsePostgreSQL(schema)
	if err != nil {
		return "", err
	}

	listener := &FilterBackupSchemaListener{
		rewriter: *antlr.NewTokenStreamRewriter(parseResult.Tokens),
	}
	antlr.ParseTreeWalkerDefault.Walk(listener, parseResult.Tree)
	return listener.rewriter.GetText(antlr.DefaultProgramName, antlr.Interval{
		Start: 0,
		Stop:  parseResult.Tokens.Size() - 1,
	}), nil
}

type FilterBackupSchemaListener struct {
	*parser.BasePostgreSQLParserListener

	rewriter antlr.TokenStreamRewriter
}

func (l *FilterBackupSchemaListener) EnterCreateschemastmt(ctx *parser.CreateschemastmtContext) {
	var schemaName string
	if ctx.Colid() != nil {
		schemaName = NormalizePostgreSQLColid(ctx.Colid())
	} else if ctx.Optschemaname() != nil {
		schemaName = NormalizePostgreSQLColid(ctx.Optschemaname().Colid())
	}
	if schemaName == backupSchemaName {
		start := ctx.GetStart().GetTokenIndex()
		stop := getFollowingTwoNewLine(ctx.GetParser(), ctx.GetStop().GetTokenIndex()+1)
		l.rewriter.DeleteDefault(start, stop)
	}
}

func (l *FilterBackupSchemaListener) EnterCreatestmt(ctx *parser.CreatestmtContext) {
	list := NormalizePostgreSQLQualifiedName(ctx.Qualified_name(0))
	if len(list) != 2 {
		return
	}
	schemaName := list[0]
	if schemaName == backupSchemaName {
		start := ctx.GetStart().GetTokenIndex()
		stop := getFollowingTwoNewLine(ctx.GetParser(), ctx.GetStop().GetTokenIndex()+1)
		l.rewriter.DeleteDefault(start, stop)
	}
}

func (l *FilterBackupSchemaListener) EnterViewstmt(ctx *parser.ViewstmtContext) {
	list := NormalizePostgreSQLQualifiedName(ctx.Qualified_name())
	if len(list) != 2 {
		return
	}
	schemaName := list[0]
	if schemaName == backupSchemaName {
		start := ctx.GetStart().GetTokenIndex()
		stop := getFollowingTwoNewLine(ctx.GetParser(), ctx.GetStop().GetTokenIndex()+1)
		l.rewriter.DeleteDefault(start, stop)
	}
}

func (l *FilterBackupSchemaListener) EnterAltertablestmt(ctx *parser.AltertablestmtContext) {
	var schemaName string

	switch {
	case ctx.Relation_expr() != nil:
		list := NormalizePostgreSQLQualifiedName(ctx.Relation_expr().Qualified_name())
		if len(list) != 2 {
			return
		}
		schemaName = list[0]
	case ctx.Qualified_name() != nil:
		list := NormalizePostgreSQLQualifiedName(ctx.Qualified_name())
		if len(list) != 2 {
			return
		}
		schemaName = list[0]
	}

	if schemaName == backupSchemaName {
		start := ctx.GetStart().GetTokenIndex()
		stop := getFollowingTwoNewLine(ctx.GetParser(), ctx.GetStop().GetTokenIndex()+1)
		l.rewriter.DeleteDefault(start, stop)
	}
}

func (l *FilterBackupSchemaListener) EnterIndexstmt(ctx *parser.IndexstmtContext) {
	var schemaName string
	if ctx.Relation_expr() != nil {
		list := NormalizePostgreSQLQualifiedName(ctx.Relation_expr().Qualified_name())
		if len(list) != 2 {
			return
		}
		schemaName = list[0]
	}

	if schemaName == backupSchemaName {
		start := ctx.GetStart().GetTokenIndex()
		stop := getFollowingTwoNewLine(ctx.GetParser(), ctx.GetStop().GetTokenIndex()+1)
		l.rewriter.DeleteDefault(start, stop)
	}
}

func (l *FilterBackupSchemaListener) EnterCreateseqstmt(ctx *parser.CreateseqstmtContext) {
	var schemaName string
	list := NormalizePostgreSQLQualifiedName(ctx.Qualified_name())
	if len(list) != 2 {
		return
	}
	schemaName = list[0]

	if schemaName == backupSchemaName {
		start := ctx.GetStart().GetTokenIndex()
		stop := getFollowingTwoNewLine(ctx.GetParser(), ctx.GetStop().GetTokenIndex()+1)
		l.rewriter.DeleteDefault(start, stop)
	}
}

func (l *FilterBackupSchemaListener) EnterAlterseqstmt(ctx *parser.AlterseqstmtContext) {
	var schemaName string
	list := NormalizePostgreSQLQualifiedName(ctx.Qualified_name())
	if len(list) != 2 {
		return
	}
	schemaName = list[0]

	if schemaName == backupSchemaName {
		start := ctx.GetStart().GetTokenIndex()
		stop := getFollowingTwoNewLine(ctx.GetParser(), ctx.GetStop().GetTokenIndex()+1)
		l.rewriter.DeleteDefault(start, stop)
	}
}

func (l *FilterBackupSchemaListener) EnterCreateextensionstmt(ctx *parser.CreateextensionstmtContext) {
	var schemaName string
	for _, item := range ctx.Create_extension_opt_list().AllCreate_extension_opt_item() {
		if item.SCHEMA() != nil {
			schemaName = normalizePostgreSQLName(item.Name())
		}
	}

	if schemaName == backupSchemaName {
		start := ctx.GetStart().GetTokenIndex()
		stop := getFollowingTwoNewLine(ctx.GetParser(), ctx.GetStop().GetTokenIndex()+1)
		l.rewriter.DeleteDefault(start, stop)
	}
}

func (l *FilterBackupSchemaListener) EnterCreatefunctionstmt(ctx *parser.CreatefunctionstmtContext) {
	list := normalizePostgreSQLFuncName(ctx.Func_name())

	if len(list) != 2 {
		return
	}
	schemaName := list[0]

	if schemaName == backupSchemaName {
		start := ctx.GetStart().GetTokenIndex()
		stop := getFollowingTwoNewLine(ctx.GetParser(), ctx.GetStop().GetTokenIndex()+1)
		l.rewriter.DeleteDefault(start, stop)
	}
}

func (l *FilterBackupSchemaListener) EnterCreatetrigstmt(ctx *parser.CreatetrigstmtContext) {
	list := NormalizePostgreSQLQualifiedName(ctx.Qualified_name())
	if len(list) != 2 {
		return
	}
	schemaName := list[0]

	if schemaName == backupSchemaName {
		start := ctx.GetStart().GetTokenIndex()
		stop := getFollowingTwoNewLine(ctx.GetParser(), ctx.GetStop().GetTokenIndex()+1)
		l.rewriter.DeleteDefault(start, stop)
	}
}

func (l *FilterBackupSchemaListener) EnterDefinestmt(ctx *parser.DefinestmtContext) {
	if ctx.TYPE_P() != nil {
		return
	}

	list := NormalizePostgreSQLAnyName(ctx.Any_name(0))
	if len(list) != 2 {
		return
	}
	schemaName := list[0]

	if schemaName == backupSchemaName {
		start := ctx.GetStart().GetTokenIndex()
		stop := getFollowingTwoNewLine(ctx.GetParser(), ctx.GetStop().GetTokenIndex()+1)
		l.rewriter.DeleteDefault(start, stop)
	}
}

func (l *FilterBackupSchemaListener) EnterCommentstmt(ctx *parser.CommentstmtContext) {
	var schemaName string

	switch {
	case ctx.Object_type_any_name() != nil:
		list := NormalizePostgreSQLAnyName(ctx.Any_name())
		if len(list) == 2 {
			schemaName = list[0]
		}
	case ctx.COLUMN() != nil:
		list := NormalizePostgreSQLAnyName(ctx.Any_name())
		if len(list) == 3 {
			schemaName = list[0]
		}
	}

	if schemaName == backupSchemaName {
		start := ctx.GetStart().GetTokenIndex()
		stop := getFollowingTwoNewLine(ctx.GetParser(), ctx.GetStop().GetTokenIndex()+1)
		l.rewriter.DeleteDefault(start, stop)
	}
}

func getFollowingTwoNewLine(p antlr.Parser, start int) int {
	i := start
	newLineCount := 0
	for {
		if i >= p.GetTokenStream().Size() {
			return i
		}
		switch p.GetTokenStream().Get(i).GetTokenType() {
		case antlr.TokenEOF:
			return i
		case parser.PostgreSQLLexerNewline:
			newLineCount++
			if newLineCount == 2 {
				return i
			}
		case parser.PostgreSQLLexerSEMI:
			i++
			continue
		default:
			if p.GetTokenStream().Get(i).GetChannel() == antlr.TokenDefaultChannel {
				return i - 1
			}
		}
		i++
	}
}
