package v1

import (
	"context"
	"encoding/json"
	"maps"

	"connectrpc.com/connect"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/component/masker"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	parserbase "github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store"
)

// mongoDBMasker implements documentMasker for MongoDB.
type mongoDBMasker struct{}

func (*mongoDBMasker) maskResults(
	ctx context.Context,
	stores *store.Store,
	database *store.DatabaseMessage,
	spans []*parserbase.QuerySpan,
	results []*v1pb.QueryResult,
	semanticTypeToMaskerMap map[string]masker.Masker,
	_ db.QueryContext,
) error {
	for i, result := range results {
		if i < len(spans) {
			if errMsg, err := checkSensitivePredicates(ctx, stores, database, spans[i]); err != nil {
				return connect.NewError(connect.CodeInternal, err)
			} else if errMsg != "" {
				result.Error = errMsg
				result.Rows = nil
				result.RowsCount = 0
				continue
			}
		}

		var analysis *parserbase.MongoDBAnalysis
		if i < len(spans) {
			analysis = spans[i].MongoDBAnalysis
		}
		if analysis == nil || analysis.Collection == "" {
			continue
		}
		if analysis.API != parserbase.MongoDBMaskableAPIFind && analysis.API != parserbase.MongoDBMaskableAPIFindOne && analysis.API != parserbase.MongoDBMaskableAPIAggregate {
			continue
		}

		objectSchema, err := getMongoDBCollectionObjectSchema(ctx, stores, database.InstanceID, database.DatabaseName, analysis.Collection)
		if err != nil {
			return connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get object schema for collection %q", analysis.Collection))
		}
		if objectSchema == nil {
			continue
		}

		if len(analysis.JoinedCollections) > 0 {
			var joined []joinedSchema
			for _, jc := range analysis.JoinedCollections {
				js, jsErr := getMongoDBCollectionObjectSchema(ctx, stores, database.InstanceID, database.DatabaseName, jc.Collection)
				if jsErr != nil {
					return connect.NewError(connect.CodeInternal, errors.Wrapf(jsErr, "failed to get object schema for joined collection %q", jc.Collection))
				}
				joined = append(joined, joinedSchema{asField: jc.AsField, schema: js})
			}
			objectSchema = injectJoinedSchemas(objectSchema, joined)
		}

		for _, row := range result.Rows {
			if len(row.Values) == 0 {
				continue
			}
			value := row.Values[0].GetStringValue()
			if value == "" {
				continue
			}

			maskedValue, maskErr := maskMongoDBDocumentString(value, objectSchema, semanticTypeToMaskerMap)
			if maskErr != nil {
				return connect.NewError(connect.CodeInternal, errors.Wrapf(maskErr, "failed to mask MongoDB response"))
			}

			row.Values[0] = &v1pb.RowValue{
				Kind: &v1pb.RowValue_StringValue{
					StringValue: maskedValue,
				},
			}
		}
	}
	return nil
}

func checkMongoDBRequestBlocked(analysis *parserbase.MongoDBAnalysis) error {
	if analysis == nil {
		return nil
	}
	if analysis.API == parserbase.MongoDBMaskableAPIUnsupportedRead {
		operation := analysis.Operation
		if operation == "" {
			operation = "unknown"
		}
		if analysis.UnsupportedStage != "" {
			return errors.Errorf("MongoDB aggregate() with stage %q on collection %q is not supported for dynamic masking. Supported operations are find(), findOne(), and aggregate() with shape-preserving stages only", analysis.UnsupportedStage, analysis.Collection)
		}
		return errors.Errorf("MongoDB operation %q on collection %q is not supported for dynamic masking in this release. Supported operations are find() and findOne()", operation+"()", analysis.Collection)
	}
	return nil
}

// injectJoinedSchemas adds array-of-objects schemas for each $lookup/$graphLookup join
// into the source objectSchema so masking covers the joined field.
// The as field holds an array of joined documents, so the injected schema is ARRAY{item: joinedSchema}.
// If the source schema has no StructKind or the joined collection has no schema, the join is skipped silently.
func injectJoinedSchemas(objectSchema *storepb.ObjectSchema, joins []joinedSchema) *storepb.ObjectSchema {
	if objectSchema == nil || len(joins) == 0 {
		return objectSchema
	}
	structKind := objectSchema.GetStructKind()
	if structKind == nil {
		return objectSchema
	}

	// Clone properties map so we don't mutate the cached schema.
	props := make(map[string]*storepb.ObjectSchema, len(structKind.Properties)+len(joins))
	maps.Copy(props, structKind.Properties)
	for _, j := range joins {
		if j.schema == nil {
			continue
		}
		props[j.asField] = &storepb.ObjectSchema{
			Type: storepb.ObjectSchema_ARRAY,
			Kind: &storepb.ObjectSchema_ArrayKind_{
				ArrayKind: &storepb.ObjectSchema_ArrayKind{
					Kind: j.schema,
				},
			},
		}
	}
	return &storepb.ObjectSchema{
		Type: storepb.ObjectSchema_OBJECT,
		Kind: &storepb.ObjectSchema_StructKind_{
			StructKind: &storepb.ObjectSchema_StructKind{
				Properties: props,
			},
		},
	}
}

// joinedSchema pairs an as-field name with its resolved ObjectSchema.
type joinedSchema struct {
	asField string
	schema  *storepb.ObjectSchema
}

func maskMongoDBDocumentString(document string, objectSchema *storepb.ObjectSchema, semanticTypeToMasker map[string]masker.Masker) (string, error) {
	if document == "" || objectSchema == nil {
		return document, nil
	}

	var parsed any
	if err := json.Unmarshal([]byte(document), &parsed); err != nil {
		return "", errors.Wrap(err, "failed to unmarshal MongoDB result document")
	}

	masked, err := walkAndMaskJSONRecursive(parsed, objectSchema, semanticTypeToMasker)
	if err != nil {
		return "", errors.Wrap(err, "failed to mask MongoDB result document")
	}

	maskedJSON, err := json.Marshal(masked)
	if err != nil {
		return "", errors.Wrap(err, "failed to marshal masked MongoDB result document")
	}
	return string(maskedJSON), nil
}
