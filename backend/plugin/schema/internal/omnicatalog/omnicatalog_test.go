package omnicatalog

import (
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

func TestDefinitionAdaptersAndRegistrations(t *testing.T) {
	require.Nil(t, ignoreTableScope(nil))
	require.Nil(t, ignoreObjectScope[string](nil))

	const testDefEngine = storepb.Engine(99901)
	schema.Register(schema.Engine{
		Definitions: MySQLFamilyDefinitions{
			Database: func(ctx schema.GetDefinitionContext, meta *metadatapb.DatabaseSchemaMetadata) (string, error) {
				if ctx.SDLFormat {
					return "sdl:" + meta.GetName(), nil
				}
				return "dump:" + meta.GetName(), nil
			},
			Table:     func(tbl *metadatapb.TableMetadata) (string, error) { return "table:" + tbl.GetName(), nil },
			View:      func(v *metadatapb.ViewMetadata) (string, error) { return "view:" + v.GetName(), nil },
			Function:  func(fn *metadatapb.FunctionMetadata) (string, error) { return "fn:" + fn.GetName(), nil },
			Procedure: func(p *metadatapb.ProcedureMetadata) (string, error) { return "proc:" + p.GetName(), nil },
		}.SchemaDefinitions(),
	}, testDefEngine)

	got, err := schema.GetDatabaseDefinition(testDefEngine, schema.GetDefinitionContext{SDLFormat: true}, &metadatapb.DatabaseSchemaMetadata{Name: "db"})
	require.NoError(t, err)
	require.Equal(t, "sdl:db", got)

	got, err = schema.GetTableDefinition(testDefEngine, "ignored", &metadatapb.TableMetadata{Name: "t"}, nil)
	require.NoError(t, err)
	require.Equal(t, "table:t", got)

	got, err = schema.GetViewDefinition(testDefEngine, "ignored", &metadatapb.ViewMetadata{Name: "v"})
	require.NoError(t, err)
	require.Equal(t, "view:v", got)

	got, err = schema.GetFunctionDefinition(testDefEngine, "ignored", &metadatapb.FunctionMetadata{Name: "f"})
	require.NoError(t, err)
	require.Equal(t, "fn:f", got)

	got, err = schema.GetProcedureDefinition(testDefEngine, "ignored", &metadatapb.ProcedureMetadata{Name: "p"})
	require.NoError(t, err)
	require.Equal(t, "proc:p", got)

	_, err = schema.GetSequenceDefinition(testDefEngine, "", &metadatapb.SequenceMetadata{Name: "s"})
	require.ErrorContains(t, err, "is not supported")
}

type fakePlan struct{ sql string }

func TestSDLAdviceHelpers(t *testing.T) {
	drop := SDLDropAdvice("drop t")
	require.Equal(t, storepb.Advice_WARNING, drop.Status)
	require.Equal(t, code.SDLDropOperation.Int32(), drop.Code)
	require.Equal(t, "DROP operation detected", drop.Title)
	require.Equal(t, "drop t", drop.Content)

	replace := SDLReplaceAdvice("replace f")
	require.Equal(t, storepb.Advice_WARNING, replace.Status)
	require.Equal(t, code.SDLReplaceOperation.Int32(), replace.Code)
	require.Equal(t, "CREATE OR REPLACE operation detected", replace.Title)
	require.Equal(t, "replace f", replace.Content)

	const testSDLEngine = storepb.Engine(99903)
	schema.Register(schema.Engine{
		Definitions: schema.Definitions{
			Database: func(ctx schema.GetDefinitionContext, meta *metadatapb.DatabaseSchemaMetadata) (string, error) {
				require.True(t, ctx.SkipBackupSchema)
				require.True(t, ctx.SDLFormat)
				return "source:" + meta.GetName(), nil
			},
		},
	}, testSDLEngine)
	meta := model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{Name: "app"}, nil, nil, testSDLEngine, true)

	collectCalled := false
	advices, err := PlanSDLDropAdvices(
		testSDLEngine,
		"target",
		meta,
		func(src, dst string) (*fakePlan, error) {
			require.Equal(t, "source:app", src)
			require.Equal(t, "target", dst)
			return nil, nil
		},
		func(*fakePlan) []*storepb.Advice {
			collectCalled = true
			return nil
		},
	)
	require.NoError(t, err)
	require.Nil(t, advices)
	require.False(t, collectCalled)

	advices, err = PlanSDLDropAdvices(
		testSDLEngine,
		"target",
		meta,
		func(_, _ string) (*fakePlan, error) { return &fakePlan{sql: "DROP TABLE t"}, nil },
		func(p *fakePlan) []*storepb.Advice { return []*storepb.Advice{SDLDropAdvice(p.sql)} },
	)
	require.NoError(t, err)
	require.Len(t, advices, 1)
	require.Equal(t, "DROP TABLE t", advices[0].Content)
}
