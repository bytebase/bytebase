package store

import (
	"testing"

	"github.com/bytebase/bytebase/api"
	"github.com/stretchr/testify/require"
)

func TestGenerateViewActions(t *testing.T) {
	tests := []struct {
		oldViewRawList []*viewRaw
		viewCreateList []*api.ViewCreate
		wantDeletes    []*api.ViewDelete
		wantCreates    []*api.ViewCreate
	}{
		{
			oldViewRawList: []*viewRaw{
				{ID: 123, Name: "view1", Definition: "def1", Comment: "comment1"},
				{ID: 124, Name: "view2", Definition: "def2", Comment: "comment2"},
			},
			viewCreateList: []*api.ViewCreate{
				{Name: "view1", Definition: "def1-change", Comment: "comment1"},
				{Name: "view2", Definition: "def2", Comment: "comment2-change"},
				{Name: "view3", Definition: "def3", Comment: "comment3"},
			},
			wantDeletes: []*api.ViewDelete{
				{ID: 123},
				{ID: 124},
			},
			wantCreates: []*api.ViewCreate{
				{Name: "view1", Definition: "def1-change", Comment: "comment1"},
				{Name: "view2", Definition: "def2", Comment: "comment2-change"},
				{Name: "view3", Definition: "def3", Comment: "comment3"},
			},
		},
		{
			oldViewRawList: []*viewRaw{
				{ID: 123, Name: "view1", Definition: "def1", Comment: "comment1"},
			},
			viewCreateList: nil,
			wantCreates:    nil,
			wantDeletes: []*api.ViewDelete{
				{ID: 123},
			},
		},
		{
			oldViewRawList: nil,
			viewCreateList: []*api.ViewCreate{
				{Name: "view1", Definition: "def1", Comment: "comment1"},
				{Name: "view2", Definition: "def2", Comment: "comment2"},
			},
			wantDeletes: nil,
			wantCreates: []*api.ViewCreate{
				{Name: "view1", Definition: "def1", Comment: "comment1"},
				{Name: "view2", Definition: "def2", Comment: "comment2"},
			},
		},
		{
			oldViewRawList: []*viewRaw{
				{ID: 123, Name: "view1", Definition: "def1", Comment: "comment1"},
			},
			viewCreateList: []*api.ViewCreate{
				{Name: "view1", Definition: "def1", Comment: "comment1"},
			},
			wantDeletes: nil,
			wantCreates: nil,
		},
	}

	for _, test := range tests {
		deletes, creates := generateViewActions(test.oldViewRawList, test.viewCreateList)
		require.Equal(t, test.wantDeletes, deletes)
		require.Equal(t, test.wantCreates, creates)
	}
}
