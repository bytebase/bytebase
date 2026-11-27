package mssql

import (
	"context"
	"errors"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/bytebase/omni/mssql/analysis"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func strPtr(s string) *string { return &s }

func TestGetQuerySpanSeam(t *testing.T) {
	dbMeta := model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{
		Name: "app",
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "dbo",
				Tables: []*metadatapb.TableMetadata{
					{
						Name: "users",
						Columns: []*metadatapb.ColumnMetadata{
							{Name: "id", Type: "int"},
							{Name: "name", Type: "nvarchar"},
						},
					},
				},
				Views: []*metadatapb.ViewMetadata{
					{Name: "v_users", Definition: "SELECT id FROM dbo.users"},
				},
			},
		},
	}, nil, nil, storepb.Engine_MSSQL, false)

	gCtx := base.GetQuerySpanContext{
		InstanceID: "inst-1",
		ListDatabaseNamesFunc: func(_ context.Context, _ string) ([]string, error) {
			return []string{"app"}, nil
		},
		GetDatabaseMetadataFunc: func(_ context.Context, _, dbName string) (string, *model.DatabaseMetadata, error) {
			switch dbName {
			case "app":
				return dbName, dbMeta, nil
			case "missing":
				return "", nil, &base.ResourceNotFoundError{Err: errors.New("missing db"), Database: strPtr("missing")}
			default:
				return "", nil, nil
			}
		},
	}
	querySpans := func(statements ...string) ([]*base.QuerySpan, error) {
		stmts := make([]base.Statement, 0, len(statements))
		for _, s := range statements {
			stmts = append(stmts, base.Statement{Text: s})
		}
		return base.GetQuerySpan(context.Background(), gCtx, storepb.Engine_MSSQL, stmts, "app", "dbo", true)
	}

	// Table and predicate column mapping via registry.
	spans, err := querySpans("SELECT name FROM dbo.users WHERE id = 1;")
	require.NoError(t, err)
	require.Len(t, spans, 1)
	span := spans[0]
	require.Equal(t, base.Select, span.Type)
	require.Len(t, span.Results, 1)
	require.True(t, span.Results[0].SourceColumns[base.ColumnResource{Database: "app", Schema: "dbo", Table: "users", Column: "name"}])
	require.True(t, span.PredicateColumns[base.ColumnResource{Database: "app", Schema: "dbo", Table: "users", Column: "id"}])

	// Temp tables carry across the statements of a script.
	spans, err = querySpans("CREATE TABLE #tmp (c1 INT);", "SELECT c1 FROM #tmp;")
	require.NoError(t, err)
	require.Len(t, spans[1].Results, 1)
	require.Equal(t, "c1", spans[1].Results[0].Name)

	// ResourceNotFoundError returned by metadata callback maps to span.NotFoundError.
	spans, err = querySpans("SELECT id FROM missing.dbo.users;")
	require.NoError(t, err)
	var notFoundErr *base.ResourceNotFoundError
	require.ErrorAs(t, spans[0].NotFoundError, &notFoundErr)
	require.Equal(t, strPtr("missing"), notFoundErr.Database)

	// Plan output carries across the statements of a script, and not into the next script.
	spans, err = querySpans("SET SHOWPLAN_ALL ON;", "SELECT id FROM dbo.users;")
	require.NoError(t, err)
	require.Equal(t, base.Explain, spans[1].Type)
	spans, err = querySpans("SELECT id FROM dbo.users;")
	require.NoError(t, err)
	require.Equal(t, base.Select, spans[0].Type)

	// Mixing user and system tables maps to base.MixUserSystemTablesError.
	_, err = querySpans("SELECT u.id, t.name FROM dbo.users u JOIN sys.tables t ON 1=1;")
	require.ErrorIs(t, err, base.MixUserSystemTablesError)

	// TypeNotSupportedError and ResourceNotFoundError convert to base's.
	var typeErr *base.TypeNotSupportedError
	_, err = querySpans("SELECT * FROM CONTAINSTABLE(dbo.users, name, 'x');")
	require.ErrorAs(t, err, &typeErr)
	require.Equal(t, "*ast.FullTextTableRef", typeErr.Type)

	var nfErr *base.ResourceNotFoundError
	spans, err = querySpans("SELECT id FROM dbo.ghost;")
	require.NoError(t, err)
	require.ErrorAs(t, spans[0].NotFoundError, &nfErr)
	require.Equal(t, strPtr("ghost"), nfErr.Table)

	require.Equal(t, base.SelectInfoSchema, base.OmniQueryType(analysis.SelectInfoSchema))
	require.Equal(t, base.DDL, base.OmniQueryType(analysis.DDL))
	require.Equal(t, base.DML, base.OmniQueryType(analysis.DML))
	require.Equal(t, base.QueryTypeUnknown, base.OmniQueryType(analysis.QueryType(999)))
}
