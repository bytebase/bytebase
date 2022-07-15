package store

import (
	"testing"

	"github.com/bytebase/bytebase/api"
	"github.com/stretchr/testify/require"
)

func TestGenerateViewActions(t *testing.T) {
	tests := []struct {
		oldViewMap  map[string]*viewRaw
		newViewMap  map[string]*api.ViewCreate
		wantDeletes []*api.ViewDelete
		wantCreates []*api.ViewCreate
	}{
		{
			oldViewMap: map[string]*viewRaw{
				"view1": {ID: 123, Definition: "def1", Comment: "comment1"},
				"view2": {ID: 124, Definition: "def2", Comment: "comment2"},
			},
			newViewMap: map[string]*api.ViewCreate{
				"view1": {Name: "view1", Definition: "def1-change", Comment: "comment1"},
				"view2": {Name: "view2", Definition: "def2", Comment: "comment2-change"},
				"view3": {Name: "view3", Definition: "def3", Comment: "comment3"},
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
			oldViewMap: map[string]*viewRaw{
				"view1": {ID: 123, Definition: "def1", Comment: "comment1"},
			},
			newViewMap: nil,
			wantDeletes: []*api.ViewDelete{
				{ID: 123},
			},
		},
		{
			oldViewMap: nil,
			newViewMap: map[string]*api.ViewCreate{
				"view1": {Name: "view1", Definition: "def1", Comment: "comment1"},
				"view2": {Name: "view2", Definition: "def2", Comment: "comment2"},
			},
			wantDeletes: nil,
			wantCreates: []*api.ViewCreate{
				{Name: "view1", Definition: "def1", Comment: "comment1"},
				{Name: "view2", Definition: "def2", Comment: "comment2"},
			},
		},
		{
			oldViewMap: map[string]*viewRaw{
				"view1": {ID: 123, Definition: "def1", Comment: "comment1"},
			},
			newViewMap: map[string]*api.ViewCreate{
				"view1": {Name: "view1", Definition: "def1", Comment: "comment1"},
			},
			wantDeletes: nil,
			wantCreates: nil,
		},
	}

	for _, test := range tests {
		deletes, creates := generateViewActions(test.oldViewMap, test.newViewMap)
		require.Equal(t, test.wantDeletes, deletes)
		require.Equal(t, test.wantCreates, creates)
	}
}
