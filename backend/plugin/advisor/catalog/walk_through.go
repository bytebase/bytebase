package catalog

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
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

// WalkThrough will collect the catalog schema in the databaseState as it walks through the stmt.
func WalkThrough(d *DatabaseState, ast any) error {
	switch d.dbType {
	case storepb.Engine_TIDB:
		return TiDBWalkThrough(d, ast)
	case storepb.Engine_MYSQL, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
		return MySQLWalkThrough(d, ast)
	case storepb.Engine_POSTGRES:
		return PgWalkThrough(d, ast)
	default:
		return errors.Errorf("Walk-through doesn't support engine type: %s", d.dbType)
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
	return compareIdentifier(d.name, database, d.ignoreCaseSensitive)
}

// getTable returns the table with the given name if it exists in the schema.
// TODO(zp): It's used for MySQL, we should refactor the package to make it more clear.
//
//nolint:revive
func (s *SchemaState) getTable(table string) (*TableState, bool) {
	for k, v := range s.tableSet {
		if compareIdentifier(k, table, s.ignoreCaseSensitive) {
			return v, true
		}
	}

	return nil, false
}

//nolint:revive
func (s *SchemaState) renameTable(oldName string, newName string) *WalkThroughError {
	if oldName == newName {
		return nil
	}

	table, exists := s.getTable(oldName)
	if !exists {
		return &WalkThroughError{
			Code:    code.TableNotExists,
			Content: fmt.Sprintf("Table `%s` does not exist", oldName),
		}
	}

	if _, exists := s.getTable(newName); exists {
		return &WalkThroughError{
			Code:    code.TableExists,
			Content: fmt.Sprintf("Table `%s` already exists", newName),
		}
	}

	table.name = newName
	delete(s.tableSet, oldName)
	s.tableSet[newName] = table
	return nil
}

func (d *DatabaseState) createSchema() *SchemaState {
	schema := &SchemaState{
		ignoreCaseSensitive: d.ignoreCaseSensitive,
		name:                "",
		tableSet:            make(tableStateMap),
		viewSet:             make(viewStateMap),
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

func (d *DatabaseState) existedViewList(viewMap map[string]bool) ([]string, error) {
	var result []string
	for viewName := range viewMap {
		schemaName, viewName, err := parseViewName(viewName)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to check view dependency")
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
				Code:    code.SchemaNotExists,
				Content: fmt.Sprintf("The schema %q doesn't exist", schemaName),
			}
		}
		schema = &SchemaState{
			ignoreCaseSensitive: d.ignoreCaseSensitive,
			name:                publicSchemaName,
			tableSet:            make(tableStateMap),
			viewSet:             make(viewStateMap),
			identifierMap:       make(identifierMap),
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
			Code:    code.ColumnNotExists,
			Content: fmt.Sprintf("The column %q does not exist in the table %q", columnName, t.name),
		}
	}
	return column, nil
}

func (s *SchemaState) pgGetTable(tableName string) (*TableState, *WalkThroughError) {
	table, exists := s.tableSet[tableName]
	if !exists {
		return nil, &WalkThroughError{
			Code:    code.TableNotExists,
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
		Code:    code.IndexNotExists,
		Content: fmt.Sprintf("Index %q does not exists in schema %q", indexName, s.name),
	}
}
