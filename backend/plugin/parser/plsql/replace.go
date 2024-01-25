package plsql

import (
	"encoding/base64"
	"fmt"
	"log/slog"
	"strings"

	"golang.org/x/crypto/sha3"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/plsql-parser"

	"github.com/bytebase/bytebase/backend/common/log"
)

type EraseContext struct {
	eraseSchemaName bool

	eraseIndexName     bool
	normalizeIndexName bool

	eraseConstraintName bool

	eraseStoreOption bool
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
	} else if l.ctx.normalizeIndexName && ctx.Index_name() != nil {
		l.rewriter.ReplaceDefault(
			ctx.Index_name().GetStart().GetTokenIndex(),
			ctx.Index_name().GetStop().GetTokenIndex(),
			getNormalizeIndexName(ctx),
		)
	}
}

func getNormalizeIndexName(ctx *parser.Create_indexContext) string {
	var buf strings.Builder
	// UNIQUE, BITMAP OR NON
	switch {
	case ctx.UNIQUE() != nil:
		if err := buf.WriteByte('1'); err != nil {
			slog.Debug("Failed to write byte", log.BBError(err))
			return ""
		}
	case ctx.BITMAP() != nil:
		if err := buf.WriteByte('2'); err != nil {
			slog.Debug("Failed to write byte", log.BBError(err))
			return ""
		}
	default:
		if err := buf.WriteByte('0'); err != nil {
			slog.Debug("Failed to write byte", log.BBError(err))
			return ""
		}
	}

	// cluster_index_clause, table_index_clause or bitmap_join_index_clause
	switch {
	case ctx.Cluster_index_clause() != nil:
		if err := buf.WriteByte('1'); err != nil {
			slog.Debug("Failed to write byte", log.BBError(err))
			return ""
		}
	case ctx.Table_index_clause() != nil:
		if err := buf.WriteByte('2'); err != nil {
			slog.Debug("Failed to write byte", log.BBError(err))
			return ""
		}
		if _, err := buf.WriteString(fmt.Sprintf("%d", len(ctx.Table_index_clause().AllIndex_expr()))); err != nil {
			slog.Debug("Failed to write byte", log.BBError(err))
			return ""
		}
	case ctx.Bitmap_join_index_clause() != nil:
		if err := buf.WriteByte('3'); err != nil {
			slog.Debug("Failed to write byte", log.BBError(err))
			return ""
		}
	}

	// USABLE or UNUSABLE
	switch {
	case ctx.USABLE() != nil:
		if err := buf.WriteByte('1'); err != nil {
			slog.Debug("Failed to write byte", log.BBError(err))
			return ""
		}
	case ctx.UNUSABLE() != nil:
		if err := buf.WriteByte('2'); err != nil {
			slog.Debug("Failed to write byte", log.BBError(err))
			return ""
		}
	default:
		if err := buf.WriteByte('0'); err != nil {
			slog.Debug("Failed to write byte", log.BBError(err))
			return ""
		}
	}

	text := EraseString(EraseContext{
		eraseSchemaName: true,
		eraseIndexName:  true,
	}, ctx, ctx.GetParser().GetTokenStream())
	sum := sha3.Sum224([]byte(text))
	s := base64.StdEncoding.EncodeToString(sum[:])
	for i, c := range s {
		if i > 32 {
			break
		}
		if _, err := buf.WriteRune(c); err != nil {
			slog.Debug("Failed to write byte", log.BBError(err))
			return ""
		}
	}
	return buf.String()
}

func (l *eraseListener) EnterConstraint_name(ctx *parser.Constraint_nameContext) {
	if l.ctx.eraseConstraintName {
		l.rewriter.DeleteDefault(
			ctx.Identifier().GetStart().GetTokenIndex(),
			ctx.Identifier().GetStop().GetTokenIndex(),
		)
	}
}
