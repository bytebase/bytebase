package common

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseCELFilterRejectsOversizedInputWithoutEchoingIt(t *testing.T) {
	filter := `query == "` + strings.Repeat("sensitive", MaxCELFilterCodePoints) + `"`
	_, err := ParseCELFilter(filter)
	require.EqualError(t, err, "filter exceeds the maximum supported size of 100000 code points")
	require.NotContains(t, err.Error(), "sensitive")
}

func TestParseCELFilterRedactsSyntaxError(t *testing.T) {
	_, err := ParseCELFilter(`query == "sensitive SQL" &&`)
	require.EqualError(t, err, "invalid filter expression")
}
