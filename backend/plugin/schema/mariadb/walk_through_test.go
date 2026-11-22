package mariadb

import (
	"context"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/component/sheet"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

// TestWalkThroughMariaDBSkipsMariaDBOnlyStatements pins what the MariaDB
// registration has to keep doing: omni's MariaDB catalog has no metadata loader,
// so MariaDB walks through omni's MySQL catalog, and a statement that catalog
// cannot parse leaves the whole script unsimulated rather than checking the rest
// against a schema missing its effect.
func TestWalkThroughMariaDBSkipsMariaDBOnlyStatements(t *testing.T) {
	state := model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{Name: "test"}, nil, nil, storepb.Engine_MARIADB, true)
	statement := "CREATE OR REPLACE TABLE t (a INT);\nALTER TABLE t ADD COLUMN b INT;"
	sm := sheet.NewManager()
	stmts, advices := sm.GetStatementsForChecks(storepb.Engine_MARIADB, statement)
	require.Empty(t, advices)

	advice := walkThrough(context.Background(), schema.WalkThroughContext{RawSQL: statement}, state, base.ExtractASTs(stmts))
	require.Nil(t, advice)
	require.Empty(t, state.GetProto().GetSchemas())
}

// TestWalkThroughMariaDBSimulatesSharedStatements is the other half: a script
// omni's MySQL catalog can read is simulated for MariaDB as it always was,
// through the MariaDB registration rather than the MySQL one.
func TestWalkThroughMariaDBSimulatesSharedStatements(t *testing.T) {
	state := model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{Name: "test"}, nil, nil, storepb.Engine_MARIADB, true)
	statement := "CREATE TABLE t (a INT);\nALTER TABLE t ADD COLUMN b INT;"
	sm := sheet.NewManager()
	stmts, advices := sm.GetStatementsForChecks(storepb.Engine_MARIADB, statement)
	require.Empty(t, advices)

	advice := walkThrough(context.Background(), schema.WalkThroughContext{RawSQL: statement}, state, base.ExtractASTs(stmts))
	require.Nil(t, advice)
	require.NotNil(t, state.GetSchemaMetadata("").GetTable("t").GetColumn("b"))
}
