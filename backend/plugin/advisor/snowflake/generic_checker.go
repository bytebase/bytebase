package snowflake

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/snowflake"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// Node type constants for consistent node type checking
const (
	NodeTypeCreateTable           = "Create_table"
	NodeTypeAlterTable            = "Alter_table"
	NodeTypeDropTable             = "Drop_table"
	NodeTypeCreateView            = "Create_view"
	NodeTypeDropView              = "Drop_view"
	NodeTypeCreateDatabase        = "Create_database"
	NodeTypeDropDatabase          = "Drop_database"
	NodeTypeCreateSchema          = "Create_schema"
	NodeTypeDropSchema            = "Drop_schema"
	NodeTypeInsertStatement       = "Insert_statement"
	NodeTypeUpdateStatement       = "Update_statement"
	NodeTypeDeleteStatement       = "Delete_statement"
	NodeTypeSelectStatement       = "Select_statement"
	NodeTypeMergeStatement        = "Merge_statement"
	NodeTypeCreateFunction        = "Create_function"
	NodeTypeDropFunction          = "Drop_function"
	NodeTypeCreateProcedure       = "Create_procedure"
	NodeTypeDropProcedure         = "Drop_procedure"
	NodeTypeCreateSequence        = "Create_sequence"
	NodeTypeDropSequence          = "Drop_sequence"
	NodeTypeCreateStage           = "Create_stage"
	NodeTypeDropStage             = "Drop_stage"
	NodeTypeCreateStream          = "Create_stream"
	NodeTypeDropStream            = "Drop_stream"
	NodeTypeCreateTask            = "Create_task"
	NodeTypeDropTask              = "Drop_task"
	NodeTypeUseStatement          = "Use_statement"
	NodeTypeColumnDeclItemList    = "Column_decl_item_list"
	NodeTypeTableColumnAction     = "Table_column_action"
	NodeTypeQueryStatement        = "Query_statement"
	NodeTypeDmlCommand            = "Dml_command"
	NodeTypeDdlCommand            = "Ddl_command"
	NodeTypeOtherCommand          = "Other_command"
	NodeTypeColumnDecl            = "Col_decl"
	NodeTypeColumnList            = "Column_list"
	NodeTypeTableName             = "Table_name"
	NodeTypeObjectName            = "Object_name"
	NodeTypeSelectListElem        = "Select_list_elem"
	NodeTypeSelectClause          = "Select_clause"
	NodeTypeFromClause            = "From_clause"
	NodeTypeWhereClause           = "Where_clause"
	NodeTypeJoinClause            = "Join_clause"
	NodeTypeSetOperators          = "Set_operators"
	NodeTypeID                    = "Id_"
	NodeTypeFullColDecl           = "Full_col_decl"
	NodeTypeColumnName            = "Column_name"
	NodeTypeDataType              = "Data_type"
	NodeTypeVarcharType           = "Varchar"
	NodeTypeStringType            = "String"
	NodeTypeNumberType            = "Number"
	NodeTypeFloatType             = "Float"
	NodeTypeIntType               = "Int"
	NodeTypeSmallintType          = "Smallint"
	NodeTypeTinyintType           = "Tinyint"
	NodeTypeBigintType            = "Bigint"
	NodeTypeByteintType           = "Byteint"
	NodeTypeIdentifier            = "Id_"
	NodeTypeTableConstraints      = "Table_constraints"
	NodeTypeOutOfLineConstraint   = "Out_of_line_constraint"
	NodeTypeInlineConstraint      = "Inline_constraint"
	NodeTypePrimaryKeyClause      = "Primary_key_clause"
	NodeTypeForeignKeyClause      = "Foreign_key_clause"
	NodeTypeSearchCondition       = "Search_condition"
	NodeTypeExpr                  = "Expr"
	NodeTypePredicate             = "Predicate"
	NodeTypePredicatePartLike     = "Predicate_part_like"
	NodeTypeExprListInParentheses = "Expr_list_in_parentheses"
	NodeTypeExecuteImmediate      = "Execute_immediate"
	NodeTypeGrantPrivileges       = "Grant_privileges"
	NodeTypeRevokePrivileges      = "Revoke_privileges"
	NodeTypeCreateExternalTable   = "Create_external_table"
	NodeTypeDropExternalTable     = "Drop_external_table"
	NodeTypeTruncateTable         = "Truncate_table"
	NodeTypeSQLCommand            = "Sql_command"
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

// GenericChecker embeds the base Snowflake parser listener and dispatches events to registered rules.
// This design ensures only one copy of the listener type metadata in the binary.
type GenericChecker struct {
	*parser.BaseSnowflakeParserListener

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
