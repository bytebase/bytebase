package mysql

import (
	"fmt"

	"github.com/bytebase/omni/mysql/catalog"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

func init() {
	schema.RegisterGetDatabaseMetadata(storepb.Engine_MYSQL, GetDatabaseMetadataOmni)
	schema.RegisterGetDatabaseMetadata(storepb.Engine_OCEANBASE, GetDatabaseMetadataOmni)
}

// GetDatabaseMetadataOmni parses MySQL schema DDL text and returns database metadata
// using the omni catalog. This replaces the ANTLR-based GetDatabaseMetadata.
func GetDatabaseMetadataOmni(schemaText string) (*storepb.DatabaseSchemaMetadata, error) {
	if schemaText == "" {
		return &storepb.DatabaseSchemaMetadata{}, nil
	}

	const dbName = "tmp"
	c := catalog.New()
	initSQL := fmt.Sprintf("SET foreign_key_checks = 0;\nCREATE DATABASE IF NOT EXISTS `%s`;\nUSE `%s`;", dbName, dbName)
	if _, err := c.Exec(initSQL, nil); err != nil {
		return nil, errors.Wrap(err, "failed to initialize catalog")
	}

	results, err := c.Exec(schemaText, &catalog.ExecOptions{ContinueOnError: true})
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse schema DDL")
	}

	// Check for hard errors (not per-statement errors).
	for _, r := range results {
		if r.Error != nil {
			return nil, errors.Wrapf(r.Error, "failed to execute schema DDL")
		}
	}

	proto := catalogToProto(c, dbName)
	return proto, nil
}
