package pgantlr

import (
	"context"
	"fmt"

	"github.com/antlr4-go/antlr/v4"

	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/pg"
)

var (
	_ advisor.Advisor = (*IndexPrimaryKeyTypeAllowlistAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.SchemaRuleIndexPrimaryKeyTypeAllowlist, &IndexPrimaryKeyTypeAllowlistAdvisor{})
}

// IndexPrimaryKeyTypeAllowlistAdvisor is the advisor checking for primary key type allowlist.
type IndexPrimaryKeyTypeAllowlistAdvisor struct {
}

// Check checks for primary key type allowlist.
func (*IndexPrimaryKeyTypeAllowlistAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, err := getANTLRTree(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	payload, err := advisor.UnmarshalStringArrayTypeRulePayload(checkCtx.Rule.Payload)
	if err != nil {
		return nil, err
	}

	checker := &indexPrimaryKeyTypeAllowlistChecker{
		BasePostgreSQLParserListener: &parser.BasePostgreSQLParserListener{},
		level:                        level,
		title:                        string(checkCtx.Rule.Type),
		allowlist:                    payload.List,
	}

	antlr.ParseTreeWalkerDefault.Walk(checker, tree.Tree)

	return checker.adviceList, nil
}

type indexPrimaryKeyTypeAllowlistChecker struct {
	*parser.BasePostgreSQLParserListener

	adviceList []*storepb.Advice
	level      storepb.Advice_Status
	title      string
	allowlist  []string
}

// EnterCreatestmt checks CREATE TABLE with inline PRIMARY KEY constraints
func (c *indexPrimaryKeyTypeAllowlistChecker) EnterCreatestmt(ctx *parser.CreatestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Build maps of column name to column type and line number
	columnTypes := make(map[string]string)
	columnLines := make(map[string]int)

	if ctx.Opttableelementlist() != nil && ctx.Opttableelementlist().Tableelementlist() != nil {
		allElements := ctx.Opttableelementlist().Tableelementlist().AllTableelement()
		for _, elem := range allElements {
			if elem.ColumnDef() != nil {
				colDef := elem.ColumnDef()
				if colDef.Colid() != nil && colDef.Typename() != nil {
					columnName := pg.NormalizePostgreSQLColid(colDef.Colid())
					columnType := c.getTypeName(colDef.Typename())
					columnTypes[columnName] = columnType
					columnLines[columnName] = colDef.GetStart().GetLine()

					// Check if this column has PRIMARY KEY constraint
					if c.hasColumnPrimaryKeyConstraint(colDef) {
						if !c.isTypeAllowed(columnType) {
							c.addAdvice(columnName, columnType, colDef.GetStart().GetLine())
						}
					}
				}
			}
		}

		// Check table-level PRIMARY KEY constraints
		for _, elem := range allElements {
			if elem.Tableconstraint() != nil {
				c.checkTablePrimaryKey(elem.Tableconstraint(), columnTypes, columnLines)
			}
		}
	}
}

func (*indexPrimaryKeyTypeAllowlistChecker) hasColumnPrimaryKeyConstraint(colDef parser.IColumnDefContext) bool {
	if colDef.Colquallist() == nil {
		return false
	}

	allConstraints := colDef.Colquallist().AllColconstraint()
	for _, constraint := range allConstraints {
		if constraint.Colconstraintelem() != nil {
			elem := constraint.Colconstraintelem()
			if elem.PRIMARY() != nil && elem.KEY() != nil {
				return true
			}
		}
	}

	return false
}

func (c *indexPrimaryKeyTypeAllowlistChecker) checkTablePrimaryKey(constraint parser.ITableconstraintContext, columnTypes map[string]string, columnLines map[string]int) {
	if constraint == nil || constraint.Constraintelem() == nil {
		return
	}

	elem := constraint.Constraintelem()

	// Check if this is a PRIMARY KEY constraint
	if elem.PRIMARY() != nil && elem.KEY() != nil {
		if elem.Columnlist() != nil {
			// Get all columns in the PRIMARY KEY
			allColumns := elem.Columnlist().AllColumnElem()
			for _, col := range allColumns {
				if col.Colid() != nil {
					columnName := pg.NormalizePostgreSQLColid(col.Colid())
					if columnType, exists := columnTypes[columnName]; exists {
						if !c.isTypeAllowed(columnType) {
							// Use the column definition's line, not the constraint's line
							line := columnLines[columnName]
							c.addAdvice(columnName, columnType, line)
						}
					}
				}
			}
		}
	}
}

func (*indexPrimaryKeyTypeAllowlistChecker) getTypeName(typename parser.ITypenameContext) string {
	if typename == nil {
		return ""
	}

	// Get the type text and normalize it
	return normalizePostgreSQLType(typename.GetText())
}

func (c *indexPrimaryKeyTypeAllowlistChecker) isTypeAllowed(columnType string) bool {
	// Check if the column type is equivalent to any type in the allowlist
	return isTypeInList(columnType, c.allowlist)
}

func (c *indexPrimaryKeyTypeAllowlistChecker) addAdvice(columnName, columnType string, line int) {
	c.adviceList = append(c.adviceList, &storepb.Advice{
		Status:  c.level,
		Code:    advisor.IndexPKType.Int32(),
		Title:   c.title,
		Content: fmt.Sprintf("The column %q is one of the primary key, but its type %q is not in allowlist", columnName, columnType),
		StartPosition: &storepb.Position{
			Line:   int32(line),
			Column: 0,
		},
	})
}
