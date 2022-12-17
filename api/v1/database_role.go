package v1

import "github.com/bytebase/bytebase/plugin/db"

// DatabaseRole is the API message for role in db.
type DatabaseRole struct {
	Name            string            `json:"name"`
	InstanceID      int               `json:"instanceId"`
	ConnectionLimit int               `json:"connectionLimit"`
	ValidUntil      *string           `json:"validUntil"`
	Attribute       *db.RoleAttribute `json:"attribute"`
}
