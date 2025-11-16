package pg

import (
	"context"
	"fmt"

	"github.com/bytebase/bytebase/backend/plugin/advisor/code"

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

	rule := &indexPrimaryKeyTypeAllowlistRule{
		BaseRule:  BaseRule{level: level, title: string(checkCtx.Rule.Type)},
		allowlist: payload.List,
	}

	checker := NewGenericChecker([]Rule{rule})
	antlr.ParseTreeWalkerDefault.Walk(checker, tree.Tree)

	return checker.GetAdviceList(), nil
}

type indexPrimaryKeyTypeAllowlistRule struct {
	BaseRule

	allowlist []string
}

func (*indexPrimaryKeyTypeAllowlistRule) Name() string {
	return "index_primary_key_type_allowlist"
}

func (r *indexPrimaryKeyTypeAllowlistRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case "Createstmt":
		r.handleCreatestmt(ctx.(*parser.CreatestmtContext))
	default:
		// Do nothing for other node types
	}
	return nil
}

func (*indexPrimaryKeyTypeAllowlistRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

// handleCreatestmt checks CREATE TABLE with inline PRIMARY KEY constraints
func (r *indexPrimaryKeyTypeAllowlistRule) handleCreatestmt(ctx *parser.CreatestmtContext) {
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
					columnType := r.getTypeName(colDef.Typename())
					columnTypes[columnName] = columnType
					columnLines[columnName] = colDef.GetStart().GetLine()

					// Check if this column has PRIMARY KEY constraint
					if r.hasColumnPrimaryKeyConstraint(colDef) {
						if !r.isTypeAllowed(columnType) {
							r.addAdvice(columnName, columnType, colDef.GetStart().GetLine())
						}
					}
				}
			}
		}

		// Check table-level PRIMARY KEY constraints
		for _, elem := range allElements {
			if elem.Tableconstraint() != nil {
				r.checkTablePrimaryKey(elem.Tableconstraint(), columnTypes, columnLines)
			}
		}
	}
}

func (*indexPrimaryKeyTypeAllowlistRule) hasColumnPrimaryKeyConstraint(colDef parser.IColumnDefContext) bool {
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

func (r *indexPrimaryKeyTypeAllowlistRule) checkTablePrimaryKey(constraint parser.ITableconstraintContext, columnTypes map[string]string, columnLines map[string]int) {
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
						if !r.isTypeAllowed(columnType) {
							// Use the column definition's line, not the constraint's line
							line := columnLines[columnName]
							r.addAdvice(columnName, columnType, line)
						}
					}
				}
			}
		}
	}
}

func (*indexPrimaryKeyTypeAllowlistRule) getTypeName(typename parser.ITypenameContext) string {
	if typename == nil {
		return ""
	}

	// Get the type text and normalize it
	return normalizePostgreSQLType(typename.GetText())
}

func (r *indexPrimaryKeyTypeAllowlistRule) isTypeAllowed(columnType string) bool {
	// Check if the column type is equivalent to any type in the allowlist
	return isTypeInList(columnType, r.allowlist)
}

func (r *indexPrimaryKeyTypeAllowlistRule) addAdvice(columnName, columnType string, line int) {
	r.AddAdvice(&storepb.Advice{
		Status:  r.level,
		Code:    code.IndexPKType.Int32(),
		Title:   r.title,
		Content: fmt.Sprintf("The column %q is one of the primary key, but its type %q is not in allowlist", columnName, columnType),
		StartPosition: &storepb.Position{
			Line:   int32(line),
			Column: 0,
		},
	})
}
