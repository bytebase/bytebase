package common

import (
	"fmt"
	"regexp"
	"strings"
)

// classificationIDPattern is the pattern for classification, it should joined id (numbers) by "-", like 1-2, 1-2-3.
const classificationIDPattern = "^[0-9]+(-([0-9])+){0,}"

// GetClassificationAndUserComment parses classification and user comment from the given comment.
func GetClassificationAndUserComment(comment string) (string, string) {
	classificationIDReg := regexp.MustCompile(classificationIDPattern)
	classification := classificationIDReg.FindString(comment)
	if classification == comment {
		// the extract classification id matches full comment, for example, raw comment is "1-2-3"
		return classification, ""
	}

	// we will handle "{classification id}-{comment}" and "{classification id} {comment}"
	classificationIDReg = regexp.MustCompile(fmt.Sprintf(`%s[-|\s]{1}`, classificationIDPattern))
	classification = classificationIDReg.FindString(comment)

	userComment := strings.TrimPrefix(comment, classification)
	classification = strings.TrimSuffix(classification, "-")
	classification = strings.TrimSuffix(classification, " ")
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
