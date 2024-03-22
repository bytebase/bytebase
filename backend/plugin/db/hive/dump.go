package hive

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/pkg/errors"
)

const (
	schemaStmtFmt = "" +
		"--\n" +
		"-- %s structure for `%s`\n" +
		"--\n" +
		"%s;\n"
	// TODO(tommy): more rigorous grammar analysis is needed.
	mtViewDDLFormat = "CREATE MATERIALIZED VIEW %s\nAS\n%s\n"
)

type IndexDDLOptions struct {
	indexName             string
	tableName             string
	indexType             string
	colNames              []string
	isWithDeferredRebuild bool
	idxProperties         []string
	indexTableName        string
	rowFmt                string
	storedAs              string
	storedBy              string
	location              string
	tableProperties       []string
	comment               string
}

func (d *Driver) Dump(ctx context.Context, _ io.Writer, _ bool) (string, error) {
	instanceMetadata, err := d.SyncInstance(ctx)
	if err != nil {
		return "", errors.Wrapf(err, "failed to sync instance")
	}

	var dumpStrBuilder strings.Builder
	for _, database := range instanceMetadata.Databases[0].Schemas {
		// dump databases.
		databaseDDL, err := d.showCreateSchemaDDL(ctx, "DATABASE", database.Name, "")
		if err != nil {
			return "", errors.Wrapf(err, "failed to dump database %s", database.Name)
		}
		_, _ = dumpStrBuilder.WriteString(databaseDDL)

		// dump managed tables.
		for _, table := range database.Tables {
			tabDDL, err := d.showCreateSchemaDDL(ctx, "TABLE", table.Name, database.Name)
			if err != nil {
				return "", errors.Wrapf(err, "failed to dump table %s", table.Name)
			}

			// dump indexes.
			for _, index := range table.Indexes {
				var (
					tabNameWithDB = fmt.Sprintf("`%s`.`%s`", database.Name, table.Name)
				)
				// TODO(tommy): get more index info from SyncInstance.
				indexDDL, err := genIndexDDL(&IndexDDLOptions{
					indexName:             index.Name,
					tableName:             table.Name,
					indexType:             index.Type,
					colNames:              index.Expressions,
					isWithDeferredRebuild: true,
					idxProperties:         nil,
					indexTableName:        "",
					rowFmt:                "",
					storedAs:              "",
					storedBy:              "",
					location:              "",
					tableProperties:       nil,
					comment:               index.Comment,
				})
				if err != nil {
					return "", errors.Wrapf(err, "failed to generate DDL for index %s", index.Name)
				}

				_, _ = dumpStrBuilder.WriteString(fmt.Sprintf(schemaStmtFmt, "INDEX", tabNameWithDB, indexDDL))
			}
			_, _ = dumpStrBuilder.WriteString(tabDDL)
		}

		// dump external tables.
		for _, extTable := range database.ExternalTables {
			tabDDL, err := d.showCreateSchemaDDL(ctx, "TABLE", extTable.Name, database.Name)
			if err != nil {
				return "", errors.Wrapf(err, "failed to dump table %s", extTable.Name)
			}
			_, _ = dumpStrBuilder.WriteString(tabDDL)
		}

		// dump views.
		for _, view := range database.Views {
			viewDDL, err := d.showCreateSchemaDDL(ctx, "VIEW", view.Name, database.Name)
			if err != nil {
				return "", errors.Wrapf(err, "failed to dump view %s", view.Name)
			}
			_, _ = dumpStrBuilder.WriteString(viewDDL)
		}

		// dump materialized views.
		for _, mtView := range database.MaterializedViews {
			mtViewDDL := fmt.Sprintf(mtViewDDLFormat, mtView.Name, mtView.Definition)
			_, _ = dumpStrBuilder.WriteString(fmt.Sprintf(schemaStmtFmt, "MATERIALIZED VIEW", mtView.Name, mtViewDDL))
		}
	}

	return dumpStrBuilder.String(), nil
}

// Restore the database from src, which is a full backup.
func (*Driver) Restore(_ context.Context, _ io.Reader) error {
	return errors.Errorf("Not implemeted")
}

// This function shows DDLs for creating certain type of schema [VIEW, DATABASE, TABLE].
func (d *Driver) showCreateSchemaDDL(ctx context.Context, schemaType string, schemaName string, belongTo string) (string, error) {
	// 'SHOW CREATE TABLE' can also be used for dumping views.
	queryStatement := fmt.Sprintf("SHOW CREATE %s `%s`", schemaType, schemaName)
	if schemaType == "VIEW" {
		queryStatement = fmt.Sprintf("SHOW CREATE TABLE `%s`", schemaName)
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

	return fmt.Sprintf(schemaStmtFmt,
		schemaType, newSchemaName, schemaDDL), nil
}

func genIndexDDL(opts *IndexDDLOptions) (string, error) {
	var builder strings.Builder

	// index name, table name.
	_, _ = builder.WriteString(fmt.Sprintf("CREATE INDEX `%s`\nON TABLE `%s` (\n", opts.indexName, opts.tableName))

	// column names.
	for idx, colName := range opts.colNames {
		_, _ = builder.WriteString(fmt.Sprintf("  `%s`", colName))
		if idx != len(opts.colNames)-1 {
			_, _ = builder.WriteString(",\n")
		}
	}

	// index type.
	_, _ = builder.WriteString(fmt.Sprintf(")\nAS '%s'\n", opts.indexType))

	// with deferred rebuild.
	if opts.isWithDeferredRebuild {
		_, _ = builder.WriteString("WITH DEFERRED REBUILD\n")
	}

	// index properties.
	if len(opts.idxProperties) != 0 {
		_, _ = builder.WriteString("IDXPROPERTIES ")
		for idx, prop := range opts.idxProperties {
			_, _ = builder.WriteString(prop)
			if idx != len(opts.idxProperties)-1 {
				_, _ = builder.WriteString(", ")
			}
		}
		_, _ = builder.WriteRune('\n')
	}

	// index table.
	if opts.indexTableName != "" {
		_, _ = builder.WriteString(fmt.Sprintf("IN TABLE %s\n", opts.indexTableName))
	}

	// row format.
	if opts.rowFmt != "" {
		_, _ = builder.WriteString(opts.rowFmt)
		_, _ = builder.WriteRune('\n')
	}

	// stored as or stored by.
	if opts.storedAs != "" && opts.storedBy != "" {
		return "", errors.New("keywords 'STORED AS' and 'STORED BY' cannot appear at the same time")
	} else if opts.storedAs != "" {
		_, _ = builder.WriteString(fmt.Sprintf("STORED AS %s\n", opts.storedAs))
	} else if opts.storedBy != "" {
		_, _ = builder.WriteString(fmt.Sprintf("STORED BY '%s'\n", opts.storedBy))
	}

	// location.
	if opts.location != "" {
		_, _ = builder.WriteString(opts.location)
		_, _ = builder.WriteRune('\n')
	}

	// table properties.
	if opts.tableProperties != nil {
		_, _ = builder.WriteString("TBLPROPERTIES (")
		for idx, prop := range opts.tableProperties {
			_, _ = builder.WriteString(prop)
			if idx != len(opts.tableProperties)-1 {
				_, _ = builder.WriteString(", ")
			}
		}
		_, _ = builder.WriteString(")\n")
	}

	// comment.
	if opts.comment != "" {
		_, _ = builder.WriteString(fmt.Sprintf("COMMENT \"%s\"\n", opts.comment))
	}

	return builder.String(), nil
}
