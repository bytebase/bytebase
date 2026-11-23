package base

import (
	"cmp"
	"context"
	"slices"
	"strings"

	metadatapb "github.com/bytebase/omni/metadata"
)

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
