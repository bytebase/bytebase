package store

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildStableOrderBy(t *testing.T) {
	tests := []struct {
		name     string
		keys     []*OrderByKey
		tieBreak []string
		want     string
	}{
		{
			name:     "tiebreak follows the only key",
			keys:     []*OrderByKey{{Key: "issue.id", SortOrder: DESC}},
			tieBreak: []string{"issue.project"},
			want:     "ORDER BY issue.id DESC, issue.project DESC",
		},
		{
			name: "tiebreak follows the last key, not the first",
			keys: []*OrderByKey{
				{Key: "issue.created_at", SortOrder: ASC},
				{Key: "issue.id", SortOrder: DESC},
			},
			tieBreak: []string{"issue.project"},
			want:     "ORDER BY issue.created_at ASC, issue.id DESC, issue.project DESC",
		},
		{
			name:     "composite tiebreak keeps its column order",
			keys:     []*OrderByKey{{Key: "release.created_at", SortOrder: DESC}},
			tieBreak: []string{"release.project", "release.train", "release.iteration"},
			want:     "ORDER BY release.created_at DESC, release.project DESC, release.train DESC, release.iteration DESC",
		},
		{
			name:     "a column already sorted on is not repeated as a tiebreak",
			keys:     []*OrderByKey{{Key: "db.name", SortOrder: ASC}},
			tieBreak: []string{"db.instance", "db.name"},
			want:     "ORDER BY db.name ASC, db.instance ASC",
		},
		{
			name:     "no keys sorts by the tiebreak ascending",
			tieBreak: []string{"project.resource_id"},
			want:     "ORDER BY project.resource_id ASC",
		},
		{
			name:     "a duplicate trailing key does not decide the tiebreak direction",
			keys:     []*OrderByKey{{Key: "db.name", SortOrder: DESC}, {Key: "db.name", SortOrder: ASC}},
			tieBreak: []string{"db.instance"},
			want:     "ORDER BY db.name DESC, db.instance DESC",
		},
		{
			name:     "a question mark in a key is escaped for qb",
			keys:     []*OrderByKey{{Key: "payload ? 'urgent'", SortOrder: DESC}},
			tieBreak: []string{"t.id"},
			want:     "ORDER BY payload ?? 'urgent' DESC, t.id DESC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, buildStableOrderBy(tt.keys, tt.tieBreak[0], tt.tieBreak[1:]...))
		})
	}
}
