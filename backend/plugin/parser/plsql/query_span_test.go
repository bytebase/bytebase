package plsql

import (
	"context"
	"errors"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/bytebase/omni/oracle/analysis"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func strPtr(s string) *string { return &s }

func TestGetQuerySpanSeam(t *testing.T) {
	localDB := model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{
		Name: "APP",
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "",
				Tables: []*metadatapb.TableMetadata{
					{
						Name: "EMP",
						Columns: []*metadatapb.ColumnMetadata{
							{Name: "ID", Type: "NUMBER"},
							{Name: "SECRET", Type: "VARCHAR2", IsInvisible: true},
						},
					},
				},
				Views: []*metadatapb.ViewMetadata{
					{Name: "V_EMP", Definition: "SELECT id FROM emp"},
					{Name: "V_MIX", Definition: "SELECT e.id, u.username FROM emp e JOIN sys.all_users u ON 1=1"},
				},
			},
		},
		LinkedDatabases: []*metadatapb.LinkedDatabaseMetadata{
			{Name: "REMOTE_LINK", Username: "REMOTE_APP"},
		},
	}, nil, nil, storepb.Engine_ORACLE, false)

	remoteDB := model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{
		Name: "REMOTE_APP",
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "",
				Tables: []*metadatapb.TableMetadata{
					{
						Name:    "DEPT",
						Columns: []*metadatapb.ColumnMetadata{{Name: "NAME", Type: "VARCHAR2"}},
					},
				},
			},
		},
	}, nil, nil, storepb.Engine_ORACLE, false)

	gCtx := base.GetQuerySpanContext{
		InstanceID: "inst-local",
		GetDatabaseMetadataFunc: func(_ context.Context, _, dbName string) (string, *model.DatabaseMetadata, error) {
			switch dbName {
			case "APP":
				return dbName, localDB, nil
			case "MISSING":
				return "", nil, &base.ResourceNotFoundError{Err: errors.New("db missing"), Database: strPtr("MISSING")}
			default:
				return "", nil, nil
			}
		},
		GetLinkedDatabaseMetadataFunc: func(_ context.Context, _, link, _ string) (string, string, *model.DatabaseMetadata, error) {
			if link == "REMOTE_LINK" {
				return "inst-remote", "REMOTE_APP", remoteDB, nil
			}
			return "", "", nil, &base.ResourceNotFoundError{Err: errors.New("link missing"), DatabaseLink: strPtr(link)}
		},
	}

	// Local table + invisible column skipping + QueryType mapping via registry.
	spans, err := base.GetQuerySpan(context.Background(), gCtx, storepb.Engine_ORACLE, []base.Statement{{Text: "SELECT * FROM emp"}}, "APP", "", false)
	require.NoError(t, err)
	require.Len(t, spans, 1)
	span := spans[0]
	require.Equal(t, base.Select, span.Type)
	require.Len(t, span.Results, 1)
	require.Equal(t, "ID", span.Results[0].Name)
	require.False(t, span.Results[0].Invisible)
	require.True(t, span.Results[0].SourceColumns[base.ColumnResource{Database: "APP", Table: "EMP", Column: "ID"}])

	// OmniQuerySpanResults preserves Invisible and IsPlainField flags across the seam.
	converted := base.OmniQuerySpanResults([]analysis.QuerySpanResult{{Name: "SECRET", IsPlainField: true, Invisible: true}})
	require.Len(t, converted, 1)
	require.True(t, converted[0].IsPlainField)
	require.True(t, converted[0].Invisible)

	// Linked database callback resolves @remote_link through GetLinkedDatabaseMetadataFunc.
	span, err = GetQuerySpan(context.Background(), gCtx, base.Statement{Text: "SELECT name FROM dept@remote_link"}, "APP", "", false)
	require.NoError(t, err)
	require.Len(t, span.Results, 1)
	require.True(t, span.Results[0].SourceColumns[base.ColumnResource{Instance: "inst-remote", Server: "REMOTE_LINK", Database: "REMOTE_APP", Table: "DEPT", Column: "NAME"}] ||
		span.Results[0].SourceColumns[base.ColumnResource{Database: "REMOTE_APP", Table: "DEPT", Column: "NAME"}])

	// ResourceNotFoundError returned from metadata callback round-trips onto QuerySpan.NotFoundError.
	span, err = GetQuerySpan(context.Background(), gCtx, base.Statement{Text: "SELECT id FROM missing.t"}, "APP", "", false)
	require.NoError(t, err)
	var notFoundErr *base.ResourceNotFoundError
	require.ErrorAs(t, span.NotFoundError, &notFoundErr)
	require.Equal(t, strPtr("MISSING"), notFoundErr.Database)

	// Mixing user and system tables directly maps to base.MixUserSystemTablesError.
	_, err = GetQuerySpan(context.Background(), gCtx, base.Statement{Text: "SELECT e.id, u.username FROM emp e JOIN sys.all_users u ON 1=1"}, "APP", "", false)
	require.ErrorIs(t, err, base.MixUserSystemTablesError)

	// Mixing user and system tables inside a view wraps ErrMixUserSystemTables via mixedTablesError.
	_, err = GetQuerySpan(context.Background(), gCtx, base.Statement{Text: "SELECT id FROM v_mix"}, "APP", "", false)
	require.ErrorIs(t, err, base.MixUserSystemTablesError)

	// Syntax error wraps with statement context.
	_, err = GetQuerySpan(context.Background(), gCtx, base.Statement{Text: "SELECT FROM ;"}, "APP", "", false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to get query span from statement")

	// QueryType conversions.
	require.Equal(t, base.SelectInfoSchema, base.OmniQueryType(analysis.SelectInfoSchema))
	require.Equal(t, base.DDL, base.OmniQueryType(analysis.DDL))
	require.Equal(t, base.DML, base.OmniQueryType(analysis.DML))
	require.Equal(t, base.Explain, base.OmniQueryType(analysis.Explain))
	require.Equal(t, base.QueryTypeUnknown, base.OmniQueryType(analysis.QueryType(999)))
}
