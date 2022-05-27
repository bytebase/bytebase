package mysql

import (
	"bytes"
	"sort"
	"strings"

	"github.com/pingcap/tidb/parser/ast"
	"github.com/pingcap/tidb/parser/format"
)

type columnSet map[string]bool

func newColumnSet(columns []string) columnSet {
	res := make(columnSet)
	for _, col := range columns {
		res[col] = true
	}
	return res
}

type tableState map[string]columnSet

// tableList returns table list in lexicographical order.
func (t tableState) tableList() []string {
	var tableList []string
	for tableName := range t {
		tableList = append(tableList, tableName)
	}
	sort.Strings(tableList)
	return tableList
}

type tablePK map[string]columnSet

// tableList returns table list in lexicographical order.
func (t tablePK) tableList() []string {
	var tableList []string
	for tableName := range t {
		tableList = append(tableList, tableName)
	}
	sort.Strings(tableList)
	return tableList
}

func restoreNode(node ast.Node, flag format.RestoreFlags) (string, error) {
	var buffer strings.Builder
	ctx := format.NewRestoreCtx(flag, &buffer)
	if err := node.Restore(ctx); err != nil {
		return "", err
	}
	return buffer.String(), nil
}

// formatSQLText will convert '\n', '\r' and '\t' to ' '(space). Then turn consecutive spaces into a single.
func formatSQLText(sql string) string {
	var buf bytes.Buffer
	var inChar, hasEscape, consecutiveSpaces bool
	var mark rune
	for _, c := range sql {
		if inChar {
			buf.WriteRune(c)
			if hasEscape {
				hasEscape = false
				continue
			}
			if c == rune('\\') {
				hasEscape = true
			}
			if c == mark {
				inChar = false
			}
			continue
		}
		switch c {
		case rune('"'), rune('`'), rune('\''):
			consecutiveSpaces = false
			inChar = true
			mark = c
			buf.WriteRune(c)
		case rune('\n'), rune('\r'), rune('\t'), rune(' '):
			if !consecutiveSpaces {
				buf.WriteString(" ")
				consecutiveSpaces = true
			}
		default:
			consecutiveSpaces = false
			buf.WriteRune(c)
		}
	}
	return strings.TrimSpace(buf.String())
}
