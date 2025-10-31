package pg

import "strings"

// PostgreSQL type equivalence helper functions

// normalizePostgreSQLType normalizes a PostgreSQL type name to its canonical form.
// This handles common aliases and returns the standard representation.
func normalizePostgreSQLType(typeName string) string {
	typeName = strings.ToLower(strings.TrimSpace(typeName))

	// Remove whitespace inside parentheses: " VARCHAR ( 20 ) " -> "varchar(20)"
	typeName = removeWhitespaceInParams(typeName)

	// Handle SERIAL types - they're aliases for INTEGER types with sequences
	switch typeName {
	case "serial":
		return "integer"
	case "bigserial":
		return "bigint"
	case "smallserial":
		return "smallint"
	}

	// Handle VARCHAR - it's an alias for CHARACTER VARYING
	if strings.HasPrefix(typeName, "varchar") {
		if typeName == "varchar" {
			return "character varying"
		}
		// varchar(N) -> character varying(N)
		if strings.HasPrefix(typeName, "varchar(") {
			params := strings.TrimPrefix(typeName, "varchar")
			return "character varying" + params
		}
	}

	return typeName
}

// removeWhitespaceInParams removes whitespace inside parentheses
// " VARCHAR ( 20 ) " -> "varchar(20)"
// "double precision" -> "double precision" (unchanged)
func removeWhitespaceInParams(typeName string) string {
	idx := strings.Index(typeName, "(")
	if idx == -1 {
		// No parameters, just normalize spaces between words to single space
		return normalizeSpaces(typeName)
	}

	// Split into base type and parameters
	baseType := normalizeSpaces(typeName[:idx])
	params := strings.ReplaceAll(typeName[idx:], " ", "")
	return baseType + params
}

// normalizeSpaces normalizes multiple spaces to single space
func normalizeSpaces(s string) string {
	// Replace multiple spaces with single space
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}
	return strings.TrimSpace(s)
}

// areTypesEquivalent checks if two PostgreSQL types are equivalent.
// This includes:
// - Exact matches after normalization
// - Type family equivalence (e.g., int/integer/int4)
func areTypesEquivalent(typeA, typeB string) bool {
	typeA = normalizePostgreSQLType(typeA)
	typeB = normalizePostgreSQLType(typeB)

	// Exact match after normalization
	if typeA == typeB {
		return true
	}

	// Check type families
	// Integer family: int, integer, int4
	intTypes := []string{"int", "integer", "int4"}
	if inSlice(typeA, intTypes) && inSlice(typeB, intTypes) {
		return true
	}

	// Bigint family: bigint, int8
	bigintTypes := []string{"bigint", "int8"}
	if inSlice(typeA, bigintTypes) && inSlice(typeB, bigintTypes) {
		return true
	}

	// Smallint family: smallint, int2
	smallintTypes := []string{"smallint", "int2"}
	if inSlice(typeA, smallintTypes) && inSlice(typeB, smallintTypes) {
		return true
	}

	// Real family: real, float4
	realTypes := []string{"real", "float4"}
	if inSlice(typeA, realTypes) && inSlice(typeB, realTypes) {
		return true
	}

	// Double precision family: double precision, float8
	doubleTypes := []string{"double precision", "float8"}
	if inSlice(typeA, doubleTypes) && inSlice(typeB, doubleTypes) {
		return true
	}

	// Boolean family: boolean, bool
	boolTypes := []string{"boolean", "bool"}
	if inSlice(typeA, boolTypes) && inSlice(typeB, boolTypes) {
		return true
	}

	// Character family: character, char (but we need to preserve length parameters)
	// For types with parameters, check base type equivalence
	if hasTypeParameters(typeA) && hasTypeParameters(typeB) {
		baseA, paramsA := splitTypeAndParams(typeA)
		baseB, paramsB := splitTypeAndParams(typeB)

		// Check if base types are equivalent
		if areBaseTypesEquivalent(baseA, baseB) {
			// For character types, parameters must also match
			return paramsA == paramsB
		}
	}

	return false
}

// hasTypeParameters checks if a type has parameters (e.g., varchar(20))
func hasTypeParameters(typeName string) bool {
	return strings.Contains(typeName, "(")
}

// splitTypeAndParams splits a type into base type and parameters
// e.g., "character varying(20)" -> ("character varying", "(20)")
func splitTypeAndParams(typeName string) (string, string) {
	idx := strings.Index(typeName, "(")
	if idx == -1 {
		return typeName, ""
	}
	return strings.TrimSpace(typeName[:idx]), typeName[idx:]
}

// areBaseTypesEquivalent checks if two base types (without parameters) are equivalent
func areBaseTypesEquivalent(baseA, baseB string) bool {
	// Normalize base types
	baseA = strings.ToLower(strings.TrimSpace(baseA))
	baseB = strings.ToLower(strings.TrimSpace(baseB))

	if baseA == baseB {
		return true
	}

	// Character varying family
	charVaryingTypes := []string{"character varying", "varchar"}
	if inSlice(baseA, charVaryingTypes) && inSlice(baseB, charVaryingTypes) {
		return true
	}

	// Character family
	charTypes := []string{"character", "char"}
	if inSlice(baseA, charTypes) && inSlice(baseB, charTypes) {
		return true
	}

	return false
}

// inSlice checks if a string is in a slice
func inSlice(str string, slice []string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

// isTypeInList checks if a column type matches any type in the given list,
// considering type equivalence.
func isTypeInList(columnType string, typeList []string) bool {
	for _, listType := range typeList {
		if areTypesEquivalent(columnType, listType) {
			return true
		}
	}
	return false
}
