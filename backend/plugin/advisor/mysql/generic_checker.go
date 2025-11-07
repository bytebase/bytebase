package mysql

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/bytebase/parser/mysql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// Node type constants for consistent node type checking
const (
	NodeTypeCreateTable             = "CreateTable"
	NodeTypeAlterTable              = "AlterTable"
	NodeTypeAlterStatement          = "AlterStatement"
	NodeTypeDropTable               = "DropTable"
	NodeTypeRenameTableStatement    = "RenameTableStatement"
	NodeTypeSetStatement            = "SetStatement"
	NodeTypeCreateIndex             = "CreateIndex"
	NodeTypeDropIndex               = "DropIndex"
	NodeTypeInsertStatement         = "InsertStatement"
	NodeTypeUpdateStatement         = "UpdateStatement"
	NodeTypeDeleteStatement         = "DeleteStatement"
	NodeTypeSelectStatement         = "SelectStatement"
	NodeTypeCreateView              = "CreateView"
	NodeTypeDropView                = "DropView"
	NodeTypeCreateProcedure         = "CreateProcedure"
	NodeTypeDropProcedure           = "DropProcedure"
	NodeTypeCreateFunction          = "CreateFunction"
	NodeTypeDropFunction            = "DropFunction"
	NodeTypeCreateEvent             = "CreateEvent"
	NodeTypeDropEvent               = "DropEvent"
	NodeTypeCreateTrigger           = "CreateTrigger"
	NodeTypeDropTrigger             = "DropTrigger"
	NodeTypeQuery                   = "Query"
	NodeTypeQueryExpression         = "QueryExpression"
	NodeTypeFunctionCall            = "FunctionCall"
	NodeTypeCreateDatabase          = "CreateDatabase"
	NodeTypeAlterDatabase           = "AlterDatabase"
	NodeTypeDropDatabase            = "DropDatabase"
	NodeTypeTruncateTableStatement  = "TruncateTableStatement"
	NodeTypeSelectStatementWithInto = "SelectStatementWithInto"
	NodeTypeSelectItemList          = "SelectItemList"
	NodeTypePureIdentifier          = "PureIdentifier"
	NodeTypeIdentifierKeyword       = "IdentifierKeyword"
	NodeTypeTransactionStatement    = "TransactionStatement"
	NodeTypePredicateExprLike       = "PredicateExprLike"
	NodeTypeSimpleStatement         = "SimpleStatement"
	NodeTypeQuerySpecification      = "QuerySpecification"
	NodeTypeLimitClause             = "LimitClause"
	NodeTypeFromClause              = "FromClause"
	NodeTypePrimaryExprCompare      = "PrimaryExprCompare"
	NodeTypeJoinedTable             = "JoinedTable"
	NodeTypeWhereClause             = "WhereClause"
	NodeTypePredicateExprIn         = "PredicateExprIn"
	NodeTypeExprList                = "ExprList"
	NodeTypeExprOr                  = "ExprOr"
	NodeTypeAlterTableActions       = "AlterTableActions"
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

// GenericChecker embeds the base MySQL parser listener and dispatches events to registered rules.
// This design ensures only one copy of the listener type metadata in the binary.
type GenericChecker struct {
	*mysql.BaseMySQLParserListener

	rules    []Rule
	baseLine int
}

// NewGenericChecker creates a new instance of GenericChecker with the given rules.
func NewGenericChecker(rules []Rule) *GenericChecker {
	return &GenericChecker{
		rules: rules,
	}
}

// SetBaseLine sets the base line number for error reporting.
func (g *GenericChecker) SetBaseLine(baseLine int) {
	g.baseLine = baseLine
}

// GetBaseLine returns the current base line number.
func (g *GenericChecker) GetBaseLine() int {
	return g.baseLine
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
func (r *BaseRule) AddAdvice(advice *storepb.Advice) {
	r.adviceList = append(r.adviceList, advice)
}
