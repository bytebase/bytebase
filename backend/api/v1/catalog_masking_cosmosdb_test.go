package v1

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/component/masker"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	cosmosdbparser "github.com/bytebase/bytebase/backend/plugin/parser/cosmosdb"
)

// cosmosDBTestSchema returns a schema for a container with:
//
//	name (string), email (string, masked), country (string), population (number),
//	_ts (number), contact.phone (string, masked), contact.city (string)
func cosmosDBTestSchema() *storepb.ObjectSchema {
	return &storepb.ObjectSchema{
		Type: storepb.ObjectSchema_OBJECT,
		Kind: &storepb.ObjectSchema_StructKind_{
			StructKind: &storepb.ObjectSchema_StructKind{
				Properties: map[string]*storepb.ObjectSchema{
					"name":       {Type: storepb.ObjectSchema_STRING},
					"email":      {Type: storepb.ObjectSchema_STRING, SemanticType: "bb.default"},
					"country":    {Type: storepb.ObjectSchema_STRING},
					"population": {Type: storepb.ObjectSchema_NUMBER},
					"status":     {Type: storepb.ObjectSchema_STRING},
					"age":        {Type: storepb.ObjectSchema_NUMBER},
					"score":      {Type: storepb.ObjectSchema_NUMBER},
					"_ts":        {Type: storepb.ObjectSchema_NUMBER},
					"_etag":      {Type: storepb.ObjectSchema_STRING},
					"contact": {
						Type: storepb.ObjectSchema_OBJECT,
						Kind: &storepb.ObjectSchema_StructKind_{
							StructKind: &storepb.ObjectSchema_StructKind{
								Properties: map[string]*storepb.ObjectSchema{
									"phone": {Type: storepb.ObjectSchema_STRING, SemanticType: "bb.default"},
									"city":  {Type: storepb.ObjectSchema_STRING},
								},
							},
						},
					},
				},
			},
		},
	}
}

func cosmosDBMaskers() map[string]masker.Masker {
	return map[string]masker.Masker{
		"bb.default": masker.NewDefaultFullMasker(),
	}
}

// getCosmosDBQuerySpan parses a CosmosDB SQL query and returns the query span.
func getCosmosDBQuerySpan(t *testing.T, query string) *base.QuerySpan {
	t.Helper()
	span, err := cosmosdbparser.GetQuerySpan(context.TODO(), base.GetQuerySpanContext{}, base.Statement{Text: query}, "", "", false)
	require.NoError(t, err)
	return span
}

// maskCosmosDBDoc parses the query, gets span, and masks the document.
func maskCosmosDBDoc(t *testing.T, query string, inputJSON string, schema *storepb.ObjectSchema, maskers map[string]masker.Masker) string {
	t.Helper()
	span := getCosmosDBQuerySpan(t, query)

	var doc map[string]any
	require.NoError(t, json.Unmarshal([]byte(inputJSON), &doc))

	masked, err := maskCosmosDB(span, doc, schema, maskers)
	require.NoError(t, err)

	out, err := json.Marshal(masked)
	require.NoError(t, err)
	return string(out)
}

// ---------------------------------------------------------------------------
// Test: Result masking with new syntax
// ---------------------------------------------------------------------------

func TestCosmosDBMaskingSelectTop(t *testing.T) {
	// SELECT TOP N should mask sensitive fields just like regular SELECT.
	query := `SELECT TOP 10 c.name, c.email FROM c`
	input := `{"name":"Alice","email":"alice@example.com"}`
	want := `{"name":"Alice","email":"******"}`

	got := maskCosmosDBDoc(t, query, input, cosmosDBTestSchema(), cosmosDBMaskers())
	requireJSONEqual(t, want, got)
}

func TestCosmosDBMaskingSelectStar(t *testing.T) {
	// SELECT * masks all fields based on schema.
	query := `SELECT * FROM c`
	input := `{"name":"Alice","email":"alice@example.com","country":"US"}`
	want := `{"name":"Alice","email":"******","country":"US"}`

	got := maskCosmosDBDoc(t, query, input, cosmosDBTestSchema(), cosmosDBMaskers())
	requireJSONEqual(t, want, got)
}

func TestCosmosDBMaskingSelectTopStar(t *testing.T) {
	// SELECT TOP N * masks like SELECT *.
	query := `SELECT TOP 5 * FROM c`
	input := `{"name":"Alice","email":"alice@example.com"}`
	want := `{"name":"Alice","email":"******"}`

	got := maskCosmosDBDoc(t, query, input, cosmosDBTestSchema(), cosmosDBMaskers())
	requireJSONEqual(t, want, got)
}

func TestCosmosDBMaskingFunctionsInSelect(t *testing.T) {
	// Functions in SELECT produce computed values. The function result column
	// (aliased) shouldn't match any schema field, so it should pass through unmasked.
	// Direct field references should still be masked.
	query := `SELECT c.name, c.email, UPPER(c.country) AS upperCountry FROM c`
	input := `{"name":"Alice","email":"alice@example.com","upperCountry":"US"}`
	want := `{"name":"Alice","email":"******","upperCountry":"US"}`

	got := maskCosmosDBDoc(t, query, input, cosmosDBTestSchema(), cosmosDBMaskers())
	requireJSONEqual(t, want, got)
}

func TestCosmosDBMaskingAggregation(t *testing.T) {
	// Aggregation result: COUNT(1) AS totalRecords.
	// The result is a computed value, not a direct field reference.
	query := `SELECT COUNT(1) AS totalRecords FROM c`
	input := `{"totalRecords":42}`
	want := `{"totalRecords":42}`

	got := maskCosmosDBDoc(t, query, input, cosmosDBTestSchema(), cosmosDBMaskers())
	requireJSONEqual(t, want, got)
}

func TestCosmosDBMaskingGroupBy(t *testing.T) {
	// GROUP BY with aggregation. Direct field (country) + function result (cnt).
	query := `SELECT c.country, COUNT(1) AS cnt FROM c GROUP BY c.country`
	input := `{"country":"US","cnt":5}`
	want := `{"country":"US","cnt":5}`

	got := maskCosmosDBDoc(t, query, input, cosmosDBTestSchema(), cosmosDBMaskers())
	requireJSONEqual(t, want, got)
}

func TestCosmosDBMaskingOrderBy(t *testing.T) {
	// ORDER BY doesn't affect masking.
	query := `SELECT c.name, c.email FROM c ORDER BY c.name ASC`
	input := `{"name":"Alice","email":"alice@example.com"}`
	want := `{"name":"Alice","email":"******"}`

	got := maskCosmosDBDoc(t, query, input, cosmosDBTestSchema(), cosmosDBMaskers())
	requireJSONEqual(t, want, got)
}

func TestCosmosDBMaskingOffsetLimit(t *testing.T) {
	// OFFSET LIMIT doesn't affect masking.
	query := `SELECT c.name, c.email FROM c ORDER BY c.name OFFSET 0 LIMIT 10`
	input := `{"name":"Alice","email":"alice@example.com"}`
	want := `{"name":"Alice","email":"******"}`

	got := maskCosmosDBDoc(t, query, input, cosmosDBTestSchema(), cosmosDBMaskers())
	requireJSONEqual(t, want, got)
}

func TestCosmosDBMaskingStringFunctions(t *testing.T) {
	// String functions in SELECT: UPPER, LOWER, LENGTH.
	query := `SELECT UPPER(c.name) AS upperName, LOWER(c.country) AS lowerCountry, LENGTH(c.name) AS nameLen FROM c`
	input := `{"upperName":"ALICE","lowerCountry":"us","nameLen":5}`
	want := `{"upperName":"ALICE","lowerCountry":"us","nameLen":5}`

	got := maskCosmosDBDoc(t, query, input, cosmosDBTestSchema(), cosmosDBMaskers())
	requireJSONEqual(t, want, got)
}

func TestCosmosDBMaskingMathFunctions(t *testing.T) {
	// Math functions in SELECT.
	query := `SELECT c.name, ROUND(c.score) AS roundedScore FROM c`
	input := `{"name":"Alice","roundedScore":95}`
	want := `{"name":"Alice","roundedScore":95}`

	got := maskCosmosDBDoc(t, query, input, cosmosDBTestSchema(), cosmosDBMaskers())
	requireJSONEqual(t, want, got)
}

func TestCosmosDBMaskingTypeCheckFunctions(t *testing.T) {
	// Type-check functions in SELECT.
	query := `SELECT c.name, IS_STRING(c.name) AS isStr, IS_NUMBER(c.population) AS isNum FROM c`
	input := `{"name":"Alice","isStr":true,"isNum":true}`
	want := `{"name":"Alice","isStr":true,"isNum":true}`

	got := maskCosmosDBDoc(t, query, input, cosmosDBTestSchema(), cosmosDBMaskers())
	requireJSONEqual(t, want, got)
}

func TestCosmosDBMaskingUnderscoreFields(t *testing.T) {
	// Underscore-prefixed fields like _ts, _etag should work with the new IDENTIFIER rule.
	query := `SELECT c._ts, c._etag, c.name FROM c`
	input := `{"_ts":1743702439,"_etag":"abc123","name":"Alice"}`
	want := `{"_ts":1743702439,"_etag":"abc123","name":"Alice"}`

	got := maskCosmosDBDoc(t, query, input, cosmosDBTestSchema(), cosmosDBMaskers())
	requireJSONEqual(t, want, got)
}

func TestCosmosDBMaskingNestedFieldProjection(t *testing.T) {
	// Projecting nested sensitive field.
	query := `SELECT c.name, c.contact.phone AS phone FROM c`
	input := `{"name":"Alice","phone":"123-456"}`
	want := `{"name":"Alice","phone":"******"}`

	got := maskCosmosDBDoc(t, query, input, cosmosDBTestSchema(), cosmosDBMaskers())
	requireJSONEqual(t, want, got)
}

func TestCosmosDBMaskingDistinctValue(t *testing.T) {
	// SELECT DISTINCT VALUE returns scalar values, not objects.
	// The result is just a plain value like "US" instead of {"country":"US"}.
	// CosmosDB returns each value as a single-field JSON, but the masking
	// code needs to handle this correctly.
	query := `SELECT DISTINCT VALUE c.country FROM c`
	input := `{"country":"US"}`
	want := `{"country":"US"}`

	got := maskCosmosDBDoc(t, query, input, cosmosDBTestSchema(), cosmosDBMaskers())
	requireJSONEqual(t, want, got)
}

func TestCosmosDBMaskingDistinctValueSensitive(t *testing.T) {
	// SELECT DISTINCT VALUE on a sensitive field.
	query := `SELECT DISTINCT VALUE c.email FROM c`
	input := `{"email":"alice@example.com"}`
	want := `{"email":"******"}`

	got := maskCosmosDBDoc(t, query, input, cosmosDBTestSchema(), cosmosDBMaskers())
	requireJSONEqual(t, want, got)
}

func TestCosmosDBMaskingGeospatial(t *testing.T) {
	// Geospatial function with JSON object literal.
	query := `SELECT c.name, ST_DISTANCE(c.location, {"type": "Point", "coordinates": [55.2708, 25.2048]}) AS dist FROM c`
	input := `{"name":"Alice","dist":1234.5}`
	want := `{"name":"Alice","dist":1234.5}`

	got := maskCosmosDBDoc(t, query, input, cosmosDBTestSchema(), cosmosDBMaskers())
	requireJSONEqual(t, want, got)
}

// ---------------------------------------------------------------------------
// Test: Predicate blocking with new WHERE operators
// ---------------------------------------------------------------------------

func TestCosmosDBPredicateNotEqual(t *testing.T) {
	// WHERE with != on a sensitive field should be detected.
	query := `SELECT * FROM c WHERE c.email != "test@example.com"`
	span := getCosmosDBQuerySpan(t, query)
	schema := cosmosDBTestSchema()

	_, ok := span.PredicatePaths["c.email"]
	require.True(t, ok, "expected c.email in predicate paths")

	semanticType := getFirstSemanticTypeInPath(span.PredicatePaths["c.email"], schema)
	require.Equal(t, "bb.default", semanticType, "email should be sensitive in predicate")
}

func TestCosmosDBPredicateIn(t *testing.T) {
	// WHERE with IN on a sensitive field should be detected.
	query := `SELECT * FROM c WHERE c.email IN ("alice@example.com", "bob@example.com")`
	span := getCosmosDBQuerySpan(t, query)
	schema := cosmosDBTestSchema()

	_, ok := span.PredicatePaths["c.email"]
	require.True(t, ok, "expected c.email in predicate paths")

	semanticType := getFirstSemanticTypeInPath(span.PredicatePaths["c.email"], schema)
	require.Equal(t, "bb.default", semanticType, "email should be sensitive in predicate")
}

func TestCosmosDBPredicateBetween(t *testing.T) {
	// WHERE with BETWEEN on a non-sensitive field should NOT be blocked.
	query := `SELECT * FROM c WHERE c.age BETWEEN 18 AND 65`
	span := getCosmosDBQuerySpan(t, query)
	schema := cosmosDBTestSchema()

	_, ok := span.PredicatePaths["c.age"]
	require.True(t, ok, "expected c.age in predicate paths")

	semanticType := getFirstSemanticTypeInPath(span.PredicatePaths["c.age"], schema)
	require.Equal(t, "", semanticType, "age should not be sensitive")
}

func TestCosmosDBPredicateInNonSensitive(t *testing.T) {
	// WHERE with IN on a non-sensitive field should NOT be blocked.
	query := `SELECT * FROM c WHERE c.country IN ("US", "UK", "CA")`
	span := getCosmosDBQuerySpan(t, query)
	schema := cosmosDBTestSchema()

	_, ok := span.PredicatePaths["c.country"]
	require.True(t, ok, "expected c.country in predicate paths")

	semanticType := getFirstSemanticTypeInPath(span.PredicatePaths["c.country"], schema)
	require.Equal(t, "", semanticType, "country should not be sensitive")
}

func TestCosmosDBPredicateNotEqualNonSensitive(t *testing.T) {
	// WHERE with != on a non-sensitive field should NOT be blocked.
	query := `SELECT * FROM c WHERE c.status != "inactive"`
	span := getCosmosDBQuerySpan(t, query)
	schema := cosmosDBTestSchema()

	_, ok := span.PredicatePaths["c.status"]
	require.True(t, ok, "expected c.status in predicate paths")

	semanticType := getFirstSemanticTypeInPath(span.PredicatePaths["c.status"], schema)
	require.Equal(t, "", semanticType, "status should not be sensitive")
}

func TestCosmosDBPredicateNestedSensitive(t *testing.T) {
	// WHERE with nested sensitive field c.contact.phone.
	query := `SELECT * FROM c WHERE c.contact.phone = "123-456"`
	span := getCosmosDBQuerySpan(t, query)
	schema := cosmosDBTestSchema()

	_, ok := span.PredicatePaths["c.contact.phone"]
	require.True(t, ok, "expected c.contact.phone in predicate paths")

	semanticType := getFirstSemanticTypeInPath(span.PredicatePaths["c.contact.phone"], schema)
	require.Equal(t, "bb.default", semanticType, "contact.phone should be sensitive")
}

func TestCosmosDBPredicateFunctionInWhere(t *testing.T) {
	// Function used in WHERE clause should extract fields from arguments.
	query := `SELECT * FROM c WHERE CONTAINS(c.email, "example")`
	span := getCosmosDBQuerySpan(t, query)
	schema := cosmosDBTestSchema()

	_, ok := span.PredicatePaths["c.email"]
	require.True(t, ok, "expected c.email in predicate paths from function argument")

	semanticType := getFirstSemanticTypeInPath(span.PredicatePaths["c.email"], schema)
	require.Equal(t, "bb.default", semanticType, "email should be sensitive even in function")
}

func TestCosmosDBPredicateCompoundWhere(t *testing.T) {
	// Compound WHERE with multiple operators including new ones.
	query := `SELECT * FROM c WHERE c.country != "US" AND c._ts > 1743702439`
	span := getCosmosDBQuerySpan(t, query)

	_, ok := span.PredicatePaths["c.country"]
	require.True(t, ok, "expected c.country in predicate paths")

	_, ok = span.PredicatePaths["c._ts"]
	require.True(t, ok, "expected c._ts in predicate paths")
}
