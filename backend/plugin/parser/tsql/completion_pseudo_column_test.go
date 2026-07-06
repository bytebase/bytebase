package tsql

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// TestCompletionPseudoColumnAction pins that the OUTPUT-clause position
// surfaces the $action pseudo-column: omni emits the pseudocolumn_action rule
// there, which previously fell through Bytebase's rule switch silently.
func TestCompletionPseudoColumnAction(t *testing.T) {
	statement, caretLine, caretPosition := getCaretPosition(
		"MERGE INTO t AS d USING s ON d.k = s.k WHEN MATCHED THEN UPDATE SET d.v = s.v OUTPUT |")
	getter, lister := buildMockDatabaseMetadataGetterLister()
	results, err := Completion(context.Background(), base.CompletionContext{
		Scene:             base.SceneTypeAll,
		DefaultDatabase:   "Company",
		Metadata:          getter,
		ListDatabaseNames: lister,
	}, statement, caretLine, caretPosition)
	require.NoError(t, err)
	require.Contains(t, results, base.Candidate{
		Type: base.CandidateTypeColumn,
		Text: "$action",
	})
}
