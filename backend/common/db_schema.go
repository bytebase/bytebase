package common

import (
	"regexp"
	"strings"
)

var numberReg = regexp.MustCompile("^[0-9]+$")

// GetClassificationAndUserComment parses classification and user comment from the given comment.
func GetClassificationAndUserComment(comment string) (string, string) {
	sections := strings.Split(comment, "-")
	classification := []string{}
	userComment := ""
	for i, section := range sections {
		if numberReg.MatchString(section) {
			classification = append(classification, section)
		} else {
			userComment = strings.Join(sections[i:], "-")
			break
		}
	}
	return strings.Join(classification, "-"), userComment
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
