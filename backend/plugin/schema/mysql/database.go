package mysql

import (
	"strings"

	"github.com/bytebase/bytebase/backend/store/model"
)

// isCurrentDatabase returns true if the given database is the current database.
func isCurrentDatabase(d *model.DatabaseMetadata, database string) bool {
	if !d.GetIsObjectCaseSensitive() {
		return strings.EqualFold(d.DatabaseName(), database)
	}
	return d.DatabaseName() == database
}
