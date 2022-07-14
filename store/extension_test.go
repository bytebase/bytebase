package store

import (
	"testing"

	"github.com/bytebase/bytebase/api"
	"github.com/stretchr/testify/require"
)

func TestGenerateDBExtensionActions(t *testing.T) {
	tests := []struct {
		oldDBExtensionMap map[extensionKey]*dbExtensionRaw
		newDBExtensionMap map[extensionKey]*api.DBExtensionCreate
		wantDeletes       []*api.DBExtensionDelete
		wantCreates       []*api.DBExtensionCreate
	}{
		{
			oldDBExtensionMap: map[extensionKey]*dbExtensionRaw{
				{"hstore", "public"}: {ID: 123, Version: "v1", Description: "desc1"},
			},
			newDBExtensionMap: map[extensionKey]*api.DBExtensionCreate{
				{"hstore", "public"}: {Name: "hstore", Schema: "public", Version: "v2", Description: "desc1"},
				{"hdd", "ddd"}:       {Name: "hdd", Schema: "ddd", Version: "v3", Description: "desc3"},
			},
			wantDeletes: []*api.DBExtensionDelete{
				{ID: 123},
			},
			wantCreates: []*api.DBExtensionCreate{
				{Name: "hstore", Schema: "public", Version: "v2", Description: "desc1"},
				{Name: "hdd", Schema: "ddd", Version: "v3", Description: "desc3"},
			},
		},
		{
			oldDBExtensionMap: map[extensionKey]*dbExtensionRaw{
				{"hstore", "public"}: {ID: 123, Version: "v1", Description: "desc1"},
			},
			newDBExtensionMap: nil,
			wantDeletes: []*api.DBExtensionDelete{
				{ID: 123},
			},
		},
		{
			oldDBExtensionMap: nil,
			newDBExtensionMap: map[extensionKey]*api.DBExtensionCreate{
				{"hstore", "public"}: {Name: "hstore", Schema: "public", Version: "v2", Description: "desc1"},
				{"hdd", "ddd"}:       {Name: "hdd", Schema: "ddd", Version: "v3", Description: "desc3"},
			},
			wantDeletes: nil,
			wantCreates: []*api.DBExtensionCreate{
				{Name: "hstore", Schema: "public", Version: "v2", Description: "desc1"},
				{Name: "hdd", Schema: "ddd", Version: "v3", Description: "desc3"},
			},
		},
		{
			oldDBExtensionMap: map[extensionKey]*dbExtensionRaw{
				{"hstore", "public"}: {ID: 123, Version: "v1", Description: "desc1"},
			},
			newDBExtensionMap: map[extensionKey]*api.DBExtensionCreate{
				{"hstore", "public"}: {Name: "hstore", Schema: "public", Version: "v1", Description: "desc1"},
			},
			wantDeletes: nil,
			wantCreates: nil,
		},
	}

	for _, test := range tests {
		deletes, creates := generateDBExtensionActions(test.oldDBExtensionMap, test.newDBExtensionMap)
		require.Equal(t, test.wantDeletes, deletes)
		require.Equal(t, test.wantCreates, creates)
	}
}
