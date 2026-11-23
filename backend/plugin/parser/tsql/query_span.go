package tsql

import (
	"context"
	"errors"

	omnianalysis "github.com/bytebase/omni/analysis"
	"github.com/bytebase/omni/metadata"
	"github.com/bytebase/omni/mssql/analysis"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

// GetQuerySpan returns the query span for the given statement.
func GetQuerySpan(
	ctx context.Context,
	gCtx base.GetQuerySpanContext,
	stmt base.Statement,
	database, schema string,
	ignoreCaseSensitive bool,
) (*base.QuerySpan, error) {
	tempTables := newTempTables(gCtx.TempTables)
	span, err := analysis.ResolveQuerySpan(stmt.Text, analysis.ResolveOptions{
		DefaultDatabase:     database,
		DefaultSchema:       schema,
		IgnoreCaseSensitive: ignoreCaseSensitive,
		Catalog: analysis.Catalog{
			ListDatabaseNames: base.QuerySpanListDatabasesFunc(ctx, gCtx, false),
			GetDatabase: base.QuerySpanDatabaseGetter(ctx, gCtx, true, func(db *model.DatabaseMetadata) analysis.Database {
				return catalogDatabase{database: db}
			}),
		},
		TempTables: tempTables.omni,
		PlanOutput: gCtx.PlanOutput,
	})
	tempTables.store(gCtx.TempTables)
	if err != nil {
		return nil, convertQuerySpanError(err)
	}
	return base.OmniResolvedSpan(&omnianalysis.ResolvedSpan{
		Type:             span.Type,
		ExplainAnalyze:   span.ExplainAnalyze,
		SourceColumns:    span.SourceColumns,
		Results:          span.Results,
		PredicateColumns: span.PredicateColumns,
		NotFoundError:    span.NotFoundError,
	}), nil
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
	if mixErr := base.ConvertOmniMixTablesError(err); mixErr != nil {
		return mixErr
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
		*target = base.OmniTypeNotSupportedError(err)
		return true
	case **base.ResourceNotFoundError:
		var err *analysis.NotFoundError
		if !errors.As(e.error, &err) {
			return false
		}
		*target = base.OmniResourceNotFoundError(err)
		return true
	default:
		return false
	}
}
