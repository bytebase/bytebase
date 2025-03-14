package base

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestString(t *testing.T) {
	testCases := []struct {
		root  string
		nodes []SelectorNode
		want  string
	}{
		{
			root: "a",
			nodes: []SelectorNode{
				NewArraySelector("b", 1),
				NewItemSelector("c"),
			},
			want: "a.b[1].c",
		},
	}

	a := require.New(t)

	for _, tc := range testCases {
		ast := NewPathAST(tc.root)
		var next SelectorNode = ast.Root
		for _, node := range tc.nodes {
			next.SetNext(node)
			next = next.GetNext()
		}

		got, err := ast.String()
		a.NoError(err)
		a.Equal(tc.want, got)
	}
}
