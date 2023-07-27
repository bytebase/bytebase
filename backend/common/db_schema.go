package common

import (
	"regexp"
	"strings"
)

var getCategoryFromCommentReg = regexp.MustCompile("^[0-9]+-[0-9]+-[0-9]+")

// GetCategoryAndUserComment parses category and user comment from the given comment.
func GetCategoryAndUserComment(comment string) (string, string) {
	category := getCategoryFromCommentReg.FindString(comment)
	userComment := strings.TrimPrefix(strings.TrimPrefix(comment, category), "-")
	return category, userComment
}

func GetCommentFromCategoryAndUserComment(category, userComment string) string {
	if category == "" {
		return userComment
	}
	if userComment == "" {
		return category
	}
	return category + "-" + userComment
}
