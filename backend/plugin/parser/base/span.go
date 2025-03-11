package base

import (
	"context"
	"sort"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/store/model"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// QueryType is the type of a query.
// The query type determines the permission to use.
type QueryType int

const (
	// Type can not be recognized for now.
	QueryTypeUnknown QueryType = iota
	// The read-only select query.
	Select
	// The explain query.
	Explain
	// The read-only select query for reading information schema and system objects.
	SelectInfoSchema
	// The DDL query that changes schema.
	DDL
	// The DML query that changes table data.
	DML
)

var (
	MixUserSystemTablesError = errors.Errorf("cannot access user and system tables at the same time")
)

type SourceColumnSet map[ColumnResource]bool

// MergeSourceColumnSet merges two source column maps, returns true if there is difference.
func MergeSourceColumnSet(m, n SourceColumnSet) (SourceColumnSet, bool) {
	r := make(SourceColumnSet)
	for k := range m {
		r[k] = true
	}
	for k := range n {
		if _, ok := r[k]; !ok {
			r[k] = true
		}
	}

	return r, len(r) != len(m)
}

// QuerySpan is the span for a query.
type QuerySpan struct {
	Type QueryType
	// Results are the result columns of a query span.
	// Currently, SourceColumns in the QuerySpanResult are only for the fields in the Query.
	Results []QuerySpanResult
	// SourceColumns are the source columns contributing to the span.
	// SourceColumns here are the source columns for the whole query span, containing fields, where conditions, join conditions, etc.
	SourceColumns SourceColumnSet
	// PredicateColumns are the source columns contributing to the span.
	// PredicateColumns here are the source columns for the where conditions.
	PredicateColumns          SourceColumnSet
	NotFoundError             error
	FunctionNotSupportedError error
}

// QuerySpanResult is the result column of a query span.
type QuerySpanResult struct {
	// Name is the result name of a query.
	Name string
	// SourceColumns are the source columns contributing to the span result.
	SourceColumns SourceColumnSet
	// IsPlainField indicates whether the field is a plain column reference (true) or an expression (false).
	IsPlainField bool
}

// ColumnResource is the resource key for a column.
type ColumnResource struct {
	// Server is the normalized server name, it's empty if the column comes from the connected server.
	Server string
	// Database is the normalized database name, it should not be empty.
	Database string
	// Schema is the normalized schema name, it should not be empty for the engines that support schema, and should be empty for the engines that don't support schema.
	Schema string
	// Table is the normalized table name, it should not be empty.
	Table string
	// Column is the normalized column name, it should not be empty.
	Column string
}

// String returns the string format of the column resource.
func (c ColumnResource) String() string {
	var list []string
	if c.Server != "" {
		list = append(list, c.Server)
	}
	if c.Database != "" {
		list = append(list, c.Database)
	}
	if c.Schema != "" {
		list = append(list, c.Schema)
	}
	if c.Table != "" {
		list = append(list, c.Table)
	}
	if c.Column != "" {
		list = append(list, c.Column)
	}
	return strings.Join(list, ".")
}

type TableSource interface {
	// Interface guard to forbid other types outside this package to implement this interface.
	isTableSource()

	GetTableName() string
	GetSchemaName() string
	GetDatabaseName() string
	GetServerName() string
	// GetQuerySpanResult returns the query span result of the table, it's callers' responsibility
	// to make a copy of the result if they want to modify it.
	GetQuerySpanResult() []QuerySpanResult
}

// baseTableSource is the base implementation table source.
type baseTableSource struct {
}

// isTableSource implements the TableSource interface.
func (baseTableSource) isTableSource() {}

// PseudoTable is the resource of table, it's useful for some pseudo/temporary tables likes CTE, AS.
type PseudoTable struct {
	baseTableSource

	// Name is the normalized table name.
	Name string

	// Columns are the columns of the table.
	Columns []QuerySpanResult
}

func NewPseudoTable(name string, columns []QuerySpanResult) *PseudoTable {
	return &PseudoTable{
		Name:    name,
		Columns: columns,
	}
}

func (p *PseudoTable) GetTableName() string {
	return p.Name
}

func (*PseudoTable) GetSchemaName() string {
	return ""
}

func (*PseudoTable) GetDatabaseName() string {
	return ""
}

func (*PseudoTable) GetServerName() string {
	return ""
}

func (p *PseudoTable) GetQuerySpanResult() []QuerySpanResult {
	return p.Columns
}

// PhysicalView is the resource of a physical view, which can be refer with schema name,
// and its columns can refer to the columns of the underlying tables.
type PhysicalView struct {
	baseTableSource

	// Server is the normalized server name, it's empty if the column comes from the connected server.
	Server string
	// Database is the normalized database name, it should not be empty.
	Database string
	// Schema is the normalized schema name, it should not be empty for the engines that support schema, and should be empty for the engines that don't support schema.
	Schema string
	// Name is the normalized table name, it should not be empty.
	Name string
	// Columns are the columns of the table.
	Columns []QuerySpanResult
}

func (p *PhysicalView) GetTableName() string {
	return p.Name
}

func (p *PhysicalView) GetSchemaName() string {
	return p.Schema
}

func (p *PhysicalView) GetDatabaseName() string {
	return p.Database
}

func (p *PhysicalView) GetServerName() string {
	return p.Server
}

func (p *PhysicalView) GetQuerySpanResult() []QuerySpanResult {
	return p.Columns
}

// PhysicalTable is the resource of a physical table.
type PhysicalTable struct {
	baseTableSource

	// Server is the normalized server name, it's empty if the column comes from the connected server.
	Server string
	// Database is the normalized database name, it should not be empty.
	Database string
	// Schema is the normalized schema name, it should not be empty for the engines that support schema, and should be empty for the engines that don't support schema.
	Schema string
	// Name is the normalized table name, it should not be empty.
	Name string
	// Columns are the columns of the table.
	Columns []string
}

func (p *PhysicalTable) GetTableName() string {
	return p.Name
}

func (p *PhysicalTable) GetSchemaName() string {
	return p.Schema
}

func (p *PhysicalTable) GetDatabaseName() string {
	return p.Database
}

func (p *PhysicalTable) GetServerName() string {
	return p.Server
}

func (p *PhysicalTable) GetQuerySpanResult() []QuerySpanResult {
	result := make([]QuerySpanResult, 0, len(p.Columns))
	for _, column := range p.Columns {
		sourceColumnSet := make(SourceColumnSet, 1)
		sourceColumnSet[ColumnResource{
			Server:   p.Server,
			Database: p.Database,
			Schema:   p.Schema,
			Table:    p.Name,
			Column:   column,
		}] = true
		result = append(result, QuerySpanResult{
			Name:          column,
			SourceColumns: sourceColumnSet,
			IsPlainField:  true,
		})
	}
	return result
}

// Sequence is the resource of a sequence.
type Sequence struct {
	baseTableSource

	// Server is the normalized server name, it's empty if the column comes from the connected server.
	Server string
	// Database is the normalized database name, it should not be empty.
	Database string
	// Schema is the normalized schema name, it should not be empty for the engines that support schema, and should be empty for the engines that don't support schema.
	Schema string
	// Name is the normalized sequence name, it should not be empty.
	Name string
	// Columns are the columns of the sequence.
	Columns []string
}

func (p *Sequence) GetTableName() string {
	return p.Name
}

func (p *Sequence) GetSchemaName() string {
	return p.Schema
}

func (p *Sequence) GetDatabaseName() string {
	return p.Database
}

func (p *Sequence) GetServerName() string {
	return p.Server
}

func (p *Sequence) GetQuerySpanResult() []QuerySpanResult {
	result := make([]QuerySpanResult, 0, len(p.Columns))
	for _, column := range p.Columns {
		sourceColumnSet := make(SourceColumnSet, 1)
		sourceColumnSet[ColumnResource{
			Server:   p.Server,
			Database: p.Database,
			Schema:   p.Schema,
			Table:    p.Name,
			Column:   column,
		}] = true
		result = append(result, QuerySpanResult{
			Name:          column,
			SourceColumns: sourceColumnSet,
		})
	}
	return result
}

type GetQuerySpanContext struct {
	InstanceID                    string
	GetDatabaseMetadataFunc       GetDatabaseMetadataFunc
	ListDatabaseNamesFunc         ListDatabaseNamesFunc
	GetLinkedDatabaseMetadataFunc GetLinkedDatabaseMetadataFunc
	// TempTables is the temporary tables created in the query span.
	// It's used to store the temporary tables declared in one batch for SQL Server.
	TempTables map[string]*PhysicalTable

	// Adding the engine information here is a trade-off between the copy-pasted and shared code.
	// For engines with more different, we implement the getQuerySpan separately.
	// For some similar engines, we can share the same getQuerySpan implementation.
	// But they may have some differences, so we need to pass the engine information here.
	// No need to set this field when call GetQuerySpan, because the base.GetQuerySpan already has the engine information.
	// We'll deal this field in the base.GetQuerySpan.
	Engine storepb.Engine
}

// GetDatabaseMetadataFunc is the function to get database metadata.
type GetDatabaseMetadataFunc func(context.Context, string, string) (string, *model.DatabaseMetadata, error)

// ListDatabaseNamesFunc is the function to list database names.
type ListDatabaseNamesFunc func(context.Context, string) ([]string, error)

type GetLinkedDatabaseMetadataFunc func(context.Context, string, string, string) (string, string, *model.DatabaseMetadata, error)

func (s *QuerySpan) ToYaml() *YamlQuerySpan {
	y := &YamlQuerySpan{
		Type:             s.Type,
		Results:          []YamlQuerySpanResult{},
		SourceColumns:    []ColumnResource{},
		PredicateColumns: []ColumnResource{},
	}
	for _, result := range s.Results {
		yamlResult := &YamlQuerySpanResult{
			Name:          result.Name,
			SourceColumns: []ColumnResource{},
			IsPlainField:  result.IsPlainField,
		}
		for k := range result.SourceColumns {
			yamlResult.SourceColumns = append(yamlResult.SourceColumns, k)
		}
		sort.Slice(yamlResult.SourceColumns, func(i, j int) bool {
			vi, vj := yamlResult.SourceColumns[i], yamlResult.SourceColumns[j]
			return vi.String() < vj.String()
		})
		y.Results = append(y.Results, *yamlResult)
	}
	for k := range s.SourceColumns {
		y.SourceColumns = append(y.SourceColumns, k)
	}
	sort.Slice(y.SourceColumns, func(i, j int) bool {
		vi, vj := y.SourceColumns[i], y.SourceColumns[j]
		return vi.String() < vj.String()
	})
	for k := range s.PredicateColumns {
		y.PredicateColumns = append(y.PredicateColumns, k)
	}
	sort.Slice(y.PredicateColumns, func(i, j int) bool {
		vi, vj := y.PredicateColumns[i], y.PredicateColumns[j]
		return vi.String() < vj.String()
	})
	return y
}

type YamlQuerySpan struct {
	Type             QueryType
	Results          []YamlQuerySpanResult
	SourceColumns    []ColumnResource
	PredicateColumns []ColumnResource
}

type YamlQuerySpanResult struct {
	Name          string
	SourceColumns []ColumnResource
	IsPlainField  bool
}
