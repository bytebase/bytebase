package v1

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

func TestSummarizeJITGrantMisses(t *testing.T) {
	now := time.Date(2026, 4, 27, 12, 0, 0, 0, time.UTC)
	past := now.Add(-1 * time.Hour)
	future := now.Add(1 * time.Hour)

	tests := []struct {
		name       string
		candidates []*store.AccessGrantMessage
		submitted  string
		want       []jitGrantMissDiag
	}{
		{
			name:       "no candidates returns empty",
			candidates: nil,
			submitted:  "SELECT 1",
			want:       nil,
		},
		{
			name: "candidate with nil payload is skipped",
			candidates: []*store.AccessGrantMessage{
				{ID: "skip-me", Payload: nil},
			},
			submitted: "SELECT 1",
			want:      nil,
		},
		{
			name: "trailing newline mismatch is surfaced as byte and hash diff",
			candidates: []*store.AccessGrantMessage{
				{
					ID:         "trailing-nl",
					ExpireTime: &future,
					Payload:    &storepb.AccessGrantPayload{Query: "SELECT 1\n"},
				},
			},
			submitted: "SELECT 1",
			want: []jitGrantMissDiag{
				{
					GrantID:        "trailing-nl",
					Expired:        false,
					StoredBytes:    9,
					SubmittedBytes: 8,
					// sha256("SELECT 1\n")
					StoredSHA256: "a0a22c9dfa428cdce6fb3e58a40d4052335e17e5fbe42efc2f17f9ca813be62c",
					// sha256("SELECT 1")
					SubmittedSHA256: "e004ebd5b5532a4b85984a62f8ad48a81aa3460c1ca07701f386135d72cdecf5",
				},
			},
		},
		{
			name: "expired candidate has expired=true",
			candidates: []*store.AccessGrantMessage{
				{
					ID:         "expired",
					ExpireTime: &past,
					Payload:    &storepb.AccessGrantPayload{Query: "SELECT 2"},
				},
			},
			submitted: "SELECT 2",
			want: []jitGrantMissDiag{
				{
					GrantID:        "expired",
					Expired:        true,
					StoredBytes:    8,
					SubmittedBytes: 8,
					// sha256("SELECT 2") on both sides — same bytes, so same hash.
					StoredSHA256:    "ebbb5b332060a3ede7047bd7528883b0a560e729848f9feb8ff742145e909b01",
					SubmittedSHA256: "ebbb5b332060a3ede7047bd7528883b0a560e729848f9feb8ff742145e909b01",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := summarizeJITGrantMisses(tt.candidates, tt.submitted, now)
			require.Equal(t, tt.want, got)
		})
	}
}
