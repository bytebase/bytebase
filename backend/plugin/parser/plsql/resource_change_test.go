package plsql

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func TestExtractChangedResources(t *testing.T) {
	statement := `CREATE TABLE t1 (c1 INT);
	CREATE VIEW hello AS SELECT * FROM world;
	`
	changedResources := model.NewChangedResources(nil /* dbSchema */)
	changedResources.AddTable(
		"DB",
		"DB",
		&storepb.ChangedResourceTable{
			Name: "T1",
			Ranges: []*storepb.Range{
				{Start: 0, End: 25},
			},
		},
		false,
	)
	changedResources.AddView(
		"DB",
		"DB",
		&storepb.ChangedResourceView{
			Name: "HELLO",
			Ranges: []*storepb.Range{
				{Start: 27, End: 67},
			},
		},
	)
	want := &base.ChangeSummary{
		ChangedResources: changedResources,
	}

	asts, _, err := ParsePLSQL(statement)
	require.NoError(t, err)
	got, err := extractChangedResources("DB", "DB", nil /* dbSchema */, asts, statement)
	require.NoError(t, err)
	require.Equal(t, want, got)
}
