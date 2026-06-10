package starrocks

import (
	"testing"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// #20562 review (P2): the system-database set must be StarRocks-specific, not the
// copied Doris list. StarRocks has a read-only `sys` metadatabase (SHOW DATABASES
// confirms: information_schema, _statistics_, sys) and no `mysql`/`__internal_schema`.
func TestIsSystemResource_StarRocks(t *testing.T) {
	cases := []struct {
		database string
		want     bool
	}{
		{"sys", true},                // StarRocks system metadatabase
		{"information_schema", true}, // shared
		{"_statistics_", true},       // statistics
		{"mysql", false},             // Doris-only, not a StarRocks system db
		{"__internal_schema", false}, // Doris-only
		{"my_user_db", false},        // user database
	}
	for _, c := range cases {
		if got := isSystemResource(base.ColumnResource{Database: c.database}, true); got != c.want {
			t.Errorf("isSystemResource(%q) = %v, want %v", c.database, got, c.want)
		}
	}
}
