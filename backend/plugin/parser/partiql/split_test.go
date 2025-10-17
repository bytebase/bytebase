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
	want := []base.SingleSQL{
		{
			Text:     "DELETE FROM Music WHERE Artist = 'Pink Floyd' AND SongTitle = 'Money';",
			BaseLine: 0,
			Start:    &storepb.Position{Line: 1, Column: 1},
			End:      &storepb.Position{Line: 1, Column: 70},
			Empty:    false,
		},
		{
			Text:     "\n\tINSERT INTO Music VALUE {'AlbumTitle': 'The Dark Side of the Moon', 'Artist': 'Pink Floyd', 'Awards': 300, 'SongTitle': 'Money'};",
			BaseLine: 0,
			Start:    &storepb.Position{Line: 2, Column: 2},
			End:      &storepb.Position{Line: 2, Column: 130},
			Empty:    false,
		},
		{
			Text:     "\n\tINSERT INTO Music VALUE {'AlbumTitle': 'The Dark Side of the Moon', 'Artist': 'Pink Floyd', 'Awards': 300, 'SongTitle': 'Money02'};",
			BaseLine: 1,
			Start:    &storepb.Position{Line: 3, Column: 2},
			End:      &storepb.Position{Line: 3, Column: 132},
			Empty:    false,
		},
		{
			Text:     "\n\tINSERT INTO Music VALUE {'AlbumTitle': 'The Dark Side of the Moon', 'Artist': 'Pink Floyd', 'Awards': 300, 'SongTitle': 'Money03'};",
			BaseLine: 2,
			Start:    &storepb.Position{Line: 4, Column: 2},
			End:      &storepb.Position{Line: 4, Column: 132},
			Empty:    false,
		},
		{
			Text:     "\n\tDELETE FROM Music WHERE Artist = 'Pink Floyd' AND SongTitle = 'Money02'",
			BaseLine: 3,
			Start:    &storepb.Position{Line: 5, Column: 2},
			End:      &storepb.Position{Line: 5, Column: 73},
			Empty:    false,
		},
	}

	list, err := SplitSQL(statement)
	require.NoError(t, err)
	require.Equal(t, want, list)
}
