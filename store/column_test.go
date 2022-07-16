package store

import (
	"testing"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/stretchr/testify/require"
)

func TestGenerateColumnActions(t *testing.T) {
	databaseID := 198
	tableID := 199
	tests := []struct {
		oldColumnList []*api.Column
		columnList    []db.Column
		wantDeletes   []*api.ColumnDelete
		wantCreates   []*api.ColumnCreate
	}{
		{
			oldColumnList: []*api.Column{
				{ID: 123, Name: "column1", CharacterSet: "def1", Comment: "comment1"},
				{ID: 124, Name: "column2", CharacterSet: "def2", Comment: "comment2"},
			},
			columnList: []db.Column{
				{Name: "column1", CharacterSet: "def1-change", Comment: "comment1"},
				{Name: "column2", CharacterSet: "def2", Comment: "comment2-change"},
				{Name: "column3", CharacterSet: "def3", Comment: "comment3"},
			},
			wantDeletes: []*api.ColumnDelete{
				{ID: 123},
				{ID: 124},
			},
			wantCreates: []*api.ColumnCreate{
				{Name: "column1", CharacterSet: "def1-change", Comment: "comment1", CreatorID: api.SystemBotID, DatabaseID: databaseID, TableID: tableID},
				{Name: "column2", CharacterSet: "def2", Comment: "comment2-change", CreatorID: api.SystemBotID, DatabaseID: databaseID, TableID: tableID},
				{Name: "column3", CharacterSet: "def3", Comment: "comment3", CreatorID: api.SystemBotID, DatabaseID: databaseID, TableID: tableID},
			},
		},
		{
			oldColumnList: []*api.Column{
				{ID: 123, Name: "column1", CharacterSet: "def1", Comment: "comment1"},
			},
			columnList:  nil,
			wantCreates: nil,
			wantDeletes: []*api.ColumnDelete{
				{ID: 123},
			},
		},
		{
			oldColumnList: nil,
			columnList: []db.Column{
				{Name: "column1", CharacterSet: "def1", Comment: "comment1"},
				{Name: "column2", CharacterSet: "def2", Comment: "comment2"},
			},
			wantDeletes: nil,
			wantCreates: []*api.ColumnCreate{
				{Name: "column1", CharacterSet: "def1", Comment: "comment1", CreatorID: api.SystemBotID, DatabaseID: databaseID, TableID: tableID},
				{Name: "column2", CharacterSet: "def2", Comment: "comment2", CreatorID: api.SystemBotID, DatabaseID: databaseID, TableID: tableID},
			},
		},
		{
			oldColumnList: []*api.Column{
				{ID: 123, Name: "column1", CharacterSet: "def1", Comment: "comment1"},
			},
			columnList: []db.Column{
				{Name: "column1", CharacterSet: "def1", Comment: "comment1"},
			},
			wantDeletes: nil,
			wantCreates: nil,
		},
	}

	for _, test := range tests {
		deletes, creates := generateColumnActions(test.oldColumnList, test.columnList, databaseID, tableID)
		require.Equal(t, test.wantDeletes, deletes)
		require.Equal(t, test.wantCreates, creates)
	}
}
