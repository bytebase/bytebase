package schema

import (
	"sync"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// FunctionChange represents a type of change detected in a function.
type FunctionChange struct {
	Type                FunctionChangeType
	Description         string
	RequiresRecreation  bool
	CanUseAlterFunction bool
}

// FunctionChangeType represents specific types of function changes.
type FunctionChangeType string

const (
	FunctionChangeDefinition FunctionChangeType = "definition"
	FunctionChangeComment    FunctionChangeType = "comment"
	FunctionChangeAttribute  FunctionChangeType = "attribute"
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
	Equal(oldFunc, newFunc *storepb.FunctionMetadata) bool

	// CompareDetailed performs detailed comparison and returns migration strategy information.
	// Returns nil if functions are equal.
	CompareDetailed(oldFunc, newFunc *storepb.FunctionMetadata) (*FunctionComparisonResult, error)
}

// DefaultFunctionComparer provides default function comparison logic that can be used by most engines.
type DefaultFunctionComparer struct{}

// Equal compares two functions using simple definition comparison.
func (*DefaultFunctionComparer) Equal(oldFunc, newFunc *storepb.FunctionMetadata) bool {
	if oldFunc == nil || newFunc == nil {
		return oldFunc == newFunc
	}

	// Simple definition comparison
	return functionsEqual(oldFunc, newFunc)
}

// CompareDetailed provides basic comparison for engines that don't have advanced comparison logic.
func (*DefaultFunctionComparer) CompareDetailed(oldFunc, newFunc *storepb.FunctionMetadata) (*FunctionComparisonResult, error) {
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
