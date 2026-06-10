package doris

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/bytebase/omni/doris/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterDiagnoseFunc(storepb.Engine_DORIS, Diagnose)
}

// Diagnose returns syntax diagnostics for the given Doris statement.
// Surfaces genuine lex/parse errors from the omni parser, filtering out
// the "not yet supported" stub messages from statement-dispatch fallthroughs.
func Diagnose(_ context.Context, _ base.DiagnoseContext, statement string) ([]base.Diagnostic, error) {
	diags := parser.Diagnose(statement)
	out := make([]base.Diagnostic, 0, len(diags))
	for _, d := range diags {
		if strings.HasSuffix(d.Msg, "statement parsing is not yet supported") {
			continue
		}
		// Convert byte offset to line/col position.
		line, col := byteOffsetToLineCol(statement, d.Loc.Start)
		syntaxErr := &base.SyntaxError{
			Position: &storepb.Position{
				Line:   int32(line),
				Column: int32(col),
			},
			Message: d.Msg,
		}
		out = append(out, base.ConvertSyntaxErrorToDiagnostic(syntaxErr, statement))
	}
	return out, nil
}

// byteOffsetToLineCol returns the 1-based line and 1-based column for a byte
// offset within the statement. Counts \n as line breaks. The 1-based column
// matches the storepb.Position convention that ConvertSyntaxErrorToDiagnostic
// expects (it subtracts 1 internally to land on the 0-based LSP offset).
func byteOffsetToLineCol(s string, offset int) (int, int) {
	if offset < 0 {
		return 1, 1
	}
	if offset > len(s) {
		offset = len(s)
	}
	line, col := 1, 1
	for i := 0; i < offset; {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == '\n' {
			line++
			col = 1
		} else {
			col++
		}
		i += size
	}
	return line, col
}
