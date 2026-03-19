package pg

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/pg/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*IndexPrimaryKeyTypeAllowlistAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_INDEX_PRIMARY_KEY_TYPE_ALLOWLIST, &IndexPrimaryKeyTypeAllowlistAdvisor{})
}

// IndexPrimaryKeyTypeAllowlistAdvisor is the advisor checking for primary key type allowlist.
type IndexPrimaryKeyTypeAllowlistAdvisor struct {
}

// Check checks for primary key type allowlist.
func (*IndexPrimaryKeyTypeAllowlistAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	stringArrayPayload := checkCtx.Rule.GetStringArrayPayload()

	rule := &indexPrimaryKeyTypeAllowlistRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
		allowlist: stringArrayPayload.List,
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type indexPrimaryKeyTypeAllowlistRule struct {
	OmniBaseRule

	allowlist []string
}

func (*indexPrimaryKeyTypeAllowlistRule) Name() string {
	return "index_primary_key_type_allowlist"
}

func (r *indexPrimaryKeyTypeAllowlistRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateStmt:
		r.handleCreateStmt(n)
	default:
	}
}

func (r *indexPrimaryKeyTypeAllowlistRule) handleCreateStmt(n *ast.CreateStmt) {
	cols, constraints := omniTableElements(n)

	// Build column name -> type mapping
	columnTypes := make(map[string]string)
	for _, col := range cols {
		typeName := normalizePostgreSQLType(omniTypeNameFull(col.TypeName))
		columnTypes[col.Colname] = typeName

		// Check column-level PRIMARY KEY constraint
		for _, c := range omniColumnConstraints(col) {
			if c.Contype == ast.CONSTR_PRIMARY {
				if !isTypeInList(typeName, r.allowlist) {
					r.addTypeAdvice(col.Colname, typeName, r.FindLineByName(col.Colname))
				}
			}
		}
	}

	// Check table-level PRIMARY KEY constraints
	for _, c := range constraints {
		if c.Contype != ast.CONSTR_PRIMARY {
			continue
		}
		for _, colName := range omniConstraintColumns(c) {
			if colType, ok := columnTypes[colName]; ok {
				if !isTypeInList(colType, r.allowlist) {
					r.addTypeAdvice(colName, colType, r.FindLineByName(colName))
				}
			}
		}
	}
}

func (r *indexPrimaryKeyTypeAllowlistRule) addTypeAdvice(columnName, columnType string, line int32) {
	r.AddAdvice(&storepb.Advice{
		Status:  r.Level,
		Code:    code.IndexPKType.Int32(),
		Title:   r.Title,
		Content: fmt.Sprintf("The column %q is one of the primary key, but its type %q is not in allowlist", columnName, columnType),
		StartPosition: &storepb.Position{
			Line:   line,
			Column: 0,
		},
	})
}
