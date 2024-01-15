package tidb

import (
	"fmt"
	"sort"
	"strings"

	tidbast "github.com/pingcap/tidb/pkg/parser/ast"
	tidbformat "github.com/pingcap/tidb/pkg/parser/format"
	"github.com/pkg/errors"

	tidbparser "github.com/bytebase/bytebase/backend/plugin/parser/tidb"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	schema.RegisterGetDesignSchema(storepb.Engine_TIDB, GetDesignSchema)
}

func GetDesignSchema(baselineSchema string, to *storepb.DatabaseSchemaMetadata) (string, error) {
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
			// Do nothing.
		}
	}
	if generator.err != nil {
		return "", generator.err
	}

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
			buf := strings.Builder{}
			if err := table.toString(&buf); err != nil {
				return "", err
			}
			generator.actions = append(generator.actions, tidbparser.NewAddTableAction(buf.String()))
		}
	}

	manipulator := tidbparser.NewStringsManipulator(baselineSchema)
	return manipulator.Manipulate(generator.actions...)
}

type tidbDesignSchemaGenerator struct {
	tidbast.Node

	to           *databaseState
	currentTable *tableState
	err          error

	actions []tidbparser.StringsManipulatorAction
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
			g.actions = append(g.actions, tidbparser.NewDropTableAction(tableName))
			return in, true
		}
		g.currentTable = table

		delete(schema.tables, tableName)
		// Column definition.
		for _, column := range node.Cols {
			columnName := column.Name.Name.String()
			stateColumn, ok := g.currentTable.columns[columnName]
			if !ok {
				g.actions = append(g.actions, tidbparser.NewDropColumnAction(tableName, columnName))
				continue
			}

			delete(g.currentTable.columns, columnName)

			// Compare column types.
			dataType := columnTypeStr(column.Tp)
			if !strings.EqualFold(dataType, stateColumn.tp) {
				g.actions = append(g.actions, tidbparser.NewModifyColumnTypeAction(tableName, columnName, stateColumn.tp))
			}

			// Column attributes.
			deleteSchemaAutoIncrement := false
			deleteAutoRand := false
			// Default value, auto increment and auto random are mutually exclusive.
			for _, option := range column.Options {
				switch option.Tp {
				case tidbast.ColumnOptionDefaultValue:
					deleteAutoRand = stateColumn.hasDefault
					deleteSchemaAutoIncrement = stateColumn.hasDefault
				case tidbast.ColumnOptionAutoIncrement:
					deleteSchemaAutoIncrement = stateColumn.hasDefault
				case tidbast.ColumnOptionAutoRandom:
					deleteAutoRand = stateColumn.hasDefault
				}
			}
			newAttr := tidbExtractNewAttrs(stateColumn, column.Options)
			for _, option := range column.Options {
				attrOrder := tidbGetAttrOrder(option)
				for ; len(newAttr) > 0 && newAttr[0].order < attrOrder; newAttr = newAttr[1:] {
					g.actions = append(g.actions, tidbparser.NewAddColumnOptionAction(tableName, columnName, newAttr[0].text))
				}

				switch option.Tp {
				case tidbast.ColumnOptionNull, tidbast.ColumnOptionNotNull:
					sameNullable := option.Tp == tidbast.ColumnOptionNull && stateColumn.nullable
					sameNullable = sameNullable || (option.Tp == tidbast.ColumnOptionNotNull && !stateColumn.nullable)

					if !sameNullable {
						if stateColumn.nullable {
							g.actions = append(g.actions, tidbparser.NewAddColumnOptionAction(tableName, columnName, "NULL"))
							if option.Tp == tidbast.ColumnOptionNotNull {
								g.actions = append(g.actions, tidbparser.NewDropColumnOptionAction(tableName, columnName, tidbast.ColumnOptionNotNull))
							}
						} else {
							g.actions = append(g.actions, tidbparser.NewAddColumnOptionAction(tableName, columnName, "NOT NULL"))
							if option.Tp == tidbast.ColumnOptionNull {
								g.actions = append(g.actions, tidbparser.NewDropColumnOptionAction(tableName, columnName, tidbast.ColumnOptionNull))
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
						continue
					}
					if stateColumn.hasDefault {
						if strings.EqualFold(stateColumn.defaultValue.toString(), autoIncrementSymbol) {
							g.actions = append(g.actions, tidbparser.NewDropColumnOptionAction(tableName, columnName, tidbast.ColumnOptionDefaultValue))
							g.actions = append(g.actions, tidbparser.NewAddColumnOptionAction(tableName, columnName, stateColumn.defaultValue.toString()))
						} else if strings.Contains(strings.ToUpper(stateColumn.defaultValue.toString()), autoRandSymbol) {
							g.actions = append(g.actions, tidbparser.NewDropColumnOptionAction(tableName, columnName, tidbast.ColumnOptionDefaultValue))
							g.actions = append(g.actions, tidbparser.NewAddColumnOptionAction(tableName, columnName, fmt.Sprintf("/*T![auto_rand] %s */", stateColumn.defaultValue.toString())))
						} else {
							g.actions = append(g.actions, tidbparser.NewModifyColumnOptionAction(tableName, columnName, tidbast.ColumnOptionDefaultValue, fmt.Sprintf("DEFAULT %s", stateColumn.defaultValue.toString())))
						}
					}
				case tidbast.ColumnOptionComment:
					commentValue, err := columnComment(column)
					if err != nil {
						g.err = err
						return in, true
					}
					if stateColumn.comment == commentValue {
						continue
					}
					if stateColumn.comment != "" {
						g.actions = append(g.actions, tidbparser.NewModifyColumnOptionAction(tableName, columnName, tidbast.ColumnOptionComment, fmt.Sprintf("COMMENT '%s'", stateColumn.comment)))
					} else {
						g.actions = append(g.actions, tidbparser.NewDropColumnOptionAction(tableName, columnName, tidbast.ColumnOptionComment))
					}
				case tidbast.ColumnOptionAutoIncrement:
					if deleteSchemaAutoIncrement {
						g.actions = append(g.actions, tidbparser.NewDropColumnOptionAction(tableName, columnName, tidbast.ColumnOptionAutoIncrement))
					}
				case tidbast.ColumnOptionAutoRandom:
					if deleteAutoRand {
						g.actions = append(g.actions, tidbparser.NewDropColumnOptionAction(tableName, columnName, tidbast.ColumnOptionAutoRandom))
					}
				}
			}

			for _, attr := range newAttr {
				g.actions = append(g.actions, tidbparser.NewAddColumnOptionAction(tableName, columnName, attr.text))
			}
		}

		// Table Constraint.
		for _, constraint := range node.Constraints {
			switch constraint.Tp {
			case tidbast.ConstraintPrimaryKey:
				if g.currentTable.indexes["PRIMARY"] != nil {
					var keys []string
					for _, key := range constraint.Keys {
						keys = append(keys, key.Column.Name.String())
					}
					if equalKeys(keys, g.currentTable.indexes["PRIMARY"].keys) {
						delete(g.currentTable.indexes, "PRIMARY")
						continue
					}
					buf := strings.Builder{}
					if err := g.currentTable.indexes["PRIMARY"].toString(&buf); err != nil {
						g.err = err
						return in, true
					}
					g.actions = append(g.actions, tidbparser.NewModifyTableConstraintAction(tableName, tidbast.ConstraintPrimaryKey, "PRIMARY", buf.String()))
					delete(g.currentTable.indexes, "PRIMARY")
				} else {
					g.actions = append(g.actions, tidbparser.NewDropTableConstraintAction(tableName, "PRIMARY"))
				}
			case tidbast.ConstraintKey, tidbast.ConstraintIndex:
				indexName := constraint.Name
				if indexName == "" {
					g.err = errors.New("empty index name")
					return in, true
				}
				if g.currentTable.indexes[indexName] != nil {
					index := g.currentTable.indexes[indexName]

					var columns []string
					for _, key := range constraint.Keys {
						var keyString string
						var err error
						if key.Column == nil {
							keyString = key.Column.Name.String()
							if key.Length > 0 {
								keyString = fmt.Sprintf("`%s`(%d)", keyString, key.Length)
							}
						} else {
							keyString, err = tidbRestoreNode(key, tidbformat.RestoreKeyWordLowercase|tidbformat.RestoreStringSingleQuotes|tidbformat.RestoreNameBackQuotes)
							if err != nil {
								g.err = err
								return in, true
							}
						}
						columns = append(columns, keyString)
					}

					if equalKeys(columns, index.keys) && !index.unique && !index.primary {
						delete(g.currentTable.indexes, indexName)
						continue
					}
					buf := strings.Builder{}
					if err := index.toString(&buf); err != nil {
						g.err = err
						return in, true
					}
					g.actions = append(g.actions, tidbparser.NewModifyTableConstraintAction(tableName, constraint.Tp, indexName, buf.String()))
					delete(g.currentTable.indexes, indexName)
				} else {
					g.actions = append(g.actions, tidbparser.NewDropTableConstraintAction(tableName, indexName))
				}
			case tidbast.ConstraintUniqKey, tidbast.ConstraintUniqIndex, tidbast.ConstraintUniq:
				indexName := constraint.Name
				if indexName == "" {
					g.err = errors.New("empty index name")
					return in, true
				}
				if g.currentTable.indexes[indexName] != nil {
					index := g.currentTable.indexes[indexName]

					var columns []string
					for _, key := range constraint.Keys {
						var keyString string
						var err error
						if key.Column == nil {
							keyString = key.Column.Name.String()
							if key.Length > 0 {
								keyString = fmt.Sprintf("`%s`(%d)", keyString, key.Length)
							}
						} else {
							keyString, err = tidbRestoreNode(key, tidbformat.RestoreKeyWordLowercase|tidbformat.RestoreStringSingleQuotes|tidbformat.RestoreNameBackQuotes)
							if err != nil {
								g.err = err
								return in, true
							}
						}
						columns = append(columns, keyString)
					}

					if equalKeys(columns, index.keys) && index.unique && !index.primary {
						delete(g.currentTable.indexes, indexName)
						continue
					}
					buf := strings.Builder{}
					if err := index.toString(&buf); err != nil {
						g.err = err
						return in, true
					}
					g.actions = append(g.actions, tidbparser.NewModifyTableConstraintAction(tableName, constraint.Tp, indexName, buf.String()))
					delete(g.currentTable.indexes, indexName)
				} else {
					g.actions = append(g.actions, tidbparser.NewDropTableConstraintAction(tableName, indexName))
				}
			case tidbast.ConstraintForeignKey:
				fkName := constraint.Name
				if fkName == "" {
					g.err = errors.New("empty foreign key name")
					return in, true
				}
				if g.currentTable.foreignKeys[fkName] != nil {
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
						delete(g.currentTable.foreignKeys, fkName)
						continue
					}
					buf := strings.Builder{}
					if err := fk.toString(&buf); err != nil {
						g.err = err
						return in, true
					}
					g.actions = append(g.actions, tidbparser.NewModifyTableConstraintAction(tableName, tidbast.ConstraintForeignKey, fkName, buf.String()))
					delete(g.currentTable.foreignKeys, fkName)
				} else {
					g.actions = append(g.actions, tidbparser.NewDropTableConstraintAction(tableName, fkName))
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
			buf := strings.Builder{}
			if err := column.toString(&buf); err != nil {
				g.err = err
				return in, true
			}
			g.actions = append(g.actions, tidbparser.NewAddColumnAction(node.Table.Name.String(), buf.String()))
		}

		// Primary key definition.
		if g.currentTable.indexes["PRIMARY"] != nil {
			buf := strings.Builder{}
			if err := g.currentTable.indexes["PRIMARY"].toString(&buf); err != nil {
				return in, true
			}
			g.actions = append(g.actions, tidbparser.NewAddTableConstraintAction(node.Table.Name.String(), tidbast.ConstraintPrimaryKey, buf.String()))
			delete(g.currentTable.indexes, "PRIMARY")
		}

		// Index definition.
		var indexes []*indexState
		for _, index := range g.currentTable.indexes {
			indexes = append(indexes, index)
		}
		sort.Slice(indexes, func(i, j int) bool {
			return indexes[i].id < indexes[j].id
		})
		for _, index := range indexes {
			buf := strings.Builder{}
			if err := index.toString(&buf); err != nil {
				return in, true
			}
			if index.unique {
				g.actions = append(g.actions, tidbparser.NewAddTableConstraintAction(node.Table.Name.String(), tidbast.ConstraintUniqKey, buf.String()))
			} else {
				g.actions = append(g.actions, tidbparser.NewAddTableConstraintAction(node.Table.Name.String(), tidbast.ConstraintKey, buf.String()))
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
			buf := strings.Builder{}
			if err := fk.toString(&buf); err != nil {
				return in, true
			}
			g.actions = append(g.actions, tidbparser.NewAddTableConstraintAction(node.Table.Name.String(), tidbast.ConstraintForeignKey, buf.String()))
		}

		// Table option.
		hasTableComment := false
		commentValue := ""
		for _, option := range node.Options {
			if option.Tp == tidbast.TableOptionComment {
				commentValue = tableComment(option)
				hasTableComment = true
			}
		}

		if hasTableComment && commentValue != g.currentTable.comment {
			if g.currentTable.comment != "" {
				g.actions = append(g.actions, tidbparser.NewModifyTableOptionAction(node.Table.Name.String(), tidbast.TableOptionComment, fmt.Sprintf("COMMENT '%s'", g.currentTable.comment)))
			} else {
				g.actions = append(g.actions, tidbparser.NewDropTableOptionAction(node.Table.Name.String(), tidbast.TableOptionComment))
			}
		}

		if !hasTableComment && g.currentTable.comment != "" {
			g.actions = append(g.actions, tidbparser.NewAddTableOptionAction(node.Table.Name.String(), fmt.Sprintf("COMMENT '%s'", g.currentTable.comment)))
		}

		g.currentTable = nil
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
		// todo(zp): refactor column attribute.
		if strings.EqualFold(column.defaultValue.toString(), autoIncrementSymbol) {
			result = append(result, columnAttr{
				text:  column.defaultValue.toString(),
				order: columnAttrOrder["DEFAULT"],
			})
		} else if strings.Contains(strings.ToUpper(column.defaultValue.toString()), autoRandSymbol) {
			result = append(result, columnAttr{
				text:  fmt.Sprintf("/*T![auto_rand] %s */", column.defaultValue.toString()),
				order: columnAttrOrder["DEFAULT"],
			})
		} else {
			result = append(result, columnAttr{
				text:  "DEFAULT " + column.defaultValue.toString(),
				order: columnAttrOrder["DEFAULT"],
			})
		}
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
	case tidbast.ColumnOptionAutoRandom:
		return columnAttrOrder["AUTO_RANDOM"]
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
