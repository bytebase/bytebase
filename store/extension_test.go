package store

import (
	"testing"

	"github.com/bytebase/bytebase/api"
	"github.com/stretchr/testify/require"
)

func TestGenerateDBExtensionActions(t *testing.T) {
	tests := []struct {
		oldDBExtensionRawList []*dbExtensionRaw
		dbExtensionCreateList []*api.DBExtensionCreate
		wantDeletes           []*api.DBExtensionDelete
		wantCreates           []*api.DBExtensionCreate
	}{
		{
			oldDBExtensionRawList: []*dbExtensionRaw{
				{ID: 123, Name: "hstore", Schema: "public", Version: "v1", Description: "desc1"},
			},
			dbExtensionCreateList: []*api.DBExtensionCreate{
				{Name: "hstore", Schema: "public", Version: "v2", Description: "desc1"},
				{Name: "hdd", Schema: "ddd", Version: "v3", Description: "desc3"},
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
			oldDBExtensionRawList: []*dbExtensionRaw{
				{ID: 123, Name: "hstore", Schema: "public", Version: "v1", Description: "desc1"},
			},
			dbExtensionCreateList: nil,
			wantDeletes: []*api.DBExtensionDelete{
				{ID: 123},
			},
		},
		{
			oldDBExtensionRawList: nil,
			dbExtensionCreateList: []*api.DBExtensionCreate{
				{Name: "hstore", Schema: "public", Version: "v2", Description: "desc1"},
				{Name: "hdd", Schema: "ddd", Version: "v3", Description: "desc3"},
			},
			wantDeletes: nil,
			wantCreates: []*api.DBExtensionCreate{
				{Name: "hstore", Schema: "public", Version: "v2", Description: "desc1"},
				{Name: "hdd", Schema: "ddd", Version: "v3", Description: "desc3"},
			},
		},
		{
			oldDBExtensionRawList: []*dbExtensionRaw{
				{ID: 123, Name: "hstore", Schema: "public", Version: "v1", Description: "desc1"},
			},
			dbExtensionCreateList: []*api.DBExtensionCreate{
				{Name: "hstore", Schema: "public", Version: "v1", Description: "desc1"},
			},
			wantDeletes: nil,
			wantCreates: nil,
		},
	}

	for _, test := range tests {
		deletes, creates := generateDBExtensionActions(test.oldDBExtensionRawList, test.dbExtensionCreateList)
		require.Equal(t, test.wantDeletes, deletes)
		require.Equal(t, test.wantCreates, creates)
	}
}
