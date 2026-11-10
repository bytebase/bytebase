package doris

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"

	// Register the Doris editor validator.
	_ "github.com/bytebase/bytebase/backend/plugin/parser/doris"
)

func TestParseVersion(t *testing.T) {
	tests := []struct {
		version  string
		want     string
		wantRest string
	}{
		{
			version:  "8.0.27",
			want:     "8.0.27",
			wantRest: "",
		},
		{
			version:  "5.7.22-log",
			want:     "5.7.22",
			wantRest: "-log",
		},
		{
			version:  "5.6.29_ddm_3.0.1.7",
			want:     "5.6.29",
			wantRest: "_ddm_3.0.1.7",
		},
		{
			version:  "10.4.7-MariaDB",
			want:     "10.4.7",
			wantRest: "-MariaDB",
		},
	}

	a := require.New(t)
	for _, tc := range tests {
		version, rest, err := parseVersion(tc.version)
		a.NoError(err)
		a.Equal(tc.want, version)
		a.Equal(tc.wantRest, rest)
	}
}

func TestContainsDelimiterDirective(t *testing.T) {
	tests := []struct {
		name string
		sql  string
		want bool
	}{
		{"basic directive", "DELIMITER ;;\nCREATE PROCEDURE p() BEGIN SELECT 1; END;;\nDELIMITER ;", true},
		{"indented directive", "  DELIMITER //\nSELECT 1//\nDELIMITER ;", true},
		{"tab-indented directive", "\tDELIMITER $$\n", true},
		{"string literal not a directive", "INSERT INTO t VALUES ('DELIMITER ');\nSELECT 1;", false},
		{"comment not a directive", "-- DELIMITER //\nSELECT 1;", false},
		{"column name not a directive", "SELECT delimiter FROM t;", false},
		{"no delimiter at all", "SELECT 1;\nSELECT 2;", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containsDelimiterDirective(tt.sql)
			require.Equal(t, tt.want, got)
		})
	}
}

// TestValidateSQLForEditorUsesDorisGrammar pins this driver to the Doris parser: a
// StarRocks-only spelling must be refused rather than quietly accepted, which is what
// routing the connection through a sibling engine's grammar would do.
func TestValidateSQLForEditorUsesDorisGrammar(t *testing.T) {
	// StarRocks generated column AS (expr); Doris wants GENERATED ALWAYS AS.
	const starRocksOnlyDDL = "CREATE TABLE t (a INT, b INT AS (a + 1)) DUPLICATE KEY(a) DISTRIBUTED BY HASH(a)"
	_, _, err := base.ValidateSQLForEditor(storepb.Engine_DORIS, starRocksOnlyDDL)
	require.Error(t, err)

	// A Doris CREATE TABLE parses, and is DDL rather than a read-only query.
	_, allQuery, err := base.ValidateSQLForEditor(storepb.Engine_DORIS, "CREATE TABLE t (a INT) DUPLICATE KEY(a) DISTRIBUTED BY HASH(a)")
	require.NoError(t, err)
	require.False(t, allQuery)
}
