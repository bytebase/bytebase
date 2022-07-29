package store

import (
	"testing"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/stretchr/testify/require"
)

func TestGenerateDBExtensionActions(t *testing.T) {
	databaseID := 198
	tests := []struct {
		oldDBExtensionRawList []*dbExtensionRaw
		dbExtensionList       []db.Extension
		wantDeletes           []*api.DBExtensionDelete
		wantCreates           []*api.DBExtensionCreate
	}{
		{
			oldDBExtensionRawList: []*dbExtensionRaw{
				{ID: 123, Name: "hstore", Schema: "public", Version: "v1", Description: "desc1"},
			},
			dbExtensionList: []db.Extension{
				{Name: "hstore", Schema: "public", Version: "v2", Description: "desc1"},
				{Name: "hdd", Schema: "ddd", Version: "v3", Description: "desc3"},
			},
			wantDeletes: []*api.DBExtensionDelete{
				{ID: 123},
			},
			wantCreates: []*api.DBExtensionCreate{
				{Name: "hstore", Schema: "public", Version: "v2", Description: "desc1", CreatorID: api.SystemBotID, DatabaseID: databaseID},
				{Name: "hdd", Schema: "ddd", Version: "v3", Description: "desc3", CreatorID: api.SystemBotID, DatabaseID: databaseID},
			},
		},
		{
			oldDBExtensionRawList: []*dbExtensionRaw{
				{ID: 123, Name: "hstore", Schema: "public", Version: "v1", Description: "desc1"},
			},
			dbExtensionList: nil,
			wantDeletes: []*api.DBExtensionDelete{
				{ID: 123},
			},
			wantCreates: nil,
		},
		{
			oldDBExtensionRawList: nil,
			dbExtensionList: []db.Extension{
				{Name: "hstore", Schema: "public", Version: "v2", Description: "desc1"},
				{Name: "hdd", Schema: "ddd", Version: "v3", Description: "desc3"},
			},
			wantDeletes: nil,
			wantCreates: []*api.DBExtensionCreate{
				{Name: "hstore", Schema: "public", Version: "v2", Description: "desc1", CreatorID: api.SystemBotID, DatabaseID: databaseID},
				{Name: "hdd", Schema: "ddd", Version: "v3", Description: "desc3", CreatorID: api.SystemBotID, DatabaseID: databaseID},
			},
		},
		{
			oldDBExtensionRawList: []*dbExtensionRaw{
				{ID: 123, Name: "hstore", Schema: "public", Version: "v1", Description: "desc1"},
			},
			dbExtensionList: []db.Extension{
				{Name: "hstore", Schema: "public", Version: "v1", Description: "desc1"},
			},
			wantDeletes: nil,
			wantCreates: nil,
		},
	}

	for _, test := range tests {
		deletes, creates := generateDBExtensionActions(test.oldDBExtensionRawList, test.dbExtensionList, databaseID)
		require.Equal(t, test.wantDeletes, deletes)
		require.Equal(t, test.wantCreates, creates)
	}
}
