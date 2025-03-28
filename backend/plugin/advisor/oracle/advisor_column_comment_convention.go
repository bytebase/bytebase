// Package oracle is the advisor for oracle database.
package oracle

import (
	"context"
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/plsql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	plsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/plsql"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*ColumnCommentConventionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, advisor.OracleColumnCommentConvention, &ColumnCommentConventionAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE_ORACLE, advisor.OracleColumnCommentConvention, &ColumnCommentConventionAdvisor{})
}

// ColumnCommentConventionAdvisor is the advisor checking for column comment convention.
type ColumnCommentConventionAdvisor struct {
}

func (*ColumnCommentConventionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, ok := checkCtx.AST.(antlr.Tree)
	if !ok {
		return nil, errors.Errorf("failed to convert to Tree")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	payload, err := advisor.UnmarshalCommentConventionRulePayload(checkCtx.Rule.Payload)
	if err != nil {
		return nil, err
	}

	listener := &columnCommentConventionListener{
		level:                level,
		title:                string(checkCtx.Rule.Type),
		currentDatabase:      checkCtx.CurrentDatabase,
		payload:              payload,
		classificationConfig: checkCtx.ClassificationConfig,
		columnNames:          []string{},
		columnComment:        make(map[string]string),
		columnLine:           make(map[string]int),
	}

	antlr.ParseTreeWalkerDefault.Walk(listener, tree)

	return listener.generateAdvices()
}

// columnCommentConventionListener is the listener for column comment convention.
type columnCommentConventionListener struct {
	*parser.BasePlSqlParserListener

	level                storepb.Advice_Status
	title                string
	currentDatabase      string
	payload              *advisor.CommentConventionRulePayload
	classificationConfig *storepb.DataClassificationSetting_DataClassificationConfig

	tableName     string
	columnNames   []string
	columnComment map[string]string
	columnLine    map[string]int
}

func (l *columnCommentConventionListener) EnterCreate_table(ctx *parser.Create_tableContext) {
	schemaName := l.currentDatabase
	if ctx.Schema_name() != nil {
		schemaName = normalizeIdentifier(ctx.Schema_name(), l.currentDatabase)
	}
	l.tableName = fmt.Sprintf("%s.%s", schemaName, normalizeIdentifier(ctx.Table_name(), schemaName))
}

func (l *columnCommentConventionListener) ExitCreate_table(_ *parser.Create_tableContext) {
	l.tableName = ""
}

func (l *columnCommentConventionListener) EnterColumn_definition(ctx *parser.Column_definitionContext) {
	if l.tableName == "" {
		return
	}
	columnName := fmt.Sprintf(`%s.%s`, l.tableName, normalizeIdentifier(ctx.Column_name(), l.currentDatabase))
	l.columnNames = append(l.columnNames, columnName)
	l.columnLine[columnName] = ctx.GetStart().GetLine()
}

func (l *columnCommentConventionListener) EnterAlter_table(ctx *parser.Alter_tableContext) {
	l.tableName = normalizeIdentifier(ctx.Tableview_name(), l.currentDatabase)
}

func (l *columnCommentConventionListener) ExitAdd_column_clause(_ *parser.Add_column_clauseContext) {
	l.tableName = ""
}

func (l *columnCommentConventionListener) EnterComment_on_column(ctx *parser.Comment_on_columnContext) {
	if ctx.Column_name() == nil {
		return
	}

	columnName := fmt.Sprintf(`%s.%s`, l.currentDatabase, normalizeIdentifier(ctx.Column_name(), ""))
	l.columnComment[columnName] = plsqlparser.NormalizeQuotedString(ctx.Quoted_string())
}

func (l *columnCommentConventionListener) generateAdvices() ([]*storepb.Advice, error) {
	advices := []*storepb.Advice{}
	for _, columnName := range l.columnNames {
		comment, ok := l.columnComment[columnName]
		if !ok || comment == "" {
			if l.payload.Required {
				advices = append(advices, &storepb.Advice{
					Status:        l.level,
					Code:          advisor.CommentEmpty.Int32(),
					Title:         l.title,
					Content:       fmt.Sprintf("Comment is required for column %s", normalizeIdentifierName(columnName)),
					StartPosition: advisor.ConvertANTLRLineToPosition(l.columnLine[columnName]),
				})
			}
		} else {
			if l.payload.MaxLength > 0 && len(comment) > l.payload.MaxLength {
				advices = append(advices, &storepb.Advice{
					Status:        l.level,
					Code:          advisor.CommentTooLong.Int32(),
					Title:         l.title,
					Content:       fmt.Sprintf("Column %s comment is too long. The length of comment should be within %d characters", normalizeIdentifierName(columnName), l.payload.MaxLength),
					StartPosition: advisor.ConvertANTLRLineToPosition(l.columnLine[columnName]),
				})
			}
			if l.payload.RequiredClassification {
				if classification, _ := common.GetClassificationAndUserComment(comment, l.classificationConfig); classification == "" {
					advices = append(advices, &storepb.Advice{
						Status:        l.level,
						Code:          advisor.CommentMissingClassification.Int32(),
						Title:         l.title,
						Content:       fmt.Sprintf("Column %s comment requires classification", normalizeIdentifierName(columnName)),
						StartPosition: advisor.ConvertANTLRLineToPosition(l.columnLine[columnName]),
					})
				}
			}
		}
	}

	return advices, nil
}
