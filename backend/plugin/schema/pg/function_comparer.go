package pg

import (
	"strings"

	"github.com/antlr4-go/antlr/v4"
	pgparser "github.com/bytebase/parser/postgresql"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/plugin/schema/pg/ast"
)

func init() {
	// Register PostgreSQL-specific function comparer
	schema.RegisterFunctionComparer(storepb.Engine_POSTGRES, &PostgreSQLFunctionComparer{})
}

// PostgreSQLFunctionComparer provides PostgreSQL-specific function comparison logic.
type PostgreSQLFunctionComparer struct {
	schema.DefaultFunctionComparer
}

// FunctionChangeType represents the type of change detected in a function.
type FunctionChangeType int

const (
	FunctionChangeNone FunctionChangeType = iota
	FunctionChangeBodyOnly
	FunctionChangeSignature
	FunctionChangeAttributes
	FunctionChangeBoth // Both signature and body changed
)

// CompareDetailed compares two functions with PostgreSQL-specific logic using ANTLR parsing.
func (*PostgreSQLFunctionComparer) CompareDetailed(oldFunc, newFunc *storepb.FunctionMetadata) (*schema.FunctionComparisonResult, error) {
	if oldFunc == nil || newFunc == nil {
		if oldFunc == newFunc {
			return nil, nil
		}
		return nil, errors.New("cannot compare nil functions")
	}

	result := &schema.FunctionComparisonResult{}

	result.SignatureChanged = compareFunctionSignature(oldFunc, newFunc)

	// Compare function attributes (comment, volatility, etc.)
	attributesChanged, changedAttrs := compareFunctionAttributes(oldFunc, newFunc)
	result.AttributesChanged = attributesChanged
	result.ChangedAttributes = changedAttrs

	// Compare function body using strict string comparison
	bodyChanged := compareFunctionBody(oldFunc.Definition, newFunc.Definition)
	result.BodyChanged = bodyChanged

	// Determine change type and migration strategy
	changeType := determineChangeType(result.SignatureChanged, bodyChanged, attributesChanged)
	requiresRecreation, canUseAlterFunction := determineMigrationStrategy(changeType, changedAttrs)
	result.RequiresRecreation = requiresRecreation
	result.CanUseAlterFunction = canUseAlterFunction

	// If there are no changes, return nil to indicate equality
	if !result.SignatureChanged && !bodyChanged && !attributesChanged {
		return nil, nil
	}

	return result, nil
}

// Equal compares two functions for equality (implements the interface expected by the schema differ).
func (c *PostgreSQLFunctionComparer) Equal(oldFunc, newFunc *storepb.FunctionMetadata) bool {
	if oldFunc == nil || newFunc == nil {
		return oldFunc == newFunc
	}

	result, err := c.CompareDetailed(oldFunc, newFunc)
	if err != nil {
		// Fallback to simple definition comparison on error
		return oldFunc.Definition == newFunc.Definition
	}

	// If result is nil, functions are equal
	if result == nil {
		return true
	}

	// Functions are not equal if there are any changes
	return false
}

// compareFunctionSignature compares function signatures using ANTLR parsing.
func compareFunctionSignature(oldFunc, newFunc *storepb.FunctionMetadata) bool {
	// Extract signature components using ANTLR parsing
	oldSig, err := parseFunctionSignature(oldFunc.Definition)
	if err != nil {
		// Fallback to simple string comparison if parsing fails
		return oldFunc.Signature != newFunc.Signature
	}

	newSig, err := parseFunctionSignature(newFunc.Definition)
	if err != nil {
		// Fallback to simple string comparison if parsing fails
		return oldFunc.Signature != newFunc.Signature
	}

	return !signatureEqual(oldSig, newSig)
}

// parseFunctionSignature parses a function definition using ANTLR and extracts signature components.
func parseFunctionSignature(definition string) (*FunctionSignature, error) {
	// Use ANTLR PostgreSQL parser for proper parsing
	inputStream := antlr.NewInputStream(definition)
	lexer := pgparser.NewPostgreSQLLexer(inputStream)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	parser := pgparser.NewPostgreSQLParser(stream)

	// Parse the SQL statement
	tree := parser.Root()
	if tree == nil {
		return nil, errors.New("failed to parse function definition")
	}

	// Create a visitor to extract function signature information
	visitor := &FunctionSignatureVisitor{}
	result := visitor.Visit(tree)

	if signature, ok := result.(*FunctionSignature); ok {
		return signature, nil
	}

	return nil, errors.New("no function signature found in definition")
}

// FunctionSignature represents the parsed signature of a PostgreSQL function.
type FunctionSignature struct {
	Name       string
	Parameters []FunctionParameter
	ReturnType string
	Language   string
	Volatility string // VOLATILE, STABLE, IMMUTABLE
	Security   string // DEFINER, INVOKER
	Parallel   string // UNSAFE, RESTRICTED, SAFE
	Cost       string
	Rows       string
	Leakproof  bool
	Strict     bool
}

// FunctionParameter represents a function parameter.
type FunctionParameter struct {
	Mode     string // IN, OUT, INOUT, VARIADIC
	Name     string
	DataType string
	Default  string
}

// FunctionSignatureVisitor extracts function signature information from ANTLR parse tree.
type FunctionSignatureVisitor struct {
	*pgparser.BasePostgreSQLParserVisitor
}

// Visit implements the visitor pattern for extracting function signature from ANTLR parse tree.
func (v *FunctionSignatureVisitor) Visit(tree antlr.ParseTree) any {
	switch t := tree.(type) {
	case *pgparser.CreatefunctionstmtContext:
		return v.visitCreateFunctionStmt(t)
	default:
		// Continue visiting children
		for i := 0; i < t.GetChildCount(); i++ {
			child := t.GetChild(i)
			if parseTree, ok := child.(antlr.ParseTree); ok {
				if result := v.Visit(parseTree); result != nil {
					return result
				}
			}
		}
	}
	return nil
}

// visitCreateFunctionStmt extracts signature information from CREATE FUNCTION statement.
func (v *FunctionSignatureVisitor) visitCreateFunctionStmt(ctx *pgparser.CreatefunctionstmtContext) any {
	// In PostgreSQL, CREATE FUNCTION and CREATE PROCEDURE use the same grammar rule
	// We support both FUNCTION and PROCEDURE
	if ctx.FUNCTION() == nil && ctx.PROCEDURE() == nil {
		// This is neither CREATE FUNCTION nor CREATE PROCEDURE - skip it
		return nil
	}

	signature := &FunctionSignature{
		Volatility: "VOLATILE", // Default volatility
		Security:   "INVOKER",  // Default security
	}

	// Extract function name
	if ctx.Func_name() != nil {
		signature.Name = v.extractFunctionName(ctx.Func_name())
	}

	// Extract parameters
	if ctx.Func_args_with_defaults() != nil {
		signature.Parameters = v.extractParameters(ctx.Func_args_with_defaults())
	}

	// Extract return type
	if ctx.Func_return() != nil {
		signature.ReturnType = v.normalizeDataType(ctx.Func_return().GetText())
	}

	// Extract function options
	if ctx.Createfunc_opt_list() != nil {
		v.extractFunctionOptions(ctx.Createfunc_opt_list(), signature)
	}

	return signature
}

// extractFunctionName extracts the function name from func_name context.
func (*FunctionSignatureVisitor) extractFunctionName(ctx pgparser.IFunc_nameContext) string {
	if ctx == nil {
		return ""
	}
	// Get the last part of the qualified name (schema.function -> function)
	text := ctx.GetText()
	parts := strings.Split(text, ".")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return text
}

// extractParameters extracts function parameters from func_args_with_defaults context.
func (v *FunctionSignatureVisitor) extractParameters(ctx pgparser.IFunc_args_with_defaultsContext) []FunctionParameter {
	var parameters []FunctionParameter

	if ctx == nil {
		return parameters
	}

	// Get the parameter list
	if argsList := ctx.Func_args_with_defaults_list(); argsList != nil {
		// Visit each parameter
		for _, argCtx := range argsList.AllFunc_arg_with_default() {
			if param := v.extractParameter(argCtx); param != nil {
				parameters = append(parameters, *param)
			}
		}
	}

	return parameters
}

// extractParameter extracts a single function parameter.
func (v *FunctionSignatureVisitor) extractParameter(ctx pgparser.IFunc_arg_with_defaultContext) *FunctionParameter {
	if ctx == nil || ctx.Func_arg() == nil {
		return nil
	}

	funcArg := ctx.Func_arg()
	param := &FunctionParameter{}

	// Extract parameter mode (IN, OUT, INOUT, VARIADIC)
	if funcArg.Arg_class() != nil {
		param.Mode = strings.ToUpper(funcArg.Arg_class().GetText())
	}

	// Extract parameter name
	if funcArg.Param_name() != nil {
		param.Name = funcArg.Param_name().GetText()
	}

	// Extract parameter type
	if funcArg.Func_type() != nil {
		param.DataType = v.normalizeDataType(funcArg.Func_type().GetText())
	}

	// Extract default value
	if ctx.DEFAULT() != nil && ctx.A_expr() != nil {
		param.Default = ctx.A_expr().GetText()
	} else if ctx.EQUAL() != nil && ctx.A_expr() != nil {
		param.Default = ctx.A_expr().GetText()
	}

	return param
}

// extractFunctionOptions extracts function options like LANGUAGE, VOLATILE, etc.
func (v *FunctionSignatureVisitor) extractFunctionOptions(ctx pgparser.ICreatefunc_opt_listContext, signature *FunctionSignature) {
	if ctx == nil {
		return
	}

	// Visit each option item
	for _, optCtx := range ctx.AllCreatefunc_opt_item() {
		v.extractFunctionOption(optCtx, signature)
	}
}

// extractFunctionOption extracts a single function option.
func (*FunctionSignatureVisitor) extractFunctionOption(ctx pgparser.ICreatefunc_opt_itemContext, signature *FunctionSignature) {
	if ctx == nil {
		return
	}

	// Extract LANGUAGE
	if ctx.LANGUAGE() != nil && ctx.Nonreservedword_or_sconst() != nil {
		signature.Language = strings.ToLower(ctx.Nonreservedword_or_sconst().GetText())
	}

	// Extract volatility
	if ctx.Common_func_opt_item() != nil {
		commonOpt := ctx.Common_func_opt_item()
		if commonOpt.VOLATILE() != nil {
			signature.Volatility = "VOLATILE"
		} else if commonOpt.STABLE() != nil {
			signature.Volatility = "STABLE"
		} else if commonOpt.IMMUTABLE() != nil {
			signature.Volatility = "IMMUTABLE"
		}

		// Extract security
		if commonOpt.SECURITY() != nil {
			if commonOpt.DEFINER() != nil {
				signature.Security = "DEFINER"
			} else if commonOpt.INVOKER() != nil {
				signature.Security = "INVOKER"
			}
		}

		// Extract parallel safety
		if commonOpt.PARALLEL() != nil && commonOpt.Colid() != nil {
			signature.Parallel = strings.ToUpper(commonOpt.Colid().GetText())
		}

		// Extract LEAKPROOF
		if commonOpt.LEAKPROOF() != nil {
			signature.Leakproof = true
		}

		// Extract STRICT
		if commonOpt.STRICT_P() != nil || commonOpt.RETURNS() != nil {
			signature.Strict = true
		}

		// Extract COST
		if commonOpt.COST() != nil && commonOpt.Numericonly() != nil {
			signature.Cost = commonOpt.Numericonly().GetText()
		}

		// Extract ROWS
		if commonOpt.ROWS() != nil && commonOpt.Numericonly() != nil {
			signature.Rows = commonOpt.Numericonly().GetText()
		}
	}
}

// normalizeDataType normalizes PostgreSQL data types for comparison.
func (*FunctionSignatureVisitor) normalizeDataType(dataType string) string {
	if dataType == "" {
		return ""
	}

	// Normalize common PostgreSQL data type aliases
	dataType = strings.ToLower(strings.TrimSpace(dataType))

	// Handle common aliases
	switch dataType {
	case "int4":
		return "integer"
	case "int8":
		return "bigint"
	case "int2":
		return "smallint"
	case "float8":
		return "double precision"
	case "float4":
		return "real"
	case "bool":
		return "boolean"
	case "varchar":
		return "character varying"
	case "char":
		return "character"
	default:
		return dataType
	}
}

// signatureEqual compares two function signatures for equality.
func signatureEqual(sig1, sig2 *FunctionSignature) bool {
	// Compare function name (case-insensitive, ignoring schema qualification)
	if !normalizedFunctionNamesEqual(sig1.Name, sig2.Name) {
		return false
	}

	// Compare parameters
	if len(sig1.Parameters) != len(sig2.Parameters) {
		return false
	}

	for i, param1 := range sig1.Parameters {
		param2 := sig2.Parameters[i]
		if !parameterEqual(param1, param2) {
			return false
		}
	}

	// Compare return type using PostgreSQL-aware type normalization
	if !postgreSQLTypeStringsEqual(sig1.ReturnType, sig2.ReturnType) {
		return false
	}

	return true
}

// parameterEqual compares two function parameters for equality.
func parameterEqual(param1, param2 FunctionParameter) bool {
	// Compare parameter mode
	if !strings.EqualFold(param1.Mode, param2.Mode) {
		return false
	}

	// Compare parameter name (case-insensitive for PostgreSQL)
	if !strings.EqualFold(param1.Name, param2.Name) {
		return false
	}

	// Compare data type using PostgreSQL-aware type normalization
	if !postgreSQLTypeStringsEqual(param1.DataType, param2.DataType) {
		return false
	}

	// Compare default value using semantic comparison
	if !ast.CompareExpressionsSemantically(param1.Default, param2.Default) {
		return false
	}

	return true
}

// compareFunctionAttributes compares function attributes like comment, volatility, etc.
func compareFunctionAttributes(oldFunc, newFunc *storepb.FunctionMetadata) (bool, []string) {
	var changedAttrs []string

	// Compare comment
	if oldFunc.Comment != newFunc.Comment {
		changedAttrs = append(changedAttrs, "comment")
	}

	// Compare other PostgreSQL-specific attributes if they exist in the metadata
	// Note: Most PostgreSQL function attributes are part of the definition,
	// so they would be captured by signature or body comparison

	return len(changedAttrs) > 0, changedAttrs
}

// compareFunctionBody compares function bodies using strict string comparison.
func compareFunctionBody(oldDef, newDef string) bool {
	// Extract the function body from the complete definition
	// Dollar quotes are already removed during extraction
	oldBody := extractFunctionBody(oldDef)
	newBody := extractFunctionBody(newDef)

	// Use strict string comparison for the extracted body content
	return strings.TrimSpace(oldBody) != strings.TrimSpace(newBody)
}

// extractFunctionBody extracts the function body from a complete function definition using ANTLR.
func extractFunctionBody(definition string) string {
	// Use ANTLR to parse and extract the function body
	inputStream := antlr.NewInputStream(definition)
	lexer := pgparser.NewPostgreSQLLexer(inputStream)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	parser := pgparser.NewPostgreSQLParser(stream)

	tree := parser.Root()
	if tree == nil {
		// Fallback to returning the entire definition if parsing fails
		return strings.TrimSpace(definition)
	}

	// Create a visitor to extract the function body
	visitor := &FunctionBodyVisitor{}
	if body := visitor.Visit(tree); body != nil {
		if bodyStr, ok := body.(string); ok {
			return strings.TrimSpace(bodyStr)
		}
	}

	// Fallback to returning the entire definition
	return strings.TrimSpace(definition)
}

// FunctionBodyVisitor extracts function body from ANTLR parse tree.
type FunctionBodyVisitor struct {
	*pgparser.BasePostgreSQLParserVisitor
}

// Visit implements the visitor pattern for extracting function body.
func (v *FunctionBodyVisitor) Visit(tree antlr.ParseTree) any {
	switch t := tree.(type) {
	case *pgparser.CreatefunctionstmtContext:
		return v.visitCreateFunctionStmt(t)
	default:
		// Continue visiting children
		for i := 0; i < t.GetChildCount(); i++ {
			child := t.GetChild(i)
			if parseTree, ok := child.(antlr.ParseTree); ok {
				if result := v.Visit(parseTree); result != nil {
					return result
				}
			}
		}
	}
	return nil
}

// visitCreateFunctionStmt extracts the function body from CREATE FUNCTION/PROCEDURE statement.
// Note: In PostgreSQL, CREATE FUNCTION and CREATE PROCEDURE use the same grammar rule.
// We support both FUNCTION and PROCEDURE as they have similar body extraction logic.
func (v *FunctionBodyVisitor) visitCreateFunctionStmt(ctx *pgparser.CreatefunctionstmtContext) any {
	// In PostgreSQL, CREATE FUNCTION and CREATE PROCEDURE use the same grammar rule
	// We support both FUNCTION and PROCEDURE
	if ctx.FUNCTION() == nil && ctx.PROCEDURE() == nil {
		// This is neither CREATE FUNCTION nor CREATE PROCEDURE - skip it
		return nil
	}

	if ctx.Createfunc_opt_list() == nil {
		return nil
	}

	// Look for AS clause in function options
	for _, optCtx := range ctx.Createfunc_opt_list().AllCreatefunc_opt_item() {
		if optCtx.AS() != nil && optCtx.Func_as() != nil {
			// Extract the function body from func_as
			return v.extractBodyFromFuncAs(optCtx.Func_as())
		}
	}

	return nil
}

// extractBodyFromFuncAs extracts the actual function body from func_as context using ANTLR-based parsing.
func (*FunctionBodyVisitor) extractBodyFromFuncAs(ctx pgparser.IFunc_asContext) string {
	if ctx == nil {
		return ""
	}

	// Use ANTLR parser to properly handle all types of string constants including dollar quotes
	sconstList := ctx.AllSconst()
	if len(sconstList) > 0 {
		// Use the postgresql parser's standard function to extract routine body
		// This properly handles dollar quotes, regular quotes, and escape sequences
		if sconstCtx, ok := sconstList[0].(*pgparser.SconstContext); ok {
			return pgparser.GetRoutineBodyString(sconstCtx)
		}
	}

	return ""
}

// determineChangeType determines the overall type of change in the function.
func determineChangeType(signatureChanged, bodyChanged, attributesChanged bool) FunctionChangeType {
	if !signatureChanged && !bodyChanged && !attributesChanged {
		return FunctionChangeNone
	}

	if signatureChanged && (bodyChanged || attributesChanged) {
		return FunctionChangeBoth
	}

	if signatureChanged {
		return FunctionChangeSignature
	}

	if bodyChanged {
		return FunctionChangeBodyOnly
	}

	if attributesChanged {
		return FunctionChangeAttributes
	}

	return FunctionChangeNone
}

// determineMigrationStrategy determines whether to use ALTER FUNCTION or DROP/CREATE.
func determineMigrationStrategy(changeType FunctionChangeType, changedAttrs []string) (requiresRecreation, canUseAlterFunction bool) {
	switch changeType {
	case FunctionChangeNone:
		return false, false

	case FunctionChangeBodyOnly:
		// Body changes can use CREATE OR REPLACE (ALTER strategy)
		return false, true

	case FunctionChangeSignature:
		// Signature changes require DROP/CREATE
		return true, false

	case FunctionChangeAttributes:
		// Some attributes can be changed with ALTER FUNCTION
		for _, attr := range changedAttrs {
			switch attr {
			case "comment":
				// Comments can be changed separately with COMMENT ON FUNCTION
				canUseAlterFunction = true
			default:
				// Most other attributes require DROP/CREATE
				requiresRecreation = true
			}
		}
		return requiresRecreation, canUseAlterFunction

	case FunctionChangeBoth:
		// Both signature and body/attributes changed - requires DROP/CREATE
		return true, false

	default:
		return true, false
	}
}

// postgreSQLTypeStringsEqual compares two PostgreSQL type strings, handling type aliases and format differences.
func postgreSQLTypeStringsEqual(type1, type2 string) bool {
	// Normalize both types and compare
	normalized1 := normalizePostgreSQLTypeAlias(type1)
	normalized2 := normalizePostgreSQLTypeAlias(type2)

	return strings.EqualFold(normalized1, normalized2)
}

// normalizePostgreSQLTypeAlias normalizes PostgreSQL type names to handle aliases and format differences.
func normalizePostgreSQLTypeAlias(typeName string) string {
	// Trim leading/trailing whitespace and convert to lowercase for comparison
	typeName = strings.TrimSpace(strings.ToLower(typeName))

	// Handle PostgreSQL type aliases
	typeAliases := map[string]string{
		"varchar":                   "character varying",
		"charactervarying":          "character varying", // Handle space-removed version
		"char":                      "character",
		"int":                       "integer",
		"int4":                      "integer",
		"int8":                      "bigint",
		"int2":                      "smallint",
		"float4":                    "real",
		"float8":                    "double precision",
		"doubleprecision":           "double precision", // Handle space-removed version
		"bool":                      "boolean",
		"decimal":                   "numeric",
		"text":                      "text",
		"bytea":                     "bytea",
		"timestamp":                 "timestamp without time zone",
		"timestampwithouttime zone": "timestamp without time zone", // Handle space-removed version
		"timestampwithtimezone":     "timestamp with time zone",    // Handle space-removed version
		"timestamptz":               "timestamp with time zone",
		"time":                      "time without time zone",
		"timewithouttime zone":      "time without time zone", // Handle space-removed version
		"timewithtimezone":          "time with time zone",    // Handle space-removed version
		"timetz":                    "time with time zone",
		"date":                      "date",
		"interval":                  "interval",
		"uuid":                      "uuid",
		"json":                      "json",
		"jsonb":                     "jsonb",
		"xml":                       "xml",
		"money":                     "money",
	}

	// Check if it's an alias
	if canonical, exists := typeAliases[typeName]; exists {
		return canonical
	}

	// Handle parameterized types like VARCHAR(255) -> character varying
	if strings.HasPrefix(typeName, "varchar(") {
		return "character varying"
	}
	if strings.HasPrefix(typeName, "char(") {
		return "character"
	}
	if strings.HasPrefix(typeName, "decimal(") || strings.HasPrefix(typeName, "numeric(") {
		return "numeric"
	}
	if strings.HasPrefix(typeName, "timestamp(") {
		if strings.Contains(typeName, "time zone") {
			return "timestamp with time zone"
		}
		return "timestamp without time zone"
	}
	if strings.HasPrefix(typeName, "time(") {
		if strings.Contains(typeName, "time zone") {
			return "time with time zone"
		}
		return "time without time zone"
	}

	return typeName
}

// normalizedFunctionNamesEqual compares function names ignoring schema qualification.
func normalizedFunctionNamesEqual(name1, name2 string) bool {
	// Extract unqualified names (remove schema prefix)
	unqualified1 := extractUnqualifiedName(name1)
	unqualified2 := extractUnqualifiedName(name2)

	// Compare case-insensitively
	return strings.EqualFold(unqualified1, unqualified2)
}

// extractUnqualifiedName removes schema qualification from a function name.
func extractUnqualifiedName(qualifiedName string) string {
	// Handle schema.function_name -> function_name
	parts := strings.Split(qualifiedName, ".")
	if len(parts) > 1 {
		return parts[len(parts)-1] // Return the last part (function name)
	}
	return qualifiedName
}
