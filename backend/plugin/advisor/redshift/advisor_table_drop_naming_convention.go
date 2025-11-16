package redshift

import (
	"context"
	"fmt"
	"regexp"

	"github.com/bytebase/bytebase/backend/plugin/advisor/code"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/redshift"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
)

var (
	_ advisor.Advisor = (*TableDropNamingConventionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_REDSHIFT, advisor.SchemaRuleTableDropNamingConvention, &TableDropNamingConventionAdvisor{})
}

// TableDropNamingConventionAdvisor is the advisor checking for table drop with naming convention.
type TableDropNamingConventionAdvisor struct {
}

// Check checks for table drop with naming convention.
func (*TableDropNamingConventionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, ok := checkCtx.AST.(antlr.Tree)
	if !ok {
		return nil, errors.Errorf("failed to convert to ANTLR Tree")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	format, _, err := advisor.UnmarshalNamingRulePayloadAsRegexp(checkCtx.Rule.Payload)
	if err != nil {
		return nil, err
	}

	listener := &tableDropNamingConventionListener{
		level:      level,
		title:      string(checkCtx.Rule.Type),
		format:     format,
		adviceList: []*storepb.Advice{},
	}

	antlr.ParseTreeWalkerDefault.Walk(listener, tree)

	return listener.adviceList, nil
}

type tableDropNamingConventionListener struct {
	*parser.BaseRedshiftParserListener

	adviceList []*storepb.Advice
	level      storepb.Advice_Status
	title      string
	format     *regexp.Regexp
}

// EnterDropstmt is called when entering a dropstmt rule.
func (l *tableDropNamingConventionListener) EnterDropstmt(ctx *parser.DropstmtContext) {
	if ctx.DROP() == nil {
		return
	}

	// Check if this is a DROP TABLE statement
	if ctx.Object_type_any_name() != nil && ctx.Object_type_any_name().TABLE() != nil {
		// Extract table names from the drop statement
		if ctx.Any_name_list() != nil {
			for _, anyName := range ctx.Any_name_list().AllAny_name() {
				tableName := getTableNameFromAnyName(anyName)
				if tableName != "" && !l.format.MatchString(tableName) {
					l.adviceList = append(l.adviceList, &storepb.Advice{
						Status:  l.level,
						Code:    code.TableDropNamingConventionMismatch.Int32(),
						Title:   l.title,
						Content: fmt.Sprintf("`%s` mismatches drop table naming convention, naming format should be %q", tableName, l.format),
						StartPosition: common.ConvertANTLRPositionToPosition(&common.ANTLRPosition{
							Line:   int32(ctx.GetStart().GetLine()),
							Column: int32(ctx.GetStart().GetColumn()),
						}, ctx.GetText()),
					})
				}
			}
		}
	}
}

func getTableNameFromAnyName(ctx parser.IAny_nameContext) string {
	if ctx == nil {
		return ""
	}

	parts := []string{}

	// First part (could be schema or table)
	if ctx.Colid() != nil {
		parts = append(parts, normalizeRedshiftIdentifier(ctx.Colid().GetText()))
	}

	// Additional parts from attrs (qualified names like schema.table)
	if ctx.Attrs() != nil {
		for _, attr := range ctx.Attrs().AllAttr_name() {
			parts = append(parts, normalizeRedshiftIdentifier(attr.GetText()))
		}
	}

	// Return the last part as the table name
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}

	return ""
}

func normalizeRedshiftIdentifier(name string) string {
	// Remove quotes if present
	if len(name) >= 2 && name[0] == '"' && name[len(name)-1] == '"' {
		return name[1 : len(name)-1]
	}
	return name
}
