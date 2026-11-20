package starrocks

import (
	"testing"

	"github.com/bytebase/omni/starrocks/analysis"
)

func TestIsSystemDatabase_StarRocks(t *testing.T) {
	cases := []struct {
		database string
		want     bool
	}{
		{"sys", true},                // StarRocks system metadatabase
		{"information_schema", true}, // shared
		{"_statistics_", false},      // ordinary DB in StarRocks (confirmed via SHOW DATABASES)
		{"mysql", false},             // Doris-only, not a StarRocks system db
		{"__internal_schema", false}, // Doris-only
		{"my_user_db", false},        // user database
	}
	for _, c := range cases {
		if got := analysis.IsSystemDatabase(c.database, true); got != c.want {
			t.Errorf("IsSystemDatabase(%q) = %v, want %v", c.database, got, c.want)
		}
	}
}
