package mysql

import (
	"fmt"
	"io"
	"log/slog"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	mysql "github.com/bytebase/mysql-parser"

	mysqldb "github.com/bytebase/bytebase/backend/plugin/db/mysql"

	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

type databaseState struct {
	name    string
	schemas map[string]*schemaState
}

func newDatabaseState() *databaseState {
	return &databaseState{
		schemas: make(map[string]*schemaState),
	}
}

func convertToDatabaseState(database *storepb.DatabaseSchemaMetadata) *databaseState {
	state := newDatabaseState()
	state.name = database.Name
	for _, schema := range database.Schemas {
		state.schemas[schema.Name] = convertToSchemaState(schema)
	}
	return state
}

func (s *databaseState) convertToDatabaseMetadata() *storepb.DatabaseSchemaMetadata {
	var schemaStates []*schemaState
	for _, schema := range s.schemas {
		schemaStates = append(schemaStates, schema)
	}
	sort.Slice(schemaStates, func(i, j int) bool {
		return schemaStates[i].id < schemaStates[j].id
	})
	var schemas []*storepb.SchemaMetadata
	for _, schema := range schemaStates {
		schemas = append(schemas, schema.convertToSchemaMetadata())
	}
	return &storepb.DatabaseSchemaMetadata{
		Name:    s.name,
		Schemas: schemas,
		// Unsupported, for tests only.
		Extensions: []*storepb.ExtensionMetadata{},
	}
}

type schemaState struct {
	id           int
	name         string
	tables       map[string]*tableState
	views        map[string]*viewState
	handledViews map[string]bool
	functions    map[string]*functionState
	procedures   map[string]*procedureState
}

func newSchemaState() *schemaState {
	return &schemaState{
		tables:       make(map[string]*tableState),
		views:        make(map[string]*viewState),
		handledViews: make(map[string]bool),
		functions:    make(map[string]*functionState),
		procedures:   make(map[string]*procedureState),
	}
}

func convertToSchemaState(schema *storepb.SchemaMetadata) *schemaState {
	state := newSchemaState()
	state.name = schema.Name
	for i, table := range schema.Tables {
		state.tables[table.Name] = convertToTableState(i, table)
	}
	for i, view := range schema.Views {
		state.views[view.Name] = convertToViewState(i, view)
	}
	for i, function := range schema.Functions {
		state.functions[function.Name] = convertToFunctionState(i, function)
	}
	for i, procedure := range schema.Procedures {
		state.procedures[procedure.Name] = convertToProcedureState(i, procedure)
	}
	return state
}

func (s *schemaState) convertToSchemaMetadata() *storepb.SchemaMetadata {
	var tableStates []*tableState
	for _, table := range s.tables {
		tableStates = append(tableStates, table)
	}
	sort.Slice(tableStates, func(i, j int) bool {
		return tableStates[i].id < tableStates[j].id
	})
	var viewStates []*viewState
	for _, view := range s.views {
		viewStates = append(viewStates, view)
	}
	sort.Slice(viewStates, func(i, j int) bool {
		return viewStates[i].id < viewStates[j].id
	})
	var functionStates []*functionState
	for _, function := range s.functions {
		functionStates = append(functionStates, function)
	}
	sort.Slice(functionStates, func(i, j int) bool {
		return functionStates[i].id < functionStates[j].id
	})
	var procedureStates []*procedureState
	for _, procedure := range s.procedures {
		procedureStates = append(procedureStates, procedure)
	}
	sort.Slice(procedureStates, func(i, j int) bool {
		return procedureStates[i].id < procedureStates[j].id
	})

	var tables []*storepb.TableMetadata
	for _, table := range tableStates {
		tables = append(tables, table.convertToTableMetadata())
	}
	var views []*storepb.ViewMetadata
	for _, view := range viewStates {
		views = append(views, view.convertToViewMetadata())
	}
	var functions []*storepb.FunctionMetadata
	for _, function := range functionStates {
		functions = append(functions, function.convertToFunctionMetadata())
	}
	var procedures []*storepb.ProcedureMetadata
	for _, procedure := range procedureStates {
		procedures = append(procedures, procedure.convertToProcedureMetadata())
	}

	return &storepb.SchemaMetadata{
		Name:       s.name,
		Tables:     tables,
		Views:      views,
		Functions:  functions,
		Procedures: procedures,
		// Unsupported, for tests only.
		Streams:           []*storepb.StreamMetadata{},
		Tasks:             []*storepb.TaskMetadata{},
		MaterializedViews: []*storepb.MaterializedViewMetadata{},
	}
}

type tableState struct {
	id          int
	name        string
	columns     map[string]*columnState
	indexes     map[string]*indexState
	foreignKeys map[string]*foreignKeyState
	comment     string
	// engine and collation is only supported in ParseToMetadata.
	engine    string
	collation string
	partition *partitionState
}

func (t *tableState) toString(buf io.StringWriter) error {
	if _, err := buf.WriteString(fmt.Sprintf("CREATE TABLE `%s` (\n  ", t.name)); err != nil {
		return err
	}
	var columns []*columnState
	for _, column := range t.columns {
		columns = append(columns, column)
	}
	sort.Slice(columns, func(i, j int) bool {
		return columns[i].id < columns[j].id
	})
	for i, column := range columns {
		if i > 0 {
			if _, err := buf.WriteString(",\n  "); err != nil {
				return err
			}
		}
		if err := column.toString(buf); err != nil {
			return err
		}
	}

	var indexes []*indexState
	for _, index := range t.indexes {
		indexes = append(indexes, index)
	}
	// Our sync query is ordered by index name, and our UI only support append-only
	// create index, to avoid introducing text diff, sorting by name instead of id.
	sort.Slice(indexes, func(i, j int) bool {
		if indexes[i].primary {
			return true
		}
		if indexes[j].primary {
			return false
		}
		return indexes[i].name < indexes[j].name
	})

	for i, index := range indexes {
		if i+len(columns) > 0 {
			if _, err := buf.WriteString(",\n  "); err != nil {
				return err
			}
		}
		if err := index.toString(buf); err != nil {
			return err
		}
	}

	var foreignKeys []*foreignKeyState
	for _, fk := range t.foreignKeys {
		foreignKeys = append(foreignKeys, fk)
	}
	sort.Slice(foreignKeys, func(i, j int) bool {
		return foreignKeys[i].name < foreignKeys[j].name
	})

	for i, fk := range foreignKeys {
		if i+len(columns)+len(indexes) > 0 {
			if _, err := buf.WriteString(",\n  "); err != nil {
				return err
			}
		}
		if err := fk.toString(buf); err != nil {
			return err
		}
	}

	if _, err := buf.WriteString("\n)"); err != nil {
		return err
	}

	if t.engine != "" {
		if _, err := buf.WriteString(fmt.Sprintf(" ENGINE=%s", t.engine)); err != nil {
			return err
		}
	}

	if t.collation != "" {
		if _, err := buf.WriteString(fmt.Sprintf(" COLLATE=%s", t.collation)); err != nil {
			return err
		}
	}

	if t.comment != "" {
		if _, err := buf.WriteString(fmt.Sprintf(" COMMENT '%s'", strings.ReplaceAll(t.comment, "'", "''"))); err != nil {
			return err
		}
	}

	if t.partition != nil {
		if err := t.partition.toString(buf, nil); err != nil {
			return err
		}
	}

	if _, err := buf.WriteString(";\n"); err != nil {
		return err
	}
	return nil
}

func newTableState(id int, name string) *tableState {
	return &tableState{
		id:          id,
		name:        name,
		columns:     make(map[string]*columnState),
		indexes:     make(map[string]*indexState),
		foreignKeys: make(map[string]*foreignKeyState),
	}
}

func convertToTableState(id int, table *storepb.TableMetadata) *tableState {
	state := newTableState(id, table.Name)
	state.comment = table.Comment
	state.engine = table.Engine
	state.collation = table.Collation
	for i, column := range table.Columns {
		state.columns[column.Name] = convertToColumnState(i, column)
	}
	for i, index := range table.Indexes {
		state.indexes[index.Name] = convertToIndexState(i, index)
	}
	for i, fk := range table.ForeignKeys {
		state.foreignKeys[fk.Name] = convertToForeignKeyState(i, fk)
	}
	state.partition = convertToPartitionState(table.Partitions)
	return state
}

func (t *tableState) convertToTableMetadata() *storepb.TableMetadata {
	var columnStates []*columnState
	for _, column := range t.columns {
		columnStates = append(columnStates, column)
	}
	sort.Slice(columnStates, func(i, j int) bool {
		return columnStates[i].id < columnStates[j].id
	})
	var columns []*storepb.ColumnMetadata
	for _, column := range columnStates {
		columns = append(columns, column.convertToColumnMetadata())
	}
	// Backfill all the column positions.
	for i, column := range columns {
		column.Position = int32(i + 1)
	}

	var indexStates []*indexState
	for _, index := range t.indexes {
		indexStates = append(indexStates, index)
	}
	sort.Slice(indexStates, func(i, j int) bool {
		return indexStates[i].id < indexStates[j].id
	})
	var indexes []*storepb.IndexMetadata
	for _, index := range indexStates {
		indexes = append(indexes, index.convertToIndexMetadata())
	}

	var fkStates []*foreignKeyState
	for _, fk := range t.foreignKeys {
		fkStates = append(fkStates, fk)
	}
	sort.Slice(fkStates, func(i, j int) bool {
		return fkStates[i].id < fkStates[j].id
	})
	var fks []*storepb.ForeignKeyMetadata
	for _, fk := range fkStates {
		fks = append(fks, fk.convertToForeignKeyMetadata())
	}

	var partitions []*storepb.TablePartitionMetadata
	if t.partition != nil {
		partitions = t.partition.convertToPartitionMetadata()
	}

	return &storepb.TableMetadata{
		Name:        t.name,
		Columns:     columns,
		Indexes:     indexes,
		ForeignKeys: fks,
		UserComment: t.comment,
		Comment:     t.comment,
		Engine:      t.engine,
		Collation:   t.collation,
		Partitions:  partitions,
	}
}

type foreignKeyState struct {
	id                int
	name              string
	columns           []string
	referencedTable   string
	referencedColumns []string
}

func (f *foreignKeyState) convertToForeignKeyMetadata() *storepb.ForeignKeyMetadata {
	return &storepb.ForeignKeyMetadata{
		Name:              f.name,
		Columns:           f.columns,
		ReferencedTable:   f.referencedTable,
		ReferencedColumns: f.referencedColumns,
	}
}

func convertToForeignKeyState(id int, foreignKey *storepb.ForeignKeyMetadata) *foreignKeyState {
	return &foreignKeyState{
		id:                id,
		name:              foreignKey.Name,
		columns:           foreignKey.Columns,
		referencedTable:   foreignKey.ReferencedTable,
		referencedColumns: foreignKey.ReferencedColumns,
	}
}

func (f *foreignKeyState) toString(buf io.StringWriter) error {
	if _, err := buf.WriteString("CONSTRAINT `"); err != nil {
		return err
	}
	if _, err := buf.WriteString(f.name); err != nil {
		return err
	}
	if _, err := buf.WriteString("` FOREIGN KEY ("); err != nil {
		return err
	}
	for i, column := range f.columns {
		if i > 0 {
			if _, err := buf.WriteString(", "); err != nil {
				return err
			}
		}
		if _, err := buf.WriteString("`"); err != nil {
			return err
		}
		if _, err := buf.WriteString(column); err != nil {
			return err
		}
		if _, err := buf.WriteString("`"); err != nil {
			return err
		}
	}
	if _, err := buf.WriteString(") REFERENCES `"); err != nil {
		return err
	}
	if _, err := buf.WriteString(f.referencedTable); err != nil {
		return err
	}
	if _, err := buf.WriteString("` ("); err != nil {
		return err
	}
	for i, column := range f.referencedColumns {
		if i > 0 {
			if _, err := buf.WriteString(", "); err != nil {
				return err
			}
		}
		if _, err := buf.WriteString("`"); err != nil {
			return err
		}
		if _, err := buf.WriteString(column); err != nil {
			return err
		}
		if _, err := buf.WriteString("`"); err != nil {
			return err
		}
	}
	if _, err := buf.WriteString(")"); err != nil {
		return err
	}
	return nil
}

type viewState struct {
	id         int
	name       string
	definition string
}

func (v *viewState) convertToViewMetadata() *storepb.ViewMetadata {
	return &storepb.ViewMetadata{
		Name:       v.name,
		Definition: v.definition,
	}
}

func convertToViewState(id int, view *storepb.ViewMetadata) *viewState {
	return &viewState{
		id:         id,
		name:       view.Name,
		definition: view.Definition,
	}
}

func (v *viewState) toString(buf io.StringWriter) error {
	stmt := fmt.Sprintf("CREATE VIEW `%s` AS %s", v.name, v.definition)
	if !strings.HasSuffix(stmt, ";") {
		stmt += ";"
	}
	if _, err := buf.WriteString(stmt); err != nil {
		return err
	}
	return nil
}

type procedureState struct {
	id         int
	name       string
	definition string
}

func (f *procedureState) convertToProcedureMetadata() *storepb.ProcedureMetadata {
	return &storepb.ProcedureMetadata{
		Name:       f.name,
		Definition: f.definition,
	}
}

func convertToProcedureState(id int, procedure *storepb.ProcedureMetadata) *procedureState {
	return &procedureState{
		id:         id,
		name:       procedure.Name,
		definition: procedure.Definition,
	}
}

func (f *procedureState) toString(buf io.StringWriter) error {
	def := f.definition
	if !strings.HasSuffix(def, " ;;") {
		def += " ;;"
	}

	stmt := fmt.Sprintf("DELIMITER ;;\n%s\nDELIMITER ;", def)
	if _, err := buf.WriteString(stmt); err != nil {
		return err
	}
	return nil
}

type functionState struct {
	id         int
	name       string
	definition string
}

func (f *functionState) convertToFunctionMetadata() *storepb.FunctionMetadata {
	return &storepb.FunctionMetadata{
		Name:       f.name,
		Definition: f.definition,
	}
}

func convertToFunctionState(id int, function *storepb.FunctionMetadata) *functionState {
	return &functionState{
		id:         id,
		name:       function.Name,
		definition: function.Definition,
	}
}

func (f *functionState) toString(buf io.StringWriter) error {
	def := f.definition
	if !strings.HasSuffix(def, " ;;") {
		def += " ;;"
	}

	stmt := fmt.Sprintf("DELIMITER ;;\n%s\nDELIMITER ;", def)
	if _, err := buf.WriteString(stmt); err != nil {
		return err
	}
	return nil
}

type indexState struct {
	id      int
	name    string
	keys    []string
	lengths []int64
	primary bool
	unique  bool
	tp      string
	comment string
}

func (i *indexState) convertToIndexMetadata() *storepb.IndexMetadata {
	return &storepb.IndexMetadata{
		Name:        i.name,
		Expressions: i.keys,
		Primary:     i.primary,
		Unique:      i.unique,
		Comment:     i.comment,
		KeyLength:   i.lengths,
		// Unsupported, for tests only.
		Visible: true,
		Type:    i.tp,
	}
}

func convertToIndexState(id int, index *storepb.IndexMetadata) *indexState {
	return &indexState{
		id:      id,
		name:    index.Name,
		keys:    index.Expressions,
		primary: index.Primary,
		unique:  index.Unique,
		tp:      index.Type,
		comment: index.Comment,
		lengths: index.KeyLength,
	}
}

func (i *indexState) toString(buf io.StringWriter) error {
	if i.primary {
		if _, err := buf.WriteString("PRIMARY KEY ("); err != nil {
			return err
		}
		for j, key := range i.keys {
			if j > 0 {
				if _, err := buf.WriteString(", "); err != nil {
					return err
				}
			}
			if _, err := buf.WriteString(fmt.Sprintf("`%s`", key)); err != nil {
				return err
			}
			if j < len(i.lengths) && i.lengths[j] > 0 {
				if _, err := buf.WriteString(fmt.Sprintf("(%d)", i.lengths[j])); err != nil {
					return err
				}
			}
		}
		if _, err := buf.WriteString(")"); err != nil {
			return err
		}
	} else {
		if strings.ToUpper(i.tp) == "FULLTEXT" {
			if _, err := buf.WriteString("FULLTEXT KEY "); err != nil {
				return err
			}
		} else if strings.ToUpper(i.tp) == "SPATIAL" {
			if _, err := buf.WriteString("SPATIAL KEY "); err != nil {
				return err
			}
		} else if i.unique {
			if _, err := buf.WriteString("UNIQUE KEY "); err != nil {
				return err
			}
		} else {
			if _, err := buf.WriteString("KEY "); err != nil {
				return err
			}
		}

		if _, err := buf.WriteString(fmt.Sprintf("`%s` (", i.name)); err != nil {
			return err
		}
		for j, key := range i.keys {
			if j > 0 {
				if _, err := buf.WriteString(","); err != nil {
					return err
				}
			}
			if len(key) > 2 && key[0] == '(' && key[len(key)-1] == ')' {
				// Expressions are surrounded by parentheses.
				if _, err := buf.WriteString(key); err != nil {
					return err
				}
			} else {
				if _, err := buf.WriteString(fmt.Sprintf("`%s`", key)); err != nil {
					return err
				}
				if j < len(i.lengths) && i.lengths[j] > 0 {
					if _, err := buf.WriteString(fmt.Sprintf("(%d)", i.lengths[j])); err != nil {
						return err
					}
				}
			}
		}
		if _, err := buf.WriteString(")"); err != nil {
			return err
		}

		if strings.ToUpper(i.tp) == "BTREE" {
			if _, err := buf.WriteString(" USING BTREE"); err != nil {
				return err
			}
		} else if strings.ToUpper(i.tp) == "HASH" {
			if _, err := buf.WriteString(" USING HASH"); err != nil {
				return err
			}
		}
	}

	if i.comment != "" {
		if _, err := buf.WriteString(fmt.Sprintf(" COMMENT '%s'", i.comment)); err != nil {
			return err
		}
	}
	return nil
}

type partitionState struct {
	info       partitionInfo
	subInfo    *partitionInfo
	partitions map[string]*partitionDefinition
}

type partitionInfo struct {
	tp         storepb.TablePartitionMetadata_Type
	useDefault int
	expr       string
}

type partitionDefinition struct {
	id            int
	name          string
	value         string
	subpartitions map[string]*partitionDefinition
}

func (p *partitionState) isValid() error {
	if p.info.useDefault == 0 && len(p.partitions) == 0 {
		return errors.New("empty partition list")
	}

	if p.info.useDefault != 0 && len(p.partitions) > 0 {
		return errors.New("specify partitions clause and use default partition are not allowed at the same time")
	}

	if p.info.tp == storepb.TablePartitionMetadata_TYPE_UNSPECIFIED {
		return errors.New("invalid partition type")
	}

	if p.subInfo != nil {
		if p.subInfo.tp == storepb.TablePartitionMetadata_TYPE_UNSPECIFIED {
			return errors.New("invalid subpartition type")
		}

		anySpecificPartition := len(p.partitions) > 0
		if anySpecificPartition {
			var anySpecificSubpartition bool
			for _, partition := range p.partitions {
				if len(partition.subpartitions) > 0 {
					anySpecificSubpartition = true
					break
				}
			}

			if anySpecificSubpartition && p.subInfo.useDefault != 0 {
				return errors.New("specify subpartitions clause and use default subpartition are not allowed at the same time")
			}
		}
	}

	return nil
}

func convertToPartitionState(partitions []*storepb.TablePartitionMetadata) *partitionState {
	if len(partitions) == 0 {
		return nil
	}
	state := &partitionState{
		partitions: make(map[string]*partitionDefinition),
	}
	for i, partition := range partitions {
		if i == 0 {
			state.info.tp = partition.Type
			state.info.expr = partition.Expression
			if partition.UseDefault != "" {
				var err error
				state.info.useDefault, err = strconv.Atoi(partition.UseDefault)
				if err != nil {
					slog.Warn(err.Error())
				}
			}
		}
		partitionDef := &partitionDefinition{
			id:            i,
			name:          partition.Name,
			value:         partition.Value,
			subpartitions: make(map[string]*partitionDefinition),
		}

		for j, subPartition := range partition.Subpartitions {
			if j == 0 {
				state.subInfo = &partitionInfo{
					tp:   subPartition.Type,
					expr: subPartition.Expression,
				}
				if subPartition.UseDefault != "" {
					var err error
					state.subInfo.useDefault, err = strconv.Atoi(subPartition.UseDefault)
					if err != nil {
						slog.Warn(err.Error())
					}
				}
			}
			partitionDef.subpartitions[subPartition.Name] = &partitionDefinition{
				id:    j,
				name:  subPartition.Name,
				value: subPartition.Value,
			}
		}
		state.partitions[partition.Name] = partitionDef
	}
	return state
}

func (p *partitionState) convertToPartitionMetadata() []*storepb.TablePartitionMetadata {
	// There are `useDefault` fields in partitionInfo structure, which may use with empty parititions fields.
	// For example, `PARTITION BY HASH (id) PARTITIONS 4 SUBPARTITION BY KEY(id) SUBPARTITIONS 5;` may cause the `partitions` field to be empty.
	// To be compatible with our protobuf definition, which requires at least one partition, we need to generate the default partition name. And, value
	// is not important, because only the KEY and HASH partition type support PARTITIONS clause.
	// TODO(parser-group): recheck the MySQL behavior of generating the default partition name and value.
	if err := p.isValid(); err != nil {
		slog.Warn(err.Error())
		return nil
	}
	var partitions []*storepb.TablePartitionMetadata
	if p.info.useDefault != 0 {
		generator := mysqlparser.NewPartitionDefaultNameGenerator("")
		for i := 0; i < p.info.useDefault; i++ {
			partitions = append(partitions, &storepb.TablePartitionMetadata{
				Name:       generator.Next(),
				Type:       p.info.tp,
				Expression: p.info.expr,
				Value:      "",
				UseDefault: strconv.Itoa(p.info.useDefault),
			})
		}
		// The reason of we do not consider subUseDefault in this case is that MySQL does not support
		// subpartitions for HASH and KEY partitioning, and they are the only partitioning types that support
		// the PARTITIONS clause.
	} else {
		sortedPartitions := make([]*partitionDefinition, 0, len(p.partitions))
		for _, partition := range p.partitions {
			sortedPartitions = append(sortedPartitions, partition)
		}
		sort.Slice(sortedPartitions, func(i, j int) bool {
			return sortedPartitions[i].id < sortedPartitions[j].id
		})
		for _, partition := range sortedPartitions {
			partitionMetadata := &storepb.TablePartitionMetadata{
				Name:       partition.name,
				Type:       p.info.tp,
				Expression: p.info.expr,
				Value:      partition.value,
			}
			if p.subInfo != nil {
				if p.subInfo.useDefault != 0 {
					subUseDefault := strconv.Itoa(p.subInfo.useDefault)
					generator := mysqlparser.NewPartitionDefaultNameGenerator(partition.name)
					for i := 0; i < p.subInfo.useDefault; i++ {
						partitionMetadata.Subpartitions = append(partitionMetadata.Subpartitions, &storepb.TablePartitionMetadata{
							Name:       generator.Next(),
							Type:       p.subInfo.tp,
							Expression: p.subInfo.expr,
							UseDefault: subUseDefault,
							Value:      "",
						})
					}
				} else {
					sortedSubpartitions := make([]*partitionDefinition, 0, len(partition.subpartitions))
					for _, subPartition := range partition.subpartitions {
						sortedSubpartitions = append(sortedSubpartitions, subPartition)
					}
					sort.Slice(sortedSubpartitions, func(i, j int) bool {
						return sortedSubpartitions[i].id < sortedSubpartitions[j].id
					})
					for _, subPartition := range sortedSubpartitions {
						partitionMetadata.Subpartitions = append(partitionMetadata.Subpartitions, &storepb.TablePartitionMetadata{
							Name:       subPartition.name,
							Type:       p.subInfo.tp,
							Expression: p.subInfo.expr,
							Value:      subPartition.value,
						})
					}
				}
			}
			partitions = append(partitions, partitionMetadata)
		}
	}

	return partitions
}

// toString() writes the partition state as SHOW CREATE TABLE syntax to buf, referencing MySQL source code:
// https://sourcegraph.com/github.com/mysql/mysql-server@824e2b4064053f7daf17d7f3f84b7a3ed92e5fb4/-/blob/sql/sql_show.cc?L2528-2550
// partitionClauseCtx is use to minimize the difference between the original one and the output, it is safe to pass nil.
func (p *partitionState) toString(buf io.StringWriter, partitionClauseCtx mysql.IPartitionClauseContext) error {
	// Write version specific comment.
	vsc := p.getVersionSpecificComment(partitionClauseCtx)
	curComment := vsc
	if _, err := buf.WriteString(vsc); err != nil {
		return err
	}
	if _, err := buf.WriteString(" PARTITION BY "); err != nil {
		return err
	}

	switch p.info.tp {
	case storepb.TablePartitionMetadata_RANGE, storepb.TablePartitionMetadata_RANGE_COLUMNS:
		if _, err := buf.WriteString("RANGE "); err != nil {
			return err
		}
		if p.info.tp == storepb.TablePartitionMetadata_RANGE {
			fields := splitPartitionExprIntoFields(p.info.expr)
			for i, field := range fields {
				fields[i] = fmt.Sprintf("`%s`", field)
			}
			if _, err := buf.WriteString(fmt.Sprintf("(%s)", strings.Join(fields, ","))); err != nil {
				return err
			}
		} else {
			// I think MySQL need to write "COLUMNS " instead of " COLUMNS" here...
			if _, err := buf.WriteString(" COLUMNS"); err != nil {
				return err
			}
			fields := splitPartitionExprIntoFields(p.info.expr)
			if _, err := buf.WriteString(fmt.Sprintf("(%s)", strings.Join(fields, ","))); err != nil {
				return err
			}
		}
	case storepb.TablePartitionMetadata_LIST, storepb.TablePartitionMetadata_LIST_COLUMNS:
		if _, err := buf.WriteString("LIST "); err != nil {
			return err
		}
		if p.info.tp == storepb.TablePartitionMetadata_LIST {
			fields := splitPartitionExprIntoFields(p.info.expr)
			for i, field := range fields {
				fields[i] = fmt.Sprintf("`%s`", field)
			}
			if _, err := buf.WriteString(fmt.Sprintf("(%s)", strings.Join(fields, ","))); err != nil {
				return err
			}
		} else {
			// I think MySQL need to write "COLUMNS " instead of " COLUMNS" here...
			if _, err := buf.WriteString(" COLUMNS"); err != nil {
				return err
			}
			fields := splitPartitionExprIntoFields(p.info.expr)
			if _, err := buf.WriteString(fmt.Sprintf("(%s)", strings.Join(fields, ","))); err != nil {
				return err
			}
		}
	case storepb.TablePartitionMetadata_HASH, storepb.TablePartitionMetadata_KEY, storepb.TablePartitionMetadata_LINEAR_HASH, storepb.TablePartitionMetadata_LINEAR_KEY:
		if p.info.tp == storepb.TablePartitionMetadata_LINEAR_HASH || p.info.tp == storepb.TablePartitionMetadata_LINEAR_KEY {
			if _, err := buf.WriteString("LINEAR "); err != nil {
				return err
			}
		}
		if p.info.tp == storepb.TablePartitionMetadata_KEY || p.info.tp == storepb.TablePartitionMetadata_LINEAR_KEY {
			if _, err := buf.WriteString("KEY "); err != nil {
				return err
			}
			// NOTE: MySQL supports an ALGORITHM option with [SUB]PARTITION BY [LINEAR KEY]. ALGORITHM=1 causes the server to use the same key-hashing function as MYSQL 5.1, and ALGORITHM=1 is the only possible output in
			// the following code. Sadly, I do not know how to get the key_algorithm from the INFORMATION_SCHEMA, AND 5.1 IS TOO LEGACY TO SUPPORT! So use the original one.
			/*
			   current_comment_start is given when called from SHOW CREATE TABLE,
			   Then only add ALGORITHM = 1, not the default 2 or non-set 0!
			   For .frm current_comment_start is NULL, then add ALGORITHM if != 0.
			*/
			// if (part_info->key_algorithm ==
			// 	enum_key_algorithm::KEY_ALGORITHM_51 ||  // SHOW
			// (!current_comment_start &&                   // .frm
			//  (part_info->key_algorithm != enum_key_algorithm::KEY_ALGORITHM_NONE))) {
			// 	/* If we already are within a comment, end that comment first. */
			// 	if (current_comment_start) err += add_string(fptr, "*/ ");
			// 	err += add_string(fptr, "/*!50611 ");
			// 	err += add_part_key_word(fptr, partition_keywords[PKW_ALGORITHM].str);
			// 	err += add_equal(fptr);
			// 	err += add_space(fptr);
			// 	err += add_int(fptr, static_cast<longlong>(part_info->key_algorithm));
			// 	err += add_space(fptr);
			// 	err += add_string(fptr, "*/ ");
			// 	if (current_comment_start) {
			// 		/* Skip new line. */
			// 		if (current_comment_start[0] == '\n') current_comment_start++;
			// 		err += add_string(fptr, current_comment_start);
			// 		err += add_space(fptr);
			// 	}
			// }
			if partitionClauseCtx != nil && partitionClauseCtx.PartitionTypeDef() != nil {
				v := partitionClauseCtx.PartitionTypeDef()
				partitionDefKeyCtx, ok := v.(*mysql.PartitionDefKeyContext)
				if ok && partitionDefKeyCtx.PartitionKeyAlgorithm() != nil {
					numText := partitionDefKeyCtx.PartitionKeyAlgorithm().Real_ulong_number().GetText()
					num, err := strconv.Atoi(numText)
					if err != nil {
						slog.Warn(err.Error())
					} else if num == 1 || (num == 0 && len(curComment) == 0) {
						if _, err := buf.WriteString(fmt.Sprintf("*/ /*!50611 ALGORITHM = %d */ ", num)); err != nil {
							return err
						}
						if len(curComment) > 0 {
							s := curComment
							if curComment[0] == '\n' {
								s = curComment[1:]
							}
							if _, err := buf.WriteString(fmt.Sprintf("%s ", s)); err != nil {
								return err
							}
						}
					}
				}
			}
			// HACK(zp): Write the part field list. In the MySQL source code, it calls append_identifier(), which considers the quote character. We should figure out the logic of it later.
			// Currently, I just found that if the expr contains more than one field, it would not be quoted by '`'.
			// KEY and LINEAR KEY can take the field list.
			// While MySQL calls append_field_list() to write the field list, it unmasks the OPTION_QUOTE_SHOW_CREATE flag,
			// for us, we do the best effort to split the expr by ',' and trim the leading and trailing '`', and write it to the buffer after joining them with ','.
			fields := splitPartitionExprIntoFields(p.info.expr)
			if _, err := buf.WriteString(fmt.Sprintf("(%s)", strings.Join(fields, ","))); err != nil {
				return err
			}
		} else {
			if _, err := buf.WriteString("HASH "); err != nil {
				return err
			}
			fields := splitPartitionExprIntoFields(p.info.expr)
			for i, field := range fields {
				fields[i] = fmt.Sprintf("`%s`", field)
			}
			if _, err := buf.WriteString(fmt.Sprintf("(%s)", strings.Join(fields, ","))); err != nil {
				return err
			}
		}
	default:
		return errors.Errorf("unsupported partition type: %v", p.info.tp)
	}

	// NOTE: MySQL writes the default partitions in the following code, which means that the server
	// takes the responsibility to generate the partitions. Sadly, we cannot get whether the user
	// use this or not in the metadata. So we skip it.
	/*
		if ((!part_info->use_default_num_partitions) &&
		    part_info->use_default_partitions) {
		    	err += add_string(fptr, "\n");
		    	err += add_string(fptr, "PARTITIONS ");
		    	err += add_int(fptr, part_info->num_parts);
		}
	*/
	if p.info.useDefault != 0 {
		if _, err := buf.WriteString(fmt.Sprintf("\nPARTITIONS %d", p.info.useDefault)); err != nil {
			return err
		}
	}

	isSubpartitioned := p.subInfo != nil && p.subInfo.tp != storepb.TablePartitionMetadata_TYPE_UNSPECIFIED
	if isSubpartitioned {
		if _, err := buf.WriteString("\nSUBPARTITION BY "); err != nil {
			return err
		}
	}
	// Subpartition must be hash or key.
	if isSubpartitioned {
		switch p.subInfo.tp {
		case storepb.TablePartitionMetadata_HASH, storepb.TablePartitionMetadata_LINEAR_HASH:
			if p.subInfo.tp == storepb.TablePartitionMetadata_LINEAR_HASH {
				if _, err := buf.WriteString("LINEAR "); err != nil {
					return err
				}
			}
			if _, err := buf.WriteString("HASH "); err != nil {
				return err
			}
			fields := splitPartitionExprIntoFields(p.subInfo.expr)
			for i, field := range fields {
				fields[i] = fmt.Sprintf("`%s`", field)
			}
			if _, err := buf.WriteString(fmt.Sprintf("(%s)", strings.Join(fields, ","))); err != nil {
				return err
			}
		case storepb.TablePartitionMetadata_KEY, storepb.TablePartitionMetadata_LINEAR_KEY:
			if p.subInfo.tp == storepb.TablePartitionMetadata_LINEAR_KEY {
				if _, err := buf.WriteString("LINEAR "); err != nil {
					return err
				}
			}
			if _, err := buf.WriteString("KEY "); err != nil {
				return err
			}
			if partitionClauseCtx != nil && partitionClauseCtx.SubPartitions() != nil {
				if v := partitionClauseCtx.SubPartitions().PartitionKeyAlgorithm(); v != nil {
					numText := v.Real_ulong_number().GetText()
					num, err := strconv.Atoi(numText)
					if err != nil {
						slog.Warn(err.Error())
					} else if num == 1 || (num == 0 && len(curComment) == 0) {
						if _, err := buf.WriteString(fmt.Sprintf("*/ /*!50611 ALGORITHM = %d */ ", num)); err != nil {
							return err
						}
						if len(curComment) > 0 {
							s := curComment
							if curComment[0] == '\n' {
								s = curComment[1:]
							}
							if _, err := buf.WriteString(fmt.Sprintf("%s ", s)); err != nil {
								return err
							}
						}
					}
				}
			}
			fields := splitPartitionExprIntoFields(p.subInfo.expr)
			if _, err := buf.WriteString(fmt.Sprintf("(%s)", strings.Join(fields, ","))); err != nil {
				return err
			}
		default:
			return errors.Errorf("invalid subpartition type: %v", p.subInfo.tp)
		}
	}

	// TODO(zp): MySQL writes the default subpartitions in the following code, which means that the server
	// takes the responsibility to generate the subpartitions. Sadly, we cannot get whether the user
	// use this or not in the metadata. So we skip it.
	/*
		if ((!part_info->use_default_num_subpartitions) &&
			part_info->use_default_subpartitions) {
				err += add_string(fptr, "\n");
				err += add_string(fptr, "SUBPARTITIONS ");
				err += add_int(fptr, part_info->num_subparts);
		}
	*/
	if isSubpartitioned && p.subInfo.useDefault != 0 {
		if _, err := buf.WriteString(fmt.Sprintf("\nSUBPARTITIONS %d", p.subInfo.useDefault)); err != nil {
			return err
		}
	}

	if p.info.useDefault == 0 {
		// Write the partition list.
		if len(p.partitions) == 0 {
			return errors.New("empty partition list")
		}
		sortedPartitions := make([]*partitionDefinition, 0, len(p.partitions))
		for _, partition := range p.partitions {
			sortedPartitions = append(sortedPartitions, partition)
		}
		sort.Slice(sortedPartitions, func(i, j int) bool {
			return sortedPartitions[i].id < sortedPartitions[j].id
		})
		if _, err := buf.WriteString("\n("); err != nil {
			return err
		}
		preposition, err := getPrepositionByType(p.info.tp)
		if err != nil {
			return err
		}
		for i, partition := range sortedPartitions {
			if i != 0 {
				if _, err := buf.WriteString(",\n "); err != nil {
					return err
				}
			}
			if _, err := buf.WriteString(fmt.Sprintf("PARTITION %s", partition.name)); err != nil {
				return err
			}
			if preposition != "" {
				if partition.value != "MAXVALUE" {
					if _, err := buf.WriteString(fmt.Sprintf(" VALUES %s (%s)", preposition, partition.value)); err != nil {
						return err
					}
				} else {
					if _, err := buf.WriteString(fmt.Sprintf(" VALUES %s %s", preposition, partition.value)); err != nil {
						return err
					}
				}
			}

			if isSubpartitioned && p.subInfo.useDefault == 0 {
				if len(partition.subpartitions) == 0 {
					return errors.New("empty subpartition list")
				}
				if _, err := buf.WriteString("\n ("); err != nil {
					return err
				}
				sortedSubpartitions := make([]*partitionDefinition, 0, len(partition.subpartitions))
				for _, subPartition := range partition.subpartitions {
					sortedSubpartitions = append(sortedSubpartitions, subPartition)
				}
				sort.Slice(sortedSubpartitions, func(i, j int) bool {
					return sortedSubpartitions[i].id < sortedSubpartitions[j].id
				})
				for j, subPartition := range sortedSubpartitions {
					if _, err := buf.WriteString(fmt.Sprintf("SUBPARTITION %s", subPartition.name)); err != nil {
						return err
					}
					if err := p.writePartitionOptions(buf); err != nil {
						return err
					}
					if j == len(sortedSubpartitions)-1 {
						if _, err := buf.WriteString(")"); err != nil {
							return err
						}
					} else {
						if _, err := buf.WriteString(",\n  "); err != nil {
							return err
						}
					}
				}
			} else {
				if err := p.writePartitionOptions(buf); err != nil {
					return err
				}
			}

			if i == len(sortedPartitions)-1 {
				if _, err := buf.WriteString(")"); err != nil {
					return err
				}
			}
		}
	}

	if _, err := buf.WriteString(" */"); err != nil {
		return err
	}

	return nil
}

func (*partitionState) writePartitionOptions(buf io.StringWriter) error {
	/*
		int err = 0;
		err += add_space(fptr);
		if (p_elem->tablespace_name) {
			err += add_string(fptr, "TABLESPACE = ");
			err += add_ident_string(fptr, p_elem->tablespace_name);
			err += add_space(fptr);
		}
		if (p_elem->nodegroup_id != UNDEF_NODEGROUP)
			err += add_keyword_int(fptr, "NODEGROUP", (longlong)p_elem->nodegroup_id);
		if (p_elem->part_max_rows)
			err += add_keyword_int(fptr, "MAX_ROWS", (longlong)p_elem->part_max_rows);
		if (p_elem->part_min_rows)
			err += add_keyword_int(fptr, "MIN_ROWS", (longlong)p_elem->part_min_rows);
		if (!(current_thd->variables.sql_mode & MODE_NO_DIR_IN_CREATE)) {
			if (p_elem->data_file_name)
			err += add_keyword_path(fptr, "DATA DIRECTORY", p_elem->data_file_name);
			if (p_elem->index_file_name)
			err += add_keyword_path(fptr, "INDEX DIRECTORY", p_elem->index_file_name);
		}
		if (p_elem->part_comment)
			err += add_keyword_string(fptr, "COMMENT", true, p_elem->part_comment);
		return err + add_engine(fptr, p_elem->engine_type);
	*/
	// TODO(zp): Get all the partition options from the metadata is too complex, just write ENGINE=InnoDB for now.
	if _, err := buf.WriteString(" ENGINE=InnoDB"); err != nil {
		return err
	}

	return nil
}

// getVersionSpecificComment is the go code equivalent of MySQL void partition_info::set_show_version_string(String *packet).
// partitionClauseCtx is use to minimize the difference between the original one and the output, it is safe to pass nil.
func (p *partitionState) getVersionSpecificComment(partitionClauseCtx mysql.IPartitionClauseContext) string {
	if p.info.tp == storepb.TablePartitionMetadata_RANGE_COLUMNS || p.info.tp == storepb.TablePartitionMetadata_LIST_COLUMNS {
		// MySQL introduce columns partitioning in 5.5+.
		return "\n/*!50500"
	} else if partitionClauseCtx != nil {
		/*
				if (part_expr)
			      part_expr->walk(&Item::intro_version, enum_walk::POSTFIX,
			                      (uchar *)&version);
			    if (subpart_expr)
			      subpart_expr->walk(&Item::intro_version, enum_walk::POSTFIX,
			                         (uchar *)&version);
		*/
		tokenStream := partitionClauseCtx.GetParser().GetTokenStream()
		startPos := partitionClauseCtx.GetStart().GetTokenIndex()
		if tokenStream != nil {
			if startPos-2 > 0 && tokenStream.Size() > startPos-2 {
				regexp := regexp.MustCompile(`\/\*![0-9]+`)
				for i := 0; i < 2; i++ {
					if tokenStream.Get(startPos-i-1).GetChannel() == antlr.TokenHiddenChannel {
						if regexp.MatchString(tokenStream.Get(startPos - i - 1).GetText()) {
							return fmt.Sprintf("\n%s", tokenStream.Get(startPos-i-1).GetText())
						}
					}
				}
			}
		}
	}
	// NOTE: Users can use function in partition expr or subpartition expr, and the intro version of function should be the infimum of the version.
	// But sadly, it's a huge work for us to copy the intro version for each function in MySQL. So we use the original one.
	return "\n/*!50100"
}

func getPrepositionByType(tp storepb.TablePartitionMetadata_Type) (string, error) {
	switch tp {
	case storepb.TablePartitionMetadata_RANGE:
		return "LESS THAN", nil
	case storepb.TablePartitionMetadata_RANGE_COLUMNS:
		return "LESS THAN", nil
	case storepb.TablePartitionMetadata_LIST:
		return "IN", nil
	case storepb.TablePartitionMetadata_LIST_COLUMNS:
		return "IN", nil
	case storepb.TablePartitionMetadata_HASH, storepb.TablePartitionMetadata_KEY, storepb.TablePartitionMetadata_LINEAR_HASH, storepb.TablePartitionMetadata_LINEAR_KEY:
		return "", nil
	default:
		return "", errors.Errorf("unsupported partition type: %v", tp)
	}
}

type defaultValue interface {
	toString() string
}

type defaultValueNull struct {
}

func (*defaultValueNull) toString() string {
	return "NULL"
}

type defaultValueString struct {
	value string
}

func (d *defaultValueString) toString() string {
	return fmt.Sprintf("'%s'", strings.ReplaceAll(d.value, "'", "''"))
}

type defaultValueExpression struct {
	value string
}

func (d *defaultValueExpression) toString() string {
	return d.value
}

type columnState struct {
	id           int
	name         string
	tp           string
	defaultValue defaultValue
	onUpdate     string
	comment      string
	nullable     bool
}

func (c *columnState) toString(buf io.StringWriter) error {
	columnCanonicalType := mysqldb.GetColumnTypeCanonicalSynonym(strings.ToLower(c.tp))
	if _, err := buf.WriteString(fmt.Sprintf("`%s` %s", c.name, columnCanonicalType)); err != nil {
		return err
	}
	if !c.nullable {
		if _, err := buf.WriteString(" NOT NULL"); err != nil {
			return err
		}
	}
	if c.defaultValue != nil {
		_, isDefaultNull := c.defaultValue.(*defaultValueNull)
		dontWriteDefaultNull := isDefaultNull && c.nullable && expressionDefaultOnlyTypes[strings.ToUpper(c.tp)]
		// Some types do not default to NULL, but support default expressions.
		if !dontWriteDefaultNull {
			// todo(zp): refactor column attribute.
			if strings.EqualFold(c.defaultValue.toString(), autoIncrementSymbol) {
				if _, err := buf.WriteString(" " + autoIncrementSymbol); err != nil {
					return err
				}
			} else if strings.Contains(strings.ToUpper(c.defaultValue.toString()), autoRandSymbol) {
				if _, err := buf.WriteString(fmt.Sprintf(" /*T![auto_rand] %s */", c.defaultValue.toString())); err != nil {
					return err
				}
			} else {
				if _, err := buf.WriteString(fmt.Sprintf(" DEFAULT %s", c.defaultValue.toString())); err != nil {
					return err
				}
			}
		}
	}
	if len(c.onUpdate) > 0 {
		if _, err := buf.WriteString(fmt.Sprintf(" ON UPDATE %s", c.onUpdate)); err != nil {
			return err
		}
	}
	if c.comment != "" {
		if _, err := buf.WriteString(fmt.Sprintf(" COMMENT '%s'", c.comment)); err != nil {
			return err
		}
	}
	return nil
}

func (c *columnState) convertToColumnMetadata() *storepb.ColumnMetadata {
	result := &storepb.ColumnMetadata{
		Name:        c.name,
		Type:        c.tp,
		Nullable:    c.nullable,
		OnUpdate:    c.onUpdate,
		Comment:     c.comment,
		UserComment: c.comment,
	}
	if c.defaultValue != nil {
		switch value := c.defaultValue.(type) {
		case *defaultValueNull:
			result.DefaultValue = &storepb.ColumnMetadata_DefaultNull{DefaultNull: true}
		case *defaultValueString:
			result.DefaultValue = &storepb.ColumnMetadata_Default{Default: wrapperspb.String(value.value)}
		case *defaultValueExpression:
			result.DefaultValue = &storepb.ColumnMetadata_DefaultExpression{DefaultExpression: value.value}
		}
	}
	if result.DefaultValue == nil && c.nullable {
		result.DefaultValue = &storepb.ColumnMetadata_DefaultNull{DefaultNull: true}
	}
	return result
}

func convertToColumnState(id int, column *storepb.ColumnMetadata) *columnState {
	result := &columnState{
		id:       id,
		name:     column.Name,
		tp:       column.Type,
		nullable: column.Nullable,
		onUpdate: normalizeOnUpdate(column.OnUpdate),
		comment:  column.Comment,
	}
	if column.GetDefaultValue() != nil {
		switch value := column.GetDefaultValue().(type) {
		case *storepb.ColumnMetadata_DefaultNull:
			result.defaultValue = &defaultValueNull{}
		case *storepb.ColumnMetadata_Default:
			if value.Default == nil {
				result.defaultValue = &defaultValueNull{}
			} else {
				result.defaultValue = &defaultValueString{value: value.Default.GetValue()}
			}
		case *storepb.ColumnMetadata_DefaultExpression:
			result.defaultValue = &defaultValueExpression{value: value.DefaultExpression}
		}
	}
	return result
}

// splitPartitioNExprIntoFields splits the partition expression by ',', and trims the leading and trailing '`' for each element.
func splitPartitionExprIntoFields(expr string) []string {
	// We do not support the expression contains parentheses, so we can split the expression by ','.
	ss := strings.Split(expr, ",")
	for i, s := range ss {
		if strings.HasPrefix(s, "`") && strings.HasSuffix(s, "`") {
			ss[i] = s[1 : len(s)-1]
		}
	}
	return ss
}
