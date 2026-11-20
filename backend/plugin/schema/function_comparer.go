package schema

import (
	"strings"
	"sync"

	metadatapb "github.com/bytebase/omni/metadata"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// FunctionComparisonResult contains detailed information about function changes.
type FunctionComparisonResult struct {
	SignatureChanged    bool
	BodyChanged         bool
	AttributesChanged   bool
	ChangedAttributes   []string
	RequiresRecreation  bool
	CanUseAlterFunction bool
}

// FunctionComparer provides an interface for engine-specific function comparison logic.
type FunctionComparer interface {
	// Equal compares two functions and returns whether they are equal.
	Equal(oldFunc, newFunc *metadatapb.FunctionMetadata) bool

	// CompareDetailed performs detailed comparison and returns migration strategy information.
	// Returns nil if functions are equal.
	CompareDetailed(oldFunc, newFunc *metadatapb.FunctionMetadata) (*FunctionComparisonResult, error)
}

// DefaultFunctionComparer provides default function comparison logic that can be used by most engines.
type DefaultFunctionComparer struct{}

// Equal compares two functions using simple definition comparison.
func (*DefaultFunctionComparer) Equal(oldFunc, newFunc *metadatapb.FunctionMetadata) bool {
	if oldFunc == nil || newFunc == nil {
		return oldFunc == newFunc
	}

	// Simple definition comparison
	return functionsEqual(oldFunc, newFunc)
}

// CompareDetailed provides basic comparison for engines that don't have advanced comparison logic.
func (*DefaultFunctionComparer) CompareDetailed(oldFunc, newFunc *metadatapb.FunctionMetadata) (*FunctionComparisonResult, error) {
	// For default implementation, if functions are equal, return nil
	if functionsEqual(oldFunc, newFunc) {
		return nil, nil
	}

	// For basic engines, treat any change as requiring recreation
	return &FunctionComparisonResult{
		SignatureChanged:    true, // Conservatively assume signature changed
		BodyChanged:         true, // Conservatively assume body changed
		AttributesChanged:   false,
		ChangedAttributes:   nil,
		CanUseAlterFunction: false, // Default engines use DROP/CREATE
	}, nil
}

// The registry is no longer read by a migration path: every engine diffs functions
// through its omni catalog, so only a test registering a comparer reads it.
var (
	functionComparerRegistryMux sync.RWMutex
	functionComparerRegistry    = make(map[storepb.Engine]FunctionComparer)
)

// RegisterFunctionComparer registers a function comparer for a specific engine.
func RegisterFunctionComparer(engine storepb.Engine, comparer FunctionComparer) {
	functionComparerRegistryMux.Lock()
	defer functionComparerRegistryMux.Unlock()
	functionComparerRegistry[engine] = comparer
}

// GetFunctionComparer returns the function comparer for a specific engine.
// If no engine-specific comparer is registered, it returns the default comparer.
func GetFunctionComparer(engine storepb.Engine) FunctionComparer {
	functionComparerRegistryMux.RLock()
	defer functionComparerRegistryMux.RUnlock()

	if comparer, exists := functionComparerRegistry[engine]; exists {
		return comparer
	}

	// Return default comparer if no engine-specific one is registered
	return &DefaultFunctionComparer{}
}

// functionsEqual checks if two functions are equal.
func functionsEqual(fn1, fn2 *metadatapb.FunctionMetadata) bool {
	if fn1.Definition != fn2.Definition {
		// Try normalized comparison for PostgreSQL functions
		norm1 := normalizePostgreSQLFunction(fn1.Definition)
		norm2 := normalizePostgreSQLFunction(fn2.Definition)
		if norm1 != norm2 {
			return false
		}
	}
	if fn1.CharacterSetClient != fn2.CharacterSetClient {
		return false
	}
	if fn1.CollationConnection != fn2.CollationConnection {
		return false
	}
	if fn1.DatabaseCollation != fn2.DatabaseCollation {
		return false
	}
	if fn1.SqlMode != fn2.SqlMode {
		return false
	}
	if fn1.Comment != fn2.Comment {
		return false
	}
	return true
}

// normalizePostgreSQLFunction normalizes PostgreSQL function definitions for comparison
func normalizePostgreSQLFunction(definition string) string {
	if definition == "" {
		return ""
	}

	normalized := strings.ToLower(definition)

	// Normalize whitespace
	normalized = strings.ReplaceAll(normalized, "\n", " ")
	normalized = strings.ReplaceAll(normalized, "\t", " ")

	// Remove extra spaces
	for strings.Contains(normalized, "  ") {
		normalized = strings.ReplaceAll(normalized, "  ", " ")
	}

	// Normalize CREATE vs CREATE OR REPLACE
	normalized = strings.ReplaceAll(normalized, "create or replace function", "create function")
	normalized = strings.ReplaceAll(normalized, "create or replace procedure", "create procedure")

	// Remove public schema prefix from function and procedure names
	normalized = strings.ReplaceAll(normalized, "function public.", "function ")
	normalized = strings.ReplaceAll(normalized, "procedure public.", "procedure ")

	// Normalize parameter types
	normalized = strings.ReplaceAll(normalized, "character varying", "varchar")
	normalized = strings.ReplaceAll(normalized, "returns numeric", "returns decimal")

	// Normalize dollar quoting - handle various dollar quote formats
	normalized = strings.ReplaceAll(normalized, "$function$", "$$")
	normalized = strings.ReplaceAll(normalized, "$procedure$", "$$")

	// Normalize language position - move to end
	if strings.Contains(normalized, "language plpgsql") && !strings.HasSuffix(strings.TrimSpace(normalized), "language plpgsql") {
		withoutLanguage := strings.ReplaceAll(normalized, " language plpgsql", "")
		withoutLanguage = strings.ReplaceAll(withoutLanguage, "language plpgsql ", "")
		normalized = strings.TrimSpace(withoutLanguage) + " language plpgsql"
	}

	return strings.TrimSpace(normalized)
}

func init() {
	// Register default comparers for engines that don't need specialized logic
	defaultComparer := &DefaultFunctionComparer{}
	RegisterFunctionComparer(storepb.Engine_MYSQL, defaultComparer)
	RegisterFunctionComparer(storepb.Engine_TIDB, defaultComparer)
	RegisterFunctionComparer(storepb.Engine_ORACLE, defaultComparer)
	RegisterFunctionComparer(storepb.Engine_MSSQL, defaultComparer)
	RegisterFunctionComparer(storepb.Engine_COCKROACHDB, defaultComparer)

	// Note: PostgreSQL will register its own specialized comparer
}
