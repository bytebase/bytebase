package base

import (
	"cmp"
	"context"
	"slices"
	"strings"

	omnianalysis "github.com/bytebase/omni/analysis"
	metadatapb "github.com/bytebase/omni/metadata"
)

// OmniCandidateType maps an omni analysis.CandidateType to a Bytebase CandidateType.
// Unmapped types (including CandidateTypeNone) return CandidateTypeNone.
func OmniCandidateType(t omnianalysis.CandidateType) CandidateType {
	switch t {
	case omnianalysis.CandidateKeyword, omnianalysis.CandidateType_, omnianalysis.CandidateShowTarget:
		return CandidateTypeKeyword
	case omnianalysis.CandidateCatalog, omnianalysis.CandidateDatabase:
		return CandidateTypeDatabase
	case omnianalysis.CandidateSchema:
		return CandidateTypeSchema
	case omnianalysis.CandidateTable, omnianalysis.CandidateSynonym, omnianalysis.CandidateCollection:
		return CandidateTypeTable
	case omnianalysis.CandidateView:
		return CandidateTypeView
	case omnianalysis.CandidateMaterializedView:
		return CandidateTypeMaterializedView
	case omnianalysis.CandidateColumn:
		return CandidateTypeColumn
	case omnianalysis.CandidateFunction,
		omnianalysis.CandidateMethod,
		omnianalysis.CandidateCursorMethod,
		omnianalysis.CandidateAggStage,
		omnianalysis.CandidateQueryOperator,
		omnianalysis.CandidateBSONHelper,
		omnianalysis.CandidateDbMethod,
		omnianalysis.CandidateRsMethod,
		omnianalysis.CandidateShMethod:
		return CandidateTypeFunction
	case omnianalysis.CandidateProcedure:
		return CandidateTypeRoutine
	case omnianalysis.CandidateSequence:
		return CandidateTypeSequence
	case omnianalysis.CandidateIndex:
		return CandidateTypeIndex
	case omnianalysis.CandidateTrigger:
		return CandidateTypeTrigger
	case omnianalysis.CandidateEvent:
		return CandidateTypeEvent
	case omnianalysis.CandidateVariable:
		return CandidateTypeSystemVar
	case omnianalysis.CandidateCharset:
		return CandidateTypeCharset
	case omnianalysis.CandidateEngine:
		return CandidateTypeEngine
	default:
		return CandidateTypeNone
	}
}

// ConvertOmniCandidates converts omni completion candidates into deduplicated,
// sorted Candidate values, dropping entries whose type maps to CandidateTypeNone.
func ConvertOmniCandidates[T any](
	items []T,
	extract func(T) (text string, typ omnianalysis.CandidateType, definition, comment string),
	compare func(a, b Candidate) int,
) []Candidate {
	return DeduplicateCandidates(items, func(item T) (Candidate, bool) {
		text, rawType, def, comment := extract(item)
		typ := OmniCandidateType(rawType)
		if typ == CandidateTypeNone {
			return Candidate{}, false
		}
		return Candidate{
			Text:       text,
			Type:       typ,
			Definition: def,
			Comment:    comment,
		}, true
	}, compare)
}

// CompletionMetadataLoaders adapts CompletionContext's Metadata and
// ListDatabaseNames callbacks into the plain callbacks expected by omni's
// session completion APIs.
func CompletionMetadataLoaders(cCtx CompletionContext) (
	loadMetadata func(ctx context.Context, name string) (*metadatapb.DatabaseSchemaMetadata, error),
	listNames func(ctx context.Context) ([]string, error),
) {
	if cCtx.Metadata != nil {
		loadMetadata = func(ctx context.Context, name string) (*metadatapb.DatabaseSchemaMetadata, error) {
			_, dbMeta, err := cCtx.Metadata(ctx, cCtx.InstanceID, name)
			if err != nil || dbMeta == nil {
				return nil, err
			}
			return dbMeta.GetProto(), nil
		}
	}
	if cCtx.ListDatabaseNames != nil {
		listNames = func(ctx context.Context) ([]string, error) {
			return cCtx.ListDatabaseNames(ctx, cCtx.InstanceID)
		}
	}
	return loadMetadata, listNames
}

// MySQLFamilyDatabaseLoaders adapts CompletionContext's Metadata and
// ListDatabaseNames callbacks into the plain callbacks expected by omni's
// MySQL-family session completion APIs.
func MySQLFamilyDatabaseLoaders(cCtx CompletionContext) (
	loadDatabase func(ctx context.Context, name string) (resolvedName string, meta *metadatapb.DatabaseSchemaMetadata, caseSensitive bool, ok bool),
	listDatabaseNames func(ctx context.Context) ([]string, error),
) {
	if cCtx.Metadata != nil {
		loadDatabase = func(ctx context.Context, name string) (string, *metadatapb.DatabaseSchemaMetadata, bool, bool) {
			resolvedName, dbMeta, err := cCtx.Metadata(ctx, cCtx.InstanceID, name)
			if err != nil || dbMeta == nil {
				return "", nil, false, false
			}
			return resolvedName, dbMeta.GetProto(), dbMeta.GetIsObjectCaseSensitive(), true
		}
	}
	if cCtx.ListDatabaseNames != nil {
		listDatabaseNames = func(ctx context.Context) ([]string, error) {
			return cCtx.ListDatabaseNames(ctx, cCtx.InstanceID)
		}
	}
	return loadDatabase, listDatabaseNames
}

// DeduplicateCandidates removes duplicate candidates keyed by Candidate.String()
// while preserving first occurrence, then sorts with compare.
func DeduplicateCandidates[T any](items []T, convert func(T) (Candidate, bool), compare func(a, b Candidate) int) []Candidate {
	result := make([]Candidate, 0, len(items))
	seen := make(map[string]bool, len(items))
	for _, item := range items {
		c, ok := convert(item)
		if !ok {
			continue
		}
		key := c.String()
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, c)
	}
	if compare != nil {
		slices.SortFunc(result, compare)
	}
	return result
}

// CompareCandidateByTypeText compares candidates by Type string then Text.
func CompareCandidateByTypeText(a, b Candidate) int {
	if a.Type != b.Type {
		return strings.Compare(string(a.Type), string(b.Type))
	}
	return strings.Compare(a.Text, b.Text)
}

// CompareCandidate compares candidates deterministically by Type, Text, Definition, and Comment.
func CompareCandidate(a, b Candidate) int {
	return cmp.Or(
		cmp.Compare(a.Type, b.Type),
		cmp.Compare(a.Text, b.Text),
		cmp.Compare(a.Definition, b.Definition),
		cmp.Compare(a.Comment, b.Comment),
	)
}
