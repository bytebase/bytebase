package catalog

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// WalkThroughErrorType is the type of WalkThroughError.
type WalkThroughErrorType int

const (
	// PrimaryKeyName is the string for PK.
	PrimaryKeyName string = "PRIMARY"
	// FullTextName is the string for FULLTEXT.
	FullTextName string = "FULLTEXT"
	// SpatialName is the string for SPATIAL.
	SpatialName string = "SPATIAL"

	publicSchemaName = "public"

	// ErrorTypeUnsupported is the error for unsupported cases.
	ErrorTypeUnsupported WalkThroughErrorType = 1
	// ErrorTypeInternal is the error for internal errors.
	ErrorTypeInternal WalkThroughErrorType = 2

	// 101 parse error type.

	// ErrorTypeParseError is the error in parsing.
	ErrorTypeParseError WalkThroughErrorType = 101
	// ErrorTypeDeparseError is the error in deparsing.
	ErrorTypeDeparseError WalkThroughErrorType = 102

	// 201 ~ 299 database error type.

	// ErrorTypeAccessOtherDatabase is the error that try to access other database.
	ErrorTypeAccessOtherDatabase = 201
	// ErrorTypeDatabaseIsDeleted is the error that try to access the deleted database.
	ErrorTypeDatabaseIsDeleted = 202
	// ErrorTypeReferenceOtherDatabase is the error that try to reference other database.
	ErrorTypeReferenceOtherDatabase = 203

	// 301 ~ 399 table error type.

	// ErrorTypeTableExists is the error that table exists.
	ErrorTypeTableExists = 301
	// ErrorTypeTableNotExists is the error that table not exists.
	ErrorTypeTableNotExists = 302
	// ErrorTypeUseCreateTableAs is the error that using CREATE TABLE AS statements.
	ErrorTypeUseCreateTableAs = 303
	// ErrorTypeTableIsReferencedByView is the error that table is referenced by view.
	ErrorTypeTableIsReferencedByView = 304

	// 401 ~ 499 column error type.

	// ErrorTypeColumnExists is the error that column exists.
	ErrorTypeColumnExists = 401
	// ErrorTypeColumnNotExists is the error that column not exists.
	ErrorTypeColumnNotExists = 402
	// ErrorTypeDropAllColumns is the error that dropping all columns in a table.
	ErrorTypeDropAllColumns = 403
	// ErrorTypeAutoIncrementExists is the error that auto_increment exists.
	ErrorTypeAutoIncrementExists = 404
	// ErrorTypeOnUpdateColumnNotDatetimeOrTimestamp is the error that the ON UPDATE column is not datetime or timestamp.
	ErrorTypeOnUpdateColumnNotDatetimeOrTimestamp = 405
	// ErrorTypeSetNullDefaultForNotNullColumn is the error that setting NULL default value for the NOT NULL column.
	ErrorTypeSetNullDefaultForNotNullColumn = 406
	// ErrorTypeInvalidColumnTypeForDefaultValue is the error that invalid column type for default value.
	ErrorTypeInvalidColumnTypeForDefaultValue = 407
	// ErrorTypeColumnIsReferencedByView is the error that column is referenced by view.
	ErrorTypeColumnIsReferencedByView = 408

	// 501 ~ 599 index error type.

	// ErrorTypePrimaryKeyExists is the error that PK exists.
	ErrorTypePrimaryKeyExists = 501
	// ErrorTypeIndexExists is the error that index exists.
	ErrorTypeIndexExists = 502
	// ErrorTypeIndexEmptyKeys is the error that index has empty keys.
	ErrorTypeIndexEmptyKeys = 503
	// ErrorTypePrimaryKeyNotExists is the error that PK does not exist.
	ErrorTypePrimaryKeyNotExists = 504
	// ErrorTypeIndexNotExists is the error that index does not exist.
	ErrorTypeIndexNotExists = 505
	// ErrorTypeIncorrectIndexName is the incorrect index name error.
	ErrorTypeIncorrectIndexName = 506
	// ErrorTypeSpatialIndexKeyNullable is the error that keys in spatial index are nullable.
	ErrorTypeSpatialIndexKeyNullable = 507

	// 701 ~ 799 schema error type.

	// ErrorTypeSchemaNotExists is the error that schema does not exist.
	ErrorTypeSchemaNotExists = 701

	// 801 ~ 899 relation error type.

	// ErrorTypeRelationExists is the error that relation already exists.
	ErrorTypeRelationExists = 801

	// 901 ~ 999 constraint error type.

	// ErrorTypeConstraintNotExists is the error that constraint doesn't exist.
	ErrorTypeConstraintNotExists = 901
)

// WalkThroughError is the error for walking-through.
type WalkThroughError struct {
	Type    WalkThroughErrorType
	Content string
	// TODO(zp): position
	Line int

	Payload any
}

// NewRelationExistsError returns a new ErrorTypeRelationExists.
func NewRelationExistsError(relationName string, schemaName string) *WalkThroughError {
	return &WalkThroughError{
		Type:    ErrorTypeRelationExists,
		Content: fmt.Sprintf("Relation %q already exists in schema %q", relationName, schemaName),
	}
}

// NewColumnNotExistsError returns a new ErrorTypeColumnNotExists.
func NewColumnNotExistsError(tableName string, columnName string) *WalkThroughError {
	return &WalkThroughError{
		Type:    ErrorTypeColumnNotExists,
		Content: fmt.Sprintf("Column `%s` does not exist in table `%s`", columnName, tableName),
	}
}

// NewIndexNotExistsError returns a new ErrorTypeIndexNotExists.
func NewIndexNotExistsError(tableName string, indexName string) *WalkThroughError {
	return &WalkThroughError{
		Type:    ErrorTypeIndexNotExists,
		Content: fmt.Sprintf("Index `%s` does not exist in table `%s`", indexName, tableName),
	}
}

// NewIndexExistsError returns a new ErrorTypeIndexExists.
func NewIndexExistsError(tableName string, indexName string) *WalkThroughError {
	return &WalkThroughError{
		Type:    ErrorTypeIndexExists,
		Content: fmt.Sprintf("Index `%s` already exists in table `%s`", indexName, tableName),
	}
}

// NewAccessOtherDatabaseError returns a new ErrorTypeAccessOtherDatabase.
func NewAccessOtherDatabaseError(current string, target string) *WalkThroughError {
	return &WalkThroughError{
		Type:    ErrorTypeAccessOtherDatabase,
		Content: fmt.Sprintf("Database `%s` is not the current database `%s`", target, current),
	}
}

// NewTableNotExistsError returns a new ErrorTypeTableNotExists.
func NewTableNotExistsError(tableName string) *WalkThroughError {
	return &WalkThroughError{
		Type:    ErrorTypeTableNotExists,
		Content: fmt.Sprintf("Table `%s` does not exist", tableName),
	}
}

// NewTableExistsError returns a new ErrorTypeTableExists.
func NewTableExistsError(tableName string) *WalkThroughError {
	return &WalkThroughError{
		Type:    ErrorTypeTableExists,
		Content: fmt.Sprintf("Table `%s` already exists", tableName),
	}
}

// Error implements the error interface.
func (e *WalkThroughError) Error() string {
	return e.Content
}

// WalkThrough will collect the catalog schema in the databaseState as it walks through the stmt.
func WalkThrough(d *DatabaseState, ast any) error {
	switch d.dbType {
	case storepb.Engine_TIDB:
		return TiDBWalkThrough(d, ast)
	case storepb.Engine_MYSQL, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
		return MySQLWalkThrough(d, ast)
	case storepb.Engine_POSTGRES:
		if err := PgWalkThrough(d, ast); err != nil {
			if d.ctx.CheckIntegrity {
				return err
			}
			d.usable = false
		}
		return nil
	default:
		return &WalkThroughError{
			Type:    ErrorTypeUnsupported,
			Content: fmt.Sprintf("Walk-through doesn't support engine type: %s", d.dbType),
		}
	}
}

// compareIdentifier returns true if the engine will regard the two identifiers as the same one.
func compareIdentifier(a, b string, ignoreCaseSensitive bool) bool {
	if ignoreCaseSensitive {
		return strings.EqualFold(a, b)
	}
	return a == b
}

// isCurrentDatabase returns true if the given database is the current database of the state.
func (d *DatabaseState) isCurrentDatabase(database string) bool {
	return compareIdentifier(d.name, database, d.ctx.IgnoreCaseSensitive)
}

// getTable returns the table with the given name if it exists in the schema.
// TODO(zp): It's used for MySQL, we should refactor the package to make it more clear.
//
//nolint:revive
func (s *SchemaState) getTable(table string) (*TableState, bool) {
	for k, v := range s.tableSet {
		if compareIdentifier(k, table, s.ctx.IgnoreCaseSensitive) {
			return v, true
		}
	}

	return nil, false
}

func (t *TableState) createIncompleteIndex(name string) *IndexState {
	index := &IndexState{
		name: name,
	}
	t.indexSet[strings.ToLower(name)] = index
	return index
}

//nolint:revive
func (s *SchemaState) renameTable(ctx *FinderContext, oldName string, newName string) *WalkThroughError {
	if oldName == newName {
		return nil
	}

	table, exists := s.getTable(oldName)
	if !exists {
		if ctx.CheckIntegrity {
			return &WalkThroughError{
				Type:    ErrorTypeTableNotExists,
				Content: fmt.Sprintf("Table `%s` does not exist", oldName),
			}
		}
		table = s.createIncompleteTable(oldName)
	}

	if _, exists := s.getTable(newName); exists {
		return &WalkThroughError{
			Type:    ErrorTypeTableExists,
			Content: fmt.Sprintf("Table `%s` already exists", newName),
		}
	}

	table.name = newName
	delete(s.tableSet, oldName)
	s.tableSet[newName] = table
	return nil
}

func (s *SchemaState) createIncompleteTable(name string) *TableState {
	table := &TableState{
		name:      name,
		columnSet: make(columnStateMap),
		indexSet:  make(IndexStateMap),
	}
	s.tableSet[name] = table
	return table
}

func (t *TableState) createIncompleteColumn(name string) *ColumnState {
	column := &ColumnState{
		name: name,
	}
	t.columnSet[strings.ToLower(name)] = column
	return column
}

func (d *DatabaseState) createSchema() *SchemaState {
	schema := &SchemaState{
		ctx:      d.ctx,
		name:     "",
		tableSet: make(tableStateMap),
		viewSet:  make(viewStateMap),
	}

	d.schemaSet[""] = schema
	return schema
}

// PostgreSQL-specific helper methods.

func parseViewName(viewName string) (string, string, error) {
	pattern := `^"(.+?)"\."(.+?)"$`

	re := regexp.MustCompile(pattern)

	match := re.FindStringSubmatch(viewName)

	if len(match) != 3 {
		return "", "", errors.Errorf("invalid view name: %s", viewName)
	}

	return match[1], match[2], nil
}

func (d *DatabaseState) existedViewList(viewMap map[string]bool) ([]string, *WalkThroughError) {
	var result []string
	for viewName := range viewMap {
		schemaName, viewName, err := parseViewName(viewName)
		if err != nil {
			return nil, &WalkThroughError{
				Type:    ErrorTypeInternal,
				Content: fmt.Sprintf("failed to check view dependency: %s", err.Error()),
			}
		}
		schemaMeta, exists := d.schemaSet[schemaName]
		if !exists {
			continue
		}
		if _, exists := schemaMeta.viewSet[viewName]; !exists {
			continue
		}

		result = append(result, fmt.Sprintf("%q.%q", schemaName, viewName))
	}
	return result, nil
}

func (s *SchemaState) pgGeneratePrimaryKeyName(tableName string) string {
	pkName := fmt.Sprintf("%s_pkey", tableName)
	if _, exists := s.identifierMap[pkName]; !exists {
		return pkName
	}
	suffix := 1
	for {
		if _, exists := s.identifierMap[fmt.Sprintf("%s%d", pkName, suffix)]; !exists {
			return fmt.Sprintf("%s%d", pkName, suffix)
		}
		suffix++
	}
}

//nolint:revive
func (d *DatabaseState) getSchema(schemaName string) (*SchemaState, *WalkThroughError) {
	if schemaName == "" {
		schemaName = publicSchemaName
	}
	schema, exists := d.schemaSet[schemaName]
	if !exists {
		if schemaName != publicSchemaName {
			return nil, &WalkThroughError{
				Type:    ErrorTypeSchemaNotExists,
				Content: fmt.Sprintf("The schema %q doesn't exist", schemaName),
			}
		}
		schema = &SchemaState{
			ctx:           d.ctx,
			name:          publicSchemaName,
			tableSet:      make(tableStateMap),
			viewSet:       make(viewStateMap),
			identifierMap: make(identifierMap),
		}
		d.schemaSet[publicSchemaName] = schema
	}
	return schema, nil
}

//nolint:revive
func (t *TableState) getColumn(columnName string) (*ColumnState, *WalkThroughError) {
	column, exists := t.columnSet[columnName]
	if !exists {
		return nil, &WalkThroughError{
			Type:    ErrorTypeColumnNotExists,
			Content: fmt.Sprintf("The column %q does not exist in the table %q", columnName, t.name),
		}
	}
	return column, nil
}

func (s *SchemaState) pgGetTable(tableName string) (*TableState, *WalkThroughError) {
	table, exists := s.tableSet[tableName]
	if !exists {
		return nil, &WalkThroughError{
			Type:    ErrorTypeTableNotExists,
			Content: fmt.Sprintf("The table %q does not exist in schema %q", tableName, s.name),
		}
	}
	return table, nil
}

func (s *SchemaState) getIndex(indexName string) (*TableState, *IndexState, *WalkThroughError) {
	for _, table := range s.tableSet {
		if index, exists := table.indexSet[indexName]; exists {
			return table, index, nil
		}
	}

	return nil, nil, &WalkThroughError{
		Type:    ErrorTypeIndexNotExists,
		Content: fmt.Sprintf("Index %q does not exists in schema %q", indexName, s.name),
	}
}
