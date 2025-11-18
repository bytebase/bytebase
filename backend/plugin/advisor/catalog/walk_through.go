package catalog

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/store/model"
)

const (
	// PrimaryKeyName is the string for PK.
	PrimaryKeyName string = "PRIMARY"
	// FullTextName is the string for FULLTEXT.
	FullTextName string = "FULLTEXT"
	// SpatialName is the string for SPATIAL.
	SpatialName string = "SPATIAL"

	publicSchemaName = "public"
)

// WalkThroughError is the error for walking-through.
// It represents SQL review errors that should be converted to advisor codes.
type WalkThroughError struct {
	Code    code.Code
	Content string
	// TODO(zp): position
	Line int
}

// NewRelationExistsError returns a new RelationExists error.
func NewRelationExistsError(relationName string, schemaName string) *WalkThroughError {
	return &WalkThroughError{
		Code:    code.RelationExists,
		Content: fmt.Sprintf("Relation %q already exists in schema %q", relationName, schemaName),
	}
}

// NewColumnNotExistsError returns a new ColumnNotExists error.
func NewColumnNotExistsError(tableName string, columnName string) *WalkThroughError {
	return &WalkThroughError{
		Code:    code.ColumnNotExists,
		Content: fmt.Sprintf("Column `%s` does not exist in table `%s`", columnName, tableName),
	}
}

// NewIndexNotExistsError returns a new IndexNotExists error.
func NewIndexNotExistsError(tableName string, indexName string) *WalkThroughError {
	return &WalkThroughError{
		Code:    code.IndexNotExists,
		Content: fmt.Sprintf("Index `%s` does not exist in table `%s`", indexName, tableName),
	}
}

// NewIndexExistsError returns a new IndexExists error.
func NewIndexExistsError(tableName string, indexName string) *WalkThroughError {
	return &WalkThroughError{
		Code:    code.IndexExists,
		Content: fmt.Sprintf("Index `%s` already exists in table `%s`", indexName, tableName),
	}
}

// NewAccessOtherDatabaseError returns a new NotCurrentDatabase error.
func NewAccessOtherDatabaseError(current string, target string) *WalkThroughError {
	return &WalkThroughError{
		Code:    code.NotCurrentDatabase,
		Content: fmt.Sprintf("Database `%s` is not the current database `%s`", target, current),
	}
}

// NewTableNotExistsError returns a new TableNotExists error.
func NewTableNotExistsError(tableName string) *WalkThroughError {
	return &WalkThroughError{
		Code:    code.TableNotExists,
		Content: fmt.Sprintf("Table `%s` does not exist", tableName),
	}
}

// NewTableExistsError returns a new TableExists error.
func NewTableExistsError(tableName string) *WalkThroughError {
	return &WalkThroughError{
		Code:    code.TableExists,
		Content: fmt.Sprintf("Table `%s` already exists", tableName),
	}
}

// Error implements the error interface.
func (e *WalkThroughError) Error() string {
	return e.Content
}

// WalkThrough will collect the catalog schema in the database metadata as it walks through the stmt.
func WalkThrough(d *model.DatabaseMetadata, engineType storepb.Engine, ast any) error {
	switch engineType {
	case storepb.Engine_TIDB:
		return TiDBWalkThrough(d, ast)
	case storepb.Engine_MYSQL, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
		return MySQLWalkThrough(d, ast)
	case storepb.Engine_POSTGRES:
		return PgWalkThrough(d, ast)
	default:
		return errors.Errorf("Walk-through doesn't support engine type: %s", engineType)
	}
}

// compareIdentifier returns true if the engine will regard the two identifiers as the same one.
func compareIdentifier(a, b string, ignoreCaseSensitive bool) bool {
	if ignoreCaseSensitive {
		return strings.EqualFold(a, b)
	}
	return a == b
}

// isCurrentDatabase returns true if the given database is the current database.
func isCurrentDatabase(d *model.DatabaseMetadata, database string) bool {
	return compareIdentifier(d.DatabaseName(), database, !d.GetIsObjectCaseSensitive())
}
