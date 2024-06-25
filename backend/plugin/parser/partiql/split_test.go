package partiql

import (
	"testing"

	"github.com/stretchr/testify/require"

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
			Text:                 "DELETE FROM Music WHERE Artist = 'Pink Floyd' AND SongTitle = 'Money';",
			BaseLine:             0,
			FirstStatementLine:   0,
			FirstStatementColumn: 0,
			LastLine:             0,
			LastColumn:           69,
			Empty:                false,
		},
		{
			Text:                 "\n\tINSERT INTO Music VALUE {'AlbumTitle': 'The Dark Side of the Moon', 'Artist': 'Pink Floyd', 'Awards': 300, 'SongTitle': 'Money'};",
			BaseLine:             0,
			FirstStatementLine:   1,
			FirstStatementColumn: 1,
			LastLine:             1,
			LastColumn:           129,
			Empty:                false,
		},
		{
			Text:                 "\n\tINSERT INTO Music VALUE {'AlbumTitle': 'The Dark Side of the Moon', 'Artist': 'Pink Floyd', 'Awards': 300, 'SongTitle': 'Money02'};",
			BaseLine:             1,
			FirstStatementLine:   2,
			FirstStatementColumn: 1,
			LastLine:             2,
			LastColumn:           131,
			Empty:                false,
		},
		{
			Text:                 "\n\tINSERT INTO Music VALUE {'AlbumTitle': 'The Dark Side of the Moon', 'Artist': 'Pink Floyd', 'Awards': 300, 'SongTitle': 'Money03'};",
			BaseLine:             2,
			FirstStatementLine:   3,
			FirstStatementColumn: 1,
			LastLine:             3,
			LastColumn:           131,
			Empty:                false,
		},
		{
			Text:                 "\n\tDELETE FROM Music WHERE Artist = 'Pink Floyd' AND SongTitle = 'Money02'",
			BaseLine:             3,
			FirstStatementLine:   4,
			FirstStatementColumn: 1,
			LastLine:             4,
			LastColumn:           63,
			Empty:                false,
		},
	}

	list, err := SplitSQL(statement)
	require.NoError(t, err)
	require.Equal(t, want, list)
}
