package mongodb

import (
	"cmp"
	"context"
	"slices"

	metadatapb "github.com/bytebase/omni/metadata"
	omnicompletion "github.com/bytebase/omni/mongo/completion"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterCompleteFunc(storepb.Engine_MONGODB, Completion)
}

// Completion is the entry point for MongoDB code completion.
func Completion(ctx context.Context, cCtx base.CompletionContext, statement string, caretLine int, caretOffset int) ([]base.Candidate, error) {
	var loadDatabase func(ctx context.Context, name string) (*metadatapb.DatabaseSchemaMetadata, error)
	if cCtx.Metadata != nil {
		loadDatabase = func(ctx context.Context, name string) (*metadatapb.DatabaseSchemaMetadata, error) {
			_, metadata, err := cCtx.Metadata(ctx, cCtx.InstanceID, name)
			if err != nil || metadata == nil {
				return nil, err
			}
			return metadata.GetProto(), nil
		}
	}

	candidates := omnicompletion.CompleteSession(ctx, omnicompletion.SessionConfig{
		DefaultDatabase: cCtx.DefaultDatabase,
		LoadDatabase:    loadDatabase,
	}, statement, caretLine, caretOffset)

	result := make([]base.Candidate, 0, len(candidates))
	for _, c := range candidates {
		result = append(result, base.Candidate{
			Type:       candidateTypeToBase(c.Type),
			Text:       c.Text,
			Definition: c.Definition,
			Comment:    c.Comment,
		})
	}

	// Omni sorts by its finer CandidateType enum; re-sort and compact after
	// collapsing into Bytebase's coarser base.CandidateType.
	slices.SortFunc(result, func(a, b base.Candidate) int {
		return cmp.Or(
			cmp.Compare(a.Type, b.Type),
			cmp.Compare(a.Text, b.Text),
			cmp.Compare(a.Definition, b.Definition),
			cmp.Compare(a.Comment, b.Comment),
		)
	})
	return slices.CompactFunc(result, func(a, b base.Candidate) bool {
		return a.Type == b.Type && a.Text == b.Text && a.Definition == b.Definition
	}), nil
}

// candidateTypeToBase maps omni completion candidate types to Bytebase base types.
func candidateTypeToBase(t omnicompletion.CandidateType) base.CandidateType {
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
