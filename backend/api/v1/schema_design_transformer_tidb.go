package v1

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	tidbast "github.com/pingcap/tidb/parser/ast"
	tidbformat "github.com/pingcap/tidb/parser/format"
	tidbmysql "github.com/pingcap/tidb/parser/mysql"
	tidbtypes "github.com/pingcap/tidb/parser/types"

	tidbparser "github.com/bytebase/bytebase/backend/plugin/parser/tidb"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func checkTiDBColumnType(tp string) bool {
	_, err := tidbparser.ParseTiDB(fmt.Sprintf("CREATE TABLE t (a %s NOT NULL)", tp), "", "")
	return err == nil
}

type tidbTransformer struct {
	tidbast.StmtNode

	state *databaseState
	err   error
}

func parseTiDBSchemaStringToDatabaseMetadata(schema string) (*v1pb.DatabaseMetadata, error) {
	stmts, err := tidbparser.ParseTiDB(schema, "", "")
	if err != nil {
		return nil, err
	}

	transformer := &tidbTransformer{
		state: newDatabaseState(),
	}
	transformer.state.schemas[""] = newSchemaState()

	for _, stmt := range stmts {
		(stmt).Accept(transformer)
	}
	return transformer.state.convertToDatabaseMetadata(), transformer.err
}

func (t *tidbTransformer) Enter(in tidbast.Node) (tidbast.Node, bool) {
	if node, ok := in.(*tidbast.CreateTableStmt); ok {
		dbInfo := node.Table.DBInfo
		databaseName := ""
		if dbInfo != nil {
			databaseName = dbInfo.Name.String()
		}
		if databaseName != "" {
			if t.state.name == "" {
				t.state.name = databaseName
			} else if t.state.name != databaseName {
				t.err = errors.New("multiple database names found: " + t.state.name + ", " + databaseName)
				return in, true
			}
		}

		tableName := node.Table.Name.String()
		schema := t.state.schemas[""]
		if _, ok := schema.tables[tableName]; ok {
			t.err = errors.New("multiple table names found: " + tableName)
			return in, true
		}
		schema.tables[tableName] = newTableState(len(schema.tables), tableName)

		table := t.state.schemas[""].tables[tableName]

		// column definition
		for _, column := range node.Cols {
			dataType := columnTypeStr(column.Tp)
			columnName := column.Name.Name.String()
			if _, ok := table.columns[columnName]; ok {
				t.err = errors.New("multiple column names found: " + columnName + " in table " + tableName)
				return in, true
			}
			defaultValue, err := columnDefaultValue(column)
			if err != nil {
				t.err = err
				return in, true
			}
			comment, err := columnComment(column)
			if err != nil {
				t.err = err
				return in, true
			}
			columnState := &columnState{
				id:       len(table.columns),
				name:     columnName,
				tp:       dataType,
				comment:  comment,
				nullable: tidbColumnCanNull(column),
			}

			if defaultValue == nil {
				columnState.hasDefault = false
			} else {
				columnState.hasDefault = true
				switch {
				case strings.EqualFold(*defaultValue, "NULL"):
					columnState.defaultValue = &defaultValueNull{}
				case strings.HasPrefix(*defaultValue, "'") && strings.HasSuffix(*defaultValue, "'"):
					columnState.defaultValue = &defaultValueString{value: strings.ReplaceAll((*defaultValue)[1:len(*defaultValue)-1], "''", "'")}
				default:
					columnState.defaultValue = &defaultValueExpression{value: *defaultValue}
				}
			}

			table.columns[columnName] = columnState
		}
		for _, tableOption := range node.Options {
			if tableOption.Tp == tidbast.TableOptionComment {
				table.comment = tableComment(tableOption)
			}
		}

		// primary and foreign key definition
		for _, constraint := range node.Constraints {
			switch constraint.Tp {
			case tidbast.ConstraintPrimaryKey:
				var pkList []string
				for _, constraint := range node.Constraints {
					if constraint.Tp == tidbast.ConstraintPrimaryKey {
						var pks []string
						for _, key := range constraint.Keys {
							columnName := key.Column.Name.String()
							pks = append(pks, columnName)
						}
						pkList = append(pkList, pks...)
					}
				}

				table.indexes["PRIMARY"] = &indexState{
					id:      len(table.indexes),
					name:    "PRIMARY",
					keys:    pkList,
					primary: true,
					unique:  true,
				}
			case tidbast.ConstraintForeignKey:
				var referencingColumnList []string
				for _, key := range constraint.Keys {
					referencingColumnList = append(referencingColumnList, key.Column.Name.String())
				}
				var referencedColumnList []string
				for _, spec := range constraint.Refer.IndexPartSpecifications {
					referencedColumnList = append(referencedColumnList, spec.Column.Name.String())
				}

				fkName := constraint.Name
				if fkName == "" {
					t.err = errors.New("empty foreign key name")
					return in, true
				}
				if table.foreignKeys[fkName] != nil {
					t.err = errors.New("multiple foreign keys found: " + fkName)
					return in, true
				}

				fk := &foreignKeyState{
					id:                len(table.foreignKeys),
					name:              fkName,
					columns:           referencingColumnList,
					referencedTable:   constraint.Refer.Table.Name.String(),
					referencedColumns: referencedColumnList,
				}
				table.foreignKeys[fkName] = fk
			}
		}
	}
	return in, false
}

// columnTypeStr returns the type string of tp.
func columnTypeStr(tp *tidbtypes.FieldType) string {
	switch tp.GetType() {
	// https://pkg.go.dev/github.com/pingcap/tidb/parser/mysql#TypeLong
	case tidbmysql.TypeLong:
		// tp.String() return int(11)
		return "int"
		// https://pkg.go.dev/github.com/pingcap/tidb/parser/mysql#TypeLonglong
	case tidbmysql.TypeLonglong:
		// tp.String() return bigint(20)
		return "bigint"
	default:
		str := tp.String()
		if strings.Contains(str, "binary") {
			tp.SetFlag(tidbmysql.BinaryFlag)
			tp.SetCharset("binary")
			tp.SetCollate("binary")
			return tp.CompactStr()
		}
		return tp.String()
	}
}

func tidbColumnCanNull(column *tidbast.ColumnDef) bool {
	for _, option := range column.Options {
		if option.Tp == tidbast.ColumnOptionNotNull || option.Tp == tidbast.ColumnOptionPrimaryKey {
			return false
		}
	}
	return true
}

func columnDefaultValue(column *tidbast.ColumnDef) (*string, error) {
	for _, option := range column.Options {
		if option.Tp == tidbast.ColumnOptionDefaultValue {
			defaultValue, err := tidbRestoreNode(option.Expr, tidbformat.RestoreStringSingleQuotes|tidbformat.RestoreStringWithoutCharset)
			if err != nil {
				return nil, err
			}
			return &defaultValue, nil
		}
	}
	// no default value.
	return nil, nil
}

func tableComment(option *tidbast.TableOption) string {
	return option.StrValue
}

func columnComment(column *tidbast.ColumnDef) (string, error) {
	for _, option := range column.Options {
		if option.Tp == tidbast.ColumnOptionComment {
			comment, err := tidbRestoreNode(option.Expr, tidbformat.RestoreStringWithoutCharset)
			if err != nil {
				return "", err
			}
			return comment, nil
		}
	}

	return "", nil
}

func tidbRestoreNode(node tidbast.Node, flag tidbformat.RestoreFlags) (string, error) {
	var buffer strings.Builder
	ctx := tidbformat.NewRestoreCtx(flag, &buffer)
	if err := node.Restore(ctx); err != nil {
		return "", err
	}
	return buffer.String(), nil
}

func tidbRestoreNodeDefault(node tidbast.Node) (string, error) {
	return tidbRestoreNode(node, tidbformat.DefaultRestoreFlags)
}

func tidbRestoreFieldType(fieldType *tidbtypes.FieldType) (string, error) {
	if strings.Contains(fieldType.String(), "binary") {
		return fieldType.CompactStr(), nil
	}
	var buffer strings.Builder
	flag := tidbformat.RestoreKeyWordLowercase | tidbformat.RestoreStringSingleQuotes
	ctx := tidbformat.NewRestoreCtx(flag, &buffer)
	if err := fieldType.Restore(ctx); err != nil {
		return "", err
	}
	return buffer.String(), nil
}

func tidbRestoreTableOption(tableOption *tidbast.TableOption) (string, error) {
	var buffer strings.Builder
	flag := tidbformat.DefaultRestoreFlags
	ctx := tidbformat.NewRestoreCtx(flag, &buffer)
	if err := tableOption.Restore(ctx); err != nil {
		return "", err
	}
	return buffer.String(), nil
}

func (*tidbTransformer) Leave(in tidbast.Node) (tidbast.Node, bool) {
	return in, true
}

func getTiDBDesignSchema(baselineSchema string, to *v1pb.DatabaseMetadata) (string, error) {
	toState := convertToDatabaseState(to)
	stmts, err := tidbparser.ParseTiDB(baselineSchema, "", "")
	if err != nil {
		return "", err
	}

	generator := &tidbDesignSchemaGenerator{
		to: toState,
	}

	for _, stmt := range stmts {
		switch stmt.(type) {
		case *tidbast.CreateTableStmt:
			stmt.Accept(generator)
		default:
			if _, err := generator.result.WriteString(stmt.OriginalText() + "\n"); err != nil {
				return "", err
			}
		}
	}
	if generator.err != nil {
		return "", generator.err
	}

	firstTable := true
	for _, schema := range to.Schemas {
		schemaState, ok := toState.schemas[schema.Name]
		if !ok {
			continue
		}
		for _, table := range schema.Tables {
			table, ok := schemaState.tables[table.Name]
			if !ok {
				continue
			}
			if firstTable {
				firstTable = false
				if _, err := generator.result.WriteString("\n"); err != nil {
					return "", err
				}
			}
			if err := table.toString(&generator.result); err != nil {
				return "", err
			}
		}
	}

	return generator.result.String(), nil
}

type tidbDesignSchemaGenerator struct {
	tidbast.Node

	to                  *databaseState
	result              strings.Builder
	currentTable        *tableState
	firstElementInTable bool
	columnDefine        strings.Builder
	tableConstraints    strings.Builder
	err                 error
}

func (g *tidbDesignSchemaGenerator) Enter(in tidbast.Node) (tidbast.Node, bool) {
	if g.err != nil {
		return in, true
	}

	if node, ok := in.(*tidbast.CreateTableStmt); ok {
		dbInfo := node.Table.DBInfo
		databaseName := ""
		if dbInfo != nil {
			databaseName = dbInfo.Name.String()
		}
		if databaseName != "" && g.to.name != "" && databaseName != g.to.name {
			g.err = errors.New("multiple database names found: " + g.to.name + ", " + databaseName)
			return in, true
		}

		schema, ok := g.to.schemas[""]
		if !ok || schema == nil {
			return in, true
		}

		tableName := node.Table.Name.String()
		table, ok := schema.tables[tableName]
		if !ok {
			return in, true
		}
		g.currentTable = table
		g.firstElementInTable = true
		g.columnDefine.Reset()
		g.tableConstraints.Reset()

		delete(schema.tables, tableName)

		// Start constructing sql.
		// Temporary keyword.
		var temporaryKeyword string
		switch node.TemporaryKeyword {
		case tidbast.TemporaryNone:
			temporaryKeyword = "CREATE TABLE "
		case tidbast.TemporaryGlobal:
			temporaryKeyword = "CREATE GLOBAL TEMPORARY TABLE "
		case tidbast.TemporaryLocal:
			temporaryKeyword = "CREATE TEMPORARY TABLE "
		}
		if _, err := g.result.WriteString(temporaryKeyword); err != nil {
			g.err = err
			return in, true
		}

		// if not exists
		if node.IfNotExists {
			if _, err := g.result.WriteString("IF NOT EXISTS "); err != nil {
				g.err = err
				return in, true
			}
		}

		if tableNameStr, err := tidbRestoreNodeDefault(tidbast.Node(node.Table)); err == nil {
			if _, err := g.result.WriteString(tableNameStr + " "); err != nil {
				g.err = err
				return in, true
			}
		}

		if node.ReferTable != nil {
			if _, err := g.result.WriteString(" LIKE "); err != nil {
				g.err = err
				return in, true
			}
			if referTableStr, err := tidbRestoreNodeDefault(tidbast.Node(node.ReferTable)); err == nil {
				if _, err := g.result.WriteString(referTableStr + " "); err != nil {
					g.err = err
					return in, true
				}
			}
		}

		if _, err := g.result.WriteString("(\n  "); err != nil {
			g.err = err
			return in, true
		}

		// Column definition.
		for _, column := range node.Cols {
			columnName := column.Name.Name.String()
			stateColumn, ok := g.currentTable.columns[columnName]
			if !ok {
				continue
			}

			delete(g.currentTable.columns, columnName)

			if g.firstElementInTable {
				g.firstElementInTable = false
			} else {
				if _, err := g.columnDefine.WriteString(",\n  "); err != nil {
					g.err = err
					return in, true
				}
			}

			// Column name.
			if columnNameStr, err := tidbRestoreNodeDefault(tidbast.Node(column.Name)); err == nil {
				if _, err := g.columnDefine.WriteString(columnNameStr + " "); err != nil {
					g.err = err
					return in, true
				}
			}

			// Compare column types.
			dataType := columnTypeStr(column.Tp)
			if !strings.EqualFold(dataType, stateColumn.tp) {
				// write lower case column type for tidb
				column.Tp = tidbNewFieldType(stateColumn.tp)
			}
			if typeStr, err := tidbRestoreFieldType(column.Tp); err == nil {
				if _, err := g.columnDefine.WriteString(typeStr); err != nil {
					g.err = err
					return in, true
				}
			}

			// Column attributes.
			newAttr := tidbExtractNewAttrs(stateColumn, column.Options)
			for _, option := range column.Options {
				attrOrder := tidbGetAttrOrder(option)
				for ; len(newAttr) > 0 && newAttr[0].order < attrOrder; newAttr = newAttr[1:] {
					if _, err := g.columnDefine.WriteString(" " + newAttr[0].text); err != nil {
						g.err = err
						return in, true
					}
				}

				switch option.Tp {
				case tidbast.ColumnOptionNull, tidbast.ColumnOptionNotNull:
					sameNullable := option.Tp == tidbast.ColumnOptionNull && stateColumn.nullable
					sameNullable = sameNullable || (option.Tp == tidbast.ColumnOptionNotNull && !stateColumn.nullable)

					if sameNullable {
						if optionStr, err := tidbRestoreNodeDefault(tidbast.Node(option)); err == nil {
							if _, err := g.columnDefine.WriteString(" " + optionStr); err != nil {
								g.err = err
								return in, true
							}
						}
					} else {
						if stateColumn.nullable {
							if _, err := g.columnDefine.WriteString(" NULL"); err != nil {
								g.err = err
								return in, true
							}
						} else {
							if _, err := g.columnDefine.WriteString(" NOT NULL"); err != nil {
								g.err = err
								return in, true
							}
						}
					}
				case tidbast.ColumnOptionDefaultValue:
					defaultValueText, err := columnDefaultValue(column)
					if err != nil {
						g.err = err
						return in, true
					}
					var defaultValue defaultValue
					switch {
					case strings.EqualFold(*defaultValueText, "NULL"):
						defaultValue = &defaultValueNull{}
					case strings.HasPrefix(*defaultValueText, "'") && strings.HasSuffix(*defaultValueText, "'"):
						defaultValue = &defaultValueString{value: strings.ReplaceAll((*defaultValueText)[1:len(*defaultValueText)-1], "''", "'")}
					default:
						defaultValue = &defaultValueExpression{value: *defaultValueText}
					}
					if stateColumn.hasDefault && stateColumn.defaultValue.toString() == defaultValue.toString() {
						if defaultStr, err := tidbRestoreNodeDefault(option); err == nil {
							if _, err := g.columnDefine.WriteString(" " + defaultStr); err != nil {
								g.err = err
								return in, true
							}
						}
					} else if stateColumn.hasDefault {
						if !strings.EqualFold(stateColumn.defaultValue.toString(), "AUTO_INCREMENT") {
							if _, err := g.columnDefine.WriteString(" DEFAUL"); err != nil {
								g.err = err
								return in, true
							}
						}
						if _, err := g.columnDefine.WriteString(" " + stateColumn.defaultValue.toString()); err != nil {
							g.err = err
							return in, true
						}
					}
				case tidbast.ColumnOptionComment:
					commentValue, err := columnComment(column)
					if err != nil {
						g.err = err
						return in, true
					}
					if stateColumn.comment == commentValue {
						if commentStr, err := tidbRestoreNodeDefault(option); err == nil {
							if _, err := g.columnDefine.WriteString(" " + commentStr); err != nil {
								g.err = err
								return in, true
							}
						}
					} else if stateColumn.comment != "" {
						if _, err := g.columnDefine.WriteString(" COMMENT"); err != nil {
							g.err = err
							return in, true
						}
						if _, err := g.columnDefine.WriteString(fmt.Sprintf(" '%s'", stateColumn.comment)); err != nil {
							g.err = err
							return in, true
						}
					}
				default:
					if optionStr, err := tidbRestoreNodeDefault(option); err == nil {
						if _, err := g.columnDefine.WriteString(" " + optionStr); err != nil {
							g.err = err
							return in, true
						}
					}
				}
			}

			for _, attr := range newAttr {
				if _, err := g.columnDefine.WriteString(" " + attr.text); err != nil {
					g.err = err
					return in, true
				}
			}
		}

		// Table Constraint.
		for _, constraint := range node.Constraints {
			switch constraint.Tp {
			case tidbast.ConstraintPrimaryKey:
				if g.currentTable.indexes["PRIMARY"] != nil {
					if g.firstElementInTable {
						g.firstElementInTable = false
					} else {
						if _, err := g.tableConstraints.WriteString(",\n  "); err != nil {
							g.err = err
							return in, true
						}
					}
					var keys []string
					for _, key := range constraint.Keys {
						keys = append(keys, key.Column.Name.String())
					}
					if equalKeys(keys, g.currentTable.indexes["PRIMARY"].keys) {
						if constraintStr, err := tidbRestoreNodeDefault(constraint); err == nil {
							if _, err := g.tableConstraints.WriteString(constraintStr); err != nil {
								g.err = err
								return in, true
							}
						}
					} else {
						if err := g.currentTable.indexes["PRIMARY"].toString(&g.tableConstraints); err != nil {
							g.err = err
							return in, true
						}
					}
					delete(g.currentTable.indexes, "PRIMARY")
				}
			case tidbast.ConstraintForeignKey:
				fkName := constraint.Name
				if fkName == "" {
					g.err = errors.New("empty foreign key name")
					return in, true
				}
				if g.currentTable.foreignKeys[fkName] != nil {
					if g.firstElementInTable {
						g.firstElementInTable = false
					} else {
						if _, err := g.tableConstraints.WriteString(",\n  "); err != nil {
							g.err = err
							return in, true
						}
					}

					fk := g.currentTable.foreignKeys[fkName]

					var columns []string
					for _, key := range constraint.Keys {
						columns = append(columns, key.Column.Name.String())
					}

					var referencedColumnList []string
					for _, spec := range constraint.Refer.IndexPartSpecifications {
						referencedColumnList = append(referencedColumnList, spec.Column.Name.String())
					}
					referencedTable := constraint.Refer.Table.Name.String()
					if equalKeys(columns, fk.columns) && referencedTable == fk.referencedTable && equalKeys(referencedColumnList, fk.referencedColumns) {
						if constraintStr, err := tidbRestoreNodeDefault(constraint); err == nil {
							if _, err := g.tableConstraints.WriteString(constraintStr); err != nil {
								g.err = err
								return in, true
							}
						}
					} else {
						if err := fk.toString(&g.tableConstraints); err != nil {
							g.err = err
							return in, true
						}
					}
					delete(g.currentTable.foreignKeys, fkName)
				}
			default:
				if g.firstElementInTable {
					g.firstElementInTable = false
				} else {
					if _, err := g.tableConstraints.WriteString(",\n  "); err != nil {
						g.err = err
						return in, true
					}
				}
				if constraintStr, err := tidbRestoreNodeDefault(constraint); err == nil {
					if _, err := g.tableConstraints.WriteString(constraintStr); err != nil {
						g.err = err
						return in, true
					}
				}
			}
		}
	}

	return in, false
}

func (g *tidbDesignSchemaGenerator) Leave(in tidbast.Node) (tidbast.Node, bool) {
	if g.err != nil || g.currentTable == nil {
		return in, true
	}
	if node, ok := in.(*tidbast.CreateTableStmt); ok {
		// Column definition.
		var columnList []*columnState
		for _, column := range g.currentTable.columns {
			columnList = append(columnList, column)
		}
		sort.Slice(columnList, func(i, j int) bool {
			return columnList[i].id < columnList[j].id
		})
		for _, column := range columnList {
			if g.firstElementInTable {
				g.firstElementInTable = false
			} else {
				if _, err := g.columnDefine.WriteString(",\n  "); err != nil {
					g.err = err
					return in, true
				}
			}
			if err := column.toString(&g.columnDefine); err != nil {
				g.err = err
				return in, true
			}
		}

		// Primary key definition.
		if g.currentTable.indexes["PRIMARY"] != nil {
			if g.firstElementInTable {
				g.firstElementInTable = false
			} else {
				if _, err := g.columnDefine.WriteString(",\n  "); err != nil {
					g.err = err
					return in, true
				}
			}
			if err := g.currentTable.indexes["PRIMARY"].toString(&g.tableConstraints); err != nil {
				return in, true
			}
		}

		// Foreign key definition.
		var fks []*foreignKeyState
		for _, fk := range g.currentTable.foreignKeys {
			fks = append(fks, fk)
		}
		sort.Slice(fks, func(i, j int) bool {
			return fks[i].id < fks[j].id
		})
		for _, fk := range fks {
			if g.firstElementInTable {
				g.firstElementInTable = false
			} else {
				if _, err := g.columnDefine.WriteString(",\n "); err != nil {
					g.err = err
					return in, true
				}
			}
			if err := fk.toString(&g.tableConstraints); err != nil {
				return in, true
			}
		}

		if _, err := g.result.WriteString(g.columnDefine.String()); err != nil {
			g.err = err
			return in, true
		}
		if _, err := g.result.WriteString(g.tableConstraints.String()); err != nil {
			g.err = err
			return in, true
		}
		if _, err := g.result.WriteString("\n)"); err != nil {
			g.err = err
			return in, true
		}

		// Table option.
		hasTableComment := false
		for _, option := range node.Options {
			if option.Tp == tidbast.TableOptionComment {
				commentValue := tableComment(option)
				if g.currentTable.comment == commentValue && g.currentTable.comment != "" {
					if _, err := g.result.WriteString(" COMMENT"); err != nil {
						g.err = err
						return in, true
					}
					if _, err := g.result.WriteString(fmt.Sprintf(" '%s'", g.currentTable.comment)); err != nil {
						g.err = err
						return in, true
					}
				}
				hasTableComment = true
			} else {
				if optionStr, err := tidbRestoreTableOption(option); err == nil {
					if _, err := g.result.WriteString(" " + optionStr); err != nil {
						g.err = err
						return in, true
					}
				}
			}
		}
		if !hasTableComment && g.currentTable.comment != "" {
			if _, err := g.result.WriteString(" COMMENT"); err != nil {
				g.err = err
				return in, true
			}
			if _, err := g.result.WriteString(fmt.Sprintf(" '%s'", g.currentTable.comment)); err != nil {
				g.err = err
				return in, true
			}
		}

		// Table partition.
		if node.Partition != nil {
			if partitionStr, err := tidbRestoreNodeDefault(node.Partition); err == nil {
				if _, err := g.result.WriteString(" " + partitionStr); err != nil {
					g.err = err
					return in, true
				}
			}
		}

		// Table select.
		if node.Select != nil {
			duplicateStr := ""
			switch node.OnDuplicate {
			case tidbast.OnDuplicateKeyHandlingError:
				duplicateStr = " AS "
			case tidbast.OnDuplicateKeyHandlingIgnore:
				duplicateStr = " IGNORE AS "
			case tidbast.OnDuplicateKeyHandlingReplace:
				duplicateStr = " REPLACE AS "
			}

			if selectStr, err := tidbRestoreNodeDefault(node.Select); err == nil {
				if _, err := g.result.WriteString(duplicateStr + selectStr); err != nil {
					g.err = err
					return in, true
				}
			}
		}

		if node.TemporaryKeyword == tidbast.TemporaryGlobal {
			if node.OnCommitDelete {
				if _, err := g.result.WriteString(" ON COMMIT DELETE ROWS"); err != nil {
					g.err = err
					return in, true
				}
			} else {
				if _, err := g.result.WriteString(" ON COMMIT PRESERVE ROWS"); err != nil {
					g.err = err
					return in, true
				}
			}
		}
		if _, err := g.result.WriteString(";\n"); err != nil {
			g.err = err
			return in, true
		}

		g.currentTable = nil
		g.firstElementInTable = false
	}
	return in, true
}

func tidbExtractNewAttrs(column *columnState, options []*tidbast.ColumnOption) []columnAttr {
	var result []columnAttr
	nullExists := false
	defaultExists := false
	commentExists := false

	for _, option := range options {
		switch option.Tp {
		case tidbast.ColumnOptionNull, tidbast.ColumnOptionNotNull:
			nullExists = true
		case tidbast.ColumnOptionDefaultValue:
			defaultExists = true
		case tidbast.ColumnOptionComment:
			commentExists = true
		}
	}

	if !nullExists && !column.nullable {
		result = append(result, columnAttr{
			text:  "NOT NULL",
			order: columnAttrOrder["NULL"],
		})
	}
	if !defaultExists && column.hasDefault {
		result = append(result, columnAttr{
			text:  "DEFAULT " + column.defaultValue.toString(),
			order: columnAttrOrder["DEFAULT"],
		})
	}
	if !commentExists && column.comment != "" {
		result = append(result, columnAttr{
			text:  "COMMENT '" + column.comment + "'",
			order: columnAttrOrder["COMMENT"],
		})
	}
	return result
}

func tidbGetAttrOrder(option *tidbast.ColumnOption) int {
	switch option.Tp {
	case tidbast.ColumnOptionDefaultValue:
		return columnAttrOrder["DEFAULT"]
	case tidbast.ColumnOptionNull, tidbast.ColumnOptionNotNull:
		return columnAttrOrder["NULL"]
	case tidbast.ColumnOptionUniqKey:
		return columnAttrOrder["UNIQUE"]
	case tidbast.ColumnOptionColumnFormat:
		return columnAttrOrder["COLUMN_FORMAT"]
	case tidbast.ColumnOptionAutoIncrement:
		return columnAttrOrder["AUTO_INCREMENT"]
	case tidbast.ColumnOptionComment:
		return columnAttrOrder["COMMENT"]
	case tidbast.ColumnOptionCollate:
		return columnAttrOrder["COLLATE"]
	case tidbast.ColumnOptionStorage:
		return columnAttrOrder["STORAGE"]
	case tidbast.ColumnOptionCheck:
		return columnAttrOrder["CHECK"]
	}
	if option.Enforced {
		return columnAttrOrder["ENFORCED"]
	}
	return len(columnAttrOrder) + 1
}

func tidbNewFieldType(tp string) *tidbtypes.FieldType {
	tpStr := strings.ToLower(tp)
	var s []byte
	var flen []byte
	var decimal []byte
	stage := 1
	for i := 0; i < len(tpStr); i++ {
		if tpStr[i] == '(' {
			stage = 2
			continue
		} else if tpStr[i] == ',' {
			stage = 3
			continue
		} else if tpStr[i] == ')' {
			continue
		}

		if stage == 1 {
			s = append(s, tpStr[i])
		} else if stage == 2 {
			flen = append(flen, tpStr[i])
		} else if stage == 3 {
			decimal = append(decimal, tpStr[i])
		}
	}
	ft := tidbtypes.NewFieldType(tidbtypes.StrToType(string(s)))
	flenInt, _ := strconv.Atoi(string(flen))
	if flenInt > 0 {
		ft.SetFlen(flenInt)
	}
	decimalInt, _ := strconv.Atoi(string(decimal))
	if decimalInt > 0 {
		ft.SetDecimal(decimalInt)
	}
	if strings.Contains(tpStr, "binary") {
		ft.SetFlag(tidbmysql.BinaryFlag)
		ft.SetCharset("binary")
		ft.SetCollate("binary")
	}
	return ft
}
