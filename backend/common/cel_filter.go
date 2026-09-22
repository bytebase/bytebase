//nolint:revive
package common

import (
	"unicode/utf8"

	"github.com/google/cel-go/cel"
	"github.com/pkg/errors"
)

const MaxCELFilterCodePoints = 100_000

// ParseCELFilter parses a bounded CEL list filter without exposing its content in errors.
func ParseCELFilter(filter string) (*cel.Ast, error) {
	if utf8.RuneCountInString(filter) > MaxCELFilterCodePoints {
		return nil, errors.New("filter exceeds the maximum supported size of 100000 code points")
	}

	env, err := cel.NewEnv(cel.ParserExpressionSizeLimit(MaxCELFilterCodePoints))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create CEL environment")
	}
	ast, issues := env.Parse(filter)
	if issues != nil && issues.Err() != nil {
		return nil, errors.New("invalid filter expression")
	}
	return ast, nil
}
