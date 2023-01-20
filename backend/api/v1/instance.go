package v1

// InstanceDatabasePatch is the API message for patching an instance database.
type InstanceDatabasePatch struct {
	// Project is the project resource ID.
	Project *string `json:"project"`
}
