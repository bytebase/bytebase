package schema

import (
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func TestRegister(t *testing.T) {
	const (
		engineA = storepb.Engine(99911)
		engineB = storepb.Engine(99912)
	)
	t.Cleanup(func() {
		registryMu.Lock()
		defer registryMu.Unlock()
		delete(engines, engineA)
		delete(engines, engineB)
	})

	Register(Engine{
		Definitions: Definitions{
			Database: func(_ GetDefinitionContext, m *metadatapb.DatabaseSchemaMetadata) (string, error) {
				return "db:" + m.GetName(), nil
			},
			Table: func(s string, tbl *metadatapb.TableMetadata, _ []*metadatapb.SequenceMetadata) (string, error) {
				return s + "." + tbl.GetName(), nil
			},
			View: func(s string, v *metadatapb.ViewMetadata) (string, error) {
				return s + "." + v.GetName(), nil
			},
		},
	}, engineA, engineB)

	// Disjoint capability registration for an already-registered engine succeeds.
	Register(Engine{
		DatabaseMetadata: func(text string) (*metadatapb.DatabaseSchemaMetadata, error) {
			return &metadatapb.DatabaseSchemaMetadata{Name: text}, nil
		},
	}, engineA)

	for _, engine := range []storepb.Engine{engineA, engineB} {
		got, err := GetDatabaseDefinition(engine, GetDefinitionContext{}, &metadatapb.DatabaseSchemaMetadata{Name: "app"})
		require.NoError(t, err)
		require.Equal(t, "db:app", got)

		got, err = GetTableDefinition(engine, "public", &metadatapb.TableMetadata{Name: "users"}, nil)
		require.NoError(t, err)
		require.Equal(t, "public.users", got)

		got, err = GetViewDefinition(engine, "public", &metadatapb.ViewMetadata{Name: "active_users"})
		require.NoError(t, err)
		require.Equal(t, "public.active_users", got)

		_, err = GetFunctionDefinition(engine, "public", &metadatapb.FunctionMetadata{Name: "fn"})
		require.ErrorContains(t, err, "is not supported")
	}

	meta, err := GetDatabaseMetadata(engineA, "app")
	require.NoError(t, err)
	require.Equal(t, "app", meta.GetName())

	require.PanicsWithValue(t, "schema: Database registered twice for 99911", func() {
		Register(Engine{
			Definitions: Definitions{
				Database: func(_ GetDefinitionContext, _ *metadatapb.DatabaseSchemaMetadata) (string, error) {
					return "", nil
				},
			},
		}, engineA)
	})
}

func TestUnsupportedEngineErrors(t *testing.T) {
	const missing = storepb.Engine(99913)

	_, err := GetSchemaDefinition(missing, &metadatapb.SchemaMetadata{})
	require.EqualError(t, err, "engine 99913 is not supported")

	_, err = GetDatabaseMetadata(missing, "")
	require.EqualError(t, err, "engine 99913 is not supported")

	_, err = GetMultiFileDatabaseDefinition(missing, GetDefinitionContext{}, &metadatapb.DatabaseSchemaMetadata{})
	require.EqualError(t, err, "engine 99913 is not supported for multi-file database definition")

	_, err = SDLDropAdvices(missing, "", nil, "")
	require.EqualError(t, err, "engine 99913 is not supported for SDL drop advices")

	_, err = DiffSDLMigration(missing, "", "", "")
	require.EqualError(t, err, "engine 99913 is not supported for SDL diff migration")

	_, err = DiffMigration(missing, nil, nil)
	require.EqualError(t, err, "engine 99913 is not supported")
}
