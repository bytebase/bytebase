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

// GetCommentFromCategoryAndUserComment returns the comment from the given category and user comment.
func GetCommentFromCategoryAndUserComment(category, userComment string) string {
	if category == "" {
		return userComment
	}
	if userComment == "" {
		return category
	}
	return category + "-" + userComment
}
