// Package idp provides the utilities for Identity Provider plugins.
package idp

import (
	"encoding/json"
)

// UserInfo contains parsed user information returned by the Identity Provider.
type UserInfo struct {
	Identifier  string `json:"identifier"`
	DisplayName string `json:"displayName"`
	Email       string `json:"email"`

	// Raw contains original fields returned by the Identity Provider.
	Raw json.RawMessage `json:"raw"`
}

// FieldMapping contains mapping relations from Bytebase to Identity Provider.
type FieldMapping struct {
	Identifier  string `json:"identifier"`
	DisplayName string `json:"displayName"`
	Email       string `json:"email"`
}
