package v1

import (
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/component/masker"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	mongoparser "github.com/bytebase/bytebase/backend/plugin/parser/mongodb"
)

func checkMongoDBRequestBlocked(analysis *mongoparser.MaskingAnalysis) error {
	if analysis == nil {
		return nil
	}
	if analysis.API == mongoparser.MaskableAPIUnsupportedRead {
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
