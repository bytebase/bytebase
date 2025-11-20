package pg

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// Rule defines the interface for individual SQL validation rules.
// Each rule implements specific checking logic without embedding the base listener.
type Rule interface {
	// OnEnter is called when entering a parse tree node
	OnEnter(ctx antlr.ParserRuleContext, nodeType string) error

	// OnExit is called when exiting a parse tree node
	OnExit(ctx antlr.ParserRuleContext, nodeType string) error

	// Name returns the rule name for logging/debugging
	Name() string

	// GetAdviceList returns the accumulated advice from this rule
	GetAdviceList() []*storepb.Advice
}

// GenericChecker embeds the base PostgreSQL parser listener and dispatches events to registered rules.
// This design ensures only one copy of the listener type metadata in the binary.
type GenericChecker struct {
	*parser.BasePostgreSQLParserListener

	rules    []Rule
	baseLine int
}

// NewGenericChecker creates a new instance of GenericChecker with the given rules.
func NewGenericChecker(rules []Rule) *GenericChecker {
	return &GenericChecker{
		BasePostgreSQLParserListener: &parser.BasePostgreSQLParserListener{},
		rules:                        rules,
	}
}

// SetBaseLine sets the base line number for error reporting.
func (g *GenericChecker) SetBaseLine(baseLine int) {
	g.baseLine = baseLine
}

// EnterEveryRule is called when any rule is entered.
// It dispatches the event to all registered rules.
func (g *GenericChecker) EnterEveryRule(ctx antlr.ParserRuleContext) {
	nodeType := g.getNodeType(ctx)
	for _, rule := range g.rules {
		if err := rule.OnEnter(ctx, nodeType); err != nil {
			// Log error if needed
			fmt.Printf("Rule %s error on enter %s: %v\n", rule.Name(), nodeType, err)
		}
	}
}

// ExitEveryRule is called when any rule is exited.
// It dispatches the event to all registered rules.
func (g *GenericChecker) ExitEveryRule(ctx antlr.ParserRuleContext) {
	nodeType := g.getNodeType(ctx)
	for _, rule := range g.rules {
		if err := rule.OnExit(ctx, nodeType); err != nil {
			// Log error if needed
			fmt.Printf("Rule %s error on exit %s: %v\n", rule.Name(), nodeType, err)
		}
	}
}

// GetAdviceList collects and returns all advice from registered rules.
func (g *GenericChecker) GetAdviceList() []*storepb.Advice {
	var allAdvice []*storepb.Advice
	for _, rule := range g.rules {
		allAdvice = append(allAdvice, rule.GetAdviceList()...)
	}
	return allAdvice
}

// getNodeType returns the type name of the parse tree node.
func (*GenericChecker) getNodeType(ctx antlr.ParserRuleContext) string {
	t := reflect.TypeOf(ctx)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	name := t.Name()
	// Remove "Context" suffix if present
	name = strings.TrimSuffix(name, "Context")
	return name
}

// BaseRule provides common functionality for rules.
// Other rules can embed this struct to get common behavior.
type BaseRule struct {
	level      storepb.Advice_Status
	title      string
	adviceList []*storepb.Advice
	baseLine   int
}

// SetBaseLine sets the base line for the rule.
func (r *BaseRule) SetBaseLine(baseLine int) {
	r.baseLine = baseLine
}

// GetAdviceList returns the accumulated advice.
func (r *BaseRule) GetAdviceList() []*storepb.Advice {
	return r.adviceList
}

// AddAdvice adds a new advice to the list.
// Automatically adds baseLine offset to the line number.
func (r *BaseRule) AddAdvice(advice *storepb.Advice) {
	if advice.StartPosition != nil {
		advice.StartPosition.Line += int32(r.baseLine)
	}
	r.adviceList = append(r.adviceList, advice)
}
