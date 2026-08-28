package store

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"regexp"
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

// TestBuildIssueOrderBy pins the ordering that keeps issue pages both stable and
// meaningful. Issue IDs restart per project — every project has an issue 101 —
// so across projects id alone is neither unique nor a recency ordering.
func TestBuildIssueOrderBy(t *testing.T) {
	rankKey := &OrderByKey{Key: "ts_rank(issue.ts_vector, query)", SortOrder: DESC}
	createTimeDesc := []*OrderByKey{{Key: "issue.created_at", SortOrder: DESC}}

	tests := []struct {
		name         string
		orderByKeys  []*OrderByKey
		rankKey      *OrderByKey
		crossProject bool
		want         string
	}{
		{
			// project is pinned to a constant, so id is exactly created order
			// here and issue_pkey serves it as an ordered index scan.
			name: "single project keeps the index-served id ordering",
			want: "ORDER BY issue.id DESC, issue.project DESC",
		},
		{
			name:         "cross project leads with created_at, since id is per-project",
			crossProject: true,
			want:         "ORDER BY issue.created_at DESC, issue.id DESC, issue.project DESC",
		},
		{
			name:        "caller keys keep id and project as tiebreaks",
			orderByKeys: createTimeDesc,
			want:        "ORDER BY issue.created_at DESC, issue.id DESC, issue.project DESC",
		},
		{
			// The caller already sorts on created_at, so the cross-project key
			// dedupes away rather than being repeated.
			name:         "an explicit create_time is not duplicated cross project",
			orderByKeys:  createTimeDesc,
			crossProject: true,
			want:         "ORDER BY issue.created_at DESC, issue.id DESC, issue.project DESC",
		},
		{
			name:    "relevance ranking leads and is still completed by project",
			rankKey: rankKey,
			want:    "ORDER BY ts_rank(issue.ts_vector, query) DESC, issue.id DESC, issue.project DESC",
		},
		{
			// Ranking must not lead here: ts_rank is a float that virtually
			// never ties, so a key behind it is never reached and an explicit
			// order_by would be inert.
			name:        "an explicit order_by leads relevance ranking",
			orderByKeys: createTimeDesc,
			rankKey:     rankKey,
			want:        "ORDER BY issue.created_at DESC, ts_rank(issue.ts_vector, query) DESC, issue.id DESC, issue.project DESC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, buildIssueOrderBy(tt.orderByKeys, tt.rankKey, tt.crossProject))
		})
	}
}

// offsetPattern matches an OFFSET clause in any spelling this package uses:
// either case, and either a qb `?` placeholder or a raw `$N` positional one.
// A literal-substring match on "OFFSET ?" misses `LIMIT $1 OFFSET $2`, which
// several raw queries here are written with.
var offsetPattern = regexp.MustCompile(`(?i)\boffset\s*(\?|\$\d+)`)

// TestPaginatedListsUseStableOrderBy fails when a store query applies OFFSET
// without building its ORDER BY through buildStableOrderBy. Offset pagination
// reads each page with a separate query, so a sort that is not a total order
// lets tied rows cross the page boundary between reads — the caller then skips
// some rows and sees others twice. See backend/store/AGENTS.md#pagination-ordering.
//
// What this cannot check: that the tiebreak columns a call site passes are
// actually unique under that query's scope. That judgment is the author's,
// against LATEST.sql — the helper's required tiebreak argument only forces one
// to be named, not to be right.
func TestPaginatedListsUseStableOrderBy(t *testing.T) {
	// Walk subpackages too, so a paginated query added under store/ but outside
	// its root is judged rather than skipped.
	var paths []string
	require.NoError(t, filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			paths = append(paths, path)
		}
		return nil
	}))

	type storeFunc struct {
		path          string
		name          string
		isMethod      bool
		appliesOffset bool
		calls         []string
	}

	fset := token.NewFileSet()
	var funcs []*storeFunc
	for _, path := range paths {
		file, err := parser.ParseFile(fset, path, nil, 0)
		require.NoErrorf(t, err, "failed to parse %s", path)

		for _, decl := range file.Decls {
			decl, ok := decl.(*ast.FuncDecl)
			if !ok || decl.Body == nil {
				continue
			}
			fn := &storeFunc{path: path, name: decl.Name.Name, isMethod: decl.Recv != nil}
			ast.Inspect(decl.Body, func(n ast.Node) bool {
				switch node := n.(type) {
				case *ast.BasicLit:
					if node.Kind == token.STRING && offsetPattern.MatchString(node.Value) {
						fn.appliesOffset = true
					}
				case *ast.CallExpr:
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
	// resolve which functions reach buildStableOrderBy transitively before
	// judging the paginated ones.
	//
	// Only receiver-less declarations may enter this set. A method is never
	// called as a bare identifier, so it can never be the callee resolved here
	// — and admitting one would let a method silently inherit the exemption of
	// an unrelated function that happens to share its name.
	reachesHelper := map[string]bool{"buildStableOrderBy": true}
	for changed := true; changed; {
		changed = false
		for _, fn := range funcs {
			if fn.isMethod || reachesHelper[fn.name] {
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
		reaches := reachesHelper[fn.name]
		if fn.isMethod {
			// Judge a method on its own body: it cannot delegate through the
			// bare-identifier call graph above without calling a package
			// function, which its own call list already records.
			reaches = false
			for _, callee := range fn.calls {
				if reachesHelper[callee] {
					reaches = true
					break
				}
			}
		}
		require.Truef(t, reaches,
			"%s: %s applies OFFSET but does not build its ORDER BY with buildStableOrderBy, "+
				"so its offset pages can skip and repeat rows; see backend/store/AGENTS.md#pagination-ordering",
			fn.path, fn.name)
	}

	// Pinned so that a list dropping out of detection — renamed, or rewritten
	// into an offset spelling the pattern misses — fails here instead of
	// silently going unguarded. Bump it deliberately when adding a paginated
	// list.
	require.Equal(t, 17, paginated,
		"expected 17 offset-paginated store queries; update this count deliberately when adding or removing one")
}
