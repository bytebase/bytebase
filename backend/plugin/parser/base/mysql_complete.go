package base

import (
	"cmp"
	"context"
	"slices"

	metadatapb "github.com/bytebase/omni/metadata"
)

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

// MapMySQLFamilyCandidates converts omni completion candidates into
// base.Candidate values, drops CandidateTypeNone, deduplicates by
// (Type, Text, Definition), and sorts deterministically.
func MapMySQLFamilyCandidates[T any](raw []T, convert func(T) Candidate) []Candidate {
	candidateMap := make(map[string]Candidate, len(raw))
	for _, item := range raw {
		c := convert(item)
		if c.Type == CandidateTypeNone {
			continue
		}
		candidateMap[c.String()] = c
	}

	result := make([]Candidate, 0, len(candidateMap))
	for _, c := range candidateMap {
		result = append(result, c)
	}
	slices.SortFunc(result, func(a, b Candidate) int {
		return cmp.Or(
			cmp.Compare(a.Type, b.Type),
			cmp.Compare(a.Text, b.Text),
			cmp.Compare(a.Definition, b.Definition),
			cmp.Compare(a.Comment, b.Comment),
		)
	})
	return result
}
