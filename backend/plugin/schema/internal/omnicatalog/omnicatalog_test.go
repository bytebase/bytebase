package omnicatalog

import (
	"context"
	"testing"

	omnianalysis "github.com/bytebase/omni/analysis"
	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

// TestSnapshotObjectFindsSequences covers the report key the TiDB catalog uses
// for a degraded sequence, so a changed sequence is refused rather than diffed
// as a stand-in.
func TestSnapshotObjectFindsSequences(t *testing.T) {
	meta := &metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{
		Name:      "public",
		Sequences: []*metadatapb.SequenceMetadata{{Name: "s", Start: "1", LastValue: "42"}},
	}}}
	object := snapshotObject(meta, "sequence:s")
	sequence, ok := object.(*metadatapb.SequenceMetadata)
	require.True(t, ok)
	require.Equal(t, "s", sequence.GetName())
	require.Equal(t, "1", sequence.GetStart())
	require.Empty(t, sequence.GetLastValue())
}

type fakeDiff struct {
	empty bool
	sql   string
}

func (d fakeDiff) IsEmpty() bool { return d.empty }

type fakePlan struct{ sql string }

func (p fakePlan) SQL() string { return p.sql }

type fakeReport struct {
	problems map[string]error
}

type fakeCatalog struct {
	loadedDB   string
	loadedMeta *metadatapb.DatabaseSchemaMetadata
	loadErr    error
	report     *fakeReport
	afterRan   bool
	execCalls  []string
	exportedDB string
}

func (c *fakeCatalog) LoadMetadataInto(_ context.Context, meta *metadatapb.DatabaseSchemaMetadata, db string) (*fakeReport, error) {
	c.loadedDB = db
	c.loadedMeta = meta
	if c.loadErr != nil {
		return nil, c.loadErr
	}
	if c.report != nil {
		return c.report, nil
	}
	return &fakeReport{}, nil
}

func (c *fakeCatalog) DatabaseMetadata(dbName string) *metadatapb.DatabaseSchemaMetadata {
	c.exportedDB = dbName
	return &metadatapb.DatabaseSchemaMetadata{Name: dbName}
}

func TestDiffSQL(t *testing.T) {
	migrateCalled := false
	fn := DiffSQL(
		func(_, _ *fakeCatalog) fakeDiff { return fakeDiff{empty: true} },
		func(_, _ *fakeCatalog, _ fakeDiff) fakePlan {
			migrateCalled = true
			return fakePlan{sql: "SHOULD NOT RUN"}
		},
	)
	sql, empty := fn(&fakeCatalog{}, &fakeCatalog{})
	require.True(t, empty)
	require.Empty(t, sql)
	require.False(t, migrateCalled)

	fn = DiffSQL(
		func(_, _ *fakeCatalog) fakeDiff { return fakeDiff{sql: "CREATE TABLE t (id int)"} },
		func(_, _ *fakeCatalog, d fakeDiff) fakePlan { return fakePlan{sql: d.sql} },
	)
	sql, empty = fn(&fakeCatalog{}, &fakeCatalog{})
	require.False(t, empty)
	require.Equal(t, "CREATE TABLE t (id int)", sql)
}

func TestMySQLFamilyEngineAndDiffMetadataMigration(t *testing.T) {
	var created []*fakeCatalog
	engine := MySQLFamilyEngine(
		func() *fakeCatalog {
			c := &fakeCatalog{}
			created = append(created, c)
			return c
		},
		func(r *fakeReport) []map[string]error {
			if r == nil || len(r.problems) == 0 {
				return nil
			}
			return []map[string]error{r.problems}
		},
		func(_, _ *fakeCatalog) fakeDiff {
			return fakeDiff{sql: "ALTER TABLE `" + Database + "`.`t` ADD COLUMN `b` int"}
		},
		func(_, _ *fakeCatalog, d fakeDiff) fakePlan { return fakePlan{sql: d.sql} },
		func(c *fakeCatalog, _ *metadatapb.DatabaseSchemaMetadata) { c.afterRan = true },
	)

	// Nil *model.DatabaseMetadata inputs are handled safely, loaded under Database,
	// run afterLoad, and strip the synthetic database qualifier.
	sql, err := DiffMetadataMigration(engine, nil, nil)
	require.NoError(t, err)
	require.Equal(t, "ALTER TABLE `t` ADD COLUMN `b` int", sql)
	require.Len(t, created, 2)
	for _, c := range created {
		require.Equal(t, Database, c.loadedDB)
		require.True(t, c.afterRan)
	}

	// Case-only renames on case-sensitive snapshots are rejected before loading.
	oldMeta := model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{
		Schemas: []*metadatapb.SchemaMetadata{{Tables: []*metadatapb.TableMetadata{{Name: "Foo"}}}},
	}, nil, nil, storepb.Engine_MYSQL, true)
	newMeta := model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{
		Schemas: []*metadatapb.SchemaMetadata{{Tables: []*metadatapb.TableMetadata{{Name: "foo"}}}},
	}, nil, nil, storepb.Engine_MYSQL, true)
	_, err = DiffMetadataMigration(engine, oldMeta, newMeta)
	require.ErrorContains(t, err, "case-only rename")

	// Load errors skip afterLoad and wrap the error with source/target context.
	afterCalled := false
	errEngine := MySQLFamilyEngine(
		func() *fakeCatalog { return &fakeCatalog{loadErr: errors.New("boom")} },
		func(*fakeReport) []map[string]error { return nil },
		func(_, _ *fakeCatalog) fakeDiff { return fakeDiff{empty: true} },
		func(_, _ *fakeCatalog, _ fakeDiff) fakePlan { return fakePlan{} },
		func(*fakeCatalog, *metadatapb.DatabaseSchemaMetadata) { afterCalled = true },
	)
	_, err = DiffMetadataMigration(errEngine, nil, nil)
	require.ErrorContains(t, err, "failed to load source schema")
	require.False(t, afterCalled)
}

func TestDiffMetadataSkipStripAndPostProcess(t *testing.T) {
	postCalls := 0
	rawSQL := "SELECT `" + Database + "`.`t`"
	engine := Engine[fakeCatalog, fakeReport]{
		New: func() *fakeCatalog { return &fakeCatalog{} },
		Load: func(_ *fakeCatalog, _ *metadatapb.DatabaseSchemaMetadata) (*fakeReport, error) {
			return &fakeReport{}, nil
		},
		Diff: func(_, _ *fakeCatalog) (string, bool) {
			return rawSQL, false
		},
		SkipStripDatabaseContext: true,
		PostProcess: func(sql string) string {
			postCalls++
			return sql + "\n"
		},
	}

	got, err := DiffMetadata(engine, nil, nil)
	require.NoError(t, err)
	require.Equal(t, rawSQL+"\n", got)
	require.Equal(t, 1, postCalls)

	// Empty diffs return "" without invoking PostProcess.
	engine.Diff = func(_, _ *fakeCatalog) (string, bool) { return "", true }
	got, err = DiffMetadata(engine, nil, nil)
	require.NoError(t, err)
	require.Empty(t, got)
	require.Equal(t, 1, postCalls)
}

func TestDatabaseProto(t *testing.T) {
	require.Nil(t, DatabaseProto(nil))
	p := &metadatapb.DatabaseSchemaMetadata{Name: "app"}
	db := model.NewDatabaseMetadata(p, nil, nil, storepb.Engine_POSTGRES, false)
	require.Same(t, p, DatabaseProto(db))
}

func TestExtractSimpleDatabaseMetadata(t *testing.T) {
	newCalled := false
	meta, err := ExtractSimpleDatabaseMetadata(
		"   \n\t ",
		func() *fakeCatalog {
			newCalled = true
			return &fakeCatalog{}
		},
		func(*fakeCatalog, string, bool) ([]ExecResult, error) { return nil, nil },
	)
	require.NoError(t, err)
	require.Equal(t, &metadatapb.DatabaseSchemaMetadata{}, meta)
	require.False(t, newCalled)

	var cat *fakeCatalog
	meta, err = ExtractSimpleDatabaseMetadata(
		"CREATE TABLE t (id INT);",
		func() *fakeCatalog {
			cat = &fakeCatalog{}
			return cat
		},
		func(c *fakeCatalog, sql string, _ bool) ([]ExecResult, error) {
			c.execCalls = append(c.execCalls, sql)
			return nil, nil
		},
	)
	require.NoError(t, err)
	require.Equal(t, "tmp", meta.GetName())
	require.Equal(t, "tmp", cat.exportedDB)
	require.Len(t, cat.execCalls, 2)
	require.Contains(t, cat.execCalls[0], "USE `tmp`;")
	require.Equal(t, "CREATE TABLE t (id INT);", cat.execCalls[1])

	_, err = ExtractSimpleDatabaseMetadata(
		"BAD SQL",
		func() *fakeCatalog { return &fakeCatalog{} },
		func(_ *fakeCatalog, _ string, continueOnError bool) ([]ExecResult, error) {
			if continueOnError {
				return []ExecResult{{Error: errors.New("syntax error")}}, nil
			}
			return nil, nil
		},
	)
	require.ErrorContains(t, err, "failed to execute schema DDL")
}

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

	const testDiffEngine = storepb.Engine(99902)
	schema.Register(schema.Engine{
		DiffMetadataMigration: MetadataMigrationFunc(MySQLFamilyEngine(
			func() *fakeCatalog { return &fakeCatalog{} },
			func(*fakeReport) []map[string]error { return nil },
			func(_, _ *fakeCatalog) fakeDiff { return fakeDiff{sql: "DROP TABLE `" + Database + "`.`old`"} },
			func(_, _ *fakeCatalog, d fakeDiff) fakePlan { return fakePlan{sql: d.sql} },
			nil,
		)),
	}, testDiffEngine)
	migration, err := schema.DiffMigration(testDiffEngine, nil, nil)
	require.NoError(t, err)
	require.Equal(t, "DROP TABLE `old`", migration)
}

type fakeSourceAST struct {
	base.AST
	node   string
	source string
}

func (a fakeSourceAST) ASTSource() string { return a.source }

func TestSDLAdviceHelpersAndInspectAST(t *testing.T) {
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

	inspect := InspectAST(
		func(a base.AST) (string, bool) {
			f, ok := a.(fakeSourceAST)
			return f.node, ok
		},
		func(n string) omnianalysis.WalkThroughStatement {
			return omnianalysis.WalkThroughStatement{Database: n}
		},
	)
	info, ok := inspect(fakeSourceAST{node: "db1", source: "  CREATE DATABASE db1; \n"})
	require.True(t, ok)
	require.Equal(t, "db1", info.Database)
	require.Equal(t, "CREATE DATABASE db1;", info.Source)

	_, ok = inspect(nil)
	require.False(t, ok)

	type rawExec struct {
		idx  int
		line int
		err  error
	}
	mapped, err := MapExecResults([]rawExec{{idx: 1, line: 4}}, nil, func(r rawExec) ExecResult {
		return ExecResult{Index: r.idx, Line: r.line, Error: r.err}
	})
	require.NoError(t, err)
	require.Equal(t, []ExecResult{{Index: 1, Line: 4}}, mapped)

	boom := errors.New("boom")
	_, err = MapExecResults([]rawExec{{idx: 1}}, boom, func(r rawExec) ExecResult {
		return ExecResult{Index: r.idx}
	})
	require.ErrorIs(t, err, boom)

	extract := ExtractError(func(e *fakeCatalogError) (int, string) { return e.code, e.msg })
	errCode, msg, ok := extract(&fakeCatalogError{code: 1050, msg: "exists"})
	require.True(t, ok)
	require.Equal(t, 1050, errCode)
	require.Equal(t, "exists", msg)
	_, _, ok = extract(boom)
	require.False(t, ok)
}

type fakeCatalogError struct {
	code int
	msg  string
}

func (e *fakeCatalogError) Error() string { return e.msg }
