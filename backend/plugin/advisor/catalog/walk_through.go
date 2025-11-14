package catalog

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
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

// SchemaViolationError represents a SQL schema validation error that should be reported as advice.
// It carries an advisor code (int32) that can be converted to advisor.Code by the caller.
type SchemaViolationError struct {
	Code    int32 // Advisor error code from backend/plugin/advisor/code.go
	Message string
}

// Error implements the error interface.
func (e *SchemaViolationError) Error() string {
	return e.Message
}

// NewSchemaViolationError creates a new SchemaViolationError.
func NewSchemaViolationError(code int32, message string) *SchemaViolationError {
	return &SchemaViolationError{
		Code:    code,
		Message: message,
	}
}

// WalkThrough will collect the catalog schema in the databaseState as it walks through the stmt.
func (d *DatabaseState) WalkThrough(ast any) error {
	switch d.dbType {
	case storepb.Engine_TIDB:
		return d.tidbWalkThrough(ast)
	case storepb.Engine_MYSQL, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
		return d.mysqlWalkThrough(ast)
	case storepb.Engine_POSTGRES:
		if err := d.pgWalkThrough(ast); err != nil {
			if d.ctx.CheckIntegrity {
				return err
			}
			d.usable = false
		}
		return nil
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
	return compareIdentifier(d.name, database, d.ctx.IgnoreCaseSensitive)
}

// getTable returns the table with the given name if it exists in the schema.
// TODO(zp): It's used for MySQL, we should refactor the package to make it more clear.
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

func (s *SchemaState) renameTable(ctx *FinderContext, oldName string, newName string) error {
	if oldName == newName {
		return nil
	}

	table, exists := s.getTable(oldName)
	if !exists {
		if ctx.CheckIntegrity {
			return NewSchemaViolationError(604, fmt.Sprintf("Table `%s` does not exist", oldName))
		}
		table = s.createIncompleteTable(oldName)
	}

	if _, exists := s.getTable(newName); exists {
		return NewSchemaViolationError(607, fmt.Sprintf("Table `%s` already exists", newName))
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
		ctx:      d.ctx.Copy(),
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

func (d *DatabaseState) getSchema(schemaName string) (*SchemaState, error) {
	if schemaName == "" {
		schemaName = publicSchemaName
	}
	schema, exists := d.schemaSet[schemaName]
	if !exists {
		if schemaName != publicSchemaName {
			return nil, NewSchemaViolationError(1901, fmt.Sprintf("The schema %q doesn't exist", schemaName))
		}
		schema = &SchemaState{
			ctx:           d.ctx.Copy(),
			name:          publicSchemaName,
			tableSet:      make(tableStateMap),
			viewSet:       make(viewStateMap),
			identifierMap: make(identifierMap),
		}
		d.schemaSet[publicSchemaName] = schema
	}
	return schema, nil
}

func (t *TableState) getColumn(columnName string) (*ColumnState, error) {
	column, exists := t.columnSet[columnName]
	if !exists {
		return nil, NewSchemaViolationError(405, fmt.Sprintf("The column %q does not exist in the table %q", columnName, t.name))
	}
	return column, nil
}

func (s *SchemaState) pgGetTable(tableName string) (*TableState, error) {
	table, exists := s.tableSet[tableName]
	if !exists {
		return nil, NewSchemaViolationError(604, fmt.Sprintf("The table %q does not exist in schema %q", tableName, s.name))
	}
	return table, nil
}

func (s *SchemaState) getIndex(indexName string) (*TableState, *IndexState, error) {
	for _, table := range s.tableSet {
		if index, exists := table.indexSet[indexName]; exists {
			return table, index, nil
		}
	}

	return nil, nil, NewSchemaViolationError(809, fmt.Sprintf("Index %q does not exists in schema %q", indexName, s.name))
}
