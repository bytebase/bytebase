package starrocks

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestIsSystemDatabase_StarRocks(t *testing.T) {
	cases := []struct {
		database string
		want     bool
	}{
		{"sys", true},                // StarRocks system metadatabase
		{"information_schema", true}, // shared
		{"_statistics_", true},       // StarRocks statistics database
		{"mysql", false},             // Doris-only, not a StarRocks system db
		{"__internal_schema", false}, // Doris-only
		{"my_user_db", false},        // user database
	}
	for _, c := range cases {
		if got := isSystemDatabase(c.database, true); got != c.want {
			t.Errorf("isSystemDatabase(%q) = %v, want %v", c.database, got, c.want)
		}
	}
}

func TestStatisticsSystemDatabaseQuerySpan(t *testing.T) {
	span, err := GetQuerySpan(
		context.Background(),
		base.GetQuerySpanContext{},
		base.Statement{Text: "SELECT * FROM _statistics_.column_statistics c JOIN information_schema.tables t ON c.table_id = t.table_name"},
		"testdb",
		"",
		true,
	)
	require.NoError(t, err)
	require.Equal(t, base.SelectInfoSchema, span.Type)
	require.True(t, span.SourceColumns[base.ColumnResource{Database: "_statistics_", Table: "column_statistics"}])
	require.True(t, span.SourceColumns[base.ColumnResource{Database: "information_schema", Table: "tables"}])

	_, err = GetQuerySpan(
		context.Background(),
		base.GetQuerySpanContext{},
		base.Statement{Text: "SELECT * FROM _statistics_.column_statistics JOIN users"},
		"testdb",
		"",
		true,
	)
	require.ErrorIs(t, err, base.MixUserSystemTablesError)

	_, err = GetQuerySpan(
		context.Background(),
		base.GetQuerySpanContext{},
		base.Statement{Text: "SELECT * FROM information_schema.tables JOIN users"},
		"testdb",
		"",
		true,
	)
	require.ErrorIs(t, err, base.MixUserSystemTablesError)
}
