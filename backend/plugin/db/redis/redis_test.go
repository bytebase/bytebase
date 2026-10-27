package redis

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSplitCommands(t *testing.T) {
	tests := []struct {
		name      string
		statement string
		wantArgs  [][]any
		wantLines []string
	}{
		{
			name:      "one command per line",
			statement: "GET a\nGET b",
			wantArgs:  [][]any{{"GET", "a"}, {"GET", "b"}},
			wantLines: []string{"GET a", "GET b"},
		},
		{
			name:      "a blank line does not shift the commands after it",
			statement: "GET a\n\nGET b\n   \nGET c",
			wantArgs:  [][]any{{"GET", "a"}, {"GET", "b"}, {"GET", "c"}},
			wantLines: []string{"GET a", "GET b", "GET c"},
		},
		{
			name:      "quoted argument",
			statement: `SET a "two words"`,
			wantArgs:  [][]any{{"SET", "a", "two words"}},
			wantLines: []string{`SET a "two words"`},
		},
		{
			name:      "no commands",
			statement: "\n  \n",
			wantArgs:  nil,
			wantLines: nil,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			args, lines, err := splitCommands(tc.statement)
			require.NoError(t, err)
			require.Equal(t, tc.wantArgs, args)
			require.Equal(t, tc.wantLines, lines)
			require.Len(t, lines, len(args))
		})
	}
}

func TestSplitCommandsRejectsAnUnclosedQuote(t *testing.T) {
	_, _, err := splitCommands(`SET a "unclosed`)
	require.Error(t, err)
}
