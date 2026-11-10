package oceanbase

import (
	"context"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/component/sheet"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/oceanbase"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

func TestWalkThroughCrossDatabaseRename(t *testing.T) {
	state := model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{Name: "test"}, nil, nil, storepb.Engine_OCEANBASE, true)
	statement := "CREATE TABLE `test`.`t1` (a INT);\nRENAME TABLE `test`.`t1` TO `other`.`t1`;"
	sm := sheet.NewManager()
	stmts, advices := sm.GetStatementsForChecks(storepb.Engine_OCEANBASE, statement)
	require.Empty(t, advices)

	advice := WalkThroughWithContext(context.Background(), schema.WalkThroughContext{RawSQL: statement}, state, base.ExtractASTs(stmts))
	require.Nil(t, advice, "a cross-database RENAME the server accepts must not fail the simulation")
}
