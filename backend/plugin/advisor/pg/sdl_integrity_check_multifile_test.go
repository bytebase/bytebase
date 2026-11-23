package pg

import (
	"testing"

	pganalysis "github.com/bytebase/omni/pg/analysis"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

func TestCheckSDLIntegrity_MultiFileMapRouting(t *testing.T) {
	t.Run("empty input map returns empty results", func(t *testing.T) {
		results, err := CheckSDLIntegrity(map[string]string{})
		require.NoError(t, err)
		require.Empty(t, results)
	})

	t.Run("valid multi-file map returns empty advice slice per file", func(t *testing.T) {
		files := map[string]string{
			"users.sql":  "CREATE TABLE public.users (id int, CONSTRAINT pk_users PRIMARY KEY (id));",
			"orders.sql": "CREATE TABLE public.orders (id int, user_id int, CONSTRAINT fk_u FOREIGN KEY (user_id) REFERENCES public.users(id));",
		}
		results, err := CheckSDLIntegrity(files)
		require.NoError(t, err)
		require.Len(t, results, 2)
		require.Empty(t, results["users.sql"])
		require.Empty(t, results["orders.sql"])
	})

	t.Run("syntax error is keyed to the failing file only", func(t *testing.T) {
		files := map[string]string{
			"valid.sql":   "CREATE TABLE public.users (id int);",
			"invalid.sql": "CREATE TABLE public.orders (id int user_id int);",
		}
		results, err := CheckSDLIntegrity(files)
		require.NoError(t, err)
		require.Empty(t, results["valid.sql"])
		require.Len(t, results["invalid.sql"], 1)
		require.Equal(t, code.StatementSyntaxError.Int32(), results["invalid.sql"][0].Code)
		require.Contains(t, results["invalid.sql"][0].Content, "file 'invalid.sql'")
		require.Nil(t, results["invalid.sql"][0].StartPosition)
	})

	t.Run("routes cross-file duplicates across file map entries", func(t *testing.T) {
		files := map[string]string{
			"a.sql": "CREATE TABLE public.t (id int, CONSTRAINT pk_dup PRIMARY KEY (id));\n" +
				"CREATE INDEX idx_dup ON public.t (id);\n" +
				"CREATE VIEW public.v_dup AS SELECT id FROM public.t;",
			"b.sql": "CREATE TABLE public.t (id int, CONSTRAINT pk_dup PRIMARY KEY (id));\n" +
				"CREATE INDEX idx_dup ON public.t (id);\n" +
				"CREATE VIEW public.v_dup AS SELECT id FROM public.t;",
		}
		results, err := CheckSDLIntegrity(files)
		require.NoError(t, err)

		var all []*storepb.Advice
		for _, advices := range results {
			all = append(all, advices...)
		}
		require.Len(t, all, 4)

		titles := make([]string, 0, len(all))
		for _, a := range all {
			require.Equal(t, storepb.Advice_ERROR, a.Status)
			titles = append(titles, a.Title)
		}
		require.ElementsMatch(t, []string{
			"Duplicate table name across files",
			"Duplicate index name across files",
			"Duplicate view name across files",
			"Duplicate constraint name across files",
		}, titles)
	})
}

func TestSDLIntegrityCrossFileAdviceFormatting(t *testing.T) {
	files := map[string]string{
		"first.sql":  "line1\nfirst_def",
		"second.sql": "line1\nline2\nsecond_def",
	}
	firstStart := len("line1\n")
	secondStart := len("line1\nline2\n")

	tests := []struct {
		name    string
		finding pganalysis.SDLFinding
		code    code.Code
		title   string
		contain []string
	}{
		{
			name: "duplicate table across files",
			finding: pganalysis.SDLFinding{
				Code:   pganalysis.SDLDuplicateTableAcrossFiles,
				Start:  secondStart,
				Schema: "public",
				Table:  "users",
				First:  &pganalysis.SDLDefinition{Path: "first.sql", Start: firstStart},
			},
			code:  code.SDLDuplicateTableName,
			title: "Duplicate table name across files",
			contain: []string{
				"Table 'public.users' is defined in multiple SDL files",
				"First definition: first.sql (line 2)",
				"Duplicate definition: second.sql (line 3)",
			},
		},
		{
			name: "duplicate index across files",
			finding: pganalysis.SDLFinding{
				Code:   pganalysis.SDLDuplicateIndexAcrossFiles,
				Start:  secondStart,
				Schema: "public",
				Name:   "idx_users",
				First:  &pganalysis.SDLDefinition{Path: "first.sql", Start: firstStart},
			},
			code:  code.SDLDuplicateIndexName,
			title: "Duplicate index name across files",
			contain: []string{
				"Index 'public.idx_users' is defined in multiple SDL files",
				"First definition: first.sql (line 2)",
				"Duplicate definition: second.sql (line 3)",
			},
		},
		{
			name: "duplicate view across files",
			finding: pganalysis.SDLFinding{
				Code:   pganalysis.SDLDuplicateViewAcrossFiles,
				Start:  secondStart,
				Schema: "public",
				Name:   "v_users",
				First:  &pganalysis.SDLDefinition{Path: "first.sql", Start: firstStart},
			},
			code:  code.SDLDuplicateTableName,
			title: "Duplicate view name across files",
			contain: []string{
				"View 'public.v_users' is defined in multiple SDL files",
				"First definition: first.sql (line 2)",
				"Duplicate definition: second.sql (line 3)",
			},
		},
		{
			name: "duplicate constraint across files",
			finding: pganalysis.SDLFinding{
				Code:           pganalysis.SDLDuplicateConstraintAcrossFiles,
				Start:          secondStart,
				Schema:         "public",
				Table:          "orders",
				Name:           "pk_common",
				ConstraintType: "UNIQUE",
				First: &pganalysis.SDLDefinition{
					Path:           "first.sql",
					Table:          "users",
					ConstraintType: "PRIMARY KEY",
					Start:          firstStart,
				},
			},
			code:  code.SDLDuplicateConstraintName,
			title: "Duplicate constraint name across files",
			contain: []string{
				"Constraint 'pk_common' in schema 'public' is defined in multiple SDL files",
				"First definition: PRIMARY KEY on table 'users' in first.sql (line 2)",
				"Duplicate definition: UNIQUE on table 'orders' in second.sql (line 3)",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			a := sdlIntegrityAdvice(files, "second.sql", tc.finding)
			require.Equal(t, storepb.Advice_ERROR, a.Status)
			require.Equal(t, tc.code.Int32(), a.Code)
			require.Equal(t, tc.title, a.Title)
			require.Equal(t, &storepb.Position{Line: 3}, a.StartPosition)
			for _, sub := range tc.contain {
				require.Contains(t, a.Content, sub)
			}
		})
	}
}
