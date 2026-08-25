package bigquery

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// base.IsAllDML resolves through the registry this package populates, and it is
// what computeNeedDump calls to decide whether a BigQuery migration skips the
// schema dump. The googlesql tests cover the type mapping one layer down; these
// pin the predicate the executor actually calls.
func TestIsAllDML(t *testing.T) {
	tests := []struct {
		name      string
		statement string
		want      bool
	}{
		{"insert", "INSERT INTO users (id) VALUES (1);", true},
		{"update", "UPDATE users SET name = 'b' WHERE id = 1;", true},
		{"delete", "DELETE FROM users WHERE id = 1;", true},
		{
			"merge writes rows, so it must not force a dump",
			"MERGE INTO users t USING staging s ON t.id = s.id WHEN MATCHED THEN UPDATE SET t.name = s.name;",
			true,
		},
		{"ddl", "ALTER TABLE users ADD COLUMN status STRING;", false},
		{"mixed", "ALTER TABLE users ADD COLUMN status STRING;\nINSERT INTO users (id) VALUES (1);", false},
		{
			// The unparsed statement classifies as UNSPECIFIED rather than
			// vanishing, so a sheet whose DDL omni cannot parse still dumps.
			"unparsed statement is not DML",
			"THIS IS NOT SQL;\nINSERT INTO users (id) VALUES (1);",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, base.IsAllDML(storepb.Engine_BIGQUERY, tt.statement))
		})
	}
}
