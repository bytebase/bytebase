package store

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
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
			name: "no tiebreak renders the keys alone",
			keys: []*OrderByKey{{Key: "plan.id", SortOrder: DESC}},
			want: "ORDER BY plan.id DESC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, buildStableOrderBy(tt.keys, tt.tieBreak...))
		})
	}
}

// TestBuildIssueOrderBy pins the cross-project ordering that keeps My Issues
// pages stable. Issue IDs restart per project, so every project has an issue
// 101, 102, and so on; ordering by id alone leaves those rows tied and offset
// paging then skips some and repeats others.
func TestBuildIssueOrderBy(t *testing.T) {
	rankKey := &OrderByKey{Key: "ts_rank(issue.ts_vector, query)", SortOrder: DESC}
	createTimeDesc := []*OrderByKey{{Key: "issue.created_at", SortOrder: DESC}}

	tests := []struct {
		name        string
		orderByKeys []*OrderByKey
		rankKey     *OrderByKey
		want        string
	}{
		{
			name: "default order is completed by project",
			want: "ORDER BY issue.id DESC, issue.project DESC",
		},
		{
			name:        "caller keys keep id and project as tiebreaks",
			orderByKeys: createTimeDesc,
			want:        "ORDER BY issue.created_at DESC, issue.id DESC, issue.project DESC",
		},
		{
			name:    "relevance ranking leads and is still completed by project",
			rankKey: rankKey,
			want:    "ORDER BY ts_rank(issue.ts_vector, query) DESC, issue.id DESC, issue.project DESC",
		},
		{
			name:        "relevance ranking does not discard the caller's order_by",
			orderByKeys: createTimeDesc,
			rankKey:     rankKey,
			want:        "ORDER BY ts_rank(issue.ts_vector, query) DESC, issue.created_at DESC, issue.id DESC, issue.project DESC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, buildIssueOrderBy(tt.orderByKeys, tt.rankKey))
		})
	}
}

// TestPaginatedListsUseStableOrderBy fails when a store query applies OFFSET
// without building its ORDER BY through buildStableOrderBy. Offset pagination
// reads each page with a separate query, so a sort that is not a total order
// lets tied rows cross the page boundary between reads — the caller then skips
// some rows and sees others twice. See backend/store/AGENTS.md#pagination-ordering.
func TestPaginatedListsUseStableOrderBy(t *testing.T) {
	paths, err := filepath.Glob("*.go")
	require.NoError(t, err)

	type storeFunc struct {
		path          string
		name          string
		appliesOffset bool
		calls         []string
	}

	fset := token.NewFileSet()
	var funcs []*storeFunc
	for _, path := range paths {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, path, nil, 0)
		require.NoErrorf(t, err, "failed to parse %s", path)

		for _, decl := range file.Decls {
			decl, ok := decl.(*ast.FuncDecl)
			if !ok || decl.Body == nil {
				continue
			}
			fn := &storeFunc{path: path, name: decl.Name.Name}
			ast.Inspect(decl.Body, func(n ast.Node) bool {
				switch node := n.(type) {
				case *ast.BasicLit:
					if node.Kind == token.STRING && strings.Contains(node.Value, "OFFSET ?") {
						fn.appliesOffset = true
					}
				case *ast.CallExpr:
					// Package-local calls only; a method call cannot be
					// resolved without type information and never reaches the
					// unexported helper anyway.
					if ident, ok := node.Fun.(*ast.Ident); ok {
						fn.calls = append(fn.calls, ident.Name)
					}
				}
				return true
			})
			funcs = append(funcs, fn)
		}
	}

	// A list may delegate its ORDER BY to a named helper (ListIssues does), so
	// resolve which package functions reach buildStableOrderBy transitively
	// before judging the paginated ones.
	reachesHelper := map[string]bool{"buildStableOrderBy": true}
	for changed := true; changed; {
		changed = false
		for _, fn := range funcs {
			if reachesHelper[fn.name] {
				continue
			}
			for _, callee := range fn.calls {
				if reachesHelper[callee] {
					reachesHelper[fn.name] = true
					changed = true
					break
				}
			}
		}
	}

	paginated := 0
	for _, fn := range funcs {
		if !fn.appliesOffset {
			continue
		}
		paginated++
		require.Truef(t, reachesHelper[fn.name],
			"%s: %s applies OFFSET but does not build its ORDER BY with buildStableOrderBy, "+
				"so its offset pages can skip and repeat rows; see backend/store/AGENTS.md#pagination-ordering",
			fn.path, fn.name)
	}

	// Guards against the detection above silently matching nothing, which would
	// make this test pass for the wrong reason.
	require.NotZero(t, paginated, "found no paginated store queries")
}
