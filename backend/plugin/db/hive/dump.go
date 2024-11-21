package hive

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/beltran/gohive"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

const (
	schemaStmtFmt = "" +
		"--\n" +
		"-- %s structure for %s\n" +
		"--\n" +
		"%s;\n"
)

type IndexDDLOptions struct {
	databaseName          string
	indexName             string
	tableName             string
	indexType             string
	colNames              []string
	isWithDeferredRebuild bool
	idxProperties         map[string]string
	indexTableName        string
	rowFmt                string
	storedAs              string
	storedBy              string
	location              string
	tableProperties       map[string]string
	comment               string
}

type MaterializedViewDDLOptions struct {
	databaseName   string
	mtViewName     string
	disableRewrite bool
	comment        string
	partitionedOn  []string
	clusteredOn    []string
	distributedOn  []string
	sortedOn       []string
	rowFmt         string
	storedAs       string
	storedBy       string
	serdProperties map[string]string
	location       string
	tblProperties  map[string]string
	as             string
}

func (d *Driver) Dump(ctx context.Context, out io.Writer, _ *storepb.DatabaseSchemaMetadata) error {
	var builder strings.Builder

	databaseName := d.config.Database
	database, err := d.SyncDBSchema(ctx)
	if err != nil {
		return err
	}
	if len(database.Schemas) == 0 {
		return errors.New("database schemas is empty")
	}
	schema := database.Schemas[0]

	cursor := d.conn.Cursor()
	if err := executeCursor(ctx, cursor, fmt.Sprintf("use %s", databaseName)); err != nil {
		return err
	}

	// dump managed tables.
	for _, table := range schema.GetTables() {
		tableDDL, err := showCreateDDL(ctx, d.conn, "TABLE", table.Name, d.config.MaximumSQLResultSize)
		if err != nil {
			return errors.Wrapf(err, "failed to dump table %s", table.Name)
		}

		// dump indexes.
		for _, index := range table.GetIndexes() {
			indexDDL, err := genIndexDDL(&IndexDDLOptions{
				databaseName:          databaseName,
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
				return errors.Wrapf(err, "failed to generate DDL for index %s", index.Name)
			}

			_, _ = builder.WriteString(fmt.Sprintf(schemaStmtFmt, "INDEX", fmt.Sprintf("`%s`", index.Name), indexDDL))
		}
		_, _ = builder.WriteString(tableDDL)
	}

	// dump external tables.
	for _, extTable := range schema.GetExternalTables() {
		tabDDL, err := showCreateDDL(ctx, d.conn, "TABLE", extTable.Name, d.config.MaximumSQLResultSize)
		if err != nil {
			return errors.Wrapf(err, "failed to dump table %s", extTable.Name)
		}
		_, _ = builder.WriteString(tabDDL)
	}

	// dump views.
	for _, view := range schema.GetViews() {
		viewDDL, err := showCreateDDL(ctx, d.conn, "VIEW", view.Name, d.config.MaximumSQLResultSize)
		if err != nil {
			return errors.Wrapf(err, "failed to dump view %s", view.Name)
		}
		_, _ = builder.WriteString(viewDDL)
	}

	// dump materialized views.
	for _, mtView := range schema.GetMaterializedViews() {
		mtViewDDL, err := genMaterializedViewDDL(&MaterializedViewDDLOptions{
			databaseName:   databaseName,
			mtViewName:     mtView.Name,
			disableRewrite: false,
			comment:        mtView.Comment,
			partitionedOn:  nil,
			clusteredOn:    nil,
			distributedOn:  nil,
			sortedOn:       nil,
			rowFmt:         "",
			storedAs:       "",
			storedBy:       "",
			serdProperties: nil,
			location:       "",
			tblProperties:  nil,
			as:             mtView.Definition,
		})
		if err != nil {
			return errors.Wrapf(err, "failed to generate DDL for materialized view %s", mtView.Name)
		}

		_, _ = builder.WriteString(fmt.Sprintf(schemaStmtFmt, "MATERIALIZED VIEW", fmt.Sprintf("`%s`.`%s`", databaseName, mtView.Name), mtViewDDL))
	}

	if _, err = io.WriteString(out, builder.String()); err != nil {
		return err
	}
	return nil
}

// This function shows DDLs for creating certain type of schema [VIEW, DATABASE, TABLE].
func showCreateDDL(ctx context.Context, conn *gohive.Connection, objectType string, objectName string, limit int64) (string, error) {
	objectName = fmt.Sprintf("`%s`", objectName)

	// 'SHOW CREATE TABLE' can also be used for dumping views.
	queryStatement := fmt.Sprintf("SHOW CREATE %s %s", objectType, objectName)
	if objectType == "VIEW" {
		queryStatement = fmt.Sprintf("SHOW CREATE TABLE %s", objectName)
	}

	schemaDDLResult, err := runSingleStatement(ctx, conn, queryStatement, limit)
	if err != nil {
		return "", err
	}

	var schemaDDL string
	for _, row := range schemaDDLResult.Rows {
		if row == nil || len(row.Values) == 0 {
			return "", errors.New("empty row")
		}
		schemaDDL += fmt.Sprintln(row.Values[0].GetStringValue())
	}

	return fmt.Sprintf(schemaStmtFmt, objectType, objectName, schemaDDL), nil
}

func genIndexDDL(opts *IndexDDLOptions) (string, error) {
	var builder strings.Builder

	// index name, table name.
	_, _ = builder.WriteString(fmt.Sprintf("CREATE INDEX `%s`\nON TABLE %s ", opts.indexName, fmt.Sprintf("`%s`.`%s`", opts.databaseName, opts.tableName)))

	// column names.
	formatColumn(&builder, opts.colNames)

	// index type.
	_, _ = builder.WriteString(fmt.Sprintf(")\nAS '%s'\n", opts.indexType))

	// with deferred rebuild.
	if opts.isWithDeferredRebuild {
		_, _ = builder.WriteString("WITH DEFERRED REBUILD\n")
	}

	// index properties.
	if opts.idxProperties != nil {
		_, _ = builder.WriteString("IDXPROPERTIES ")
		formatKVPair(&builder, opts.tableProperties)
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
	}
	if opts.storedAs != "" {
		_, _ = builder.WriteString(fmt.Sprintf("STORED AS %s\n", opts.storedAs))
	}
	if opts.storedBy != "" {
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
		formatKVPair(&builder, opts.tableProperties)
	}

	// comment.
	if opts.comment != "" {
		_, _ = builder.WriteString(fmt.Sprintf("COMMENT '%s'\n", opts.comment))
	}

	return builder.String(), nil
}

func genMaterializedViewDDL(opts *MaterializedViewDDLOptions) (string, error) {
	var builder strings.Builder
	viewNameWithDB := fmt.Sprintf("`%s`.`%s`", opts.databaseName, opts.mtViewName)

	// mtView name.
	_, _ = builder.WriteString(fmt.Sprintf("CREATE MATERIALIZED VIEW %s\n", viewNameWithDB))

	// disable rewrite.
	if opts.disableRewrite {
		_, _ = builder.WriteString("DISABLE REWRITE\n")
	}

	// comment.
	if opts.comment != "" {
		_, _ = builder.WriteString(fmt.Sprintf("COMMENT '%s'\n", opts.comment))
	}

	// partitioned on.
	if opts.partitionedOn != nil {
		_, _ = builder.WriteString("PARTITIONED ON ")
		formatColumn(&builder, opts.partitionedOn)
	}

	if opts.clusteredOn != nil && opts.distributedOn != nil {
		return "", errors.New("keywords 'CLUSTERED ON' and 'DISTRIBUTED ON' cannot appear at the same time")
	}
	//   [CLUSTERED ON (col_name, ...) | DISTRIBUTED ON (col_name, ...) SORTED ON (col_name, ...)].
	if opts.distributedOn != nil && opts.sortedOn != nil {
		// distributed on.
		_, _ = builder.WriteString("DISTRIBUTED ON ")
		formatColumn(&builder, opts.distributedOn)

		// sorted on.
		_, _ = builder.WriteString("SORTED ON ")
		formatColumn(&builder, opts.sortedOn)
	}
	if opts.clusteredOn != nil {
		// clusteredOn
		_, _ = builder.WriteString("CLUSTERED ON ")
		formatColumn(&builder, opts.clusteredOn)
	}

	// stored as or stored by.
	if opts.storedAs != "" && opts.storedBy != "" {
		return "", errors.New("keywords 'STORED AS' and 'STORED BY' cannot appear at the same time")
	}
	if opts.storedAs != "" {
		_, _ = builder.WriteString(fmt.Sprintf("STORED AS %s\n", opts.storedAs))
	}
	if opts.storedBy != "" {
		_, _ = builder.WriteString(fmt.Sprintf("STORED BY '%s'\n", opts.storedBy))

		// if with serde properties.
		if opts.serdProperties != nil {
			_, _ = builder.WriteString("WITH SERDEPROPERTIES ")
			formatKVPair(&builder, opts.serdProperties)
		}
	}

	// location.
	if opts.location != "" {
		_, _ = builder.WriteString(opts.location)
		_, _ = builder.WriteRune('\n')
	}

	// table properties.
	if opts.tblProperties != nil {
		_, _ = builder.WriteString("TBLPROPERTIES ")
		formatKVPair(&builder, opts.tblProperties)
	}

	// as.
	_, _ = builder.WriteString(fmt.Sprintf("AS %s \n", opts.as))

	return builder.String(), nil
}

func formatColumn(builder *strings.Builder, columnNames []string) {
	_, _ = builder.WriteString("(\n")
	for idx, colName := range columnNames {
		_, _ = builder.WriteString(fmt.Sprintf("  `%s`", colName))
		if idx != len(columnNames)-1 {
			_, _ = builder.WriteString(",\n")
		}
	}
	_, _ = builder.WriteString(")\n")
}

func formatKVPair(builder *strings.Builder, kvMap map[string]string) {
	_, _ = builder.WriteString("(\n")
	count := 0
	for key, value := range kvMap {
		count++
		_, _ = builder.WriteString(fmt.Sprintf("  '%s' = '%s'", key, value))
		if count != len(kvMap) {
			_, _ = builder.WriteString(",\n")
		}
	}
	_, _ = builder.WriteString(")\n")
}
