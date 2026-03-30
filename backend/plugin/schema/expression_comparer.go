package schema

import (
	"strings"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// CompareExpressionsSemantically compares two expressions semantically based on the engine type.
func CompareExpressionsSemantically(_ storepb.Engine, expr1, expr2 string) bool {
	return compareExpressionsGeneric(expr1, expr2)
}

// compareExpressionsGeneric provides a generic string-based comparison for unsupported engines
func compareExpressionsGeneric(expr1, expr2 string) bool {
	// Quick check for identical strings
	if expr1 == expr2 {
		return true
	}

	// Handle empty expressions
	if strings.TrimSpace(expr1) == "" && strings.TrimSpace(expr2) == "" {
		return true
	}
	if strings.TrimSpace(expr1) == "" || strings.TrimSpace(expr2) == "" {
		return false
	}

	// Normalize whitespace and compare
	norm1 := strings.Join(strings.Fields(expr1), " ")
	norm2 := strings.Join(strings.Fields(expr2), " ")

	// Case-insensitive comparison for SQL keywords
	return strings.EqualFold(norm1, norm2)
}
