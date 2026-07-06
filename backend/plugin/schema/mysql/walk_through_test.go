package mysql

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	omniast "github.com/bytebase/omni/mysql/ast"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/sheet"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

type testData struct {
	Statement string
	// Use custom yaml tag to avoid generate field name `ignorecasesensitive`.
	IgnoreCaseSensitive bool `yaml:"ignore_case_sensitive"`
	Want                string
	Advice              *storepb.Advice
}

func TestWalkThrough(t *testing.T) {
	originDatabase := &storepb.DatabaseSchemaMetadata{
		Name: "test",
	}

	tests := []testData{}
	filepath := filepath.Join("testdata", "walk_through.yaml")
	yamlFile, err := os.Open(filepath)
	require.NoError(t, err)
	defer yamlFile.Close()

	byteValue, err := io.ReadAll(yamlFile)
	require.NoError(t, err)
	err = yaml.Unmarshal(byteValue, &tests)
	require.NoError(t, err)
	sm := sheet.NewManager()

	for _, test := range tests {
		// Make a deep copy to avoid mutation across tests
		protoData, ok := proto.Clone(originDatabase).(*storepb.DatabaseSchemaMetadata)
		require.True(t, ok)

		// Create DatabaseMetadata for walk-through
		state := model.NewDatabaseMetadata(protoData, nil, nil, storepb.Engine_MYSQL, !test.IgnoreCaseSensitive)

		stmts, _ := sm.GetStatementsForChecks(storepb.Engine_MYSQL, test.Statement)
		asts := base.ExtractASTs(stmts)
		advice := WalkThroughOmni(schema.WalkThroughContext{RawSQL: test.Statement}, state, asts)
		if advice != nil {
			// Compare the advice fields
			require.NotNil(t, test.Advice, "unexpected advice for statement %q: %+v", test.Statement, advice)
			require.Equal(t, test.Advice.Code, advice.Code, "statement %q advice %+v", test.Statement, advice)
			require.Equal(t, test.Advice.Content, advice.Content, "statement %q advice %+v", test.Statement, advice)
			continue
		}

		// Skip comparison if want is empty (error cases)
		if test.Want == "" {
			continue
		}

		want := &storepb.DatabaseSchemaMetadata{}
		err = common.ProtojsonUnmarshaler.Unmarshal([]byte(test.Want), want)
		require.NoError(t, err)
		result := state.GetProto()
		diff := cmp.Diff(want, result, protocmp.Transform(),
			protocmp.SortRepeatedFields(&storepb.DatabaseSchemaMetadata{}, "schemas"),
			protocmp.SortRepeatedFields(&storepb.SchemaMetadata{}, "tables", "views"),
			protocmp.SortRepeatedFields(&storepb.TableMetadata{}, "indexes", "columns"),
		)
		require.Empty(t, diff, "statement %q", test.Statement)
	}
}

func TestWalkThroughOmniCreateTableIfNotExistsCTASExistingTable(t *testing.T) {
	originDatabase := &storepb.DatabaseSchemaMetadata{
		Name: "test",
		Schemas: []*storepb.SchemaMetadata{
			{
				Tables: []*storepb.TableMetadata{
					{
						Name: "t1",
						Columns: []*storepb.ColumnMetadata{
							{
								Name:     "a",
								Position: 1,
								Nullable: true,
								Type:     "int",
							},
						},
					},
				},
			},
		},
	}

	state := model.NewDatabaseMetadata(originDatabase, nil, nil, storepb.Engine_MYSQL, true)
	require.NotNil(t, state.GetSchemaMetadata("").GetTable("t1"))
	statement := "CREATE TABLE IF NOT EXISTS t1 AS SELECT 1;"
	sm := sheet.NewManager()
	stmts, _ := sm.GetStatementsForChecks(storepb.Engine_MYSQL, statement)
	asts := base.ExtractASTs(stmts)
	omniAST, ok := asts[0].(*mysqlparser.OmniAST)
	require.True(t, ok)
	createTable, ok := omniAST.Node.(*omniast.CreateTableStmt)
	require.True(t, ok)
	require.True(t, createTable.IfNotExists)

	advice := WalkThroughOmni(schema.WalkThroughContext{RawSQL: statement}, state, asts)
	require.Nil(t, advice)
	require.NotNil(t, state.GetSchemaMetadata("").GetTable("t1"))
}

// TestWalkThroughOmniSRIDInvisible verifies the omni catalog -> storepb.ColumnMetadata
// conversion (tableToProto) carries the spatial SRID (presence + value, including the
// valid explicit SRID 0) and the INVISIBLE flag. Without this the WalkThroughOmni
// simulation path — used to compute the schema a changeset would produce — would silently
// drop these attributes, mirroring the v1-converter gap.
func TestWalkThroughOmniSRIDInvisible(t *testing.T) {
	state := model.NewDatabaseMetadata(&storepb.DatabaseSchemaMetadata{Name: "test"}, nil, nil, storepb.Engine_MYSQL, true)
	statement := "CREATE TABLE t (" +
		"id INT PRIMARY KEY, " +
		"g4326 GEOMETRY NOT NULL SRID 4326, " +
		"g0 GEOMETRY NOT NULL SRID 0, " +
		"gnone GEOMETRY NOT NULL, " +
		"secret INT INVISIBLE NOT NULL DEFAULT 0);"
	sm := sheet.NewManager()
	stmts, _ := sm.GetStatementsForChecks(storepb.Engine_MYSQL, statement)
	asts := base.ExtractASTs(stmts)

	advice := WalkThroughOmni(schema.WalkThroughContext{RawSQL: statement}, state, asts)
	require.Nil(t, advice)

	cols := map[string]*storepb.ColumnMetadata{}
	for _, c := range state.GetProto().GetSchemas()[0].GetTables()[0].GetColumns() {
		cols[c.GetName()] = c
	}

	// Explicit SRID must round-trip as presence (nil vs non-nil) plus value.
	require.NotNil(t, cols["g4326"].Srid)
	require.Equal(t, uint32(4326), *cols["g4326"].Srid)
	// Explicit SRID 0 is present-and-distinct from "no SRID"; presence must survive.
	require.NotNil(t, cols["g0"].Srid, "explicit SRID 0 must be captured as present")
	require.Equal(t, uint32(0), *cols["g0"].Srid)
	// A no-SRID spatial column must stay unset (no phantom SRID 0).
	require.Nil(t, cols["gnone"].Srid)
	// INVISIBLE captured; visible columns stay visible.
	require.True(t, cols["secret"].IsInvisible)
	require.False(t, cols["g4326"].IsInvisible)
}

// TestWalkThroughOmniSRIDInvisibleSeeded verifies the reverse leg of the WalkThroughOmni
// round-trip: when the walk-through starts from already-synced metadata that carries SRID
// / INVISIBLE columns, the proto->catalog seeding (wtBuildColumnDef) must install those
// attributes into the omni catalog so that running an UNRELATED DDL does not strip them
// from the resulting proto. Without seeding, the pre-existing columns load as
// plain/no-SRID and — because columnsEqual now compares these fields — surface as phantom
// changes.
func TestWalkThroughOmniSRIDInvisibleSeeded(t *testing.T) {
	origin := &storepb.DatabaseSchemaMetadata{
		Name: "test",
		Schemas: []*storepb.SchemaMetadata{{
			Name: "",
			Tables: []*storepb.TableMetadata{
				{
					Name: "geo",
					Columns: []*storepb.ColumnMetadata{
						{Name: "id", Position: 1, Type: "int", Nullable: false, Default: "AUTO_INCREMENT"},
						{Name: "pt", Position: 2, Type: "point", Nullable: false, Srid: func() *uint32 { v := uint32(4326); return &v }()},
						{Name: "secret", Position: 3, Type: "int", Nullable: false, Default: "0", IsInvisible: true},
					},
					Indexes: []*storepb.IndexMetadata{
						{Name: "PRIMARY", Expressions: []string{"id"}, Primary: true, Unique: true},
					},
				},
			},
		}},
	}
	state := model.NewDatabaseMetadata(origin, nil, nil, storepb.Engine_MYSQL, true)

	// An unrelated DDL against a different table: it must not perturb geo's SRID/INVISIBLE.
	statement := "CREATE TABLE unrelated (x INT PRIMARY KEY);"
	sm := sheet.NewManager()
	stmts, _ := sm.GetStatementsForChecks(storepb.Engine_MYSQL, statement)
	asts := base.ExtractASTs(stmts)
	advice := WalkThroughOmni(schema.WalkThroughContext{RawSQL: statement}, state, asts)
	require.Nil(t, advice)

	geo := state.GetSchemaMetadata("").GetTable("geo")
	require.NotNil(t, geo)
	cols := map[string]*storepb.ColumnMetadata{}
	for _, c := range geo.GetProto().GetColumns() {
		cols[c.GetName()] = c
	}
	// The pre-synced SRID + INVISIBLE attributes must survive the seed -> catalog ->
	// proto round-trip unchanged.
	require.NotNil(t, cols["pt"].Srid, "seeded SRID must survive (no phantom strip)")
	require.Equal(t, uint32(4326), *cols["pt"].Srid)
	require.True(t, cols["secret"].IsInvisible, "seeded INVISIBLE must survive")
}
