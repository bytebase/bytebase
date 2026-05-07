package tidb

import (
	"context"
	"fmt"
	"regexp"

	"github.com/bytebase/omni/tidb/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*NamingColumnConventionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_NAMING_COLUMN, &NamingColumnConventionAdvisor{})
}

// NamingColumnConventionAdvisor is the advisor checking for column naming convention.
type NamingColumnConventionAdvisor struct {
}

// Check checks for column naming convention.
func (*NamingColumnConventionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, err := getTiDBOmniNodes(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	namingPayload := checkCtx.Rule.GetNamingPayload()
	if namingPayload == nil {
		return nil, errors.New("naming_payload is required for this rule")
	}

	format, err := regexp.Compile(namingPayload.Format)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compile regex format %q", namingPayload.Format)
	}

	maxLength := int(namingPayload.MaxLength)
	if maxLength == 0 {
		maxLength = advisor.DefaultNameLengthLimit
	}
	checker := &namingColumnConventionChecker{
		level:     level,
		title:     checkCtx.Rule.Type.String(),
		format:    format,
		maxLength: maxLength,
	}

	for _, ostmt := range stmts {
		checker.checkStmt(ostmt)
	}

	return checker.adviceList, nil
}

type namingColumnConventionChecker struct {
	adviceList []*storepb.Advice
	level      storepb.Advice_Status
	title      string
	format     *regexp.Regexp
	maxLength  int
}

type columnDataNamingColumn struct {
	name string
	line int
}

func (c *namingColumnConventionChecker) checkStmt(ostmt OmniStmt) {
	var columnList []columnDataNamingColumn
	var tableName string
	switch n := ostmt.Node.(type) {
	case *ast.CreateTableStmt:
		if n.Table == nil {
			return
		}
		tableName = n.Table.Name
		for _, col := range n.Columns {
			if col == nil {
				continue
			}
			columnList = append(columnList, columnDataNamingColumn{
				name: col.Name,
				line: ostmt.AbsoluteLine(col.Loc.Start),
			})
		}
	case *ast.AlterTableStmt:
		if n.Table == nil {
			return
		}
		tableName = n.Table.Name
		// Use the ALTER statement's line for all column references — pingcap parity.
		stmtLine := ostmt.AbsoluteLine(n.Loc.Start)
		for _, cmd := range n.Commands {
			if cmd == nil {
				continue
			}
			switch cmd.Type {
			case ast.ATRenameColumn:
				if cmd.NewName != "" {
					columnList = append(columnList, columnDataNamingColumn{
						name: cmd.NewName,
						line: stmtLine,
					})
				}
			case ast.ATAddColumn:
				// omni populates either Columns (multi-form) or Column (single).
				// Mutually exclusive in practice but not enforced by the type.
				if len(cmd.Columns) > 0 {
					for _, col := range cmd.Columns {
						if col != nil {
							columnList = append(columnList, columnDataNamingColumn{
								name: col.Name,
								line: stmtLine,
							})
						}
					}
				} else if cmd.Column != nil {
					columnList = append(columnList, columnDataNamingColumn{
						name: cmd.Column.Name,
						line: stmtLine,
					})
				}
			case ast.ATChangeColumn:
				// Only the new column name matters here.
				if cmd.Column != nil {
					columnList = append(columnList, columnDataNamingColumn{
						name: cmd.Column.Name,
						line: stmtLine,
					})
				}
			default:
				// MODIFY COLUMN (ATModifyColumn) and other forms intentionally
				// not handled — pre-existing inheritance from the pingcap-AST
				// version of this advisor.
			}
		}
	default:
		return
	}

	for _, column := range columnList {
		if !c.format.MatchString(column.name) {
			c.adviceList = append(c.adviceList, &storepb.Advice{
				Status:        c.level,
				Code:          code.NamingColumnConventionMismatch.Int32(),
				Title:         c.title,
				Content:       fmt.Sprintf("`%s`.`%s` mismatches column naming convention, naming format should be %q", tableName, column.name, c.format),
				StartPosition: common.ConvertANTLRLineToPosition(column.line),
			})
		}
		if c.maxLength > 0 && len(column.name) > c.maxLength {
			c.adviceList = append(c.adviceList, &storepb.Advice{
				Status:        c.level,
				Code:          code.NamingColumnConventionMismatch.Int32(),
				Title:         c.title,
				Content:       fmt.Sprintf("`%s`.`%s` mismatches column naming convention, its length should be within %d characters", tableName, column.name, c.maxLength),
				StartPosition: common.ConvertANTLRLineToPosition(column.line),
			})
		}
	}
}
