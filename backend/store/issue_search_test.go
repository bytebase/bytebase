package store

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Text with no lexeme in it must produce no tsquery at all. Interpolating it
// raw builds an invalid tsquery — `(`, `:`, `'`, `&`, `|` and `!` are tsquery
// syntax — which fails the whole issue list at cast time.
func TestGetTSQueryRejectsNonLexemeText(t *testing.T) {
	t.Parallel()
	for _, text := range []string{"(", ")", ":", "'", "&", "|", "!", "!!!", "<->", " ", `\`} {
		require.Empty(t, getTSQuery(text), "getTSQuery(%q)", text)
	}
	for _, text := range []string{"cleanup", "fix login bug", "100%", "foo-bar", "重构订单表"} {
		require.NotEmpty(t, getTSQuery(text), "getTSQuery(%q)", text)
	}
}
