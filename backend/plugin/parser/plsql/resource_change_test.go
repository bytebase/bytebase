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
	changedResources := model.NewChangedResources(nil /* dbSchema */)
	changedResources.AddTable(
		"DB",
		"",
		&storepb.ChangedResourceTable{
			Name: "T1",
			Ranges: []*storepb.Range{
				{Start: 0, End: 25},
				{Start: 70, End: 100},
			},
		},
		false,
	)
	changedResources.AddView(
		"DB",
		"",
		&storepb.ChangedResourceView{
			Name: "HELLO",
			Ranges: []*storepb.Range{
				{Start: 27, End: 67},
			},
		},
	)
	want := &base.ChangeSummary{
		ChangedResources: changedResources,
		InsertCount:      1,
	}

	asts, _, err := ParsePLSQL(statement)
	require.NoError(t, err)
	got, err := extractChangedResources("DB", "", nil /* dbSchema */, asts, statement)
	require.NoError(t, err)
	require.Equal(t, want, got)
}
