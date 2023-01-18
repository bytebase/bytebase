package idp

import (
	"encoding/json"
)

// UserInfo contains parsed user information returned by the Identity Provider.
type UserInfo struct {
	Identifier  string `json:"identifier"`
	DisplayName string `json:"displayName"`
	Email       string `json:"email"`

	// Source contains original fields returned by the Identity Provider.
	Source json.RawMessage `json:"source"`
}
