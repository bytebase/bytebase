package tidb

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/tidb/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	advisorcode "github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*WhereRequirementForSelectAdvisor)(nil)
	_ ast.Visitor     = (*whereRequireSelectVisitor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_SELECT, &WhereRequirementForSelectAdvisor{})
}

// WhereRequirementForSelectAdvisor checks the WHERE clause requirement for
// SELECT statements.
type WhereRequirementForSelectAdvisor struct {
}

// Check is Recipe B (sub-walk via omni Visitor) — the pingcap-typed
// version's Visitor returns (in, false) and so recurses into sub-selects;
// the existing fixture
//
//	SELECT id FROM tech_book WHERE id > (SELECT max(id) FROM tech_book)
//
// expects an advice on the inner SELECT (outer has a WHERE; trigger must
// come from the inner subquery). A naive Recipe-A top-level type-switch
// would silently regress on this case — same shape of bug as batch 3's
// no_select_all round-1 missed-qualified-wildcard.
func (*WhereRequirementForSelectAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, err := getTiDBOmniNodes(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice
	for _, ostmt := range stmts {
		// Pingcap parity: every match reports the OUTER statement's
		// first-token line and trimmed text, not the inner SelectStmt's
		// position — checker.text/line are set ONCE per top-level statement
		// in the pingcap version (before Accept()).
		v := &whereRequireSelectVisitor{
			level:   level,
			title:   title,
			text:    ostmt.TrimmedText(),
			line:    ostmt.FirstTokenLine(),
			advices: &adviceList,
		}
		ast.Walk(v, ostmt.Node)
	}
	return adviceList, nil
}

type whereRequireSelectVisitor struct {
	level   storepb.Advice_Status
	title   string
	text    string
	line    int
	advices *[]*storepb.Advice
}

// Visit returns v to recurse into children, including SelectStmt.Left/Right
// (UNION arms) per omni's Walk contract.
func (v *whereRequireSelectVisitor) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		return nil
	}
	sel, ok := node.(*ast.SelectStmt)
	if !ok {
		return v
	}
	// Cumulative shape divergence #10: omni's SelectStmt.From is []TableExpr
	// (slice), not pingcap's *TableRefsClause (pointer). len(From) > 0
	// is the correct "has FROM" check; `From != nil` would be wrong on
	// empty-but-non-nil slices.
	//
	// Allow SELECT without FROM (e.g. SELECT 1, SELECT CURDATE()).
	if sel.Where == nil && len(sel.From) > 0 {
		*v.advices = append(*v.advices, &storepb.Advice{
			Status:        v.level,
			Code:          advisorcode.StatementNoWhere.Int32(),
			Title:         v.title,
			Content:       fmt.Sprintf("\"%s\" requires WHERE clause", v.text),
			StartPosition: common.ConvertANTLRLineToPosition(v.line),
		})
	}
	return v
}
