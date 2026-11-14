package tsql

import (
	"context"
	"errors"

	"github.com/bytebase/omni/metadata"
	"github.com/bytebase/omni/mssql/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	base.RegisterGetQuerySpan(storepb.Engine_MSSQL, GetQuerySpan)
}

// GetQuerySpan returns the query span for the given statement.
func GetQuerySpan(
	ctx context.Context,
	gCtx base.GetQuerySpanContext,
	stmt base.Statement,
	database, schema string,
	ignoreCaseSensitive bool,
) (*base.QuerySpan, error) {
	return resolveQuerySpan(ctx, gCtx, stmt.Text, database, schema, ignoreCaseSensitive)
}

func resolveQuerySpan(ctx context.Context, gCtx base.GetQuerySpanContext, statement, database, schema string, ignoreCaseSensitive bool) (*base.QuerySpan, error) {
	tempTables := newTempTables(gCtx.TempTables)
	span, err := analysis.ResolveQuerySpan(statement, analysis.ResolveOptions{
		DefaultDatabase:     database,
		DefaultSchema:       schema,
		IgnoreCaseSensitive: ignoreCaseSensitive,
		Catalog:             newCatalog(ctx, gCtx),
		TempTables:          tempTables.omni,
		PlanOutput:          gCtx.PlanOutput,
	})
	tempTables.store(gCtx.TempTables)
	if err != nil {
		return nil, convertQuerySpanError(err)
	}
	querySpan := &base.QuerySpan{
		Type:             convertQueryType(span.Type),
		ExplainAnalyze:   span.ExplainAnalyze,
		SourceColumns:    convertColumnSet(span.SourceColumns),
		Results:          convertResults(span.Results),
		PredicateColumns: convertColumnSet(span.PredicateColumns),
	}
	if span.NotFoundError != nil {
		querySpan.NotFoundError = convertNotFoundError(span.NotFoundError)
	}
	return querySpan, nil
}

// newCatalog reads the instance's synced metadata for omni. A lookup error
// carrying a ResourceNotFoundError becomes omni's NotFoundError, which omni
// reports on the span rather than as an error.
func newCatalog(ctx context.Context, gCtx base.GetQuerySpanContext) analysis.Catalog {
	var catalog analysis.Catalog
	if gCtx.ListDatabaseNamesFunc != nil {
		catalog.ListDatabaseNames = func() ([]string, error) {
			return gCtx.ListDatabaseNamesFunc(ctx, gCtx.InstanceID)
		}
	}
	if gCtx.GetDatabaseMetadataFunc != nil {
		catalog.GetDatabase = func(name string) (analysis.Database, error) {
			_, database, err := gCtx.GetDatabaseMetadataFunc(ctx, gCtx.InstanceID, name)
			if err != nil {
				var notFound *base.ResourceNotFoundError
				if errors.As(err, &notFound) {
					return nil, &analysis.NotFoundError{
						Err:      notFound.Err,
						Database: notFound.Database,
						Schema:   notFound.Schema,
						Table:    notFound.Table,
						Column:   notFound.Column,
					}
				}
				return nil, err
			}
			if database == nil {
				return nil, nil
			}
			return catalogDatabase{database: database}, nil
		}
	}
	return catalog
}

type catalogDatabase struct {
	database *model.DatabaseMetadata
}

func (d catalogDatabase) ListSchemaNames() []string { return d.database.ListSchemaNames() }

func (d catalogDatabase) GetSchema(name string) analysis.Schema {
	schema := d.database.GetSchemaMetadata(name)
	if schema == nil {
		return nil
	}
	return catalogSchema{schema: schema}
}

type catalogSchema struct {
	schema *model.SchemaMetadata
}

func (s catalogSchema) ListTableNames() []string { return s.schema.ListTableNames() }

func (s catalogSchema) GetTable(name string) *metadata.TableMetadata {
	table := s.schema.GetTable(name)
	if table == nil {
		return nil
	}
	return table.GetProto()
}

func (s catalogSchema) ListViewNames() []string { return s.schema.ListViewNames() }

func (s catalogSchema) GetView(name string) *metadata.ViewMetadata { return s.schema.GetView(name) }

// tempTables carries a batch's temp tables to omni and back. omni adds the
// ones a statement declares; store copies those into the batch's map.
type tempTables struct {
	omni     map[string]*analysis.TempTable
	original map[string]*analysis.TempTable
}

func newTempTables(batch map[string]*base.PhysicalTable) tempTables {
	if batch == nil {
		return tempTables{}
	}
	t := tempTables{
		omni:     make(map[string]*analysis.TempTable, len(batch)),
		original: make(map[string]*analysis.TempTable, len(batch)),
	}
	for name, table := range batch {
		var converted *analysis.TempTable
		if table != nil {
			converted = &analysis.TempTable{
				Server:   table.Server,
				Database: table.Database,
				Schema:   table.Schema,
				Name:     table.Name,
				Columns:  table.Columns,
			}
		}
		t.omni[name] = converted
		t.original[name] = converted
	}
	return t
}

func (t tempTables) store(batch map[string]*base.PhysicalTable) {
	for name, table := range t.omni {
		if original, ok := t.original[name]; ok && original == table {
			continue
		}
		batch[name] = &base.PhysicalTable{
			Server:   table.Server,
			Database: table.Database,
			Schema:   table.Schema,
			Name:     table.Name,
			Columns:  table.Columns,
		}
	}
}

func convertQuerySpanError(err error) error {
	if err == analysis.ErrMixUserSystemTables {
		return base.MixUserSystemTablesError
	}
	var typeNotSupported *analysis.TypeNotSupportedError
	var notFound *analysis.NotFoundError
	if errors.As(err, &typeNotSupported) || errors.As(err, &notFound) {
		return querySpanError{err}
	}
	return err
}

// querySpanError is an omni resolution error that matches the parser/base
// error types its omni counterparts convert to.
type querySpanError struct {
	error
}

func (e querySpanError) Unwrap() error { return e.error }

func (e querySpanError) As(target any) bool {
	switch target := target.(type) {
	case **base.TypeNotSupportedError:
		var err *analysis.TypeNotSupportedError
		if !errors.As(e.error, &err) {
			return false
		}
		*target = &base.TypeNotSupportedError{Err: err.Err, Type: err.Type, Name: err.Name, Extra: err.Extra}
		return true
	case **base.ResourceNotFoundError:
		var err *analysis.NotFoundError
		if !errors.As(e.error, &err) {
			return false
		}
		*target = convertNotFoundError(err)
		return true
	default:
		return false
	}
}

func convertNotFoundError(err *analysis.NotFoundError) *base.ResourceNotFoundError {
	return &base.ResourceNotFoundError{
		Err:      err.Err,
		Database: err.Database,
		Schema:   err.Schema,
		Table:    err.Table,
		Column:   err.Column,
	}
}

func convertQueryType(t analysis.QueryType) base.QueryType {
	switch t {
	case analysis.Select:
		return base.Select
	case analysis.Explain:
		return base.Explain
	case analysis.SelectInfoSchema:
		return base.SelectInfoSchema
	case analysis.DDL:
		return base.DDL
	case analysis.DML:
		return base.DML
	default:
		return base.QueryTypeUnknown
	}
}

func convertColumnSet(set analysis.ColumnSet) base.SourceColumnSet {
	if set == nil {
		return nil
	}
	result := make(base.SourceColumnSet, len(set))
	for column := range set {
		result[base.ColumnResource{
			Server:   column.Server,
			Database: column.Database,
			Schema:   column.Schema,
			Table:    column.Table,
			Column:   column.Column,
		}] = true
	}
	return result
}

func convertResults(results []analysis.QuerySpanResult) []base.QuerySpanResult {
	if results == nil {
		return nil
	}
	converted := make([]base.QuerySpanResult, 0, len(results))
	for _, result := range results {
		converted = append(converted, base.QuerySpanResult{
			Name:          result.Name,
			SourceColumns: convertColumnSet(result.SourceColumns),
			IsPlainField:  result.IsPlainField,
		})
	}
	return converted
}
