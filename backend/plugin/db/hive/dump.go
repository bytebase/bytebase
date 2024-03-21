package hive

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/pkg/errors"
)

const (
	settingsStmt = "" +
		"SET character_set_client  = %s;\n" +
		"SET character_set_results = %s;\n" +
		"SET collation_connection  = %s;\n" +
		"SET sql_mode              = '%s';\n"
	schemaStmtFmt = "" +
		"--\n" +
		"-- %s structure for `%s`\n" +
		"--\n" +
		"%s;\n"
)

func (d *Driver) Dump(ctx context.Context, _ io.Writer, schemaOnly bool) (string, error) {
	instanceMetadata, err := d.SyncInstance(ctx)
	if err != nil {
		return "", errors.Wrapf(err, "failed to sync instance")
	}

	var dumpString string
	for _, database := range instanceMetadata.Databases[0].Schemas {
		// dump databases.
		databaseDDL, err := d.showCreateSchemaDDL(ctx, "DATABASE", database.Name, "")
		if err != nil {
			return "", errors.Wrapf(err, "failed to dump database %s", database.Name)
		}
		dumpString += databaseDDL

		// dump tables
		for _, table := range database.Tables {
			tabDDL, err := d.showCreateSchemaDDL(ctx, "TABLE", table.Name, database.Name)
			if err != nil {
				return "", errors.Wrapf(err, "failed to dump table %s", table.Name)
			}
			dumpString += tabDDL
		}

		// dump views.
		for _, view := range database.Views {
			viewDDL, err := d.showCreateSchemaDDL(ctx, "VIEW", view.Name, database.Name)
			if err != nil {
				return "", errors.Wrapf(err, "failed to dump view %s", view.Name)
			}
			dumpString += viewDDL
		}

		// TODO(tommy): dump indexes.
	}

	if schemaOnly {
		return dumpString, nil
	}

	return dumpString, nil
}

// Restore the database from src, which is a full backup.
func (*Driver) Restore(_ context.Context, _ io.Reader) error {
	return errors.Errorf("Not implemeted")
}

// This function shows DDLs for creating certain type of schema [VIEW, DATABASE, TABLE].
func (d *Driver) showCreateSchemaDDL(ctx context.Context, schemaType string, schemaName string, belongTo string) (string, error) {
	// 'SHOW CREATE TABLE' can also be used for dumping views.
	queryStatement := fmt.Sprintf("SHOW CREATE %s %s", schemaType, schemaName)
	if schemaType == "VIEW" {
		queryStatement = fmt.Sprintf("SHOW CREATE TABLE %s", schemaName)
	}

	schemaDDLResults, err := d.QueryConn(ctx, nil, queryStatement, nil)
	if err != nil {
		return "", errors.Wrapf(err, "failed to dump %s %s", schemaType, schemaName)
	}

	var schemaDDL string
	for _, row := range schemaDDLResults[0].Rows {
		schemaDDL += fmt.Sprintln(row.Values[0].GetStringValue())
	}

	// rename table to format: [DatabaseName].[TableName].
	newSchemaName := schemaName
	if belongTo != "" {
		newSchemaName = fmt.Sprintf("%s.%s", belongTo, schemaName)
		schemaDDL = strings.Replace(schemaDDL, schemaName, newSchemaName, 1)
	}

	return fmt.Sprintf(schemaStmtFmt, schemaType, newSchemaName, schemaDDL), nil
}
