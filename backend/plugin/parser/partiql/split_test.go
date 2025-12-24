package partiql

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestSplitSQL(t *testing.T) {
	statement := `DELETE FROM Music WHERE Artist = 'Pink Floyd' AND SongTitle = 'Money';
	INSERT INTO Music VALUE {'AlbumTitle': 'The Dark Side of the Moon', 'Artist': 'Pink Floyd', 'Awards': 300, 'SongTitle': 'Money'};
	INSERT INTO Music VALUE {'AlbumTitle': 'The Dark Side of the Moon', 'Artist': 'Pink Floyd', 'Awards': 300, 'SongTitle': 'Money02'};
	INSERT INTO Music VALUE {'AlbumTitle': 'The Dark Side of the Moon', 'Artist': 'Pink Floyd', 'Awards': 300, 'SongTitle': 'Money03'};
	DELETE FROM Music WHERE Artist = 'Pink Floyd' AND SongTitle = 'Money02'`
	want := []base.Statement{
		{
			Text:  "DELETE FROM Music WHERE Artist = 'Pink Floyd' AND SongTitle = 'Money';",
			Range: &storepb.Range{Start: 0, End: 70},
			Start: &storepb.Position{Line: 1, Column: 1},
			End:   &storepb.Position{Line: 1, Column: 70},
			Empty: false,
		},
		{
			Text:  "\n\tINSERT INTO Music VALUE {'AlbumTitle': 'The Dark Side of the Moon', 'Artist': 'Pink Floyd', 'Awards': 300, 'SongTitle': 'Money'};",
			Range: &storepb.Range{Start: 70, End: 201},
			Start: &storepb.Position{Line: 2, Column: 2},
			End:   &storepb.Position{Line: 2, Column: 130},
			Empty: false,
		},
		{
			Text:  "\n\tINSERT INTO Music VALUE {'AlbumTitle': 'The Dark Side of the Moon', 'Artist': 'Pink Floyd', 'Awards': 300, 'SongTitle': 'Money02'};",
			Range: &storepb.Range{Start: 201, End: 334},
			Start: &storepb.Position{Line: 3, Column: 2},
			End:   &storepb.Position{Line: 3, Column: 132},
			Empty: false,
		},
		{
			Text:  "\n\tINSERT INTO Music VALUE {'AlbumTitle': 'The Dark Side of the Moon', 'Artist': 'Pink Floyd', 'Awards': 300, 'SongTitle': 'Money03'};",
			Range: &storepb.Range{Start: 334, End: 467},
			Start: &storepb.Position{Line: 4, Column: 2},
			End:   &storepb.Position{Line: 4, Column: 132},
			Empty: false,
		},
		{
			Text:  "\n\tDELETE FROM Music WHERE Artist = 'Pink Floyd' AND SongTitle = 'Money02'",
			Range: &storepb.Range{Start: 467, End: 540},
			Start: &storepb.Position{Line: 5, Column: 2},
			End:   &storepb.Position{Line: 5, Column: 73},
			Empty: false,
		},
	}

	list, err := SplitSQL(statement)
	require.NoError(t, err)
	require.Equal(t, want, list)
}
