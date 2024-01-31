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
			optionMap := make(map[tidbast.ColumnOptionType]*tidbast.ColumnOption)
			for _, option := range column.Options {
				optionMap[option.Tp] = option
			}

			// NULL and NOT NULL are mutually exclusive.
			oldNullable := true
			if _, exists := optionMap[tidbast.ColumnOptionNotNull]; exists {
				oldNullable = false
			}
			if oldNullable != stateColumn.nullable {
				if stateColumn.nullable {
					g.actions = append(g.actions, tidbparser.NewDropColumnOptionAction(tableName, columnName, tidbast.ColumnOptionNotNull))
					g.actions = append(g.actions, tidbparser.NewAddColumnOptionAction(tableName, columnName, tidbast.ColumnOptionNull, "NULL"))
				} else {
					g.actions = append(g.actions, tidbparser.NewDropColumnOptionAction(tableName, columnName, tidbast.ColumnOptionNull))
					g.actions = append(g.actions, tidbparser.NewAddColumnOptionAction(tableName, columnName, tidbast.ColumnOptionNotNull, "NOT NULL"))
				}
			}

			// Default value, auto increment and auto random are mutually exclusive.
			if option, exists := optionMap[tidbast.ColumnOptionDefaultValue]; exists {
				if stateColumn.defaultValue != nil {
					switch {
					case stateColumn.hasAutoIncrement():
						g.actions = append(g.actions, tidbparser.NewDropColumnOptionAction(tableName, columnName, tidbast.ColumnOptionDefaultValue))
						g.actions = append(g.actions, tidbparser.NewAddColumnOptionAction(tableName, columnName, tidbast.ColumnOptionAutoIncrement, stateColumn.defaultValue.toString()))
					case stateColumn.hasAutoRand():
						g.actions = append(g.actions, tidbparser.NewDropColumnOptionAction(tableName, columnName, tidbast.ColumnOptionDefaultValue))
						g.actions = append(g.actions, tidbparser.NewAddColumnOptionAction(tableName, columnName, tidbast.ColumnOptionAutoRandom, fmt.Sprintf("/*T![auto_rand] %s */", stateColumn.defaultValue.toString())))
					default:
						expr, err := restoreExpr(option.Expr)
						if err != nil {
							g.err = err
							return in, true
						}
						if *expr != stateColumn.defaultValue.toString() {
							g.actions = append(g.actions, tidbparser.NewModifyColumnOptionAction(tableName, columnName, tidbast.ColumnOptionDefaultValue, fmt.Sprintf("DEFAULT %s", stateColumn.defaultValue.toString())))
						}
					}
				} else {
					g.actions = append(g.actions, tidbparser.NewDropColumnOptionAction(tableName, columnName, tidbast.ColumnOptionDefaultValue))
				}
			} else if _, exists := optionMap[tidbast.ColumnOptionAutoIncrement]; exists {
				if stateColumn.defaultValue != nil {
					switch {
					case stateColumn.hasAutoIncrement():
					// Do nothing.
					case stateColumn.hasAutoRand():
						g.actions = append(g.actions, tidbparser.NewDropColumnOptionAction(tableName, columnName, tidbast.ColumnOptionAutoIncrement))
						g.actions = append(g.actions, tidbparser.NewAddColumnOptionAction(tableName, columnName, tidbast.ColumnOptionAutoRandom, fmt.Sprintf("/*T![auto_rand] %s */", stateColumn.defaultValue.toString())))
					default:
						g.actions = append(g.actions, tidbparser.NewDropColumnOptionAction(tableName, columnName, tidbast.ColumnOptionAutoIncrement))
						g.actions = append(g.actions, tidbparser.NewAddColumnOptionAction(tableName, columnName, tidbast.ColumnOptionDefaultValue, fmt.Sprintf("DEFAULT %s", stateColumn.defaultValue.toString())))
					}
				} else {
					g.actions = append(g.actions, tidbparser.NewDropColumnOptionAction(tableName, columnName, tidbast.ColumnOptionAutoIncrement))
				}
			} else if _, exists := optionMap[tidbast.ColumnOptionAutoRandom]; exists {
				if stateColumn.defaultValue != nil {
					switch {
					case stateColumn.hasAutoIncrement():
						g.actions = append(g.actions, tidbparser.NewDropColumnOptionAction(tableName, columnName, tidbast.ColumnOptionAutoRandom))
						g.actions = append(g.actions, tidbparser.NewAddColumnOptionAction(tableName, columnName, tidbast.ColumnOptionAutoIncrement, stateColumn.defaultValue.toString()))
					case stateColumn.hasAutoRand():
					// Do nothing.
					default:
						g.actions = append(g.actions, tidbparser.NewDropColumnOptionAction(tableName, columnName, tidbast.ColumnOptionAutoRandom))
						g.actions = append(g.actions, tidbparser.NewAddColumnOptionAction(tableName, columnName, tidbast.ColumnOptionDefaultValue, fmt.Sprintf("DEFAULT %s", stateColumn.defaultValue.toString())))
					}
				} else {
					g.actions = append(g.actions, tidbparser.NewDropColumnOptionAction(tableName, columnName, tidbast.ColumnOptionAutoRandom))
				}
			} else {
				if stateColumn.defaultValue != nil {
					switch {
					case stateColumn.hasAutoIncrement():
						g.actions = append(g.actions, tidbparser.NewAddColumnOptionAction(tableName, columnName, tidbast.ColumnOptionAutoIncrement, stateColumn.defaultValue.toString()))
					case stateColumn.hasAutoRand():
						g.actions = append(g.actions, tidbparser.NewAddColumnOptionAction(tableName, columnName, tidbast.ColumnOptionAutoRandom, fmt.Sprintf("/*T![auto_rand] %s */", stateColumn.defaultValue.toString())))
					default:
						g.actions = append(g.actions, tidbparser.NewAddColumnOptionAction(tableName, columnName, tidbast.ColumnOptionDefaultValue, fmt.Sprintf("DEFAULT %s", stateColumn.defaultValue.toString())))
					}
				}
			}

			// On update.
			if option, exists := optionMap[tidbast.ColumnOptionOnUpdate]; exists {
				onUpdate, err := restoreExpr(option.Expr)
				if err != nil {
					g.err = err
					return in, true
				}
				if *onUpdate != stateColumn.onUpdate {
					if stateColumn.onUpdate != "" {
						g.actions = append(g.actions, tidbparser.NewModifyColumnOptionAction(tableName, columnName, tidbast.ColumnOptionOnUpdate, fmt.Sprintf("ON UPDATE %s", stateColumn.onUpdate)))
					} else {
						g.actions = append(g.actions, tidbparser.NewDropColumnOptionAction(tableName, columnName, tidbast.ColumnOptionOnUpdate))
					}
				}
			} else if stateColumn.onUpdate != "" {
				g.actions = append(g.actions, tidbparser.NewAddColumnOptionAction(tableName, columnName, tidbast.ColumnOptionOnUpdate, fmt.Sprintf("ON UPDATE %s", stateColumn.onUpdate)))
			}

			// Comment.
			if option, exists := optionMap[tidbast.ColumnOptionComment]; exists {
				comment, err := restoreComment(option.Expr)
				if err != nil {
					g.err = err
					return in, true
				}
				if comment != stateColumn.comment {
					if stateColumn.comment != "" {
						g.actions = append(g.actions, tidbparser.NewModifyColumnOptionAction(tableName, columnName, tidbast.ColumnOptionComment, fmt.Sprintf("COMMENT '%s'", stateColumn.comment)))
					} else {
						g.actions = append(g.actions, tidbparser.NewDropColumnOptionAction(tableName, columnName, tidbast.ColumnOptionComment))
					}
				}
			} else if stateColumn.comment != "" {
				g.actions = append(g.actions, tidbparser.NewAddColumnOptionAction(tableName, columnName, tidbast.ColumnOptionComment, fmt.Sprintf("COMMENT '%s'", stateColumn.comment)))
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
					if equalKeys(keys, nil /* length */, g.currentTable.indexes["PRIMARY"].keys, nil /* length */) {
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
					var length []int64
					for _, key := range constraint.Keys {
						var keyString string
						var err error
						if key.Column != nil {
							keyString = key.Column.Name.String()
							if key.Length > 0 {
								length = append(length, int64(key.Length))
							} else {
								length = append(length, -1)
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

					if equalKeys(columns, length, index.keys, index.length) && !index.unique && !index.primary {
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
					var length []int64
					for _, key := range constraint.Keys {
						var keyString string
						var err error
						if key.Column != nil {
							keyString = key.Column.Name.String()
							if key.Length > 0 {
								length = append(length, int64(key.Length))
							} else {
								length = append(length, -1)
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

					if equalKeys(columns, length, index.keys, index.length) && index.unique && !index.primary {
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
					if equalKeys(columns, nil /* length */, fk.columns, nil /* length */) && referencedTable == fk.referencedTable &&
						equalKeys(referencedColumnList, nil /* length */, fk.referencedColumns, nil /* length */) {
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
