package tidb

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/tidb/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/store/model"
)

var (
	_ advisor.Advisor = (*IndexTypeNoBlobAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_INDEX_TYPE_NO_BLOB, &IndexTypeNoBlobAdvisor{})
}

// IndexTypeNoBlobAdvisor is the advisor checking for index type no blob.
type IndexTypeNoBlobAdvisor struct {
}

// Check checks for index type no blob.
func (*IndexTypeNoBlobAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, err := getTiDBOmniNodes(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	checker := &indexTypeNoBlobChecker{
		level:            level,
		title:            checkCtx.Rule.Type.String(),
		originalMetadata: checkCtx.OriginalMetadata,
		tablesNewColumns: make(tableNewOmniColumn),
	}

	for _, ostmt := range stmts {
		checker.checkStmt(ostmt)
	}

	return checker.adviceList, nil
}

type indexTypeNoBlobChecker struct {
	adviceList       []*storepb.Advice
	level            storepb.Advice_Status
	title            string
	originalMetadata *model.DatabaseMetadata
	tablesNewColumns tableNewOmniColumn
}

func (c *indexTypeNoBlobChecker) checkStmt(ostmt OmniStmt) {
	cb := indexFamilyCallbacks{
		onColumn:       c.addNewColumn,
		onConstraint:   c.addConstraint,
		onChangeColumn: c.changeColumn,
	}
	var pkDataList []pkData
	switch n := ostmt.Node.(type) {
	case *ast.CreateTableStmt:
		pkDataList = collectIndexFamilyCreateTable(ostmt, n, cb)
	case *ast.AlterTableStmt:
		pkDataList = collectIndexFamilyAlterTable(ostmt, n, cb)
	case *ast.CreateIndexStmt:
		pkDataList = c.collectCreateIndex(ostmt, n)
	default:
		return
	}

	for _, pd := range pkDataList {
		c.adviceList = append(c.adviceList, &storepb.Advice{
			Status:        c.level,
			Code:          code.IndexTypeNoBlob.Int32(),
			Title:         c.title,
			Content:       fmt.Sprintf("Columns in index must not be BLOB but `%s`.`%s` is %s", pd.table, pd.column, pd.columnType),
			StartPosition: common.ConvertANTLRLineToPosition(pd.line),
		})
	}
}

// collectCreateIndex handles standalone `CREATE INDEX ... ON t (...)`
// statements (the BLOB-only top-level type — neither pk_type nor
// allowlist inspect this form).
//
// Note: pingcap-typed advisor does NOT exclude FULLTEXT/SPATIAL indexes
// from the BLOB check, even though the mysql omni rule does. Preserve
// pingcap behavior; aligning to mysql's stricter exclusion is a separate
// intentional behavior change (cumulative #9 territory).
func (c *indexTypeNoBlobChecker) collectCreateIndex(ostmt OmniStmt, n *ast.CreateIndexStmt) []pkData {
	if n.Table == nil {
		return nil
	}
	tableName := n.Table.Name
	stmtLine := ostmt.AbsoluteLine(n.Loc.Start)
	var pkDataList []pkData
	for _, columnName := range omniIndexColumns(n.Columns) {
		columnType, err := c.getColumnType(tableName, columnName)
		if err != nil {
			continue
		}
		if isBlobType(columnType) {
			pkDataList = append(pkDataList, pkData{
				table:      tableName,
				column:     columnName,
				columnType: columnType,
				line:       stmtLine,
			})
		}
	}
	return pkDataList
}

func (c *indexTypeNoBlobChecker) addNewColumn(tableName string, line int, colDef *ast.ColumnDef) []pkData {
	var pkDataList []pkData
	// Note: pingcap-typed addNewColumn checks ColumnOptionUniqKey only (NOT
	// ColumnOptionPrimaryKey), so column-level `PRIMARY KEY` on a BLOB inside
	// CREATE TABLE is NOT flagged by the pingcap version. Preserve that
	// behavior. The change-column path checks both (matching the pingcap
	// inconsistency); replicate as-is.
	tp := omniDataTypeNameCompact(colDef.TypeName)
	for _, constraint := range colDef.Constraints {
		if constraint != nil && constraint.Type == ast.ColConstrUnique {
			if isBlobType(tp) {
				pkDataList = append(pkDataList, pkData{
					table:      tableName,
					column:     colDef.Name,
					columnType: tp,
					line:       line,
				})
			}
		}
	}
	c.tablesNewColumns.set(tableName, colDef.Name, colDef)
	return pkDataList
}

func (c *indexTypeNoBlobChecker) changeColumn(tableName, oldColumnName string, line int, newColumnDef *ast.ColumnDef) []pkData {
	var pkDataList []pkData
	c.tablesNewColumns.delete(tableName, oldColumnName)
	tp := omniDataTypeNameCompact(newColumnDef.TypeName)
	for _, constraint := range newColumnDef.Constraints {
		if constraint != nil && (constraint.Type == ast.ColConstrPrimaryKey || constraint.Type == ast.ColConstrUnique) {
			if isBlobType(tp) {
				pkDataList = append(pkDataList, pkData{
					table:      tableName,
					column:     newColumnDef.Name,
					columnType: tp,
					line:       line,
				})
			}
		}
	}
	c.tablesNewColumns.set(tableName, newColumnDef.Name, newColumnDef)
	return pkDataList
}

func (c *indexTypeNoBlobChecker) addConstraint(tableName string, line int, constraint *ast.Constraint) []pkData {
	var pkDataList []pkData
	// Mirror pingcap match: PRIMARY KEY + UNIQUE family + INDEX/KEY family.
	// Pingcap had 6 distinct ConstraintTp values (ConstraintPrimaryKey,
	// ConstraintUniqKey, ConstraintKey, ConstraintIndex, ConstraintUniqIndex,
	// ConstraintUniq); omni unifies the unique forms under ConstrUnique and
	// the index/key forms under ConstrIndex (cumulative #2). Pingcap-typed
	// advisor does NOT include ConstraintForeignKey; the mysql omni analog
	// does. Preserve pingcap behavior — FK is a separate behavior change.
	switch constraint.Type {
	case ast.ConstrPrimaryKey, ast.ConstrUnique, ast.ConstrIndex:
	default:
		return pkDataList
	}
	for _, columnName := range constraint.Columns {
		columnType, err := c.getColumnType(tableName, columnName)
		if err != nil {
			continue
		}
		if isBlobType(columnType) {
			pkDataList = append(pkDataList, pkData{
				table:      tableName,
				column:     columnName,
				columnType: columnType,
				line:       line,
			})
		}
	}
	return pkDataList
}

// getColumnType returns the lowercase rendered type for tableName.columnName,
// preferring the column we observed in this check pass and falling back to
// the catalog.
func (c *indexTypeNoBlobChecker) getColumnType(tableName, columnName string) (string, error) {
	if colDef, ok := c.tablesNewColumns.get(tableName, columnName); ok {
		return omniDataTypeNameCompact(colDef.TypeName), nil
	}
	column := c.originalMetadata.GetSchemaMetadata("").GetTable(tableName).GetColumn(columnName)
	if column != nil {
		return strings.ToLower(column.GetProto().Type), nil
	}
	return "", errors.Errorf(errCannotFindColumnTypeFmt, tableName, columnName)
}

// isBlobType matches the BLOB *and* TEXT type families. The rule is
// named INDEX_TYPE_NO_BLOB and pingcap rendered everything as "blob" in
// advice content, but pingcap-tidb's `mysql.TypeBlob` enum value covers
// both BLOB and TEXT (TEXT is BLOB with non-binary charset), so the
// pingcap-typed advisor flagged TEXT-in-INDEX too. Omni gives TEXT its
// own DataType.Name (cumulative #14 territory): a mechanical port that
// only matched the four BLOB names dropped TEXT coverage. Match both
// families to preserve the rule's effective behavior; advice content
// now reflects the accurate type name (e.g. "is text" rather than
// pingcap's accidental "is blob").
func isBlobType(columnType string) bool {
	switch strings.ToLower(columnType) {
	case "blob", "tinyblob", "mediumblob", "longblob",
		"text", "tinytext", "mediumtext", "longtext":
		return true
	default:
		return false
	}
}
