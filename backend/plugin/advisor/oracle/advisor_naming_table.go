// Package oracle is the advisor for oracle database.
package oracle

import (
	"fmt"
	"regexp"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/plsql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*NamingTableAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, advisor.OracleNamingTableConvention, &NamingTableAdvisor{})
	advisor.Register(storepb.Engine_DM, advisor.OracleNamingTableConvention, &NamingTableAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE_ORACLE, advisor.OracleNamingTableConvention, &NamingTableAdvisor{})
}

// NamingTableAdvisor is the advisor checking for table naming convention.
type NamingTableAdvisor struct {
}

// Check checks for table naming convention.
func (*NamingTableAdvisor) Check(ctx advisor.Context) ([]*storepb.Advice, error) {
	tree, ok := ctx.AST.(antlr.Tree)
	if !ok {
		return nil, errors.Errorf("failed to convert to Tree")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	format, maxLength, err := advisor.UnmarshalNamingRulePayloadAsRegexp(ctx.Rule.Payload)
	if err != nil {
		return nil, err
	}

	listener := &namingTableListener{
		level:           level,
		title:           string(ctx.Rule.Type),
		currentDatabase: ctx.CurrentDatabase,
		format:          format,
		maxLength:       maxLength,
	}

	antlr.ParseTreeWalkerDefault.Walk(listener, tree)

	return listener.generateAdvice()
}

// namingTableListener is the listener for table naming convention.
type namingTableListener struct {
	*parser.BasePlSqlParserListener

	level           storepb.Advice_Status
	title           string
	currentDatabase string
	format          *regexp.Regexp
	maxLength       int

	adviceList []*storepb.Advice
}

func (l *namingTableListener) generateAdvice() ([]*storepb.Advice, error) {
	return l.adviceList, nil
}

// EnterCreate_table is called when production create_table is entered.
func (l *namingTableListener) EnterCreate_table(ctx *parser.Create_tableContext) {
	tableName := normalizeIdentifier(ctx.Table_name(), l.currentDatabase)
	if !l.format.MatchString(tableName) {
		l.adviceList = append(l.adviceList, &storepb.Advice{
			Status:  l.level,
			Code:    advisor.NamingTableConventionMismatch.Int32(),
			Title:   l.title,
			Content: fmt.Sprintf(`"%s" mismatches table naming convention, naming format should be %q`, tableName, l.format),
			StartPosition: &storepb.Position{
				Line: int32(ctx.GetStart().GetLine()),
			},
		})
	}
	if l.maxLength > 0 && len(tableName) > l.maxLength {
		l.adviceList = append(l.adviceList, &storepb.Advice{
			Status:  l.level,
			Code:    advisor.NamingTableConventionMismatch.Int32(),
			Title:   l.title,
			Content: fmt.Sprintf("\"%s\" mismatches table naming convention, its length should be within %d characters", tableName, l.maxLength),
			StartPosition: &storepb.Position{
				Line: int32(ctx.GetStart().GetLine()),
			},
		})
	}
}

// EnterAlter_table_properties is called when production alter_table_properties is entered.
func (l *namingTableListener) EnterAlter_table_properties(ctx *parser.Alter_table_propertiesContext) {
	if ctx.Tableview_name() == nil {
		return
	}
	tableName := lastIdentifier(normalizeIdentifier(ctx.Tableview_name(), l.currentDatabase))
	if tableName == "" {
		return
	}
	if !l.format.MatchString(tableName) {
		l.adviceList = append(l.adviceList, &storepb.Advice{
			Status:  l.level,
			Code:    advisor.NamingTableConventionMismatch.Int32(),
			Title:   l.title,
			Content: fmt.Sprintf(`"%s" mismatches table naming convention, naming format should be %q`, tableName, l.format),
			StartPosition: &storepb.Position{
				Line: int32(ctx.GetStart().GetLine()),
			},
		})
	}
	if l.maxLength > 0 && len(tableName) > l.maxLength {
		l.adviceList = append(l.adviceList, &storepb.Advice{
			Status:  l.level,
			Code:    advisor.NamingTableConventionMismatch.Int32(),
			Title:   l.title,
			Content: fmt.Sprintf("\"%s\" mismatches table naming convention, its length should be within %d characters", tableName, l.maxLength),
			StartPosition: &storepb.Position{
				Line: int32(ctx.GetStart().GetLine()),
			},
		})
	}
}
