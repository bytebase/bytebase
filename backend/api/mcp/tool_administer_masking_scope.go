package mcp

import (
	"slices"
	"strings"
	"sync"

	"github.com/google/cel-go/cel"
	celast "github.com/google/cel-go/common/ast"
	"github.com/google/cel-go/common/operators"

	"github.com/bytebase/bytebase/backend/common"
)

// maskingScopeAttributes are the string attributes of a masking condition, in
// the order a scope phrase names them.
var maskingScopeAttributes = []string{"environment_id", "project_id", "instance_id", "database_name", "schema_name", "table_name", "column_name"}

// maskingScopeEnv is the CEL environment a masking condition is parsed in,
// built once: a form parses a condition for each rule it shows, and cel-go's
// parser keeps its state per call (parser.Parser.Parse), so every form shares
// one environment.
var maskingScopeEnv = sync.OnceValues(func() (*cel.Env, error) {
	return cel.NewEnv(common.MaskingRulePolicyCELAttributes...)
})

// maskingScope is what a masking condition matches when it is an && chain of
// string tests: the values each attribute it names may hold.
type maskingScope map[string][]string

// parseMaskingScope reads condition as an && chain with one term per attribute,
// each resource.<attribute> == "v" or resource.<attribute> in ["v", …]. Every
// attribute is bound when masking evaluates a rule, so such a chain matches
// exactly the columns its terms name, in any term order. Any other condition
// reports false.
func parseMaskingScope(condition string) (maskingScope, bool) {
	env, err := maskingScopeEnv()
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
	if db := s["database_name"]; db != nil {
		parts = append(parts, "db "+scopeValues(db))
	} else if schema != nil || table != nil || columns != nil {
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
	case len(columns) == 1:
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

// conditionText states a masking rule's condition: as its scope phrase, or as
// typed when it has none, or when another rule of the stored or the requested
// list reads as the same scope. Two conditions can spell one scope, and a line
// that removes or changes one of the two rules has to say which.
func (d *maskingRulesDiff) conditionText(condition string) string {
	if text, ok := d.texts[condition]; ok {
		return text
	}
	text := conditionText(condition)
	if scope := d.scope(condition); scope != nil && !d.sharesScope(condition, scope) {
		text = scope.phrase()
	}
	if d.texts == nil {
		d.texts = map[string]string{}
	}
	d.texts[condition] = text
	return text
}

// scope is condition's scope, or nil when it has none.
func (d *maskingRulesDiff) scope(condition string) maskingScope {
	if scope, ok := d.scopes[condition]; ok {
		return scope
	}
	scope, _ := parseMaskingScope(condition)
	if d.scopes == nil {
		d.scopes = map[string]maskingScope{}
	}
	d.scopes[condition] = scope
	return scope
}

// sharesScope reports whether another condition of the lists reads as scope.
// A condition that does holds each value of the scope, so only the conditions
// that hold the value the fewest hold are parsed: a form states a few rules of
// a list that can hold thousands.
func (d *maskingRulesDiff) sharesScope(condition string, scope maskingScope) bool {
	if d.literals == nil {
		d.indexLiterals()
	}
	var fewest []string
	picked := false
	for _, values := range scope {
		for _, value := range values {
			if held := d.literals[value]; !picked || len(held) < len(fewest) {
				fewest, picked = held, true
			}
		}
	}
	phrase := scope.phrase()
	for _, others := range [][]string{fewest, d.unruled} {
		for _, other := range others {
			if other == condition {
				continue
			}
			if otherScope := d.scope(other); otherScope != nil && otherScope.phrase() == phrase {
				return true
			}
		}
	}
	return false
}

// indexLiterals maps each text that stands between two quotes of one kind in
// a condition to the conditions it stands in. A string literal with no escape
// holds its value as written between two such quotes, unless it is
// triple-quoted: only such a literal can hold a quote of its own kind, or a
// carriage return, which the parser reads as a line feed (cel-go,
// parser/gen/CEL.g4, STRING; parser/unescape.go). An escape starts with a
// backslash. So a condition with a backslash or a triple quote goes to
// unruled, which every search looks at.
func (d *maskingRulesDiff) indexLiterals() {
	d.literals = map[string][]string{}
	for _, condition := range d.conditions {
		if strings.Contains(condition, `\`) || strings.Contains(condition, `"""`) || strings.Contains(condition, `'''`) {
			d.unruled = append(d.unruled, condition)
			continue
		}
		for _, quote := range []string{`"`, `'`} {
			for _, text := range strings.Split(condition, quote) {
				d.literals[text] = append(d.literals[text], condition)
			}
		}
	}
}
