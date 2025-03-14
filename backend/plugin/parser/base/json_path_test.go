package base

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestString(t *testing.T) {
	testCases := []struct {
		nodes []SelectorNode
		want  string
	}{
		{
			nodes: []SelectorNode{
				NewItemSelector("a"),
				NewArraySelector("b", 1),
				NewItemSelector("c"),
			},
			want: "a.b[1].c",
		},
	}

	a := require.New(t)

	for _, tc := range testCases {
		if len(tc.nodes) == 0 {
			continue
		}

		ast := NewPathAST(tc.nodes[0])
		next := ast.Root
		for i := 1; i < len(tc.nodes); i++ {
			next.SetNext(tc.nodes[i])
			next = next.GetNext()
		}

		got, err := ast.String()
		a.NoError(err)
		a.Equal(tc.want, got)
	}
}
