package catalog

import (
	"fmt"
	"strings"

	tidbast "github.com/pingcap/tidb/pkg/parser/ast"
	"github.com/pingcap/tidb/pkg/parser/format"
	"github.com/pingcap/tidb/pkg/parser/mysql"
	"github.com/pingcap/tidb/pkg/types"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/store/model"
)

// TiDBWalkThrough walks through TiDB AST and updates the database state.
func TiDBWalkThrough(d *model.DatabaseMetadata, ast any) error {
	// We define the Catalog as Database -> Schema -> Table. The Schema is only for PostgreSQL.
	// So we use a Schema whose name is empty for other engines, such as MySQL.
	// If there is no empty-string-name schema, create it to avoid corner cases.
	if d.GetSchema("") == nil {
		d.CreateSchema("")
	}

	nodeList, ok := ast.([]tidbast.StmtNode)
	if !ok {
		return errors.Errorf("invalid ast type %T", ast)
	}
	for _, node := range nodeList {
		// change state
		if err := changeStateTiDB(d, node); err != nil {
			return err
		}
	}

	return nil
}

func changeStateTiDB(d *model.DatabaseMetadata, in tidbast.StmtNode) (err *WalkThroughError) {
	defer func() {
		if err == nil {
			return
		}
		if err.Line == 0 {
			err.Line = in.OriginTextPosition()
		}
	}()
	// Note: removed database deleted check as it's not stored in protobuf
	switch node := in.(type) {
	case *tidbast.CreateTableStmt:
		return tidbCreateTable(d, node)
	case *tidbast.DropTableStmt:
		return tidbDropTable(d, node)
	case *tidbast.AlterTableStmt:
		return tidbAlterTable(d, node)
	case *tidbast.CreateIndexStmt:
		return tidbHandleCreateIndex(d, node)
	case *tidbast.DropIndexStmt:
		return tidbDropIndex(d, node)
	case *tidbast.AlterDatabaseStmt:
		return tidbAlterDatabase(d, node)
	case *tidbast.DropDatabaseStmt:
		return tidbDropDatabase(d, node)
	case *tidbast.CreateDatabaseStmt:
		return NewAccessOtherDatabaseError(d.GetProto().Name, node.Name.O)
	case *tidbast.RenameTableStmt:
		return tidbRenameTable(d, node)
	default:
		return nil
	}
}

func tidbRenameTable(d *model.DatabaseMetadata, node *tidbast.RenameTableStmt) *WalkThroughError {
	for _, tableToTable := range node.TableToTables {
		schema := d.GetSchema("")
		if schema == nil {
			schema = d.CreateSchema("")
		}
		oldTableName := tableToTable.OldTable.Name.O
		newTableName := tableToTable.NewTable.Name.O
		if tidbTheCurrentDatabase(d, tableToTable) {
			if strings.EqualFold(oldTableName, newTableName) {
				return nil
			}
			table := schema.GetTable(oldTableName)
			if table == nil {
				return NewTableNotExistsError(oldTableName)
			}
			if schema.GetTable(newTableName) != nil {
				return NewTableExistsError(newTableName)
			}
			if err := schema.RenameTable(table.GetProto().Name, newTableName); err != nil {
				return &WalkThroughError{Code: code.TableNotExists, Content: err.Error()}
			}
		} else if tidbMoveToOtherDatabase(d, tableToTable) {
			if schema.GetTable(tableToTable.OldTable.Name.O) == nil {
				return NewTableNotExistsError(tableToTable.OldTable.Name.O)
			}
			if err := schema.DropTable(tableToTable.OldTable.Name.O); err != nil {
				return &WalkThroughError{Code: code.TableNotExists, Content: err.Error()}
			}
		} else {
			return NewAccessOtherDatabaseError(d.GetProto().Name, tidbTargetDatabase(d, tableToTable))
		}
	}
	return nil
}

func tidbTargetDatabase(d *model.DatabaseMetadata, node *tidbast.TableToTable) string {
	if node.OldTable.Schema.O != "" && !isCurrentDatabase(d, node.OldTable.Schema.O) {
		return node.OldTable.Schema.O
	}
	return node.NewTable.Schema.O
}

func tidbMoveToOtherDatabase(d *model.DatabaseMetadata, node *tidbast.TableToTable) bool {
	if node.OldTable.Schema.O != "" && !isCurrentDatabase(d, node.OldTable.Schema.O) {
		return false
	}
	return node.OldTable.Schema.O != node.NewTable.Schema.O
}

func tidbTheCurrentDatabase(d *model.DatabaseMetadata, node *tidbast.TableToTable) bool {
	if node.NewTable.Schema.O != "" && !isCurrentDatabase(d, node.NewTable.Schema.O) {
		return false
	}
	if node.OldTable.Schema.O != "" && !isCurrentDatabase(d, node.OldTable.Schema.O) {
		return false
	}
	return true
}

func tidbDropDatabase(d *model.DatabaseMetadata, node *tidbast.DropDatabaseStmt) *WalkThroughError {
	if !isCurrentDatabase(d, node.Name.O) {
		return NewAccessOtherDatabaseError(d.GetProto().Name, node.Name.O)
	}

	// Note: In walk-through, we don't need to actually delete the database metadata
	// The check is sufficient for validation
	return nil
}

func tidbAlterDatabase(d *model.DatabaseMetadata, node *tidbast.AlterDatabaseStmt) *WalkThroughError {
	if !node.AlterDefaultDatabase && !isCurrentDatabase(d, node.Name.O) {
		return NewAccessOtherDatabaseError(d.GetProto().Name, node.Name.O)
	}

	for _, option := range node.Options {
		switch option.Tp {
		case tidbast.DatabaseOptionCharset:
			d.GetProto().CharacterSet = option.Value
		case tidbast.DatabaseOptionCollate:
			d.GetProto().Collation = option.Value
		default:
			// Other database options
		}
	}
	return nil
}

func tidbFindTableState(d *model.DatabaseMetadata, tableName *tidbast.TableName) (*model.TableMetadata, *WalkThroughError) {
	if tableName.Schema.O != "" && !isCurrentDatabase(d, tableName.Schema.O) {
		return nil, NewAccessOtherDatabaseError(d.GetProto().Name, tableName.Schema.O)
	}

	schema := d.GetSchema("")
	if schema == nil {
		schema = d.CreateSchema("")
	}

	table := schema.GetTable(tableName.Name.O)
	if table == nil {
		return nil, NewTableNotExistsError(tableName.Name.O)
	}

	return table, nil
}

func tidbDropIndex(d *model.DatabaseMetadata, node *tidbast.DropIndexStmt) *WalkThroughError {
	table, err := tidbFindTableState(d, node.Table)
	if err != nil {
		return err
	}

	if err := table.DropIndex(node.IndexName); err != nil {
		return &WalkThroughError{Code: code.IndexNotExists, Content: err.Error()}
	}
	return nil
}

func tidbHandleCreateIndex(d *model.DatabaseMetadata, node *tidbast.CreateIndexStmt) *WalkThroughError {
	table, err := tidbFindTableState(d, node.Table)
	if err != nil {
		return err
	}

	unique := false
	tp := tidbast.IndexTypeBtree.String()
	isSpatial := false

	switch node.KeyType {
	case tidbast.IndexKeyTypeNone:
	case tidbast.IndexKeyTypeUnique:
		unique = true
	case tidbast.IndexKeyTypeFulltext:
		tp = FullTextName
	case tidbast.IndexKeyTypeSpatial:
		isSpatial = true
		tp = SpatialName
	default:
		// Other index key types
	}

	keyList, err := tidbValidateAndGetKeyStringList(table, node.IndexPartSpecifications, false /* primary */, isSpatial)
	if err != nil {
		return err
	}

	return tidbCreateIndexHelper(table, node.IndexName, keyList, unique, tp, node.IndexOption)
}

func tidbAlterTable(d *model.DatabaseMetadata, node *tidbast.AlterTableStmt) *WalkThroughError {
	table, err := tidbFindTableState(d, node.Table)
	if err != nil {
		return err
	}

	for _, spec := range node.Specs {
		switch spec.Tp {
		case tidbast.AlterTableOption:
			for _, option := range spec.Options {
				switch option.Tp {
				case tidbast.TableOptionCollate:
					table.GetProto().Collation = option.StrValue
				case tidbast.TableOptionComment:
					table.GetProto().Comment = option.StrValue
				case tidbast.TableOptionEngine:
					table.GetProto().Engine = option.StrValue
				default:
					// Other table options
				}
			}
		case tidbast.AlterTableAddColumns:
			for _, column := range spec.NewColumns {
				var pos *tidbast.ColumnPosition
				if len(spec.NewColumns) == 1 {
					pos = spec.Position
				}
				if err := tidbCreateColumnHelper(table, column, pos); err != nil {
					return err
				}
			}
			// MySQL can add table constraints in ALTER TABLE ADD COLUMN statements.
			for _, constraint := range spec.NewConstraints {
				if err := tidbCreateConstraint(table, constraint); err != nil {
					return err
				}
			}
		case tidbast.AlterTableAddConstraint:
			if err := tidbCreateConstraint(table, spec.Constraint); err != nil {
				return err
			}
		case tidbast.AlterTableDropColumn:
			columnName := spec.OldColumnName.Name.O
			// Validate column exists
			if table.GetColumn(columnName) == nil {
				return NewColumnNotExistsError(table.GetProto().Name, columnName)
			}
			if err := table.DropColumn(columnName); err != nil {
				return &WalkThroughError{Code: code.Internal, Content: fmt.Sprintf("failed to drop column: %v", err)}
			}
		case tidbast.AlterTableDropPrimaryKey:
			if err := table.DropIndex(PrimaryKeyName); err != nil {
				return &WalkThroughError{Code: code.IndexNotExists, Content: err.Error()}
			}
		case tidbast.AlterTableDropIndex:
			if err := table.DropIndex(spec.Name); err != nil {
				return &WalkThroughError{Code: code.IndexNotExists, Content: err.Error()}
			}
		case tidbast.AlterTableDropForeignKey:
			// we do not deal with DROP FOREIGN KEY statements.
		case tidbast.AlterTableModifyColumn:
			if err := tidbChangeColumn(table, spec.NewColumns[0].Name.Name.O, spec.NewColumns[0], spec.Position); err != nil {
				return err
			}
		case tidbast.AlterTableChangeColumn:
			if err := tidbChangeColumn(table, spec.OldColumnName.Name.O, spec.NewColumns[0], spec.Position); err != nil {
				return err
			}
		case tidbast.AlterTableRenameColumn:
			oldColumnName := spec.OldColumnName.Name.O
			newColumnName := spec.NewColumnName.Name.O
			// Validate old column exists
			if table.GetColumn(oldColumnName) == nil {
				return NewColumnNotExistsError(table.GetProto().Name, oldColumnName)
			}
			// Validate new column doesn't already exist
			if table.GetColumn(newColumnName) != nil {
				return &WalkThroughError{
					Code:    code.ColumnExists,
					Content: fmt.Sprintf("Column `%s` already exists in table `%s`", newColumnName, table.GetProto().Name),
				}
			}
			if err := table.RenameColumn(oldColumnName, newColumnName); err != nil {
				return &WalkThroughError{Code: code.Internal, Content: fmt.Sprintf("failed to rename column: %v", err)}
			}
		case tidbast.AlterTableRenameTable:
			schema := d.GetSchema("")
			if err := schema.RenameTable(table.GetProto().Name, spec.NewTable.Name.O); err != nil {
				return &WalkThroughError{Code: code.TableNotExists, Content: err.Error()}
			}
		case tidbast.AlterTableAlterColumn:
			if err := tidbChangeColumnDefault(table, spec.NewColumns[0]); err != nil {
				return err
			}
		case tidbast.AlterTableRenameIndex:
			if err := table.RenameIndex(spec.FromKey.O, spec.ToKey.O); err != nil {
				return &WalkThroughError{Code: code.IndexNotExists, Content: err.Error()}
			}
		case tidbast.AlterTableIndexInvisible:
			if err := tidbChangeIndexVisibility(table, spec.IndexName.O, spec.Visibility); err != nil {
				return err
			}
		default:
			// Other ALTER TABLE types
		}
	}

	return nil
}

func tidbChangeIndexVisibility(t *model.TableMetadata, indexName string, visibility tidbast.IndexVisibility) *WalkThroughError {
	index := t.GetIndex(indexName)
	if index == nil {
		return NewIndexNotExistsError(t.GetProto().Name, indexName)
	}
	switch visibility {
	case tidbast.IndexVisibilityVisible:
		index.GetProto().Visible = true
	case tidbast.IndexVisibilityInvisible:
		index.GetProto().Visible = false
	default:
		// Keep current visibility
	}
	return nil
}

func tidbChangeColumnDefault(t *model.TableMetadata, column *tidbast.ColumnDef) *WalkThroughError {
	columnName := column.Name.Name.L
	col := t.GetColumn(columnName)
	if col == nil {
		return NewColumnNotExistsError(t.GetProto().Name, columnName)
	}

	if len(column.Options) == 1 {
		// SET DEFAULT
		if column.Options[0].Expr.GetType().GetType() != mysql.TypeNull {
			if col.Type != "" {
				switch strings.ToLower(col.Type) {
				case "blob", "tinyblob", "mediumblob", "longblob",
					"text", "tinytext", "mediumtext", "longtext",
					"json",
					"geometry":
					return &WalkThroughError{
						Code: code.InvalidColumnDefault,
						// Content comes from MySQL Error content.
						Content: fmt.Sprintf("BLOB, TEXT, GEOMETRY or JSON column `%s` can't have a default value", columnName),
					}
				default:
					// Other column types allow default values
				}
			}

			defaultValue, err := restoreNode(column.Options[0].Expr, format.RestoreStringWithoutCharset)
			if err != nil {
				return &WalkThroughError{
					Code:    code.Internal,
					Content: fmt.Sprintf("Failed to deparse default value: %v", err),
				}
			}
			col.Default = defaultValue
		} else {
			if !col.Nullable {
				return &WalkThroughError{
					Code: code.SetNullDefaultForNotNullColumn,
					// Content comes from MySQL Error content.
					Content: fmt.Sprintf("Invalid default value for column `%s`", columnName),
				}
			}
			col.Default = ""
		}
	} else {
		// DROP DEFAULT
		col.Default = ""
	}
	return nil
}

func tidbRenameColumnInIndexKey(t *model.TableMetadata, oldName string, newName string) {
	if strings.EqualFold(oldName, newName) {
		return
	}
	for _, index := range t.GetProto().Indexes {
		for i, key := range index.Expressions {
			if strings.EqualFold(key, oldName) {
				index.Expressions[i] = newName
			}
		}
	}
}

// tidbCompleteTableChangeColumn changes column definition.
// It works as:
// 1. drop column from table, but do not drop column from index expressions.
// 2. rename column in index expressions.
// 3. create a new column.
func tidbCompleteTableChangeColumn(t *model.TableMetadata, oldName string, newColumn *tidbast.ColumnDef, position *tidbast.ColumnPosition) *WalkThroughError {
	column := t.GetColumn(oldName)
	if column == nil {
		return NewColumnNotExistsError(t.GetProto().Name, oldName)
	}

	pos := column.Position

	// generate Position struct for creating new column
	// Create a local copy to avoid modifying the input parameter
	var localPosition *tidbast.ColumnPosition
	if position == nil {
		localPosition = &tidbast.ColumnPosition{Tp: tidbast.ColumnPositionNone}
	} else {
		// Create a copy of the position to avoid modifying the original
		localPosition = &tidbast.ColumnPosition{
			Tp:             position.Tp,
			RelativeColumn: position.RelativeColumn,
		}
	}

	if localPosition.Tp == tidbast.ColumnPositionNone {
		if pos == 1 {
			localPosition.Tp = tidbast.ColumnPositionFirst
		} else {
			for _, col := range t.GetColumns() {
				if col.Position == pos-1 {
					localPosition.Tp = tidbast.ColumnPositionAfter
					localPosition.RelativeColumn = &tidbast.ColumnName{Name: tidbast.NewCIStr(col.Name)}
					break
				}
			}
		}
	}
	position = localPosition

	// drop column from column list
	for _, col := range t.GetColumns() {
		if col.Position > pos {
			col.Position--
		}
	}
	if err := t.DropColumn(strings.ToLower(column.Name)); err != nil {
		return &WalkThroughError{Code: code.Internal, Content: fmt.Sprintf("failed to drop column: %v", err)}
	}

	// rename column from index expressions
	tidbRenameColumnInIndexKey(t, oldName, newColumn.Name.Name.O)

	// create a new column
	return tidbCreateColumnHelper(t, newColumn, position)
}

func tidbChangeColumn(t *model.TableMetadata, oldName string, newColumn *tidbast.ColumnDef, position *tidbast.ColumnPosition) *WalkThroughError {
	return tidbCompleteTableChangeColumn(t, oldName, newColumn, position)
}

// tidbReorderColumn reorders the columns for new column and returns the new column position.
func tidbReorderColumn(t *model.TableMetadata, position *tidbast.ColumnPosition) (int, *WalkThroughError) {
	switch position.Tp {
	case tidbast.ColumnPositionNone:
		return len(t.GetColumns()) + 1, nil
	case tidbast.ColumnPositionFirst:
		for _, column := range t.GetColumns() {
			column.Position++
		}
		return 1, nil
	case tidbast.ColumnPositionAfter:
		columnName := position.RelativeColumn.Name.L
		column := t.GetColumn(columnName)
		if column == nil {
			return 0, NewColumnNotExistsError(t.GetProto().Name, columnName)
		}
		for _, col := range t.GetColumns() {
			if col.Position > column.Position {
				col.Position++
			}
		}
		return int(column.Position) + 1, nil
	default:
		return 0, &WalkThroughError{
			Code:    code.Unsupported,
			Content: fmt.Sprintf("Unsupported column position type: %d", position.Tp),
		}
	}
}

func tidbDropTable(d *model.DatabaseMetadata, node *tidbast.DropTableStmt) *WalkThroughError {
	// TODO(rebelice): deal with DROP VIEW statement.
	if !node.IsView {
		for _, name := range node.Tables {
			if name.Schema.O != "" && !isCurrentDatabase(d, name.Schema.O) {
				return &WalkThroughError{
					Code:    code.NotCurrentDatabase,
					Content: fmt.Sprintf("Database `%s` is not the current database `%s`", name.Schema.O, d.GetProto().Name),
				}
			}

			schema := d.GetSchema("")
			if schema == nil {
				schema = d.CreateSchema("")
			}

			table := schema.GetTable(name.Name.O)
			if table == nil {
				if node.IfExists {
					return nil
				}
				return &WalkThroughError{
					Code:    code.TableNotExists,
					Content: fmt.Sprintf("Table `%s` does not exist", name.Name.O),
				}
			}

			if err := schema.DropTable(table.GetProto().Name); err != nil {
				return &WalkThroughError{Code: code.TableNotExists, Content: err.Error()}
			}
		}
	}
	return nil
}

func tidbCopyTable(d *model.DatabaseMetadata, node *tidbast.CreateTableStmt) *WalkThroughError {
	targetTable, err := tidbFindTableState(d, node.ReferTable)
	if err != nil {
		if err.Code == code.NotCurrentDatabase {
			return &WalkThroughError{
				Code:    code.ReferenceOtherDatabase,
				Content: fmt.Sprintf("Reference table `%s` in other database `%s`, skip walkthrough", node.ReferTable.Name.O, node.ReferTable.Schema.O),
			}
		}
		return err
	}

	schema := d.GetSchema("")
	// Create new table
	newTable, createErr := schema.CreateTable(node.Table.Name.O)
	if createErr != nil {
		return &WalkThroughError{Code: code.TableExists, Content: createErr.Error()}
	}

	// Copy columns and indexes from the target table
	for _, col := range targetTable.GetColumns() {
		colCopy, ok := proto.Clone(col).(*storepb.ColumnMetadata)
		if !ok {
			return &WalkThroughError{Code: code.Internal, Content: "failed to clone column metadata"}
		}
		if err := newTable.CreateColumn(colCopy); err != nil {
			return &WalkThroughError{Code: code.ColumnExists, Content: err.Error()}
		}
	}
	for _, idx := range targetTable.GetProto().Indexes {
		idxCopy, ok := proto.Clone(idx).(*storepb.IndexMetadata)
		if !ok {
			return &WalkThroughError{Code: code.Internal, Content: "failed to clone index metadata"}
		}
		if err := newTable.CreateIndex(idxCopy); err != nil {
			return &WalkThroughError{Code: code.IndexExists, Content: err.Error()}
		}
	}

	return nil
}

func tidbCreateTable(d *model.DatabaseMetadata, node *tidbast.CreateTableStmt) *WalkThroughError {
	if node.Table.Schema.O != "" && !isCurrentDatabase(d, node.Table.Schema.O) {
		return &WalkThroughError{
			Code:    code.NotCurrentDatabase,
			Content: fmt.Sprintf("Database `%s` is not the current database `%s`", node.Table.Schema.O, d.GetProto().Name),
		}
	}

	schema := d.GetSchema("")
	if schema == nil {
		schema = d.CreateSchema("")
	}

	if schema.GetTable(node.Table.Name.O) != nil {
		if node.IfNotExists {
			return nil
		}
		return &WalkThroughError{
			Code:    code.TableExists,
			Content: fmt.Sprintf("Table `%s` already exists", node.Table.Name.O),
		}
	}

	if node.Select != nil {
		return &WalkThroughError{
			Code:    code.StatementCreateTableAs,
			Content: fmt.Sprintf("Disallow the CREATE TABLE AS statement but \"%s\" uses", node.Text()),
		}
	}

	if node.ReferTable != nil {
		return tidbCopyTable(d, node)
	}

	table, createErr := schema.CreateTable(node.Table.Name.O)
	if createErr != nil {
		return &WalkThroughError{Code: code.TableExists, Content: createErr.Error()}
	}

	hasAutoIncrement := false

	for _, column := range node.Cols {
		if isAutoIncrement(column) {
			if hasAutoIncrement {
				return &WalkThroughError{
					Code: code.AutoIncrementExists,
					// The content comes from MySQL error content.
					Content: fmt.Sprintf("There can be only one auto column for table `%s`", table.GetProto().Name),
				}
			}
			hasAutoIncrement = true
		}
		if err := tidbCreateColumnHelper(table, column, nil /* position */); err != nil {
			err.Line = column.OriginTextPosition()
			return err
		}
	}

	for _, constraint := range node.Constraints {
		if err := tidbCreateConstraint(table, constraint); err != nil {
			err.Line = constraint.OriginTextPosition()
			return err
		}
	}

	return nil
}

func tidbCreateConstraint(t *model.TableMetadata, constraint *tidbast.Constraint) *WalkThroughError {
	switch constraint.Tp {
	case tidbast.ConstraintPrimaryKey:
		keyList, err := tidbValidateAndGetKeyStringList(t, constraint.Keys, true /* primary */, false /* isSpatial */)
		if err != nil {
			return err
		}
		if err := tidbCreatePrimaryKeyHelper(t, keyList, getIndexType(constraint.Option)); err != nil {
			return err
		}
	case tidbast.ConstraintKey, tidbast.ConstraintIndex:
		keyList, err := tidbValidateAndGetKeyStringList(t, constraint.Keys, false /* primary */, false /* isSpatial */)
		if err != nil {
			return err
		}
		if err := tidbCreateIndexHelper(t, constraint.Name, keyList, false /* unique */, getIndexType(constraint.Option), constraint.Option); err != nil {
			return err
		}
	case tidbast.ConstraintUniq, tidbast.ConstraintUniqKey, tidbast.ConstraintUniqIndex:
		keyList, err := tidbValidateAndGetKeyStringList(t, constraint.Keys, false /* primary */, false /* isSpatial */)
		if err != nil {
			return err
		}
		if err := tidbCreateIndexHelper(t, constraint.Name, keyList, true /* unique */, getIndexType(constraint.Option), constraint.Option); err != nil {
			return err
		}
	case tidbast.ConstraintForeignKey:
		// we do not deal with FOREIGN KEY constraints
	case tidbast.ConstraintFulltext:
		keyList, err := tidbValidateAndGetKeyStringList(t, constraint.Keys, false /* primary */, false /* isSpatial */)
		if err != nil {
			return err
		}
		if err := tidbCreateIndexHelper(t, constraint.Name, keyList, false /* unique */, FullTextName, constraint.Option); err != nil {
			return err
		}
	case tidbast.ConstraintCheck:
		// we do not deal with CHECK constraints
	default:
		// Other constraint types
	}

	return nil
}

func tidbValidateAndGetKeyStringList(t *model.TableMetadata, keyList []*tidbast.IndexPartSpecification, primary bool, isSpatial bool) ([]string, *WalkThroughError) {
	var res []string
	for _, key := range keyList {
		if key.Expr != nil {
			str, err := restoreNode(key, format.DefaultRestoreFlags)
			if err != nil {
				return nil, &WalkThroughError{
					Code:    code.Internal,
					Content: fmt.Sprintf("Failed to deparse index key: %v", err),
				}
			}
			res = append(res, str)
		} else {
			columnName := key.Column.Name.L
			column := t.GetColumn(columnName)
			if column == nil {
				return nil, NewColumnNotExistsError(t.GetProto().Name, columnName)
			}
			if primary {
				column.Nullable = false
			}
			if isSpatial && column.Nullable {
				return nil, &WalkThroughError{
					Code: code.SpatialIndexKeyNullable,
					// The error content comes from MySQL.
					Content: fmt.Sprintf("All parts of a SPATIAL index must be NOT NULL, but `%s` is nullable", column.Name),
				}
			}

			res = append(res, columnName)
		}
	}
	return res, nil
}

func isAutoIncrement(column *tidbast.ColumnDef) bool {
	for _, option := range column.Options {
		if option.Tp == tidbast.ColumnOptionAutoIncrement {
			return true
		}
	}
	return false
}

func checkDefault(columnName string, columnType *types.FieldType, value tidbast.ExprNode) *WalkThroughError {
	if value.GetType().GetType() != mysql.TypeNull {
		switch columnType.GetType() {
		case mysql.TypeBlob, mysql.TypeTinyBlob, mysql.TypeMediumBlob, mysql.TypeLongBlob, mysql.TypeJSON, mysql.TypeGeometry:
			return &WalkThroughError{
				Code: code.InvalidColumnDefault,
				// Content comes from MySQL Error content.
				Content: fmt.Sprintf("BLOB, TEXT, GEOMETRY or JSON column `%s` can't have a default value", columnName),
			}
		default:
			// Other column types allow default values
		}
	}

	if valueExpr, yes := value.(tidbast.ValueExpr); yes {
		datum := types.NewDatum(valueExpr.GetValue())
		if _, err := datum.ConvertTo(types.Context{}, columnType); err != nil {
			return &WalkThroughError{
				Code:    code.InvalidColumnDefault,
				Content: err.Error(),
			}
		}
	}
	return nil
}

func tidbCreateColumnHelper(t *model.TableMetadata, column *tidbast.ColumnDef, position *tidbast.ColumnPosition) *WalkThroughError {
	if t.GetColumn(column.Name.Name.L) != nil {
		return &WalkThroughError{
			Code:    code.ColumnExists,
			Content: fmt.Sprintf("Column `%s` already exists in table `%s`", column.Name.Name.O, t.GetProto().Name),
		}
	}

	pos := len(t.GetColumns()) + 1
	if position != nil {
		var err *WalkThroughError
		pos, err = tidbReorderColumn(t, position)
		if err != nil {
			return err
		}
	}

	col := &storepb.ColumnMetadata{
		Name:         column.Name.Name.L,
		Position:     int32(pos),
		Default:      "",
		Nullable:     true,
		Type:         column.Tp.CompactStr(),
		CharacterSet: column.Tp.GetCharset(),
		Collation:    column.Tp.GetCollate(),
	}
	setNullDefault := false

	for _, option := range column.Options {
		switch option.Tp {
		case tidbast.ColumnOptionPrimaryKey:
			col.Nullable = false
			if err := tidbCreatePrimaryKeyHelper(t, []string{col.Name}, tidbast.IndexTypeBtree.String()); err != nil {
				return err
			}
		case tidbast.ColumnOptionNotNull:
			col.Nullable = false
		case tidbast.ColumnOptionAutoIncrement:
			// we do not deal with AUTO-INCREMENT
		case tidbast.ColumnOptionDefaultValue:
			if err := checkDefault(col.Name, column.Tp, option.Expr); err != nil {
				return err
			}
			if option.Expr.GetType().GetType() != mysql.TypeNull {
				defaultValue, err := restoreNode(option.Expr, format.RestoreStringWithoutCharset)
				if err != nil {
					return &WalkThroughError{
						Code:    code.Internal,
						Content: fmt.Sprintf("Failed to deparse default value: %v", err),
					}
				}
				col.Default = defaultValue
			} else {
				setNullDefault = true
			}
		case tidbast.ColumnOptionUniqKey:
			if err := tidbCreateIndexHelper(t, "", []string{col.Name}, true /* unique */, tidbast.IndexTypeBtree.String(), nil); err != nil {
				return err
			}
		case tidbast.ColumnOptionNull:
			col.Nullable = true
		case tidbast.ColumnOptionOnUpdate:
			// we do not deal with ON UPDATE
			if column.Tp.GetType() != mysql.TypeDatetime && column.Tp.GetType() != mysql.TypeTimestamp {
				return &WalkThroughError{
					Code:    code.OnUpdateColumnNotDatetimeOrTimestamp,
					Content: fmt.Sprintf("Column `%s` use ON UPDATE but is not DATETIME or TIMESTAMP", col.Name),
				}
			}
		case tidbast.ColumnOptionComment:
			comment, err := restoreNode(option.Expr, format.RestoreStringWithoutCharset)
			if err != nil {
				return &WalkThroughError{
					Code:    code.Internal,
					Content: fmt.Sprintf("Failed to deparse comment: %v", err),
				}
			}
			col.Comment = comment
		case tidbast.ColumnOptionGenerated:
			// we do not deal with GENERATED ALWAYS AS
		case tidbast.ColumnOptionReference:
			// MySQL will ignore the inline REFERENCE
			// https://dev.mysql.com/doc/refman/8.0/en/create-table.html
		case tidbast.ColumnOptionCollate:
			col.Collation = option.StrValue
		case tidbast.ColumnOptionCheck:
			// we do not deal with CHECK constraint
		case tidbast.ColumnOptionColumnFormat:
			// we do not deal with COLUMN_FORMAT
		case tidbast.ColumnOptionStorage:
			// we do not deal with STORAGE
		case tidbast.ColumnOptionAutoRandom:
			// we do not deal with AUTO_RANDOM
		default:
			// Other column options
		}
	}

	if !col.Nullable && setNullDefault {
		return &WalkThroughError{
			Code: code.SetNullDefaultForNotNullColumn,
			// Content comes from MySQL Error content.
			Content: fmt.Sprintf("Invalid default value for column `%s`", col.Name),
		}
	}

	if err := t.CreateColumn(col); err != nil {
		return &WalkThroughError{Code: code.ColumnExists, Content: err.Error()}
	}
	return nil
}

func tidbCreateIndexHelper(t *model.TableMetadata, name string, keyList []string, unique bool, tp string, option *tidbast.IndexOption) *WalkThroughError {
	if len(keyList) == 0 {
		return &WalkThroughError{
			Code:    code.IndexEmptyKeys,
			Content: fmt.Sprintf("Index `%s` in table `%s` has empty key", name, t.GetProto().Name),
		}
	}
	if name != "" {
		if t.GetIndex(name) != nil {
			return NewIndexExistsError(t.GetProto().Name, name)
		}
	} else {
		suffix := 1
		for {
			name = keyList[0]
			if suffix > 1 {
				name = fmt.Sprintf("%s_%d", keyList[0], suffix)
			}
			if t.GetIndex(name) == nil {
				break
			}
			suffix++
		}
	}

	visible := option == nil || option.Visibility != tidbast.IndexVisibilityInvisible

	index := &storepb.IndexMetadata{
		Name:        name,
		Expressions: keyList,
		Type:        tp,
		Unique:      unique,
		Primary:     false,
		Visible:     visible,
	}

	if err := t.CreateIndex(index); err != nil {
		return &WalkThroughError{Code: code.IndexExists, Content: err.Error()}
	}
	return nil
}

func tidbCreatePrimaryKeyHelper(t *model.TableMetadata, keys []string, tp string) *WalkThroughError {
	if t.GetIndex(PrimaryKeyName) != nil {
		return &WalkThroughError{
			Code:    code.PrimaryKeyExists,
			Content: fmt.Sprintf("Primary key exists in table `%s`", t.GetProto().Name),
		}
	}

	pk := &storepb.IndexMetadata{
		Name:        PrimaryKeyName,
		Expressions: keys,
		Type:        tp,
		Unique:      true,
		Primary:     true,
		Visible:     true,
	}
	if err := t.CreateIndex(pk); err != nil {
		return &WalkThroughError{Code: code.IndexExists, Content: err.Error()}
	}
	return nil
}

func restoreNode(node tidbast.Node, flag format.RestoreFlags) (string, error) {
	var buffer strings.Builder
	ctx := format.NewRestoreCtx(flag, &buffer)
	if err := node.Restore(ctx); err != nil {
		return "", errors.Wrapf(err, "failed to deparse node")
	}
	return buffer.String(), nil
}

func getIndexType(option *tidbast.IndexOption) string {
	if option != nil {
		switch option.Tp {
		case tidbast.IndexTypeBtree,
			tidbast.IndexTypeHash,
			tidbast.IndexTypeRtree:
			return option.Tp.String()
		default:
			// Other index types
		}
	}
	return tidbast.IndexTypeBtree.String()
}
