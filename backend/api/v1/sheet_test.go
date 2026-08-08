package v1

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestSheetContentGetterConfinement asserts nothing under backend/api/v1
// calls the unscoped sheet getter: user-facing reads go through the
// project-scoped store accessors (GetSheetsForProject and friends), and
// GetSheetFull exists for runners and components only. The scoped/unscoped
// split is otherwise compiler-enforced — the raw batch getters and the ref
// filter are unexported from the store.
func TestSheetContentGetterConfinement(t *testing.T) {
	forbidden := map[string]bool{
		"GetSheetFull": true,
	}
	fset := token.NewFileSet()
	entries, err := os.ReadDir(".")
	require.NoError(t, err)
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, name, nil, 0)
		require.NoError(t, err)
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			if forbidden[sel.Sel.Name] {
				t.Errorf("%s: %s is the unscoped runner getter; user-facing reads must use the project-scoped store accessors",
					fset.Position(call.Pos()), sel.Sel.Name)
			}
			return true
		})
	}
}
