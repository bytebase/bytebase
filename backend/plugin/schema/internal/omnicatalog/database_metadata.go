package omnicatalog

import (
	"fmt"
	"strings"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/pkg/errors"
)

// ExtractDatabaseMetadata initializes an omni catalog with dbName, executes schemaText,
// and exports the resulting DatabaseSchemaMetadata. When currentDB is non-nil, it
// selects the database name to export (used by TiDB when schemaText selects its own
// database via USE).
func ExtractDatabaseMetadata[Catalog any](
	schemaText string,
	dbName string,
	newCatalog func() *Catalog,
	exec func(c *Catalog, sql string, continueOnError bool) ([]ExecResult, error),
	currentDB func(c *Catalog) string,
	export func(c *Catalog, dbName string) *metadatapb.DatabaseSchemaMetadata,
) (*metadatapb.DatabaseSchemaMetadata, error) {
	if strings.TrimSpace(schemaText) == "" {
		return &metadatapb.DatabaseSchemaMetadata{}, nil
	}

	c := newCatalog()
	initSQL := fmt.Sprintf("SET foreign_key_checks = 0;\nCREATE DATABASE IF NOT EXISTS `%s`;\nUSE `%s`;", dbName, dbName)
	if _, err := exec(c, initSQL, false); err != nil {
		return nil, errors.Wrap(err, "failed to initialize catalog")
	}

	results, err := exec(c, schemaText, true)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse schema DDL")
	}
	for _, r := range results {
		if r.Error != nil {
			return nil, errors.Wrap(r.Error, "failed to execute schema DDL")
		}
	}

	targetDB := dbName
	if currentDB != nil {
		targetDB = currentDB(c)
	}
	return export(c, targetDB), nil
}
