package service

import (
	v1 "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// DatabaseChange represents a database change that may involve sensitive data.
type DatabaseChange struct {
	// ChangeID is the unique identifier for the database change.
	ChangeID string

	// Requester is the user who initiated the change.
	Requester string

	// Database is the name of the database being changed.
	Database string

	// Schema is the schema of the database being changed.
	Schema string

	// Table is the table being changed.
	Table string

	// SQL is the SQL statement that is being executed.
	SQL string

	// Description is a description of the change.
	Description string

	// SensitiveData is the list of sensitive data fields involved in the change.
	// This may be populated by the sensitive data service.
	SensitiveData []*v1.SensitiveData
}
