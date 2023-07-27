package common

import (
	"regexp"
	"strings"
)

var getClassificationFromCommentReg = regexp.MustCompile("^[0-9]+-[0-9]+-[0-9]+")

// GetClassificationAndUserComment parses classification and user comment from the given comment.
func GetClassificationAndUserComment(comment string) (string, string) {
	classification := getClassificationFromCommentReg.FindString(comment)
	userComment := strings.TrimPrefix(strings.TrimPrefix(comment, classification), "-")
	return classification, userComment
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
