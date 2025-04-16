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
	_ advisor.Advisor = (*TableCommentConventionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, advisor.OracleTableCommentConvention, &TableCommentConventionAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE_ORACLE, advisor.OracleTableCommentConvention, &TableCommentConventionAdvisor{})
}

// TableCommentConventionAdvisor is the advisor checking for table comment convention.
type TableCommentConventionAdvisor struct {
}

func (*TableCommentConventionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
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

	listener := &tableCommentConventionListener{
		level:                level,
		title:                string(checkCtx.Rule.Type),
		currentDatabase:      checkCtx.CurrentDatabase,
		payload:              payload,
		classificationConfig: checkCtx.ClassificationConfig,
		tableNames:           []string{},
		tableComment:         make(map[string]string),
		tableLine:            make(map[string]int),
	}

	antlr.ParseTreeWalkerDefault.Walk(listener, tree)

	return listener.generateAdvices()
}

// tableCommentConventionListener is the listener for table comment convention.
type tableCommentConventionListener struct {
	*parser.BasePlSqlParserListener

	level                storepb.Advice_Status
	title                string
	currentDatabase      string
	payload              *advisor.CommentConventionRulePayload
	classificationConfig *storepb.DataClassificationSetting_DataClassificationConfig

	tableNames   []string
	tableComment map[string]string
	tableLine    map[string]int
}

func (l *tableCommentConventionListener) EnterCreate_table(ctx *parser.Create_tableContext) {
	schemaName := l.currentDatabase
	if ctx.Schema_name() != nil {
		schemaName = normalizeIdentifier(ctx.Schema_name(), l.currentDatabase)
	}

	tableName := fmt.Sprintf("%s.%s", schemaName, normalizeIdentifier(ctx.Table_name(), l.currentDatabase))
	l.tableNames = append(l.tableNames, tableName)
	l.tableLine[tableName] = ctx.GetStart().GetLine()
}

func (l *tableCommentConventionListener) EnterComment_on_table(ctx *parser.Comment_on_tableContext) {
	if ctx.Tableview_name() == nil {
		return
	}

	tableName := normalizeIdentifier(ctx.Tableview_name(), l.currentDatabase)
	l.tableComment[tableName] = plsqlparser.NormalizeQuotedString(ctx.Quoted_string())
}

func (l *tableCommentConventionListener) generateAdvices() ([]*storepb.Advice, error) {
	advices := []*storepb.Advice{}
	for _, tableName := range l.tableNames {
		comment, ok := l.tableComment[tableName]
		if !ok || comment == "" {
			if l.payload.Required {
				advices = append(advices, &storepb.Advice{
					Status:        l.level,
					Code:          advisor.CommentEmpty.Int32(),
					Title:         l.title,
					Content:       fmt.Sprintf("Comment is required for table %s", normalizeIdentifierName(tableName)),
					StartPosition: common.ConvertANTLRLineToPosition(l.tableLine[tableName]),
				})
			}
		} else {
			if l.payload.MaxLength > 0 && len(comment) > l.payload.MaxLength {
				advices = append(advices, &storepb.Advice{
					Status:        l.level,
					Code:          advisor.CommentTooLong.Int32(),
					Title:         l.title,
					Content:       fmt.Sprintf("Table %s comment is too long. The length of comment should be within %d characters", normalizeIdentifierName(tableName), l.payload.MaxLength),
					StartPosition: common.ConvertANTLRLineToPosition(l.tableLine[tableName]),
				})
			}
			if l.payload.RequiredClassification {
				if classification, _ := common.GetClassificationAndUserComment(comment, l.classificationConfig); classification == "" {
					advices = append(advices, &storepb.Advice{
						Status:        l.level,
						Code:          advisor.CommentMissingClassification.Int32(),
						Title:         l.title,
						Content:       fmt.Sprintf("Table %s comment requires classification", normalizeIdentifierName(tableName)),
						StartPosition: common.ConvertANTLRLineToPosition(l.tableLine[tableName]),
					})
				}
			}
		}
	}

	return advices, nil
}
