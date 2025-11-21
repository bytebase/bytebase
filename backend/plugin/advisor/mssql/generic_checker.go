package mssql

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/tsql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// Node type constants for consistent node type checking
const (
	NodeTypeCreateTable               = "Create_table"
	NodeTypeAlterTable                = "Alter_table"
	NodeTypeDropTable                 = "Drop_table"
	NodeTypeCreateIndex               = "Create_index"
	NodeTypeDropIndex                 = "Drop_index"
	NodeTypeInsertStatement           = "Insert_statement"
	NodeTypeUpdateStatement           = "Update_statement"
	NodeTypeDeleteStatement           = "Delete_statement"
	NodeTypeSelectStatement           = "Select_statement"
	NodeTypeCreateView                = "Create_view"
	NodeTypeDropView                  = "Drop_view"
	NodeTypeCreateProcedure           = "Create_procedure"
	NodeTypeDropProcedure             = "Drop_procedure"
	NodeTypeCreateFunction            = "Create_function"
	NodeTypeDropFunction              = "Drop_function"
	NodeTypeCreateDatabase            = "Create_database"
	NodeTypeAlterDatabase             = "Alter_database"
	NodeTypeDropDatabase              = "Drop_database"
	NodeTypeTruncateTable             = "Truncate_table"
	NodeTypeColumnDefinition          = "Column_definition"
	NodeTypeColumnDefinitionElement   = "Column_definition_element"
	NodeTypeTableConstraint           = "Table_constraint"
	NodeTypeSearchCondition           = "Search_condition"
	NodeTypeSelectListElem            = "Select_list_elem"
	NodeTypeTableConstraintElement    = "Table_constraint_element"
	NodeTypeDdlStatement              = "Ddl_statement"
	NodeTypeDmlStatement              = "Dml_statement"
	NodeTypeAnothStatement            = "Another_statement"
	NodeTypeCreateOrAlterProcedure    = "Create_or_alter_procedure"
	NodeTypeCreateOrAlterFunction     = "Create_or_alter_function"
	NodeTypeTableValueConstructor     = "Table_value_constructor"
	NodeTypeExpressionElem            = "Expression_elem"
	NodeTypeTableNameWithHint         = "Table_name_with_hint"
	NodeTypeTableName                 = "Table_name"
	NodeTypeSimpleID                  = "Simple_id"
	NodeTypeID                        = "Id_"
	NodeTypeKeyword                   = "Keyword"
	NodeTypeFullColumnNameList        = "Full_column_name_list"
	NodeTypeColumnElemList            = "Column_elem_list"
	NodeTypeCreateType                = "Create_type"
	NodeTypePrimaryKeyOptions         = "Primary_key_options"
	NodeTypeForeignKeyOptions         = "Foreign_key_options"
	NodeTypeTableConstraintPrimaryKey = "Pk"
	NodeTypeTableConstraintForeignKey = "Fk"
	NodeTypeCommonTableExpression     = "Common_table_expression"
	NodeTypeWithExpression            = "With_expression"
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

// GenericChecker embeds the base TSQL parser listener and dispatches events to registered rules.
// This design ensures only one copy of the listener type metadata in the binary.
type GenericChecker struct {
	*parser.BaseTSqlParserListener

	rules []Rule
}

// NewGenericChecker creates a new instance of GenericChecker with the given rules.
func NewGenericChecker(rules []Rule) *GenericChecker {
	return &GenericChecker{
		rules: rules,
	}
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

// GetAdviceList returns the accumulated advice.
func (r *BaseRule) GetAdviceList() []*storepb.Advice {
	return r.adviceList
}

// AddAdvice adds a new advice to the list.
// It automatically adjusts the StartPosition by adding the baseLine offset.
func (r *BaseRule) AddAdvice(advice *storepb.Advice) {
	// Adjust the line number by adding baseLine offset
	if advice.StartPosition != nil && r.baseLine > 0 {
		advice.StartPosition.Line += int32(r.baseLine)
	}
	r.adviceList = append(r.adviceList, advice)
}

// SetBaseLine sets the base line number for multi-statement SQL.
func (r *BaseRule) SetBaseLine(baseLine int) {
	r.baseLine = baseLine
}
