package mcp

import (
	"strconv"
	"strings"
	"testing"

	"github.com/google/cel-go/cel"
	celast "github.com/google/cel-go/common/ast"
	"github.com/google/cel-go/common/types"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
)

func TestMaskingScopePhrase(t *testing.T) {
	for condition, want := range map[string]string{
		`resource.environment_id == "prod" && resource.database_name == "hr_prod" && resource.table_name == "employee" && resource.column_name == "birth_date"`: `env "prod", db "hr_prod", "employee"."birth_date"`,
		`resource.environment_id == "prod" && resource.table_name == "employee" && resource.column_name in ["birth_date", "gender"]`:                            `env "prod", any db, "employee".("birth_date", "gender")`,
		`resource.table_name == "customers" && resource.column_name == "ssn"`:                                                                                   `any db, "customers"."ssn"`,
		`resource.column_name == "ssn"`:                                                                   `any db, any table, column "ssn"`,
		`resource.column_name in ["email", "ssn"]`:                                                        `any db, any table, columns ("email", "ssn")`,
		`resource.environment_id in ["test"]`:                                                             `env "test", every column`,
		`resource.database_name == "hr"`:                                                                  `db "hr", every column`,
		`resource.table_name == "salary"`:                                                                 `any db, table "salary", every column`,
		`resource.schema_name == "hr"`:                                                                    `any db, schema "hr", every column`,
		`resource.schema_name == "hr" && resource.column_name == "ssn"`:                                   `any db, schema "hr", any table, column "ssn"`,
		`resource.schema_name == "hr" && resource.table_name == "t"`:                                      `any db, schema "hr", table "t", every column`,
		`resource.column_name == "c" && resource.table_name == "t" && resource.schema_name == "hr"`:       `any db, "hr"."t"."c"`,
		`resource.project_id == "p" && resource.instance_id in ["a", "b"]`:                                `project "p", instance ("a", "b"), every column`,
		`(resource.database_name == "hr" && (resource.table_name == "t")) && resource.column_name == "c"`: `db "hr", "t"."c"`,
		`resource.table_name == "a.b" && resource.column_name == "c\"d"`:                                  `any db, "a.b"."c\"d"`,
		`resource.schema_name == ""`:                                                                      `any db, schema "", every column`,
	} {
		scope, ok := parseMaskingScope(condition)
		require.True(t, ok, condition)
		require.Equal(t, want, scope.phrase(), condition)
	}
}

// These shapes are not an && chain of string tests on distinct attributes, so
// the form shows their CEL.
func TestMaskingScopeLeavesOtherConditionsAsCEL(t *testing.T) {
	for _, condition := range []string{
		`resource.table_name != "t"`,
		`resource.column_name.startsWith("x")`,
		`resource.table_name == "t" || resource.column_name == "c"`,
		`resource.table_name == "t" && resource.table_name == "u"`,
		`resource.column_name in []`,
		`"t" == resource.table_name`,
		`has(resource.table_name)`,
		`.resource.table_name == "t"`,
		`resource.classification_level >= 2`,
		`resource.classification_level == 2`,
		`resource.environment_name == "prod"`,
		`resource.table_name == b"t"`,
		`resource.table_name == resource.column_name`,
		`true`,
		`resource.table_name == "t" && true`,
		`!(resource.table_name == "t")`,
		`not CEL`,
	} {
		_, ok := parseMaskingScope(condition)
		require.False(t, ok, condition)
		require.Equal(t, conditionText(condition), maskingConditionText(condition), condition)
	}
}

// The phrase is what the user approves, so it must state exactly what masking
// evaluates: decode the phrase as a reader would, and compare its reach with
// cel-go over every mix of the values the condition names, another value and
// "", with every attribute bound as masking binds it.
func TestMaskingScopePhraseAgreesWithMaskingEvaluation(t *testing.T) {
	env, err := cel.NewEnv(common.MaskingRulePolicyCELAttributes...)
	require.NoError(t, err)
	for _, condition := range []string{
		`resource.environment_id == "prod" && resource.database_name == "hr_prod" && resource.table_name == "employee" && resource.column_name == "birth_date"`,
		`resource.environment_id == "prod" && resource.table_name == "employee" && resource.column_name in ["birth_date", "gender"]`,
		`resource.column_name in ["email", "ssn", "email"]`,
		`resource.environment_id in ["test"] && resource.project_id == "p"`,
		`resource.instance_id == "i" && resource.database_name in ["a", "b"]`,
		`resource.schema_name == "hr" && resource.table_name == "t" && resource.column_name == "c"`,
		`resource.schema_name == "hr" && resource.column_name in ["c", "d"]`,
		`resource.table_name in ["t", "u"] && resource.schema_name == ""`,
		`resource.table_name == "a.b" && resource.column_name == "x, y"`,
		`resource.column_name == "(c)" && resource.table_name == "名前"`,
		`resource.column_name == "line\nbreak" && resource.database_name == "q\"uote"`,
		`resource.environment_id == ""`,
	} {
		scope, ok := parseMaskingScope(condition)
		require.True(t, ok, condition)
		reach := decodeScopePhrase(t, scope.phrase())
		ast, issues := env.Compile(condition)
		require.NoError(t, issues.Err(), condition)
		program, err := env.Program(ast)
		require.NoError(t, err)
		named := conditionValues(t, env, condition)
		var values [][]string
		for _, attribute := range maskingScopeAttributes {
			values = append(values, append(append([]string{"zz", ""}, named[attribute]...), reach[attribute]...))
		}
		rows := 0
		for row := range attributeRows(values) {
			bound := map[string]any{common.CELAttributeResourceClassificationLevel: int64(0)}
			matches := true
			for i, attribute := range maskingScopeAttributes {
				bound["resource."+attribute] = row[i]
				if want, ok := reach[attribute]; ok && !containsString(want, row[i]) {
					matches = false
				}
			}
			out, _, err := program.Eval(bound)
			require.NoError(t, err, condition)
			require.Equal(t, out.Value(), matches, "%s\nphrase: %s\nrow: %v", condition, scope.phrase(), bound)
			rows++
		}
		require.Positive(t, rows)
	}
}

// The BYT-10598 rule now reads in one line, so its removal fits with the
// unmask line of its own.
func TestAdministerMaskingRulesNamesARuleByItsScope(t *testing.T) {
	mock := &maskingMock{}
	mock.setStored(maskingRule("r1", "Salary amounts", `resource.database_name == "hr_test" && resource.table_name == "salary" && resource.column_name == "amount"`, "bb.default"))
	user := &asker{answer: &mcp.ElicitResult{Action: "decline"}}
	_, session := connectToMaskingServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})

	res := callAdminister(t, session, maskingParams(maskingBody(t, maskingRequest(false))))
	require.Contains(t, resultText(t, res), "Not changed: the user declined")
	lines := strings.Split(user.lastParams(t).Message, "\n")
	require.Equal(t, []string{
		`Change the masking rules, as alice@example.com?`,
		`Remove, Full mask: db "hr_test", "salary"."amount"`,
		maskingRulesUnmask,
	}, lines[:3])
}

// decodeScopePhrase reads a phrase back into the values each named attribute
// may hold, by the grammar a user reads it with.
func decodeScopePhrase(t *testing.T, phrase string) map[string][]string {
	t.Helper()
	labels := map[string]string{"env": "environment_id", "project": "project_id", "instance": "instance_id", "db": "database_name", "schema": "schema_name", "table": "table_name", "column": "column_name", "columns": "column_name"}
	reach := map[string][]string{}
	for _, part := range splitScope(t, phrase, ", ") {
		if part == "any db" || part == "any table" || part == "every column" {
			continue
		}
		if label, value, found := strings.Cut(part, " "); found && labels[label] != "" {
			reach[labels[label]] = decodeScopeValues(t, value)
			continue
		}
		path := splitScope(t, part, ".")
		require.Contains(t, []int{2, 3}, len(path), part)
		levels := []string{"table_name", "column_name"}
		if len(path) == 3 {
			levels = []string{"schema_name", "table_name", "column_name"}
		}
		for i, level := range levels {
			reach[level] = decodeScopeValues(t, path[i])
		}
	}
	return reach
}

func decodeScopeValues(t *testing.T, text string) []string {
	t.Helper()
	if inner, ok := strings.CutPrefix(text, "("); ok {
		inner = strings.TrimSuffix(inner, ")")
		var values []string
		for _, item := range splitScope(t, inner, ", ") {
			values = append(values, decodeScopeValues(t, item)...)
		}
		return values
	}
	value, err := strconv.Unquote(text)
	require.NoError(t, err, text)
	return []string{value}
}

// splitScope splits text on sep outside quoted values and parentheses.
func splitScope(t *testing.T, text, sep string) []string {
	t.Helper()
	var parts []string
	depth, start := 0, 0
	for i := 0; i < len(text); {
		if text[i] == '"' {
			quoted, err := strconv.QuotedPrefix(text[i:])
			require.NoError(t, err, text)
			i += len(quoted)
			continue
		}
		if depth == 0 && strings.HasPrefix(text[i:], sep) {
			parts = append(parts, text[start:i])
			i += len(sep)
			start = i
			continue
		}
		switch text[i] {
		case '(':
			depth++
		case ')':
			depth--
		default:
		}
		i++
	}
	return append(parts, text[start:])
}

// conditionValues collects the string literals each attribute is compared
// with, from the condition's own tree.
func conditionValues(t *testing.T, env *cel.Env, condition string) map[string][]string {
	t.Helper()
	parsed, issues := env.Parse(condition)
	require.NoError(t, issues.Err())
	values := map[string][]string{}
	var walk func(e celast.Expr)
	walk = func(e celast.Expr) {
		if e.Kind() != celast.CallKind {
			return
		}
		args := e.AsCall().Args()
		if len(args) == 2 && args[0].Kind() == celast.SelectKind {
			attribute := args[0].AsSelect().FieldName()
			literals := []celast.Expr{args[1]}
			if args[1].Kind() == celast.ListKind {
				literals = args[1].AsList().Elements()
			}
			for _, literal := range literals {
				if literal.Kind() == celast.LiteralKind {
					if s, ok := literal.AsLiteral().(types.String); ok {
						values[attribute] = append(values[attribute], string(s))
					}
				}
			}
		}
		for _, arg := range args {
			walk(arg)
		}
	}
	walk(parsed.NativeRep().Expr())
	return values
}

// attributeRows yields every row that takes one value per attribute.
func attributeRows(values [][]string) func(yield func([]string) bool) {
	return func(yield func([]string) bool) {
		row := make([]string, len(values))
		var fill func(i int) bool
		fill = func(i int) bool {
			if i == len(values) {
				return yield(append([]string(nil), row...))
			}
			for _, value := range values[i] {
				row[i] = value
				if !fill(i + 1) {
					return false
				}
			}
			return true
		}
		fill(0)
	}
}

func containsString(values []string, value string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}
