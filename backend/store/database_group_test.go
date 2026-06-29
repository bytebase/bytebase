package store

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildListDatabaseGroupsQueryMultiProjectFilter(t *testing.T) {
	resourceID := "group-a"
	tests := []struct {
		name        string
		find        *FindDatabaseGroupMessage
		wantSQLPart string
		wantArgs    []any
		wantErr     string
	}{
		{
			name:    "rejects empty project filter",
			find:    &FindDatabaseGroupMessage{},
			wantErr: "invalid project filter",
		},
		{
			name: "single project uses equality",
			find: &FindDatabaseGroupMessage{
				ProjectIDs: []string{"project-a"},
			},
			wantSQLPart: "WHERE TRUE AND project = $1 ORDER BY project, resource_id ASC",
			wantArgs:    []any{"project-a"},
		},
		{
			name: "multiple projects use any",
			find: &FindDatabaseGroupMessage{
				ProjectIDs: []string{"project-a", "project-b"},
			},
			wantSQLPart: "WHERE TRUE AND project = ANY($1) ORDER BY project, resource_id ASC",
			wantArgs:    []any{[]string{"project-a", "project-b"}},
		},
		{
			name: "single project with resource id",
			find: &FindDatabaseGroupMessage{
				ProjectIDs: []string{"project-a"},
				ResourceID: &resourceID,
			},
			wantSQLPart: "WHERE TRUE AND project = $1 AND resource_id = $2 ORDER BY project, resource_id ASC",
			wantArgs:    []any{"project-a", "group-a"},
		},
		{
			name: "multiple projects with resource id",
			find: &FindDatabaseGroupMessage{
				ProjectIDs: []string{"project-a", "project-b"},
				ResourceID: &resourceID,
			},
			wantSQLPart: "WHERE TRUE AND project = ANY($1) AND resource_id = $2 ORDER BY project, resource_id ASC",
			wantArgs:    []any{[]string{"project-a", "project-b"}, "group-a"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, args, err := buildListDatabaseGroupsQuery(tt.find)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			require.Contains(t, strings.Join(strings.Fields(query), " "), tt.wantSQLPart)
			require.Equal(t, tt.wantArgs, args)
		})
	}
}
