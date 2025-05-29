package tsql

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func TestExtractChangedResources(t *testing.T) {
	statement := `CREATE TABLE t1 (c1 INT);
	DROP TABLE t1;
	ALTER TABLE t1 ADD colb INT;
	INSERT INTO t1 (c1) VALUES (1), (5);
	UPDATE t1 SET c1 = 5;
	`
	changedResources := model.NewChangedResources(nil /* dbSchema */)
	changedResources.AddTable(
		"DB",
		"dbo",
		&storepb.ChangedResourceTable{
			Name: "t1",
			Ranges: []*storepb.Range{
				{Start: 0, End: 25},
				{Start: 27, End: 41},
				{Start: 43, End: 71},
				{Start: 73, End: 109},
				{Start: 111, End: 132},
			},
		},
		true,
	)
	want := &base.ChangeSummary{
		ChangedResources: changedResources,
		SampleDMLS: []string{
			"UPDATE t1 SET c1 = 5;",
		},
		DMLCount:    1,
		InsertCount: 2,
	}

	asts, err := ParseTSQL(statement)
	require.NoError(t, err)
	got, err := extractChangedResources("DB", "dbo", nil /* dbSchema */, asts.Tree, statement)
	require.NoError(t, err)
	require.Equal(t, want, got)
}
