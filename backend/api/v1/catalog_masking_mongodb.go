package v1

import (
	"encoding/json"
	"maps"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/component/masker"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	parserbase "github.com/bytebase/bytebase/backend/plugin/parser/base"
)

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
