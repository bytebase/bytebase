package tidb

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"

	tidbast "github.com/pingcap/tidb/pkg/parser/ast"
	tidbformat "github.com/pingcap/tidb/pkg/parser/format"
	"github.com/pingcap/tidb/pkg/parser/mysql"
	tidbtypes "github.com/pingcap/tidb/pkg/parser/types"

	tidbparser "github.com/bytebase/bytebase/backend/plugin/parser/tidb"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	schema.RegisterParseToMetadatas(storepb.Engine_TIDB, ParseToMetadata)
}

const (
	autoIncrementSymbol = "AUTO_INCREMENT"
	autoRandSymbol      = "AUTO_RANDOM"
)

func ParseToMetadata(schema string) (*storepb.DatabaseSchemaMetadata, error) {
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

type tidbTransformer struct {
	tidbast.StmtNode

	state *databaseState
	err   error
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

			columnState := &columnState{
				id:       len(table.columns),
				name:     columnName,
				tp:       dataType,
				comment:  "",
				nullable: tidbColumnCanNull(column),
			}

			for _, option := range column.Options {
				switch option.Tp {
				case tidbast.ColumnOptionDefaultValue:
					defaultValue, err := restoreExpr(option.Expr)
					if err != nil {
						t.err = err
						return in, true
					}
					if defaultValue != nil {
						switch {
						case strings.EqualFold(*defaultValue, "NULL"):
							columnState.defaultValue = &defaultValueNull{}
						case strings.HasPrefix(*defaultValue, "'") && strings.HasSuffix(*defaultValue, "'"):
							columnState.defaultValue = &defaultValueString{value: strings.ReplaceAll((*defaultValue)[1:len(*defaultValue)-1], "''", "'")}
						default:
							columnState.defaultValue = &defaultValueExpression{value: *defaultValue}
						}
					}
				case tidbast.ColumnOptionComment:
					comment, err := restoreComment(option.Expr)
					if err != nil {
						t.err = err
						return in, true
					}
					columnState.comment = comment
				case tidbast.ColumnOptionAutoIncrement:
					defaultValue := autoIncrementSymbol
					columnState.defaultValue = &defaultValueExpression{value: defaultValue}
				case tidbast.ColumnOptionAutoRandom:
					defaultValue := autoRandSymbol
					unspecifiedLength := -1
					if option.AutoRandOpt.ShardBits != unspecifiedLength {
						if option.AutoRandOpt.RangeBits != unspecifiedLength {
							defaultValue += fmt.Sprintf("(%d, %d)", option.AutoRandOpt.ShardBits, option.AutoRandOpt.RangeBits)
						} else {
							defaultValue += fmt.Sprintf("(%d)", option.AutoRandOpt.ShardBits)
						}
					}
					columnState.defaultValue = &defaultValueExpression{value: defaultValue}
				case tidbast.ColumnOptionOnUpdate:
					onUpdate, err := restoreExpr(option.Expr)
					if err != nil {
						t.err = err
						return in, true
					}
					columnState.onUpdate = *onUpdate
				}
			}
			table.columns[columnName] = columnState
		}
		for _, tableOption := range node.Options {
			switch tableOption.Tp {
			case tidbast.TableOptionComment:
				table.comment = tableComment(tableOption)
			case tidbast.TableOptionEngine:
				table.engine = tableOption.StrValue
			case tidbast.TableOptionCollate:
				table.collation = tableOption.StrValue
			}
		}

		// primary and foreign key definition
		for _, constraint := range node.Constraints {
			constraintType := constraint.Tp
			switch constraintType {
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
			case tidbast.ConstraintIndex, tidbast.ConstraintUniq, tidbast.ConstraintUniqKey, tidbast.ConstraintUniqIndex, tidbast.ConstraintKey:
				var referencingColumnList []string
				var lengthList []int64
				for _, spec := range constraint.Keys {
					var specString string
					var err error
					if spec.Column != nil {
						specString = spec.Column.Name.String()
						if spec.Length > 0 {
							lengthList = append(lengthList, int64(spec.Length))
						} else {
							lengthList = append(lengthList, -1)
						}
					} else {
						specString, err = tidbRestoreNode(spec, tidbformat.RestoreKeyWordLowercase|tidbformat.RestoreStringSingleQuotes|tidbformat.RestoreNameBackQuotes)
						if err != nil {
							t.err = err
							return in, true
						}
					}
					referencingColumnList = append(referencingColumnList, specString)
				}

				var indexName string
				if constraint.Name != "" {
					indexName = constraint.Name
				} else {
					t.err = errors.New("empty index name")
					return in, true
				}

				if table.indexes[indexName] != nil {
					t.err = errors.New("multiple foreign keys found: " + indexName)
					return in, true
				}

				table.indexes[indexName] = &indexState{
					id:      len(table.indexes),
					name:    indexName,
					keys:    referencingColumnList,
					length:  lengthList,
					primary: false,
					unique:  constraintType == tidbast.ConstraintUniq || constraintType == tidbast.ConstraintUniqKey || constraintType == tidbast.ConstraintUniqIndex,
				}
			}
		}
	}
	return in, false
}

// columnTypeStr returns the type string of tp.
func columnTypeStr(tp *tidbtypes.FieldType) string {
	// This logic is copy from tidb/pkg/parser/model/model.go:GetTypeDesc()
	// DO NOT TOUCH!
	desc := tp.CompactStr()
	if mysql.HasUnsignedFlag(tp.GetFlag()) && tp.GetType() != mysql.TypeBit && tp.GetType() != mysql.TypeYear {
		desc += " unsigned"
	}
	if mysql.HasZerofillFlag(tp.GetFlag()) && tp.GetType() != mysql.TypeYear {
		desc += " zerofill"
	}
	return desc
}

func tidbColumnCanNull(column *tidbast.ColumnDef) bool {
	for _, option := range column.Options {
		if option.Tp == tidbast.ColumnOptionNotNull || option.Tp == tidbast.ColumnOptionPrimaryKey {
			return false
		}
	}
	return true
}

func restoreExpr(expr tidbast.ExprNode) (*string, error) {
	if expr == nil {
		return nil, nil
	}
	result, err := tidbRestoreNode(expr, tidbformat.RestoreStringSingleQuotes|tidbformat.RestoreStringWithoutCharset)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func tableComment(option *tidbast.TableOption) string {
	return option.StrValue
}

func restoreComment(expr tidbast.ExprNode) (string, error) {
	comment, err := tidbRestoreNode(expr, tidbformat.RestoreStringWithoutCharset)
	if err != nil {
		return "", err
	}
	return comment, nil
}

func equalKeys(a []string, aLength []int64, b []string, bLength []int64) bool {
	if len(a) != len(b) {
		return false
	}
	for i, key := range a {
		if key != b[i] {
			return false
		}
		lenA := int64(-1)
		lenB := int64(-1)
		if len(aLength) > i {
			lenA = aLength[i]
		}
		if len(bLength) > i {
			lenB = bLength[i]
		}
		if lenA != lenB {
			return false
		}
	}
	return true
}

func tidbRestoreNode(node tidbast.Node, flag tidbformat.RestoreFlags) (string, error) {
	var buffer strings.Builder
	ctx := tidbformat.NewRestoreCtx(flag, &buffer)
	if err := node.Restore(ctx); err != nil {
		return "", err
	}
	return buffer.String(), nil
}

func (*tidbTransformer) Leave(in tidbast.Node) (tidbast.Node, bool) {
	return in, true
}
