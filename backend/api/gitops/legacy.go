package gitops

import (
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/store/model"
)

// migrationDetail is the detail for database migration such as Migrate, Data.
type migrationDetail struct {
	// migrationType is the type of a migration.
	// migrationType can be empty for gh-ost type of migration.
	migrationType db.MigrationType
	// databaseID is the ID of a database.
	// This should be unset when a project is in tenant mode. The ProjectID is derived from IssueCreate.
	databaseID int
	// sheetID is the ID of a sheet. Statement and sheet ID is mutually exclusive.
	sheetID int
	// schemaVersion is parsed from VCS file name.
	// It is automatically generated in the UI workflow.
	schemaVersion model.Version
}

// MigrationFileYAMLDatabase contains the information of a database in a YAML
// format migration file.
type MigrationFileYAMLDatabase struct {
	Name string `yaml:"name"` // The name of the database
}

// MigrationFileYAML contains the information in a YAML format migration file.
type MigrationFileYAML struct {
	Databases []MigrationFileYAMLDatabase `yaml:"databases"` // The list of databases and how to identify them
	Statement string                      `yaml:"statement"` // The SQL statement to be executed to specified list of databases
}
