package mongodb

import (
	"context"
	"slices"

	"github.com/bytebase/omni/mongo/catalog"
	omnicompletion "github.com/bytebase/omni/mongo/completion"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterCompleteFunc(storepb.Engine_MONGODB, Completion)
}

// Completion is the entry point for MongoDB code completion.
func Completion(ctx context.Context, cCtx base.CompletionContext, statement string, caretLine int, caretOffset int) ([]base.Candidate, error) {
	// Build omni catalog from Bytebase metadata.
	cat := buildCatalog(ctx, cCtx)

	// Convert caret line:column to byte offset.
	byteOffset := caretToByteOffset(statement, caretLine, caretOffset)

	// Get omni completion candidates.
	candidates := omnicompletion.Complete(statement, byteOffset, cat)

	// Convert to base.Candidate.
	var result []base.Candidate
	for _, c := range candidates {
		result = append(result, base.Candidate{
			Type: omniCandidateTypeToBase(c.Type),
			Text: c.Text,
		})
	}

	// Sort and deduplicate.
	slices.SortFunc(result, func(a, b base.Candidate) int {
		if a.Type != b.Type {
			if a.Type < b.Type {
				return -1
			}
			return 1
		}
		if a.Text < b.Text {
			return -1
		}
		if a.Text > b.Text {
			return 1
		}
		return 0
	})

	return slices.CompactFunc(result, func(a, b base.Candidate) bool {
		return a.Type == b.Type && a.Text == b.Text
	}), nil
}

// buildCatalog creates an omni catalog from Bytebase completion context.
func buildCatalog(ctx context.Context, cCtx base.CompletionContext) *catalog.Catalog {
	cat := catalog.New()

	if cCtx.DefaultDatabase == "" || cCtx.Metadata == nil {
		return cat
	}

	_, metadata, err := cCtx.Metadata(ctx, cCtx.InstanceID, cCtx.DefaultDatabase)
	if err != nil || metadata == nil {
		return cat
	}

	for _, schema := range metadata.ListSchemaNames() {
		schemaMeta := metadata.GetSchemaMetadata(schema)
		if schemaMeta == nil {
			continue
		}
		for _, table := range schemaMeta.ListTableNames() {
			cat.AddCollection(table)
		}
	}

	return cat
}

// caretToByteOffset converts a 1-based line and 0-based column offset to a byte offset.
func caretToByteOffset(statement string, caretLine int, caretOffset int) int {
	line := 1
	col := 0
	for i, r := range statement {
		if line == caretLine && col == caretOffset {
			return i
		}
		if r == '\n' {
			line++
			col = 0
		} else {
			col++
		}
	}
	return len(statement)
}

// omniCandidateTypeToBase maps omni completion candidate types to Bytebase base types.
func omniCandidateTypeToBase(t omnicompletion.CandidateType) base.CandidateType {
	switch t {
	case omnicompletion.CandidateCollection:
		return base.CandidateTypeTable
	case omnicompletion.CandidateKeyword, omnicompletion.CandidateShowTarget:
		return base.CandidateTypeKeyword
	case omnicompletion.CandidateMethod,
		omnicompletion.CandidateCursorMethod,
		omnicompletion.CandidateAggStage,
		omnicompletion.CandidateQueryOperator,
		omnicompletion.CandidateBSONHelper,
		omnicompletion.CandidateDbMethod,
		omnicompletion.CandidateRsMethod,
		omnicompletion.CandidateShMethod:
		return base.CandidateTypeFunction
	default:
		return base.CandidateTypeFunction
	}
}
