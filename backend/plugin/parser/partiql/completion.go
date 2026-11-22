package partiql

import (
	"context"
	"strings"
	"unicode/utf8"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/bytebase/omni/partiql/catalog"
	"github.com/bytebase/omni/partiql/completion"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterCompleteFunc(storepb.Engine_DYNAMODB, Completion)
}

// Completion provides auto-complete candidates for PartiQL statements,
// delegating keyword, table, and column resolution to omni's PartiQL completer.
func Completion(ctx context.Context, cCtx base.CompletionContext, statement string, caretLine int, caretOffset int) ([]base.Candidate, error) {
	cat := catalog.New()
	var protoMeta *metadatapb.DatabaseSchemaMetadata
	if cCtx.Metadata != nil && cCtx.DefaultDatabase != "" {
		_, dbMeta, err := cCtx.Metadata(ctx, cCtx.InstanceID, cCtx.DefaultDatabase)
		if err == nil && dbMeta != nil {
			protoMeta = dbMeta.GetProto()
			cat.LoadMetadata(protoMeta)
		}
	}

	pos := lineOffsetToBytePos(statement, caretLine, caretOffset)
	candidates := completion.CompleteWithMetadata(statement, pos, cat, protoMeta)

	return base.DeduplicateCandidates(candidates, func(c completion.Candidate) (base.Candidate, bool) {
		var candidateType base.CandidateType
		switch c.Kind {
		case "keyword":
			if cCtx.Scene == base.SceneTypeQuery && isDMLKeyword(c.Text) {
				return base.Candidate{}, false
			}
			candidateType = base.CandidateTypeKeyword
		case "table":
			candidateType = base.CandidateTypeTable
		case "column":
			candidateType = base.CandidateTypeColumn
		default:
			candidateType = base.CandidateTypeNone
		}
		return base.Candidate{
			Type: candidateType,
			Text: c.Text,
		}, true
	}, base.CompareCandidateByTypeText), nil
}

// lineOffsetToBytePos converts a 1-based line number and 0-based
// character (rune) offset to a 0-based byte position in s.
func lineOffsetToBytePos(s string, line, offset int) int {
	pos := 0
	currentLine := 1
	for pos < len(s) && currentLine < line {
		if s[pos] == '\n' {
			currentLine++
		}
		pos++
	}
	for i := 0; i < offset && pos < len(s); i++ {
		_, size := utf8.DecodeRuneInString(s[pos:])
		pos += size
	}
	if pos > len(s) {
		return len(s)
	}
	return pos
}

func isDMLKeyword(w string) bool {
	switch strings.ToUpper(w) {
	case "INSERT", "UPDATE", "DELETE", "CREATE", "DROP", "REPLACE",
		"UPSERT", "REMOVE", "SET":
		return true
	}
	return false
}
