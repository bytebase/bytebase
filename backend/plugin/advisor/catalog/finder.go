package catalog

import (
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// FinderContext is the context for finder.
type FinderContext struct {
	// EngineType is the engine type for database engine.
	EngineType storepb.Engine

	// Ignore case sensitive is the policy for identifier name comparison case-sensitive.
	// It has different behavior for different database engine.
	// MySQL:
	// This controls the following identifier comparisons:
	// Database, Table
	IgnoreCaseSensitive bool
}
