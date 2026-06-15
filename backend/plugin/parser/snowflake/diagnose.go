package snowflake

import (
	"context"

	"github.com/bytebase/omni/snowflake/diagnostics"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterDiagnoseFunc(storepb.Engine_SNOWFLAKE, Diagnose)
}

// Diagnose returns syntax diagnostics for the given Snowflake statement. It
// surfaces genuine lex/parse errors from the omni best-effort parser.
//
// The omni Snowflake parser emits a "<NAME> statement parsing is not yet
// supported" / "unknown or unsupported statement starting with <token>" stub
// error from a handful of dispatch fallthroughs. Unlike Trino (where DESCRIBE
// INPUT / DESCRIBE OUTPUT are valid statements that still hit a stub), every
// such Snowflake stub fires ONLY on genuinely malformed input: an unrecognized
// CREATE/ALTER/DROP object keyword, or an unknown leading keyword. All valid
// object forms (CREATE TABLE/VIEW/WAREHOUSE/SEQUENCE/STREAM/TASK/FUNCTION/...,
// ALTER WAREHOUSE/DATABASE/USER/..., DROP <obj>/UNDROP <obj>) parse cleanly, so
// these stubs represent true syntax errors and are intentionally passed through
// (no suppression filter is needed here). When omni grows coverage for a new
// object the stub simply stops firing for it.
func Diagnose(_ context.Context, _ base.DiagnoseContext, statement string) ([]base.Diagnostic, error) {
	diags := diagnostics.Analyze(statement)
	out := make([]base.Diagnostic, 0, len(diags))
	// Re-derive line:column from the byte offset so the column is rune-based
	// (the position that base.ConvertSyntaxErrorToDiagnostic expects). omni's
	// own Range.Column is byte-based, so we deliberately use Offset instead to
	// keep multibyte input positions correct — mirroring the Trino diagnose.go.
	mapper := base.NewByteOffsetPositionMapper(statement)
	for _, d := range diags {
		syntaxErr := &base.SyntaxError{
			Position: mapper.Position(d.Range.Start.Offset),
			Message:  d.Message,
		}
		out = append(out, base.ConvertSyntaxErrorToDiagnostic(syntaxErr, statement))
	}
	return out, nil
}

// parseSnowflakeSQL parses statement with the omni best-effort parser and
// returns the first syntax error as a *base.SyntaxError (nil when the input is
// syntactically valid or empty). It mirrors the Trino parseTrinoSQL helper and
// preserves the "first syntax error" shape the legacy implementation exposed.
func parseSnowflakeSQL(statement string) *base.SyntaxError {
	diags := diagnostics.Analyze(statement)
	if len(diags) == 0 {
		return nil
	}
	d := diags[0]
	mapper := base.NewByteOffsetPositionMapper(statement)
	return &base.SyntaxError{
		Position: mapper.Position(d.Range.Start.Offset),
		Message:  d.Message,
	}
}
