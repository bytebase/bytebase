package tidb

import (
	"fmt"
	"slices"
	"strings"

	tidbast "github.com/pingcap/tidb/pkg/parser/ast"
	"github.com/pingcap/tidb/pkg/parser/format"
	"github.com/pingcap/tidb/pkg/parser/mysql"
	"github.com/pingcap/tidb/pkg/types"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/parser/tidb"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

const (
	// PrimaryKeyName is the string for PK.
	PrimaryKeyName string = "PRIMARY"
	// FullTextName is the string for FULLTEXT.
	FullTextName string = "FULLTEXT"
	// SpatialName is the string for SPATIAL.
	SpatialName string = "SPATIAL"
)

func compareIdentifier(a, b string, ignoreCaseSensitive bool) bool {
	if ignoreCaseSensitive {
		return strings.EqualFold(a, b)
	}
	return a == b
}

func init() {
	schema.RegisterWalkThrough(storepb.Engine_TIDB, WalkThrough)
}

// WalkThrough walks through TiDB AST and updates the database state.
func WalkThrough(d *model.DatabaseMetadata, ast []base.AST) *storepb.Advice {
	// We define the Catalog as Database -> Schema -> Table. The Schema is only for PostgreSQL.
	// So we use a Schema whose name is empty for other engines, such as MySQL.
	// If there is no empty-string-name schema, create it to avoid corner cases.
	if d.GetSchemaMetadata("") == nil {
		d.CreateSchema("")
	}

	// Extract TiDB nodes from AST
	var nodeList []tidbast.StmtNode
	for _, unifiedAST := range ast {
		tidbAST, ok := tidb.GetTiDBAST(unifiedAST)
		if !ok {
			return &storepb.Advice{
				Status:  storepb.Advice_ERROR,
				Code:    code.Internal.Int32(),
				Title:   "TiDB walk-through expects TiDB parser result",
				Content: "TiDB walk-through expects TiDB parser result",
				StartPosition: &storepb.Position{
					Line: 0,
				},
			}
		}
		nodeList = append(nodeList, tidbAST.Node)
	}

	for _, node := range nodeList {
		// change state
		if err := changeStateTiDB(d, node); err != nil {
			return err
		}
	}

	return nil
}

func changeStateTiDB(d *model.DatabaseMetadata, in tidbast.StmtNode) (err *storepb.Advice) {
	defer func() {
		if err == nil {
			return
		}
		if err.StartPosition.Line == 0 {
			err.StartPosition.Line = int32(in.OriginTextPosition())
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
		content := fmt.Sprintf("Database `%s` is not the current database `%s`", node.Name.O, d.GetProto().Name)
		return &storepb.Advice{
			Status:        storepb.Advice_WARNING,
			Code:          code.NotCurrentDatabase.Int32(),
			Title:         content,
			Content:       content,
			StartPosition: &storepb.Position{Line: 0},
		}
	case *tidbast.RenameTableStmt:
		return tidbRenameTable(d, node)
	default:
		return nil
	}
}

func tidbRenameTable(d *model.DatabaseMetadata, node *tidbast.RenameTableStmt) *storepb.Advice {
	for _, tableToTable := range node.TableToTables {
		schema := d.GetSchemaMetadata("")
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
				content := fmt.Sprintf("Table `%s` does not exist", oldTableName)
				return &storepb.Advice{
					Status:        storepb.Advice_ERROR,
					Code:          code.TableNotExists.Int32(),
					Title:         content,
					Content:       content,
					StartPosition: &storepb.Position{Line: 0},
				}
			}
			if schema.GetTable(newTableName) != nil {
				content := fmt.Sprintf("Table `%s` already exists", newTableName)
				return &storepb.Advice{
					Status:        storepb.Advice_ERROR,
					Code:          code.TableExists.Int32(),
					Title:         content,
					Content:       content,
					StartPosition: &storepb.Position{Line: 0},
				}
			}
			if err := schema.RenameTable(table.GetProto().Name, newTableName); err != nil {
				return &storepb.Advice{
					Status:        storepb.Advice_ERROR,
					Code:          code.TableNotExists.Int32(),
					Title:         err.Error(),
					Content:       err.Error(),
					StartPosition: &storepb.Position{Line: 0},
				}
			}
		} else if tidbMoveToOtherDatabase(d, tableToTable) {
			if schema.GetTable(tableToTable.OldTable.Name.O) == nil {
				content := fmt.Sprintf("Table `%s` does not exist", tableToTable.OldTable.Name.O)
				return &storepb.Advice{
					Status:        storepb.Advice_ERROR,
					Code:          code.TableNotExists.Int32(),
					Title:         content,
					Content:       content,
					StartPosition: &storepb.Position{Line: 0},
				}
			}
			if err := schema.DropTable(tableToTable.OldTable.Name.O); err != nil {
				return &storepb.Advice{
					Status:        storepb.Advice_ERROR,
					Code:          code.TableNotExists.Int32(),
					Title:         err.Error(),
					Content:       err.Error(),
					StartPosition: &storepb.Position{Line: 0},
				}
			}
		} else {
			content := fmt.Sprintf("Database `%s` is not the current database `%s`", tidbTargetDatabase(d, tableToTable), d.GetProto().Name)
			return &storepb.Advice{
				Status:        storepb.Advice_WARNING,
				Code:          code.NotCurrentDatabase.Int32(),
				Title:         content,
				Content:       content,
				StartPosition: &storepb.Position{Line: 0},
			}
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

func tidbDropDatabase(d *model.DatabaseMetadata, node *tidbast.DropDatabaseStmt) *storepb.Advice {
	if !isCurrentDatabase(d, node.Name.O) {
		content := fmt.Sprintf("Database `%s` is not the current database `%s`", node.Name.O, d.GetProto().Name)
		return &storepb.Advice{
			Status:        storepb.Advice_WARNING,
			Code:          code.NotCurrentDatabase.Int32(),
			Title:         content,
			Content:       content,
			StartPosition: &storepb.Position{Line: 0},
		}
	}

	// Note: In walk-through, we don't need to actually delete the database metadata
	// The check is sufficient for validation
	return nil
}

func tidbAlterDatabase(d *model.DatabaseMetadata, node *tidbast.AlterDatabaseStmt) *storepb.Advice {
	if !node.AlterDefaultDatabase && !isCurrentDatabase(d, node.Name.O) {
		content := fmt.Sprintf("Database `%s` is not the current database `%s`", node.Name.O, d.GetProto().Name)
		return &storepb.Advice{
			Status:        storepb.Advice_WARNING,
			Code:          code.NotCurrentDatabase.Int32(),
			Title:         content,
			Content:       content,
			StartPosition: &storepb.Position{Line: 0},
		}
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

func tidbFindTableState(d *model.DatabaseMetadata, tableName *tidbast.TableName) (*model.TableMetadata, *storepb.Advice) {
	if tableName.Schema.O != "" && !isCurrentDatabase(d, tableName.Schema.O) {
		content := fmt.Sprintf("Database `%s` is not the current database `%s`", tableName.Schema.O, d.GetProto().Name)
		return nil, &storepb.Advice{
			Status:        storepb.Advice_WARNING,
			Code:          code.NotCurrentDatabase.Int32(),
			Title:         content,
			Content:       content,
			StartPosition: &storepb.Position{Line: 0},
		}
	}

	schema := d.GetSchemaMetadata("")
	if schema == nil {
		schema = d.CreateSchema("")
	}

	table := schema.GetTable(tableName.Name.O)
	if table == nil {
		content := fmt.Sprintf("Table `%s` does not exist", tableName.Name.O)
		return nil, &storepb.Advice{
			Status:        storepb.Advice_ERROR,
			Code:          code.TableNotExists.Int32(),
			Title:         content,
			Content:       content,
			StartPosition: &storepb.Position{Line: 0},
		}
	}

	return table, nil
}

func tidbDropIndex(d *model.DatabaseMetadata, node *tidbast.DropIndexStmt) *storepb.Advice {
	table, err := tidbFindTableState(d, node.Table)
	if err != nil {
		return err
	}

	if err := table.DropIndex(node.IndexName); err != nil {
		return &storepb.Advice{
			Status:        storepb.Advice_ERROR,
			Code:          code.IndexNotExists.Int32(),
			Title:         err.Error(),
			Content:       err.Error(),
			StartPosition: &storepb.Position{Line: 0},
		}
	}
	return nil
}

func tidbHandleCreateIndex(d *model.DatabaseMetadata, node *tidbast.CreateIndexStmt) *storepb.Advice {
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

func tidbAlterTable(d *model.DatabaseMetadata, node *tidbast.AlterTableStmt) *storepb.Advice {
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
			// Sort and renumber columns after adding
			// CreateColumn appends to end, but tidbReorderColumn sets position values
			tableProto := table.GetProto()
			slices.SortFunc(tableProto.Columns, func(a, b *storepb.ColumnMetadata) int {
				if a.Position < b.Position {
					return -1
				} else if a.Position > b.Position {
					return 1
				}
				return 0
			})
			for i, col := range tableProto.Columns {
				col.Position = int32(i + 1)
			}
		case tidbast.AlterTableAddConstraint:
			if err := tidbCreateConstraint(table, spec.Constraint); err != nil {
				return err
			}
		case tidbast.AlterTableDropColumn:
			columnName := spec.OldColumnName.Name.O
			// Validate column exists
			if table.GetColumn(columnName) == nil {
				content := fmt.Sprintf("Column `%s` does not exist in table `%s`", columnName, table.GetProto().Name)
				return &storepb.Advice{
					Status:        storepb.Advice_ERROR,
					Code:          code.ColumnNotExists.Int32(),
					Title:         content,
					Content:       content,
					StartPosition: &storepb.Position{Line: 0},
				}
			}
			if err := table.DropColumn(columnName); err != nil {
				content := fmt.Sprintf("failed to drop column: %v", err)
				return &storepb.Advice{
					Status:        storepb.Advice_ERROR,
					Code:          code.Internal.Int32(),
					Title:         content,
					Content:       content,
					StartPosition: &storepb.Position{Line: 0},
				}
			}
		case tidbast.AlterTableDropPrimaryKey:
			if err := table.DropIndex(PrimaryKeyName); err != nil {
				return &storepb.Advice{
					Status:        storepb.Advice_ERROR,
					Code:          code.IndexNotExists.Int32(),
					Title:         err.Error(),
					Content:       err.Error(),
					StartPosition: &storepb.Position{Line: 0},
				}
			}
		case tidbast.AlterTableDropIndex:
			if err := table.DropIndex(spec.Name); err != nil {
				return &storepb.Advice{
					Status:        storepb.Advice_ERROR,
					Code:          code.IndexNotExists.Int32(),
					Title:         err.Error(),
					Content:       err.Error(),
					StartPosition: &storepb.Position{Line: 0},
				}
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
				content := fmt.Sprintf("Column `%s` does not exist in table `%s`", oldColumnName, table.GetProto().Name)
				return &storepb.Advice{
					Status:        storepb.Advice_ERROR,
					Code:          code.ColumnNotExists.Int32(),
					Title:         content,
					Content:       content,
					StartPosition: &storepb.Position{Line: 0},
				}
			}
			// Validate new column doesn't already exist
			if table.GetColumn(newColumnName) != nil {
				content := fmt.Sprintf("Column `%s` already exists in table `%s`", newColumnName, table.GetProto().Name)
				return &storepb.Advice{
					Status:        storepb.Advice_ERROR,
					Code:          code.ColumnExists.Int32(),
					Title:         content,
					Content:       content,
					StartPosition: &storepb.Position{Line: 0},
				}
			}
			if err := table.RenameColumn(oldColumnName, newColumnName); err != nil {
				content := fmt.Sprintf("failed to rename column: %v", err)
				return &storepb.Advice{
					Status:        storepb.Advice_ERROR,
					Code:          code.Internal.Int32(),
					Title:         content,
					Content:       content,
					StartPosition: &storepb.Position{Line: 0},
				}
			}
		case tidbast.AlterTableRenameTable:
			schema := d.GetSchemaMetadata("")
			if err := schema.RenameTable(table.GetProto().Name, spec.NewTable.Name.O); err != nil {
				return &storepb.Advice{
					Status:        storepb.Advice_ERROR,
					Code:          code.TableNotExists.Int32(),
					Title:         err.Error(),
					Content:       err.Error(),
					StartPosition: &storepb.Position{Line: 0},
				}
			}
		case tidbast.AlterTableAlterColumn:
			if err := tidbChangeColumnDefault(table, spec.NewColumns[0]); err != nil {
				return err
			}
		case tidbast.AlterTableRenameIndex:
			// Validate old index exists
			if table.GetIndex(spec.FromKey.O) == nil {
				content := fmt.Sprintf("index %q does not exist in table %q", spec.FromKey.O, table.GetProto().Name)
				return &storepb.Advice{
					Status:        storepb.Advice_ERROR,
					Code:          code.IndexNotExists.Int32(),
					Title:         content,
					Content:       content,
					StartPosition: &storepb.Position{Line: 0},
				}
			}
			// Validate new index doesn't already exist
			if table.GetIndex(spec.ToKey.O) != nil {
				content := fmt.Sprintf("index %q already exists in table %q", spec.ToKey.O, table.GetProto().Name)
				return &storepb.Advice{
					Status:        storepb.Advice_ERROR,
					Code:          code.IndexExists.Int32(),
					Title:         content,
					Content:       content,
					StartPosition: &storepb.Position{Line: 0},
				}
			}
			if err := table.RenameIndex(spec.FromKey.O, spec.ToKey.O); err != nil {
				content := fmt.Sprintf("failed to rename index: %v", err)
				return &storepb.Advice{
					Status:        storepb.Advice_ERROR,
					Code:          code.Internal.Int32(),
					Title:         content,
					Content:       content,
					StartPosition: &storepb.Position{Line: 0},
				}
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

func tidbChangeIndexVisibility(t *model.TableMetadata, indexName string, visibility tidbast.IndexVisibility) *storepb.Advice {
	index := t.GetIndex(indexName)
	if index == nil {
		content := fmt.Sprintf("Index `%s` does not exist in table `%s`", indexName, t.GetProto().Name)
		return &storepb.Advice{
			Status:        storepb.Advice_ERROR,
			Code:          code.IndexNotExists.Int32(),
			Title:         content,
			Content:       content,
			StartPosition: &storepb.Position{Line: 0},
		}
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

func tidbChangeColumnDefault(t *model.TableMetadata, column *tidbast.ColumnDef) *storepb.Advice {
	columnName := column.Name.Name.L
	col := t.GetColumn(columnName)
	if col == nil {
		content := fmt.Sprintf("Column `%s` does not exist in table `%s`", columnName, t.GetProto().Name)
		return &storepb.Advice{
			Status:        storepb.Advice_ERROR,
			Code:          code.ColumnNotExists.Int32(),
			Title:         content,
			Content:       content,
			StartPosition: &storepb.Position{Line: 0},
		}
	}

	colProto := col.GetProto()
	if len(column.Options) == 1 {
		// SET DEFAULT
		if column.Options[0].Expr.GetType().GetType() != mysql.TypeNull {
			if colProto.Type != "" {
				switch strings.ToLower(colProto.Type) {
				case "blob", "tinyblob", "mediumblob", "longblob",
					"text", "tinytext", "mediumtext", "longtext",
					"json",
					"geometry":
					content := fmt.Sprintf("BLOB, TEXT, GEOMETRY or JSON column `%s` can't have a default value", columnName)
					return &storepb.Advice{
						Status:        storepb.Advice_ERROR,
						Code:          code.InvalidColumnDefault.Int32(),
						Title:         content,
						Content:       content,
						StartPosition: &storepb.Position{Line: 0},
					}
				default:
					// Other column types allow default values
				}
			}

			defaultValue, err := restoreNode(column.Options[0].Expr, format.RestoreStringWithoutCharset)
			if err != nil {
				content := fmt.Sprintf("Failed to deparse default value: %v", err)
				return &storepb.Advice{
					Status:        storepb.Advice_ERROR,
					Code:          code.Internal.Int32(),
					Title:         content,
					Content:       content,
					StartPosition: &storepb.Position{Line: 0},
				}
			}
			colProto.Default = defaultValue
		} else {
			if !colProto.Nullable {
				content := fmt.Sprintf("Invalid default value for column `%s`", columnName)
				return &storepb.Advice{
					Status:        storepb.Advice_ERROR,
					Code:          code.SetNullDefaultForNotNullColumn.Int32(),
					Title:         content,
					Content:       content,
					StartPosition: &storepb.Position{Line: 0},
				}
			}
			colProto.Default = ""
		}
	} else {
		// DROP DEFAULT
		colProto.Default = ""
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
// It works by:
// 1. Dropping the old column from the column list (but keeping it in index expressions)
// 2. Renaming the column in index expressions
// 3. Creating a new column with the new definition
func tidbCompleteTableChangeColumn(t *model.TableMetadata, oldName string, newColumn *tidbast.ColumnDef, position *tidbast.ColumnPosition) *storepb.Advice {
	column := t.GetColumn(oldName)
	if column == nil {
		content := fmt.Sprintf("Column `%s` does not exist in table `%s`", oldName, t.GetProto().Name)
		return &storepb.Advice{
			Status:        storepb.Advice_ERROR,
			Code:          code.ColumnNotExists.Int32(),
			Title:         content,
			Content:       content,
			StartPosition: &storepb.Position{Line: 0},
		}
	}

	// Store the current position info if no position specified
	var localPosition *tidbast.ColumnPosition
	if position == nil {
		localPosition = &tidbast.ColumnPosition{Tp: tidbast.ColumnPositionNone}
	} else {
		localPosition = &tidbast.ColumnPosition{
			Tp:             position.Tp,
			RelativeColumn: position.RelativeColumn,
		}
	}

	// If position not specified, preserve current position by finding the column before it
	if localPosition.Tp == tidbast.ColumnPositionNone {
		tableProto := t.GetProto()
		var currentIdx int
		for i, col := range tableProto.Columns {
			if col == column.GetProto() {
				currentIdx = i
				break
			}
		}

		if currentIdx == 0 {
			localPosition.Tp = tidbast.ColumnPositionFirst
		} else {
			// Position after the previous column
			localPosition.Tp = tidbast.ColumnPositionAfter
			previousColumn := tableProto.Columns[currentIdx-1]
			localPosition.RelativeColumn = &tidbast.ColumnName{Name: tidbast.NewCIStr(previousColumn.Name)}
		}
	}

	// Remove the old column from the table (but keep it in index expressions)
	// We use DropColumnWithoutUpdatingIndexes to remove from internal map and proto
	// without affecting index expressions
	if err := t.DropColumnWithoutUpdatingIndexes(oldName); err != nil {
		content := fmt.Sprintf("failed to drop column: %v", err)
		return &storepb.Advice{
			Status:        storepb.Advice_ERROR,
			Code:          code.Internal.Int32(),
			Title:         content,
			Content:       content,
			StartPosition: &storepb.Position{Line: 0},
		}
	}

	// Rename column in index expressions
	tidbRenameColumnInIndexKey(t, oldName, newColumn.Name.Name.O)

	// Create the new column
	if err := tidbCreateColumnHelper(t, newColumn, localPosition); err != nil {
		return err
	}

	// Sort columns by position value to get the correct array order
	// CreateColumn appends to the end, but tidbReorderColumn sets position values,
	// so we need to sort the array to match the position values
	tableProto := t.GetProto()
	slices.SortFunc(tableProto.Columns, func(a, b *storepb.ColumnMetadata) int {
		if a.Position < b.Position {
			return -1
		} else if a.Position > b.Position {
			return 1
		}
		return 0
	})

	// Renumber all column positions to be sequential (1, 2, 3, ...)
	// This closes any gaps left by DropColumnWithoutUpdatingIndexes
	for i, col := range tableProto.Columns {
		col.Position = int32(i + 1)
	}

	return nil
}

func tidbChangeColumn(t *model.TableMetadata, oldName string, newColumn *tidbast.ColumnDef, position *tidbast.ColumnPosition) *storepb.Advice {
	return tidbCompleteTableChangeColumn(t, oldName, newColumn, position)
}

// tidbReorderColumn reorders the columns for new column and returns the new column position.
func tidbReorderColumn(t *model.TableMetadata, position *tidbast.ColumnPosition) (int, *storepb.Advice) {
	switch position.Tp {
	case tidbast.ColumnPositionNone:
		return len(t.GetProto().GetColumns()) + 1, nil
	case tidbast.ColumnPositionFirst:
		for _, column := range t.GetProto().GetColumns() {
			column.Position++
		}
		return 1, nil
	case tidbast.ColumnPositionAfter:
		columnName := position.RelativeColumn.Name.L
		column := t.GetColumn(columnName)
		if column == nil {
			content := fmt.Sprintf("Column `%s` does not exist in table `%s`", columnName, t.GetProto().Name)
			return 0, &storepb.Advice{
				Status:        storepb.Advice_ERROR,
				Code:          code.ColumnNotExists.Int32(),
				Title:         content,
				Content:       content,
				StartPosition: &storepb.Position{Line: 0},
			}
		}
		columnProto := column.GetProto()
		for _, col := range t.GetProto().GetColumns() {
			if col.Position > columnProto.Position {
				col.Position++
			}
		}
		return int(columnProto.Position) + 1, nil
	default:
		content := fmt.Sprintf("Unsupported column position type: %d", position.Tp)
		return 0, &storepb.Advice{
			Status:        storepb.Advice_ERROR,
			Code:          code.Unsupported.Int32(),
			Title:         content,
			Content:       content,
			StartPosition: &storepb.Position{Line: 0},
		}
	}
}

func tidbDropTable(d *model.DatabaseMetadata, node *tidbast.DropTableStmt) *storepb.Advice {
	// TODO(rebelice): deal with DROP VIEW statement.
	if !node.IsView {
		for _, name := range node.Tables {
			if name.Schema.O != "" && !isCurrentDatabase(d, name.Schema.O) {
				content := fmt.Sprintf("Database `%s` is not the current database `%s`", name.Schema.O, d.GetProto().Name)
				return &storepb.Advice{
					Status:        storepb.Advice_WARNING,
					Code:          code.NotCurrentDatabase.Int32(),
					Title:         content,
					Content:       content,
					StartPosition: &storepb.Position{Line: 0},
				}
			}

			schema := d.GetSchemaMetadata("")
			if schema == nil {
				schema = d.CreateSchema("")
			}

			table := schema.GetTable(name.Name.O)
			if table == nil {
				if node.IfExists {
					return nil
				}
				content := fmt.Sprintf("Table `%s` does not exist", name.Name.O)
				return &storepb.Advice{
					Status:        storepb.Advice_ERROR,
					Code:          code.TableNotExists.Int32(),
					Title:         content,
					Content:       content,
					StartPosition: &storepb.Position{Line: 0},
				}
			}

			if err := schema.DropTable(table.GetProto().Name); err != nil {
				return &storepb.Advice{
					Status:        storepb.Advice_ERROR,
					Code:          code.TableNotExists.Int32(),
					Title:         err.Error(),
					Content:       err.Error(),
					StartPosition: &storepb.Position{Line: 0},
				}
			}
		}
	}
	return nil
}

func tidbCopyTable(d *model.DatabaseMetadata, node *tidbast.CreateTableStmt) *storepb.Advice {
	targetTable, err := tidbFindTableState(d, node.ReferTable)
	if err != nil {
		if err.Code == code.NotCurrentDatabase.Int32() {
			content := fmt.Sprintf("Reference table `%s` in other database `%s`, skip walkthrough", node.ReferTable.Name.O, node.ReferTable.Schema.O)
			return &storepb.Advice{
				Status:        storepb.Advice_WARNING,
				Code:          code.ReferenceOtherDatabase.Int32(),
				Title:         content,
				Content:       content,
				StartPosition: &storepb.Position{Line: 0},
			}
		}
		return err
	}

	schema := d.GetSchemaMetadata("")
	// Create new table
	newTable, createErr := schema.CreateTable(node.Table.Name.O)
	if createErr != nil {
		return &storepb.Advice{
			Status:        storepb.Advice_ERROR,
			Code:          code.TableExists.Int32(),
			Title:         createErr.Error(),
			Content:       createErr.Error(),
			StartPosition: &storepb.Position{Line: 0},
		}
	}

	// Copy columns and indexes from the target table
	for _, col := range targetTable.GetProto().GetColumns() {
		colCopy, ok := proto.Clone(col).(*storepb.ColumnMetadata)
		if !ok {
			return &storepb.Advice{
				Status:        storepb.Advice_ERROR,
				Code:          code.Internal.Int32(),
				Title:         "failed to clone column metadata",
				Content:       "failed to clone column metadata",
				StartPosition: &storepb.Position{Line: 0},
			}
		}
		if err := newTable.CreateColumn(colCopy, nil /* columnCatalog */); err != nil {
			return &storepb.Advice{
				Status:        storepb.Advice_ERROR,
				Code:          code.ColumnExists.Int32(),
				Title:         err.Error(),
				Content:       err.Error(),
				StartPosition: &storepb.Position{Line: 0},
			}
		}
	}
	for _, idx := range targetTable.GetProto().Indexes {
		idxCopy, ok := proto.Clone(idx).(*storepb.IndexMetadata)
		if !ok {
			return &storepb.Advice{
				Status:        storepb.Advice_ERROR,
				Code:          code.Internal.Int32(),
				Title:         "failed to clone index metadata",
				Content:       "failed to clone index metadata",
				StartPosition: &storepb.Position{Line: 0},
			}
		}
		if err := newTable.CreateIndex(idxCopy); err != nil {
			return &storepb.Advice{
				Status:        storepb.Advice_ERROR,
				Code:          code.IndexExists.Int32(),
				Title:         err.Error(),
				Content:       err.Error(),
				StartPosition: &storepb.Position{Line: 0},
			}
		}
	}

	return nil
}

func tidbCreateTable(d *model.DatabaseMetadata, node *tidbast.CreateTableStmt) *storepb.Advice {
	if node.Table.Schema.O != "" && !isCurrentDatabase(d, node.Table.Schema.O) {
		content := fmt.Sprintf("Database `%s` is not the current database `%s`", node.Table.Schema.O, d.GetProto().Name)
		return &storepb.Advice{
			Status:        storepb.Advice_WARNING,
			Code:          code.NotCurrentDatabase.Int32(),
			Title:         content,
			Content:       content,
			StartPosition: &storepb.Position{Line: 0},
		}
	}

	schema := d.GetSchemaMetadata("")
	if schema == nil {
		schema = d.CreateSchema("")
	}

	if schema.GetTable(node.Table.Name.O) != nil {
		if node.IfNotExists {
			return nil
		}
		content := fmt.Sprintf("Table `%s` already exists", node.Table.Name.O)
		return &storepb.Advice{
			Status:        storepb.Advice_ERROR,
			Code:          code.TableExists.Int32(),
			Title:         content,
			Content:       content,
			StartPosition: &storepb.Position{Line: 0},
		}
	}

	if node.Select != nil {
		// Trim leading whitespace for display - node.Text() may include leading whitespace
		// to maintain position consistency, but error messages should show clean SQL.
		content := fmt.Sprintf("CREATE TABLE AS statement is used in \"%s\"", strings.TrimSpace(node.Text()))
		return &storepb.Advice{
			Status:        storepb.Advice_WARNING,
			Code:          code.StatementCreateTableAs.Int32(),
			Title:         content,
			Content:       content,
			StartPosition: &storepb.Position{Line: 0},
		}
	}

	if node.ReferTable != nil {
		return tidbCopyTable(d, node)
	}

	table, createErr := schema.CreateTable(node.Table.Name.O)
	if createErr != nil {
		return &storepb.Advice{
			Status:        storepb.Advice_ERROR,
			Code:          code.TableExists.Int32(),
			Title:         createErr.Error(),
			Content:       createErr.Error(),
			StartPosition: &storepb.Position{Line: 0},
		}
	}

	hasAutoIncrement := false

	for _, column := range node.Cols {
		if isAutoIncrementColumn(column) {
			if hasAutoIncrement {
				content := fmt.Sprintf("There can be only one auto column for table `%s`", table.GetProto().Name)
				return &storepb.Advice{
					Status:        storepb.Advice_ERROR,
					Code:          code.AutoIncrementExists.Int32(),
					Title:         content,
					Content:       content,
					StartPosition: &storepb.Position{Line: int32(column.OriginTextPosition())},
				}
			}
			hasAutoIncrement = true
		}
		if err := tidbCreateColumnHelper(table, column, nil /* position */); err != nil {
			err.StartPosition.Line = int32(column.OriginTextPosition())
			return err
		}
	}

	for _, constraint := range node.Constraints {
		if err := tidbCreateConstraint(table, constraint); err != nil {
			err.StartPosition.Line = int32(constraint.OriginTextPosition())
			return err
		}
	}

	return nil
}

func tidbCreateConstraint(t *model.TableMetadata, constraint *tidbast.Constraint) *storepb.Advice {
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

func tidbValidateAndGetKeyStringList(t *model.TableMetadata, keyList []*tidbast.IndexPartSpecification, primary bool, isSpatial bool) ([]string, *storepb.Advice) {
	var res []string
	for _, key := range keyList {
		if key.Expr != nil {
			str, err := restoreNode(key, format.DefaultRestoreFlags)
			if err != nil {
				content := fmt.Sprintf("Failed to deparse index key: %v", err)
				return nil, &storepb.Advice{
					Status:        storepb.Advice_ERROR,
					Code:          code.Internal.Int32(),
					Title:         content,
					Content:       content,
					StartPosition: &storepb.Position{Line: 0},
				}
			}
			res = append(res, str)
		} else {
			columnName := key.Column.Name.L
			column := t.GetColumn(columnName)
			if column == nil {
				content := fmt.Sprintf("Column `%s` does not exist in table `%s`", columnName, t.GetProto().Name)
				return nil, &storepb.Advice{
					Status:        storepb.Advice_ERROR,
					Code:          code.ColumnNotExists.Int32(),
					Title:         content,
					Content:       content,
					StartPosition: &storepb.Position{Line: 0},
				}
			}
			if primary {
				column.GetProto().Nullable = false
			}
			if isSpatial && column.GetProto().Nullable {
				content := fmt.Sprintf("All parts of a SPATIAL index must be NOT NULL, but `%s` is nullable", column.GetProto().Name)
				return nil, &storepb.Advice{
					Status:        storepb.Advice_ERROR,
					Code:          code.SpatialIndexKeyNullable.Int32(),
					Title:         content,
					Content:       content,
					StartPosition: &storepb.Position{Line: 0},
				}
			}

			res = append(res, columnName)
		}
	}
	return res, nil
}

func isAutoIncrementColumn(column *tidbast.ColumnDef) bool {
	for _, option := range column.Options {
		if option.Tp == tidbast.ColumnOptionAutoIncrement {
			return true
		}
	}
	return false
}

func checkDefault(columnName string, columnType *types.FieldType, value tidbast.ExprNode) *storepb.Advice {
	if value.GetType().GetType() != mysql.TypeNull {
		switch columnType.GetType() {
		case mysql.TypeBlob, mysql.TypeTinyBlob, mysql.TypeMediumBlob, mysql.TypeLongBlob, mysql.TypeJSON, mysql.TypeGeometry:
			content := fmt.Sprintf("BLOB, TEXT, GEOMETRY or JSON column `%s` can't have a default value", columnName)
			return &storepb.Advice{
				Status:        storepb.Advice_ERROR,
				Code:          code.InvalidColumnDefault.Int32(),
				Title:         content,
				Content:       content,
				StartPosition: &storepb.Position{Line: 0},
			}
		default:
			// Other column types allow default values
		}
	}

	if valueExpr, yes := value.(tidbast.ValueExpr); yes {
		datum := types.NewDatum(valueExpr.GetValue())
		if _, err := datum.ConvertTo(types.Context{}, columnType); err != nil {
			return &storepb.Advice{
				Status:        storepb.Advice_ERROR,
				Code:          code.InvalidColumnDefault.Int32(),
				Title:         err.Error(),
				Content:       err.Error(),
				StartPosition: &storepb.Position{Line: 0},
			}
		}
	}
	return nil
}

func tidbCreateColumnHelper(t *model.TableMetadata, column *tidbast.ColumnDef, position *tidbast.ColumnPosition) *storepb.Advice {
	if t.GetColumn(column.Name.Name.L) != nil {
		content := fmt.Sprintf("Column `%s` already exists in table `%s`", column.Name.Name.O, t.GetProto().Name)
		return &storepb.Advice{
			Status:        storepb.Advice_ERROR,
			Code:          code.ColumnExists.Int32(),
			Title:         content,
			Content:       content,
			StartPosition: &storepb.Position{Line: 0},
		}
	}

	pos := len(t.GetProto().GetColumns()) + 1
	if position != nil {
		var err *storepb.Advice
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
					content := fmt.Sprintf("Failed to deparse default value: %v", err)
					return &storepb.Advice{
						Status:        storepb.Advice_ERROR,
						Code:          code.Internal.Int32(),
						Title:         content,
						Content:       content,
						StartPosition: &storepb.Position{Line: 0},
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
				content := fmt.Sprintf("Column `%s` use ON UPDATE but is not DATETIME or TIMESTAMP", col.Name)
				return &storepb.Advice{
					Status:        storepb.Advice_ERROR,
					Code:          code.OnUpdateColumnNotDatetimeOrTimestamp.Int32(),
					Title:         content,
					Content:       content,
					StartPosition: &storepb.Position{Line: 0},
				}
			}
		case tidbast.ColumnOptionComment:
			comment, err := restoreNode(option.Expr, format.RestoreStringWithoutCharset)
			if err != nil {
				content := fmt.Sprintf("Failed to deparse comment: %v", err)
				return &storepb.Advice{
					Status:        storepb.Advice_ERROR,
					Code:          code.Internal.Int32(),
					Title:         content,
					Content:       content,
					StartPosition: &storepb.Position{Line: 0},
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
		content := fmt.Sprintf("Invalid default value for column `%s`", col.Name)
		return &storepb.Advice{
			Status:        storepb.Advice_ERROR,
			Code:          code.SetNullDefaultForNotNullColumn.Int32(),
			Title:         content,
			Content:       content,
			StartPosition: &storepb.Position{Line: 0},
		}
	}

	if err := t.CreateColumn(col, nil /* columnCatalog */); err != nil {
		return &storepb.Advice{
			Status:        storepb.Advice_ERROR,
			Code:          code.ColumnExists.Int32(),
			Title:         err.Error(),
			Content:       err.Error(),
			StartPosition: &storepb.Position{Line: 0},
		}
	}
	return nil
}

func tidbCreateIndexHelper(t *model.TableMetadata, name string, keyList []string, unique bool, tp string, option *tidbast.IndexOption) *storepb.Advice {
	if len(keyList) == 0 {
		content := fmt.Sprintf("Index `%s` in table `%s` has empty key", name, t.GetProto().Name)
		return &storepb.Advice{
			Status:        storepb.Advice_ERROR,
			Code:          code.IndexEmptyKeys.Int32(),
			Title:         content,
			Content:       content,
			StartPosition: &storepb.Position{Line: 0},
		}
	}
	if name != "" {
		if t.GetIndex(name) != nil {
			content := fmt.Sprintf("Index `%s` already exists in table `%s`", name, t.GetProto().Name)
			return &storepb.Advice{
				Status:        storepb.Advice_ERROR,
				Code:          code.IndexExists.Int32(),
				Title:         content,
				Content:       content,
				StartPosition: &storepb.Position{Line: 0},
			}
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
		return &storepb.Advice{
			Status:        storepb.Advice_ERROR,
			Code:          code.IndexExists.Int32(),
			Title:         err.Error(),
			Content:       err.Error(),
			StartPosition: &storepb.Position{Line: 0},
		}
	}
	return nil
}

func tidbCreatePrimaryKeyHelper(t *model.TableMetadata, keys []string, tp string) *storepb.Advice {
	if t.GetIndex(PrimaryKeyName) != nil {
		content := fmt.Sprintf("Primary key exists in table `%s`", t.GetProto().Name)
		return &storepb.Advice{
			Status:        storepb.Advice_ERROR,
			Code:          code.PrimaryKeyExists.Int32(),
			Title:         content,
			Content:       content,
			StartPosition: &storepb.Position{Line: 0},
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
		return &storepb.Advice{
			Status:        storepb.Advice_ERROR,
			Code:          code.IndexExists.Int32(),
			Title:         err.Error(),
			Content:       err.Error(),
			StartPosition: &storepb.Position{Line: 0},
		}
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

// isCurrentDatabase returns true if the given database is the current database.
func isCurrentDatabase(d *model.DatabaseMetadata, database string) bool {
	return compareIdentifier(d.DatabaseName(), database, !d.GetIsObjectCaseSensitive())
}
