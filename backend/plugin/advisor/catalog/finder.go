package catalog

import (
	"github.com/bytebase/bytebase/backend/plugin/advisor/db"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// FinderContext is the context for finder.
type FinderContext struct {
	// CheckIntegrity defines the policy for integrity checking.
	// There are two cases that will cause database to have an empty catalog:
	//   1. we cannot fetch the catalog, such as GitHub App/Actions.
	//   2. the database is indeed empty.
	// We need different logic to deal with these two cases separately.
	// If DROP TABLE t and t not exists:
	//   1. For case one, just ignore this statement.
	//   2. For case two, return the error that table t not exists.
	// In addition, We need fine-grained CheckIntegrity.
	// Consider the case one, and then create a table t by CREATE TABLE statement.
	// After this, drop column a in table t, but column a not exists.
	// In this case, we need return the error that column a does not exist in table t,
	// instead of ignoring this drop-column statement.
	CheckIntegrity bool

	// EngineType is the engine type for database engine.
	EngineType db.Type
}

// Copy returns the deep copy.
func (ctx *FinderContext) Copy() *FinderContext {
	return &FinderContext{
		CheckIntegrity: ctx.CheckIntegrity,
	}
}

// Finder is the service for finding schema information in database.
type Finder struct {
	Origin *DatabaseState
	Final  *DatabaseState
}

// NewFinder creates a new finder.
func NewFinder(database *storepb.DatabaseSchemaMetadata, ctx *FinderContext) *Finder {
	return &Finder{Origin: newDatabaseState(database, ctx), Final: newDatabaseState(database, ctx)}
}

// NewEmptyFinder creates a finder with empty databse.
func NewEmptyFinder(ctx *FinderContext) *Finder {
	return &Finder{Origin: newDatabaseState(&storepb.DatabaseSchemaMetadata{}, ctx), Final: newDatabaseState(&storepb.DatabaseSchemaMetadata{}, ctx)}
}

// WalkThrough does the walk through.
func (f *Finder) WalkThrough(statements string) error {
	return f.Final.WalkThrough(statements)
}
