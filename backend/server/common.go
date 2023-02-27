package server

import "regexp"

var (
	// resourceIDMatcher is a regular expression that matches a valid resource ID.
	// same with backend/api/v1/common.go
	resourceIDMatcher = regexp.MustCompile("^[a-z]([a-z0-9-]{0,61}[a-z0-9])?$")
)

func isValidResourceID(resourceID string) bool {
	return resourceIDMatcher.MatchString(resourceID)
}
