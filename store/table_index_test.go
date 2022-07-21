package store

import (
	"testing"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/stretchr/testify/require"
)

func TestGenerateIndexActions(t *testing.T) {
	databaseID := 198
	tableID := 199
	tests := []struct {
		oldIndexList []*api.Index
		indexList    []db.Index
		wantDeletes  []*api.IndexDelete
		wantCreates  []*api.IndexCreate
	}{
		{
			oldIndexList: []*api.Index{
				{ID: 123, Name: "index1", Expression: "def1", Comment: "comment1"},
				{ID: 124, Name: "index2", Expression: "def2", Position: 1, Comment: "comment2"},
				{ID: 125, Name: "index2", Expression: "def2", Position: 2, Comment: "comment2"},
			},
			indexList: []db.Index{
				{Name: "index1", Expression: "def1-change", Comment: "comment1"},
				{Name: "index2", Expression: "def2", Position: 1, Comment: "comment2"},
				{Name: "index2", Expression: "def2", Position: 2, Comment: "comment2-change"},
				{Name: "index2", Expression: "def2", Position: 3, Comment: "comment2-new"},
				{Name: "index3", Expression: "def3", Comment: "comment3"},
			},
			wantDeletes: []*api.IndexDelete{
				{ID: 123},
				{ID: 125},
			},
			wantCreates: []*api.IndexCreate{
				{Name: "index1", Expression: "def1-change", Comment: "comment1", CreatorID: api.SystemBotID, DatabaseID: databaseID, TableID: tableID},
				{Name: "index2", Expression: "def2", Position: 2, Comment: "comment2-change", CreatorID: api.SystemBotID, DatabaseID: databaseID, TableID: tableID},
				{Name: "index2", Expression: "def2", Position: 3, Comment: "comment2-new", CreatorID: api.SystemBotID, DatabaseID: databaseID, TableID: tableID},
				{Name: "index3", Expression: "def3", Comment: "comment3", CreatorID: api.SystemBotID, DatabaseID: databaseID, TableID: tableID},
			},
		},
		{
			oldIndexList: []*api.Index{
				{ID: 123, Name: "index1", Expression: "def1", Comment: "comment1"},
			},
			indexList:   nil,
			wantCreates: nil,
			wantDeletes: []*api.IndexDelete{
				{ID: 123},
			},
		},
		{
			oldIndexList: nil,
			indexList: []db.Index{
				{Name: "index1", Expression: "def1", Comment: "comment1"},
				{Name: "index2", Expression: "def2", Comment: "comment2"},
			},
			wantDeletes: nil,
			wantCreates: []*api.IndexCreate{
				{Name: "index1", Expression: "def1", Comment: "comment1", CreatorID: api.SystemBotID, DatabaseID: databaseID, TableID: tableID},
				{Name: "index2", Expression: "def2", Comment: "comment2", CreatorID: api.SystemBotID, DatabaseID: databaseID, TableID: tableID},
			},
		},
		{
			oldIndexList: []*api.Index{
				{ID: 123, Name: "index1", Expression: "def1", Comment: "comment1"},
			},
			indexList: []db.Index{
				{Name: "index1", Expression: "def1", Comment: "comment1"},
			},
			wantDeletes: nil,
			wantCreates: nil,
		},
	}

	for _, test := range tests {
		deletes, creates := generateIndexActions(test.oldIndexList, test.indexList, databaseID, tableID)
		require.Equal(t, test.wantDeletes, deletes)
		require.Equal(t, test.wantCreates, creates)
	}
}
