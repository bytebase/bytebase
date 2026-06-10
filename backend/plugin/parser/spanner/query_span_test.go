package spanner

import (
	"testing"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/googlesql/googlesqltest"
)

// TestGetQuerySpan runs the Spanner query-span differential corpus. The
// goldens were RECORDED FROM the legacy ANTLR resolver — they are the
// masking-parity bar and must be reproduced, never re-recorded against this
// implementation (record=true exists for legacy-worktree recording only).
func TestGetQuerySpan(t *testing.T) {
	googlesqltest.RunQuerySpanCorpus(t, storepb.Engine_SPANNER, GetQuerySpan,
		"test-data/query-span/standard.yaml", false /* record */)
}
