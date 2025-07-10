package mysql

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func TestExtractChangedResources(t *testing.T) {
	statement := `CREATE TABLE t1 (c1 INT);
	DROP TABLE t1;
	ALTER TABLE t1 ADD COLUMN c1 INT;
	RENAME TABLE t1 TO t2;
	INSERT INTO t1 (c1) VALUES (1), (5);
	UPDATE t1 SET c1 = 5;
	CREATE PROCEDURE getUser(id INT) SELECT * FROM hello WHERE uid = id;
	`
	changedResources := model.NewChangedResources(nil /* dbSchema */)
	changedResources.AddTable(
		"db",
		"",
		&storepb.ChangedResourceTable{
			Name: "t1",
			Ranges: []*storepb.Range{
				{Start: 0, End: 25},
				{Start: 27, End: 41},
				{Start: 43, End: 76},
				{Start: 78, End: 100},
			},
		},
		true,
	)
	changedResources.AddTable(
		"db",
		"",
		&storepb.ChangedResourceTable{
			Name:   "t2",
			Ranges: []*storepb.Range{{Start: 78, End: 100}},
		},
		false,
	)
	changedResources.AddProcedure(
		"db",
		"",
		&storepb.ChangedResourceProcedure{
			Name:   "getUser",
			Ranges: []*storepb.Range{{Start: 163, End: 231}},
		},
	)
	want := &base.ChangeSummary{
		ChangedResources: changedResources,
		SampleDMLS: []string{
			"UPDATE t1 SET c1 = 5;",
		},
		DMLCount:    1,
		InsertCount: 2,
	}

	asts, _ := ParseMySQL(statement)
	got, err := extractChangedResources("db", "", nil /* dbSchema */, asts, statement)
	require.NoError(t, err)
	require.Equal(t, want, got)
}
