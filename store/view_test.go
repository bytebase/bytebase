package store

import (
	"testing"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/stretchr/testify/require"
)

func TestGenerateViewActions(t *testing.T) {
	databaseID := 198
	tests := []struct {
		oldViewRawList []*viewRaw
		viewList       []db.View
		wantDeletes    []*api.ViewDelete
		wantCreates    []*api.ViewCreate
	}{
		{
			oldViewRawList: []*viewRaw{
				{ID: 123, Name: "view1", Definition: "def1", Comment: "comment1"},
				{ID: 124, Name: "view2", Definition: "def2", Comment: "comment2"},
			},
			viewList: []db.View{
				{Name: "view1", Definition: "def1-change", Comment: "comment1"},
				{Name: "view2", Definition: "def2", Comment: "comment2-change"},
				{Name: "view3", Definition: "def3", Comment: "comment3"},
			},
			wantDeletes: []*api.ViewDelete{
				{ID: 123},
				{ID: 124},
			},
			wantCreates: []*api.ViewCreate{
				{Name: "view1", Definition: "def1-change", Comment: "comment1", CreatorID: api.SystemBotID, DatabaseID: databaseID},
				{Name: "view2", Definition: "def2", Comment: "comment2-change", CreatorID: api.SystemBotID, DatabaseID: databaseID},
				{Name: "view3", Definition: "def3", Comment: "comment3", CreatorID: api.SystemBotID, DatabaseID: databaseID},
			},
		},
		{
			oldViewRawList: []*viewRaw{
				{ID: 123, Name: "view1", Definition: "def1", Comment: "comment1"},
			},
			viewList:    nil,
			wantCreates: nil,
			wantDeletes: []*api.ViewDelete{
				{ID: 123},
			},
		},
		{
			oldViewRawList: nil,
			viewList: []db.View{
				{Name: "view1", Definition: "def1", Comment: "comment1"},
				{Name: "view2", Definition: "def2", Comment: "comment2"},
			},
			wantDeletes: nil,
			wantCreates: []*api.ViewCreate{
				{Name: "view1", Definition: "def1", Comment: "comment1", CreatorID: api.SystemBotID, DatabaseID: databaseID},
				{Name: "view2", Definition: "def2", Comment: "comment2", CreatorID: api.SystemBotID, DatabaseID: databaseID},
			},
		},
		{
			oldViewRawList: []*viewRaw{
				{ID: 123, Name: "view1", Definition: "def1", Comment: "comment1"},
			},
			viewList: []db.View{
				{Name: "view1", Definition: "def1", Comment: "comment1"},
			},
			wantDeletes: nil,
			wantCreates: nil,
		},
	}

	for _, test := range tests {
		deletes, creates := generateViewActions(test.oldViewRawList, test.viewList, databaseID)
		require.Equal(t, test.wantDeletes, deletes)
		require.Equal(t, test.wantCreates, creates)
	}
}
