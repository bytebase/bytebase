package oracle

import (
	"github.com/antlr4-go/antlr/v4"
	"github.com/bytebase/omni/oracle/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	plsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/plsql"
)

// OmniRule defines the Oracle omni SQL review rule interface.
type OmniRule interface {
	OnStatement(node ast.Node)
	Name() string
	GetAdviceList() ([]*storepb.Advice, error)
}

// RunOmniRules dispatches parsed Oracle omni AST nodes to rules.
func RunOmniRules(stmts []base.ParsedStatement, rules []OmniRule) ([]*storepb.Advice, error) {
	for _, stmt := range stmts {
		if stmt.AST == nil {
			continue
		}
		node, ok := plsqlparser.GetOmniNode(stmt.AST)
		if !ok {
			rulesForANTLR := make([]Rule, 0, len(rules))
			for _, rule := range rules {
				if legacyRule, ok := rule.(Rule); ok {
					rulesForANTLR = append(rulesForANTLR, legacyRule)
				}
			}
			if len(rulesForANTLR) == 0 {
				continue
			}
			antlrAST, ok := base.GetANTLRAST(stmt.AST)
			if !ok {
				continue
			}
			checker := NewGenericChecker(rulesForANTLR)
			for _, rule := range rulesForANTLR {
				if br, ok := rule.(interface{ SetStatement(int, string) }); ok {
					br.SetStatement(stmt.BaseLine(), stmt.Text)
				}
			}
			checker.SetBaseLine(stmt.BaseLine())
			antlr.ParseTreeWalkerDefault.Walk(checker, antlrAST.Tree)
			continue
		}
		for _, rule := range rules {
			if br, ok := rule.(interface{ SetStatement(int, string) }); ok {
				br.SetStatement(stmt.BaseLine(), stmt.Text)
			}
			rule.OnStatement(node)
		}
	}

	var adviceList []*storepb.Advice
	for _, rule := range rules {
		list, err := rule.GetAdviceList()
		if err != nil {
			return nil, err
		}
		adviceList = append(adviceList, list...)
	}
	return adviceList, nil
}
