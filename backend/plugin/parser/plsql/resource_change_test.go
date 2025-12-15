package plsql

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func TestExtractChangedResources(t *testing.T) {
	statement := `CREATE TABLE t1 (c1 INT);
	CREATE VIEW hello AS SELECT * FROM world;
	INSERT INTO T1 (c1) VALUES (1);
	`
	changedResources := model.NewChangedResources(nil /* dbMetadata */)
	changedResources.AddTable(
		"DB",
		"",
		&storepb.ChangedResourceTable{
			Name: "T1",
		},
		false,
	)
	want := &base.ChangeSummary{
		ChangedResources: changedResources,
		InsertCount:      1,
	}

	asts, err := base.Parse(storepb.Engine_ORACLE, statement)
	require.NoError(t, err)
	require.NotEmpty(t, asts)

	// Pass the full asts array to extractChangedResources
	got, err := extractChangedResources("DB", "", nil /* dbMetadata */, asts, statement)
	require.NoError(t, err)
	require.Equal(t, want, got)
}
