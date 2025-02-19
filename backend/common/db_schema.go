package common

import (
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// GetClassificationAndUserComment parses classification and user comment from the given comment.
func GetClassificationAndUserComment(comment string, classificationConfig *storepb.DataClassificationSetting_DataClassificationConfig) (string, string) {
	if classificationConfig == nil {
		return "", comment
	}
	if _, ok := classificationConfig.Classification[comment]; ok {
		return comment, ""
	}
	for i := len(comment) - 1; i >= 0; i-- {
		if comment[i] != '-' {
			continue
		}
		if _, ok := classificationConfig.Classification[comment[:i]]; ok {
			return comment[:i], comment[i+1:]
		}
	}
	return "", comment
}

// GetCommentFromClassificationAndUserComment returns the comment from the given classification and user comment.
func GetCommentFromClassificationAndUserComment(classification, userComment string) string {
	if classification == "" {
		return userComment
	}
	if userComment == "" {
		return classification
	}
	return classification + "-" + userComment
}
