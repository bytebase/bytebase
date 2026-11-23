package omnicatalog

import (
	"fmt"
	"strings"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/pkg/errors"
)

// metadataCatalog is an omni catalog that exports a database snapshot by name.
type metadataCatalog interface {
	DatabaseMetadata(dbName string) *metadatapb.DatabaseSchemaMetadata
}

// ExtractSimpleDatabaseMetadata parses schemaText in a fresh "tmp" database
// and exports the resulting metadata via c.DatabaseMetadata.
func ExtractSimpleDatabaseMetadata[C any, PC interface {
	*C
	metadataCatalog
}](
	schemaText string,
	newCatalog func() PC,
	exec func(c PC, sql string, continueOnError bool) ([]ExecResult, error),
) (*metadatapb.DatabaseSchemaMetadata, error) {
	return ExtractDatabaseMetadata(
		schemaText,
		"tmp",
		func() *C { return newCatalog() },
		func(c *C, sql string, continueOnError bool) ([]ExecResult, error) {
			return exec(PC(c), sql, continueOnError)
		},
		nil,
		func(c *C, dbName string) *metadatapb.DatabaseSchemaMetadata {
			return PC(c).DatabaseMetadata(dbName)
		},
	)
}

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
