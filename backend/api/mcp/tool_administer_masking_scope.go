package mcp

import (
	"slices"
	"strings"

	"github.com/google/cel-go/cel"
	celast "github.com/google/cel-go/common/ast"
	"github.com/google/cel-go/common/operators"

	"github.com/bytebase/bytebase/backend/common"
)

// maskingScopeAttributes are the string attributes of a masking condition, in
// the order a scope phrase names them.
var maskingScopeAttributes = []string{"environment_id", "project_id", "instance_id", "database_name", "schema_name", "table_name", "column_name"}

// maskingScope is what a masking condition matches when it is an && chain of
// string tests: the values each attribute it names may hold.
type maskingScope map[string][]string

// parseMaskingScope reads condition as an && chain with one term per attribute,
// each resource.<attribute> == "v" or resource.<attribute> in ["v", …]. Every
// attribute is bound when masking evaluates a rule, so such a chain matches
// exactly the columns its terms name, in any term order. Any other condition
// reports false.
func parseMaskingScope(condition string) (maskingScope, bool) {
	env, err := cel.NewEnv(common.MaskingRulePolicyCELAttributes...)
	if err != nil {
		return nil, false
	}
	parsed, issues := env.Parse(condition)
	if issues != nil && issues.Err() != nil {
		return nil, false
	}
	scope := maskingScope{}
	for _, term := range conjuncts(parsed.NativeRep().Expr()) {
		attribute, values, ok := maskingScopeTerm(term)
		if !ok || !slices.Contains(maskingScopeAttributes, attribute) || scope[attribute] != nil {
			return nil, false
		}
		scope[attribute] = values
	}
	return scope, true
}

// maskingScopeTerm returns the attribute and values of one term. Unlike
// resourceValues, a list may stand on any attribute, and an empty list, which
// matches nothing, is refused.
func maskingScopeTerm(e celast.Expr) (string, []string, bool) {
	if e.Kind() != celast.CallKind || len(e.AsCall().Args()) != 2 {
		return "", nil, false
	}
	attribute, ok := resourceAttribute(e.AsCall().Args()[0])
	if !ok {
		return "", nil, false
	}
	right := e.AsCall().Args()[1]
	switch e.AsCall().FunctionName() {
	case operators.Equals:
		value, ok := stringLiteral(right)
		return attribute, []string{value}, ok
	case operators.In:
		values, ok := stringLiterals(right)
		return attribute, values, ok && len(values) > 0
	default:
		return "", nil, false
	}
}

// phrase states the scope in the order environment, project, instance,
// database, then the objects. A level the condition does not name matches any
// value: "any db" says so when a lower level is named, a table and its columns
// read as a quoted path, and a path with no schema matches the table in every
// schema, as the console's masking rules do.
func (s maskingScope) phrase() string {
	var parts []string
	for _, level := range []struct{ attribute, label string }{
		{"environment_id", "env"}, {"project_id", "project"}, {"instance_id", "instance"},
	} {
		if values := s[level.attribute]; values != nil {
			parts = append(parts, level.label+" "+scopeValues(values))
		}
	}
	schema, table, columns := s["schema_name"], s["table_name"], s["column_name"]
	switch {
	case s["database_name"] != nil:
		parts = append(parts, "db "+scopeValues(s["database_name"]))
	case schema != nil || table != nil || columns != nil:
		parts = append(parts, "any db")
	}
	if schema != nil && (table == nil || columns == nil) {
		parts = append(parts, "schema "+scopeValues(schema))
	}
	switch {
	case table != nil && columns != nil:
		path := []string{scopeValues(table), scopeValues(columns)}
		if schema != nil {
			path = append([]string{scopeValues(schema)}, path...)
		}
		parts = append(parts, strings.Join(path, "."))
	case table != nil:
		parts = append(parts, "table "+scopeValues(table)+", every column")
	case columns != nil && len(columns) == 1:
		parts = append(parts, "any table, column "+scopeValues(columns))
	case columns != nil:
		parts = append(parts, "any table, columns "+scopeValues(columns))
	default:
		parts = append(parts, "every column")
	}
	return strings.Join(parts, ", ")
}

// scopeValues quotes one value alone and a list in parentheses, so no value can
// pose as a label or a separator of the phrase.
func scopeValues(values []string) string {
	if len(values) == 1 {
		return quoteAll(values)
	}
	return "(" + quoteAll(values) + ")"
}

// maskingConditionText shows a masking rule's condition as its scope phrase
// when it has one, and as CEL otherwise.
func maskingConditionText(condition string) string {
	if scope, ok := parseMaskingScope(condition); ok {
		return scope.phrase()
	}
	return conditionText(condition)
}
