// Copied from plugin/db/oracle/dump.go
package obo

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"sort"
	"strings"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// Dump dumps the database.
func (driver *Driver) Dump(ctx context.Context, out io.Writer, _ *storepb.DatabaseSchemaMetadata) error {
	txn, err := driver.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}
	defer txn.Rollback()

	if err := driver.dumpTxn(ctx, txn, []string{driver.databaseName}, out); err != nil {
		return err
	}

	if err := txn.Commit(); err != nil {
		return err
	}
	return err
}

func (driver *Driver) dumpTxn(ctx context.Context, txn *sql.Tx, schemas []string, out io.Writer) error {
	for _, schema := range schemas {
		if err := driver.dumpSchemaTxn(ctx, txn, schema, out); err != nil {
			return err
		}
	}
	return nil
}

func (*Driver) dumpSchemaTxn(ctx context.Context, txn *sql.Tx, schema string, out io.Writer) error {
	if err := dumpTableTxn(ctx, txn, schema, out); err != nil {
		return errors.Wrapf(err, "failed to dump table")
	}
	if err := dumpIndexTxn(ctx, txn, schema, out); err != nil {
		return errors.Wrapf(err, "failed to dump index")
	}
	return nil
}

func assembleTableStatement(tableMap map[string]*tableSchema, out io.Writer) error {
	var tableList []*tableSchema
	for _, table := range tableMap {
		switch {
		case !table.meta.TableName.Valid,
			!table.meta.Owner.Valid:
			continue
		}
		tableList = append(tableList, table)
	}
	sort.Slice(tableList, func(i, j int) bool {
		return tableList[i].meta.TableName.String < tableList[j].meta.TableName.String
	})

	for _, table := range tableList {
		if err := table.assembleStatement(out); err != nil {
			return err
		}
		if _, err := out.Write([]byte("\n")); err != nil {
			return err
		}
	}
	return nil
}

type tableSchema struct {
	meta        *tableMeta
	fields      []*fieldMeta
	constraints []*mergedConstraintMeta
}

func (t *tableSchema) assembleStatement(out io.Writer) error {
	if _, err := out.Write([]byte(`CREATE TABLE "`)); err != nil {
		return err
	}
	if _, err := out.Write([]byte(t.meta.Owner.String)); err != nil {
		return err
	}
	if _, err := out.Write([]byte(`"."`)); err != nil {
		return err
	}
	if _, err := out.Write([]byte(t.meta.TableName.String)); err != nil {
		return err
	}
	if _, err := out.Write([]byte("\" (\n")); err != nil {
		return err
	}
	for i, field := range t.fields {
		if i > 0 {
			if _, err := out.Write([]byte(",\n")); err != nil {
				return err
			}
		}
		if _, err := out.Write([]byte(`  `)); err != nil {
			return err
		}
		if err := field.assembleStatement(out); err != nil {
			return err
		}
	}
	for i, constraint := range t.constraints {
		if i+len(t.fields) > 0 {
			if _, err := out.Write([]byte(",\n")); err != nil {
				return err
			}
		}
		if _, err := out.Write([]byte(`  `)); err != nil {
			return err
		}
		if err := constraint.assembleStatement(out); err != nil {
			return err
		}
	}
	if _, err := out.Write([]byte("\n)")); err != nil {
		return err
	}

	if t.meta.Logging.Valid && t.meta.Logging.String == "YES" {
		if _, err := out.Write([]byte("\nLOGGING")); err != nil {
			return err
		}
	} else if t.meta.Logging.Valid && t.meta.Logging.String == "NO" {
		if _, err := out.Write([]byte("\nNOLOGGING")); err != nil {
			return err
		}
	}

	if err := t.assembleCompression(out); err != nil {
		return err
	}

	if t.meta.PctFree.Valid {
		if _, err := out.Write([]byte(fmt.Sprintf("\nPCTFREE %d", t.meta.PctFree.Int64))); err != nil {
			return err
		}
	}

	if t.meta.IniTrans.Valid {
		if _, err := out.Write([]byte(fmt.Sprintf("\nINITRANS %d", t.meta.IniTrans.Int64))); err != nil {
			return err
		}
	}

	if err := t.assembleStorage(out); err != nil {
		return err
	}

	if t.meta.Cache.Valid {
		if t.meta.Cache.String == "Y" {
			if _, err := out.Write([]byte("\nCACHE")); err != nil {
				return err
			}
		} else {
			if _, err := out.Write([]byte("\nNOCACHE")); err != nil {
				return err
			}
		}
	}

	if t.meta.Degree.Valid {
		if t.meta.Degree.String == "DEFAULT" {
			if _, err := out.Write([]byte("\nPARALLEL")); err != nil {
				return err
			}
		} else {
			if _, err := out.Write([]byte(fmt.Sprintf("\nPARALLEL %s", t.meta.Degree.String))); err != nil {
				return err
			}
		}
	}

	if t.meta.RowMovement.Valid {
		if _, err := out.Write([]byte("\n")); err != nil {
			return err
		}
		if t.meta.RowMovement.String == "ENABLED" {
			if _, err := out.Write([]byte("ENABLE")); err != nil {
				return err
			}
		} else {
			if _, err := out.Write([]byte("DISABLE")); err != nil {
				return err
			}
		}
		if _, err := out.Write([]byte(" ROW MOVEMENT")); err != nil {
			return err
		}
	}

	if _, err := out.Write([]byte("\n;\n")); err != nil {
		return err
	}
	return nil
}

func (t *tableSchema) assembleStorage(out io.Writer) error {
	switch {
	case t.meta.InitialExtent.Valid,
		t.meta.NextExtent.Valid,
		t.meta.MinExtents.Valid,
		t.meta.MaxExtents.Valid,
		t.meta.PctIncrease.Valid,
		t.meta.FreeLists.Valid,
		t.meta.FreeListGroups.Valid,
		t.meta.BufferPool.Valid && t.meta.BufferPool.String != "NULL":
	default:
		// No need storage.
		return nil
	}
	if _, err := out.Write([]byte("\nSTORAGE (")); err != nil {
		return err
	}

	switch {
	case t.meta.InitialExtent.Valid:
		if _, err := out.Write([]byte(fmt.Sprintf("\n  INITIAL %d", t.meta.InitialExtent.Int64))); err != nil {
			return err
		}
	case t.meta.NextExtent.Valid:
		if _, err := out.Write([]byte(fmt.Sprintf("\n  NEXT %d", t.meta.NextExtent.Int64))); err != nil {
			return err
		}
	case t.meta.MinExtents.Valid:
		if _, err := out.Write([]byte(fmt.Sprintf("\n  MINEXTENTS %d", t.meta.MinExtents.Int64))); err != nil {
			return err
		}
	case t.meta.MaxExtents.Valid:
		if _, err := out.Write([]byte(fmt.Sprintf("\n  MAXEXTENTS %d", t.meta.MaxExtents.Int64))); err != nil {
			return err
		}
	case t.meta.PctIncrease.Valid:
		if _, err := out.Write([]byte(fmt.Sprintf("\n  PCTINCREASE %d", t.meta.PctIncrease.Int64))); err != nil {
			return err
		}
	case t.meta.FreeLists.Valid:
		if _, err := out.Write([]byte(fmt.Sprintf("\n  FREELISTS %d", t.meta.FreeLists.Int64))); err != nil {
			return err
		}
	case t.meta.FreeListGroups.Valid:
		if _, err := out.Write([]byte(fmt.Sprintf("\n  FREELIST GROUPS %d", t.meta.FreeListGroups.Int64))); err != nil {
			return err
		}
	case t.meta.BufferPool.Valid && t.meta.BufferPool.String != "NULL":
		if _, err := out.Write([]byte(fmt.Sprintf("\n  BUFFER_POOL %s", t.meta.BufferPool.String))); err != nil {
			return err
		}
	}

	if _, err := out.Write([]byte("\n)")); err != nil {
		return err
	}
	return nil
}

func (t *tableSchema) assembleCompression(out io.Writer) error {
	if t.meta.Compression.Valid && t.meta.Compression.String == "DISABLED" {
		if _, err := out.Write([]byte("\nNOCOMPRESS")); err != nil {
			return err
		}
	} else if t.meta.Compression.Valid && t.meta.Compression.String == "ENABLED" {
		switch {
		case !t.meta.CompressFor.Valid || t.meta.CompressFor.String == "NULL":
			if _, err := out.Write([]byte("\nCOMPRESS")); err != nil {
				return err
			}
		case t.meta.CompressFor.Valid && t.meta.CompressFor.String == "BASIC":
			if _, err := out.Write([]byte("\nROW STORE COMPRESS BASIC")); err != nil {
				return err
			}
		case t.meta.CompressFor.Valid && t.meta.CompressFor.String == "ADVANCED":
			if _, err := out.Write([]byte("\nROW STORE COMPRESS ADVANCED")); err != nil {
				return err
			}
		case t.meta.CompressFor.Valid && t.meta.CompressFor.String == "QUERY LOW":
			if _, err := out.Write([]byte("\nCOLUMN STORE COMPRESS FOR QUERY LOW")); err != nil {
				return err
			}
		case t.meta.CompressFor.Valid && t.meta.CompressFor.String == "QUERY HIGH":
			if _, err := out.Write([]byte("\nCOLUMN STORE COMPRESS FOR QUERY HIGH")); err != nil {
				return err
			}
		case t.meta.CompressFor.Valid && t.meta.CompressFor.String == "ARCHIVE LOW":
			if _, err := out.Write([]byte("\nCOLUMN STORE COMPRESS FOR ARCHIVE LOW")); err != nil {
				return err
			}
		case t.meta.CompressFor.Valid && t.meta.CompressFor.String == "ARCHIVE HIGH":
			if _, err := out.Write([]byte("\nCOLUMN STORE COMPRESS FOR ARCHIVE HIGH")); err != nil {
				return err
			}
		}
	}
	return nil
}

type tableMeta struct {
	TableName              sql.NullString
	Owner                  sql.NullString
	TableSpaceName         sql.NullString
	ClusterName            sql.NullString
	IotName                sql.NullString
	PctFree                sql.NullInt64
	PctUsed                sql.NullInt64
	IniTrans               sql.NullInt64
	MaxTrans               sql.NullInt64
	InitialExtent          sql.NullInt64
	NextExtent             sql.NullInt64
	MinExtents             sql.NullInt64
	MaxExtents             sql.NullInt64
	PctIncrease            sql.NullInt64
	FreeLists              sql.NullInt64
	FreeListGroups         sql.NullInt64
	Logging                sql.NullString
	BackedUp               sql.NullString
	NumRows                sql.NullFloat64
	Blocks                 sql.NullInt64
	EmptyBlocks            sql.NullInt64
	AvgSpace               sql.NullInt64
	ChainCnt               sql.NullInt64
	AvgRowLen              sql.NullInt64
	AvgSpaceFreeListBlocks sql.NullInt64
	NumFreeBlocks          sql.NullInt64
	Degree                 sql.NullString
	Instances              sql.NullString
	Cache                  sql.NullString
	TableLock              sql.NullString
	SampleSize             sql.NullInt64
	LastAnalyzed           sql.NullTime
	Partitioned            sql.NullString
	IotType                sql.NullString
	Temporary              sql.NullString
	Secondary              sql.NullString
	Nested                 sql.NullString
	BufferPool             sql.NullString
	Monitoring             sql.NullString
	ClusterOwner           sql.NullString
	Comments               sql.NullString
	ObjectIDType           sql.NullString
	TableTypeOwner         sql.NullString
	TableType              sql.NullString
	GlobalStats            sql.NullString
	UserStats              sql.NullString
	Duration               sql.NullString
	SkipCorrupt            sql.NullString
	RowMovement            sql.NullString
	ExtTableName           sql.NullString
	Dependencies           sql.NullString
	Compression            sql.NullString
	Dropped                sql.NullString
	DropStatus             sql.NullString
	CompressFor            sql.NullString
	Status                 sql.NullString
	Generated              sql.NullString
}

type fieldMeta struct {
	IotType       sql.NullString
	ExtTableName  sql.NullString
	TableName     sql.NullString
	ColumnName    sql.NullString
	DataType      sql.NullString
	DataTypeOwner sql.NullString
	DataLength    sql.NullInt64
	DataPrecision sql.NullInt64
	DataScale     sql.NullInt64
	Nullable      sql.NullString
	ColumnID      sql.NullInt64
	DataDefault   sql.NullString
	CharLength    sql.NullInt64
	CharUsed      sql.NullString
	Collation     sql.NullString
	DefaultOnNull sql.NullString
	IsInvisible   sql.NullString
	Comments      sql.NullString
}

func (f *fieldMeta) assembleStatement(out io.Writer) error {
	if _, err := out.Write([]byte(`"`)); err != nil {
		return err
	}
	if _, err := out.Write([]byte(f.ColumnName.String)); err != nil {
		return err
	}
	if _, err := out.Write([]byte(`" `)); err != nil {
		return err
	}
	if err := f.assembleType(out); err != nil {
		return err
	}
	if f.Collation.Valid {
		if _, err := out.Write([]byte(` COLLATE `)); err != nil {
			return err
		}
		if _, err := out.Write([]byte(f.Collation.String)); err != nil {
			return err
		}
	}
	if f.IsInvisible.Valid {
		if _, err := out.Write([]byte(` VISIBLE`)); err != nil {
			return err
		}
	} else {
		if _, err := out.Write([]byte(` INVISIBLE`)); err != nil {
			return err
		}
	}
	if err := f.assembleDefault(out); err != nil {
		return err
	}
	if f.Nullable.Valid && f.Nullable.String == "N" {
		if _, err := out.Write([]byte(` NOT NULL`)); err != nil {
			return err
		}
	}
	return nil
}

func (f *fieldMeta) assembleDefault(out io.Writer) error {
	if !f.DataDefault.Valid {
		return nil
	}

	if _, err := out.Write([]byte(` DEFAULT `)); err != nil {
		return err
	}
	if f.DefaultOnNull.Valid && f.DefaultOnNull.String == "YES" {
		if _, err := out.Write([]byte(`ON NULL `)); err != nil {
			return err
		}
		return nil
	}
	if _, err := out.Write([]byte(f.DataDefault.String)); err != nil {
		return err
	}
	return nil
}

func (f *fieldMeta) assembleType(out io.Writer) error {
	if _, err := out.Write([]byte(f.DataType.String)); err != nil {
		return err
	}
	switch f.DataType.String {
	case "VARCHAR2", "CHAR":
		if _, err := out.Write([]byte(fmt.Sprintf("(%d BYTE)", f.DataLength.Int64))); err != nil {
			return err
		}
	case "NVARCHAR2", "RAW", "UROWID", "NCHAR":
		if _, err := out.Write([]byte(fmt.Sprintf("(%d)", f.DataLength.Int64))); err != nil {
			return err
		}
	case "NUMBER":
		switch {
		case !f.DataPrecision.Valid || f.DataPrecision.Int64 == 0:
		// do nothing
		case f.DataPrecision.Valid && f.DataPrecision.Int64 > 0 && (!f.DataScale.Valid || f.DataScale.Int64 == 0):
			if _, err := out.Write([]byte(fmt.Sprintf("(%d)", f.DataPrecision.Int64))); err != nil {
				return err
			}
		case f.DataPrecision.Valid && f.DataPrecision.Int64 > 0 && f.DataScale.Valid && f.DataScale.Int64 > 0:
			if _, err := out.Write([]byte(fmt.Sprintf("(%d,%d)", f.DataPrecision.Int64, f.DataScale.Int64))); err != nil {
				return err
			}
		}
	case "FLOAT":
		switch {
		case !f.DataPrecision.Valid || f.DataPrecision.Int64 == 0:
		// do nothing
		case f.DataPrecision.Valid && f.DataPrecision.Int64 > 0:
			if _, err := out.Write([]byte(fmt.Sprintf("(%d)", f.DataPrecision.Int64))); err != nil {
				return err
			}
		}
	}
	return nil
}

type constraintMeta struct {
	IotType         sql.NullString
	ExtTableName    sql.NullString
	TableName       sql.NullString
	ConstraintName  sql.NullString
	ConstraintType  sql.NullString
	DeleteRule      sql.NullString
	Deferrable      sql.NullString
	Deferred        sql.NullString
	Validated       sql.NullString
	Rely            sql.NullString
	SearchCondition sql.NullString
	Status          sql.NullString
	ColumnName      sql.NullString
	ROwner          sql.NullString
	RTableName      sql.NullString
	RConstraintName sql.NullString
	RColumnName     sql.NullString
}

type mergedConstraintMeta struct {
	IotType         sql.NullString
	ExtTableName    sql.NullString
	TableName       sql.NullString
	ConstraintName  sql.NullString
	ConstraintType  sql.NullString
	DeleteRule      sql.NullString
	Deferrable      sql.NullString
	Deferred        sql.NullString
	Validated       sql.NullString
	Rely            sql.NullString
	SearchCondition sql.NullString
	Status          sql.NullString
	ColumnName      []sql.NullString
	ROwner          sql.NullString
	RTableName      sql.NullString
	RConstraintName sql.NullString
	RColumnName     []sql.NullString
}

func (c *mergedConstraintMeta) assembleStatement(out io.Writer) error {
	if _, err := out.Write([]byte(`CONSTRAINT "`)); err != nil {
		return err
	}
	if _, err := out.Write([]byte(c.ConstraintName.String)); err != nil {
		return err
	}
	if _, err := out.Write([]byte(`"`)); err != nil {
		return err
	}

	switch c.ConstraintType.String {
	case "P":
		if _, err := out.Write([]byte(` PRIMARY KEY (`)); err != nil {
			return err
		}
		for i, column := range c.ColumnName {
			if i != 0 {
				if _, err := out.Write([]byte(", ")); err != nil {
					return err
				}
			}
			if _, err := out.Write([]byte(`"`)); err != nil {
				return err
			}
			if _, err := out.Write([]byte(column.String)); err != nil {
				return err
			}
			if _, err := out.Write([]byte(`"`)); err != nil {
				return err
			}
		}
		if _, err := out.Write([]byte(`)`)); err != nil {
			return err
		}
	case "U":
		if _, err := out.Write([]byte(` UNIQUE (`)); err != nil {
			return err
		}
		for i, column := range c.ColumnName {
			if i != 0 {
				if _, err := out.Write([]byte(", ")); err != nil {
					return err
				}
			}
			if _, err := out.Write([]byte(`"`)); err != nil {
				return err
			}
			if _, err := out.Write([]byte(column.String)); err != nil {
				return err
			}
			if _, err := out.Write([]byte(`"`)); err != nil {
				return err
			}
		}
		if _, err := out.Write([]byte(`)`)); err != nil {
			return err
		}

		if err := c.assembleConstraintState(out); err != nil {
			return err
		}
	case "C":
		if _, err := out.Write([]byte(` CHECK (`)); err != nil {
			return err
		}
		if _, err := out.Write([]byte(c.SearchCondition.String)); err != nil {
			return err
		}
		if _, err := out.Write([]byte(`)`)); err != nil {
			return err
		}
		if err := c.assembleConstraintState(out); err != nil {
			return err
		}
	case "R":
		if _, err := out.Write([]byte(` FOREIGN KEY (`)); err != nil {
			return err
		}
		for i, column := range c.ColumnName {
			if i != 0 {
				if _, err := out.Write([]byte(", ")); err != nil {
					return err
				}
			}
			if _, err := out.Write([]byte(`"`)); err != nil {
				return err
			}
			if _, err := out.Write([]byte(column.String)); err != nil {
				return err
			}
			if _, err := out.Write([]byte(`"`)); err != nil {
				return err
			}
		}
		if _, err := out.Write([]byte(`) REFERENCES "`)); err != nil {
			return err
		}
		if _, err := out.Write([]byte(c.ROwner.String)); err != nil {
			return err
		}
		if _, err := out.Write([]byte(`"."`)); err != nil {
			return err
		}
		if _, err := out.Write([]byte(c.RTableName.String)); err != nil {
			return err
		}
		if _, err := out.Write([]byte(`" (`)); err != nil {
			return err
		}
		for i, column := range c.RColumnName {
			if i != 0 {
				if _, err := out.Write([]byte(", ")); err != nil {
					return err
				}
			}
			if _, err := out.Write([]byte(`"`)); err != nil {
				return err
			}
			if _, err := out.Write([]byte(column.String)); err != nil {
				return err
			}
			if _, err := out.Write([]byte(`"`)); err != nil {
				return err
			}
		}
		if _, err := out.Write([]byte(`)`)); err != nil {
			return err
		}
		if err := c.assembleConstraintState(out); err != nil {
			return err
		}
	default:
		slog.Warn("Unsupported constraint type", slog.String("type", c.ConstraintType.String))
	}
	return nil
}

func (c *mergedConstraintMeta) assembleConstraintState(out io.Writer) error {
	if c.Deferrable.Valid && c.Deferrable.String == "DEFERRABLE" {
		if _, err := out.Write([]byte(` DEFERRABLE`)); err != nil {
			return err
		}
	} else if c.Deferrable.Valid && c.Deferrable.String == "NOT DEFERRABLE" {
		if _, err := out.Write([]byte(` NOT DEFERRABLE`)); err != nil {
			return err
		}
	}

	if c.Deferred.Valid && c.Deferred.String == "DEFERRED" {
		if _, err := out.Write([]byte(` INITIALLY DEFERRED`)); err != nil {
			return err
		}
	} else if c.Deferred.Valid && c.Deferred.String == "IMMEDIATE" {
		if _, err := out.Write([]byte(` INITIALLY IMMEDIATE`)); err != nil {
			return err
		}
	}

	if c.Validated.Valid && c.Validated.String == "NOT VALIDATED" {
		if !c.Rely.Valid {
			if _, err := out.Write([]byte(` NORELY`)); err != nil {
				return err
			}
		} else if c.Rely.String == "RELY" {
			if _, err := out.Write([]byte(` RELY`)); err != nil {
				return err
			}
		}

		if _, err := out.Write([]byte(` NOVALIDATE`)); err != nil {
			return err
		}
	} else {
		if _, err := out.Write([]byte(` VALIDATE`)); err != nil {
			return err
		}
	}
	return nil
}

func assembleIndexes(indexes []*mergedIndexMeta, out io.Writer) error {
	for _, index := range indexes {
		if err := assembleIndexStatement(index, out); err != nil {
			return err
		}
		if _, err := out.Write([]byte("\n;\n\n")); err != nil {
			return err
		}
	}
	return nil
}

func assembleIndexStatement(index *mergedIndexMeta, out io.Writer) error {
	if _, err := out.Write([]byte(`CREATE`)); err != nil {
		return err
	}

	switch index.IndexType.String {
	case "BITMAP", "FUNCTION-BASED BITMAP":
		if _, err := out.Write([]byte(` BITMAP`)); err != nil {
			return err
		}
	}

	if index.Uniqueness.String == "UNIQUE" {
		if _, err := out.Write([]byte(` UNIQUE`)); err != nil {
			return err
		}
	}

	if _, err := out.Write([]byte(` INDEX "`)); err != nil {
		return err
	}

	if _, err := out.Write([]byte(index.Owner.String)); err != nil {
		return err
	}

	if _, err := out.Write([]byte(`"."`)); err != nil {
		return err
	}

	if _, err := out.Write([]byte(index.IndexName.String)); err != nil {
		return err
	}

	if _, err := out.Write([]byte(`" ON "`)); err != nil {
		return err
	}

	if _, err := out.Write([]byte(index.TableOwner.String)); err != nil {
		return err
	}

	if _, err := out.Write([]byte(`"."`)); err != nil {
		return err
	}

	if _, err := out.Write([]byte(index.TableName.String)); err != nil {
		return err
	}

	if _, err := out.Write([]byte(`" (`)); err != nil {
		return err
	}

	if strings.Contains(index.IndexType.String, "FUNCTION-BASED") {
		for i, expression := range index.ColumnExpression {
			if i != 0 {
				if _, err := out.Write([]byte(", ")); err != nil {
					return err
				}
			}
			if _, err := out.Write([]byte(expression.String)); err != nil {
				return err
			}

			if index.Descend[i].Valid {
				if index.Descend[i].String == "DESC" {
					if _, err := out.Write([]byte(` DESC`)); err != nil {
						return err
					}
				} else {
					if _, err := out.Write([]byte(` ASC`)); err != nil {
						return err
					}
				}
			}
		}
	} else {
		for i, column := range index.ColumnName {
			if i != 0 {
				if _, err := out.Write([]byte(", ")); err != nil {
					return err
				}
			}
			if _, err := out.Write([]byte(`"`)); err != nil {
				return err
			}
			if _, err := out.Write([]byte(column.String)); err != nil {
				return err
			}
			if _, err := out.Write([]byte(`"`)); err != nil {
				return err
			}
			if index.Descend[i].Valid {
				if index.Descend[i].String == "DESC" {
					if _, err := out.Write([]byte(` DESC`)); err != nil {
						return err
					}
				} else {
					if _, err := out.Write([]byte(` ASC`)); err != nil {
						return err
					}
				}
			}
		}
	}

	if _, err := out.Write([]byte(`)`)); err != nil {
		return err
	}

	return assembleIndexProperties(index, out)
}

func assembleIndexProperties(index *mergedIndexMeta, out io.Writer) error {
	if index.Logging.String == "YES" {
		if _, err := out.Write([]byte("\nLOGGING")); err != nil {
			return err
		}
	} else {
		if _, err := out.Write([]byte("\nNOLOGGING")); err != nil {
			return err
		}
	}

	if index.Visibility.String == "INVISIBLE" {
		if _, err := out.Write([]byte("\nINVISIBLE")); err != nil {
			return err
		}
	} else {
		if _, err := out.Write([]byte("\nVISIBLE")); err != nil {
			return err
		}
	}

	if index.PctFree.Valid {
		if _, err := out.Write([]byte(fmt.Sprintf("\nPCTFREE %d", index.PctFree.Int64))); err != nil {
			return err
		}
	}

	if index.IniTrans.Valid {
		if _, err := out.Write([]byte(fmt.Sprintf("\nINITRANS %d", index.IniTrans.Int64))); err != nil {
			return err
		}
	}

	if err := index.assembleStorage(out); err != nil {
		return err
	}

	if index.Status.Valid {
		if index.Status.String == "VALID" {
			if _, err := out.Write([]byte("\nUSABLE")); err != nil {
				return err
			}
		} else {
			if _, err := out.Write([]byte("\nUNUSABLE")); err != nil {
				return err
			}
		}
	}

	return nil
}

type mergedIndexMeta struct {
	IndexName            sql.NullString
	Owner                sql.NullString
	IndexType            sql.NullString
	Status               sql.NullString
	TableOwner           sql.NullString
	TableName            sql.NullString
	TableType            sql.NullString
	Uniqueness           sql.NullString
	Logging              sql.NullString
	TablespaceName       sql.NullString
	NumRows              sql.NullFloat64
	LastAnalyzed         sql.NullTime
	Degree               sql.NullString
	Instances            sql.NullString
	Partitioned          sql.NullString
	Temporary            sql.NullString
	Generated            sql.NullString
	BufferPool           sql.NullString
	IniTrans             sql.NullInt64
	MaxTrans             sql.NullInt64
	InitialExtent        sql.NullInt64
	NextExtent           sql.NullInt64
	MinExtents           sql.NullInt64
	MaxExtents           sql.NullInt64
	PctFree              sql.NullInt64
	PctThreshold         sql.NullInt64
	PctIncrease          sql.NullInt64
	IncludeColumn        sql.NullString
	FreeLists            sql.NullInt64
	FreeListGroups       sql.NullInt64
	BLevel               sql.NullInt64
	LeafBlocks           sql.NullInt64
	DistinctKeys         sql.NullInt64
	AvgLeafBlocksPerKey  sql.NullInt64
	AvgDataBlocksPerKey  sql.NullInt64
	ClusteringFactor     sql.NullInt64
	SampleSize           sql.NullInt64
	Compression          sql.NullString
	PrefixLength         sql.NullInt64
	Secondary            sql.NullString
	UserStats            sql.NullString
	Duration             sql.NullString
	PctDirectAccess      sql.NullInt64
	ColumnExpression     []sql.NullString
	Descend              []sql.NullString
	IndexTypeOwner       sql.NullString
	IndexTypeName        sql.NullString
	Parameters           sql.NullString
	DomidxStatus         sql.NullString
	DomidxOpstatus       sql.NullString
	FuncidxStatus        sql.NullString
	GlobalStats          sql.NullString
	IotRedundantPkeyElim sql.NullString
	JoinIndex            sql.NullString
	Dropped              sql.NullString
	Visibility           sql.NullString
	DomidxManagement     sql.NullString
	FlashCache           sql.NullString
	ColTabOwner          sql.NullString
	ColTabName           sql.NullString
	ColumnName           []sql.NullString
	ConstraintName       sql.NullString
	ConstraintType       sql.NullString
}

func (i *mergedIndexMeta) assembleStorage(out io.Writer) error {
	switch {
	case i.InitialExtent.Valid,
		i.NextExtent.Valid,
		i.MinExtents.Valid,
		i.MaxExtents.Valid,
		i.PctIncrease.Valid,
		i.FreeLists.Valid,
		i.FreeListGroups.Valid,
		i.BufferPool.Valid && i.BufferPool.String != "NULL",
		i.FlashCache.Valid:
	default:
		// No need storage.
		return nil
	}
	if _, err := out.Write([]byte("\nSTORAGE (")); err != nil {
		return err
	}

	switch {
	case i.InitialExtent.Valid:
		if _, err := out.Write([]byte(fmt.Sprintf("\n  INITIAL %d", i.InitialExtent.Int64))); err != nil {
			return err
		}
	case i.NextExtent.Valid:
		if _, err := out.Write([]byte(fmt.Sprintf("\n  NEXT %d", i.NextExtent.Int64))); err != nil {
			return err
		}
	case i.MinExtents.Valid:
		if _, err := out.Write([]byte(fmt.Sprintf("\n  MINEXTENTS %d", i.MinExtents.Int64))); err != nil {
			return err
		}
	case i.MaxExtents.Valid:
		if _, err := out.Write([]byte(fmt.Sprintf("\n  MAXEXTENTS %d", i.MaxExtents.Int64))); err != nil {
			return err
		}
	case i.PctIncrease.Valid:
		if _, err := out.Write([]byte(fmt.Sprintf("\n  PCTINCREASE %d", i.PctIncrease.Int64))); err != nil {
			return err
		}
	case i.FreeLists.Valid:
		if _, err := out.Write([]byte(fmt.Sprintf("\n  FREELISTS %d", i.FreeLists.Int64))); err != nil {
			return err
		}
	case i.FreeListGroups.Valid:
		if _, err := out.Write([]byte(fmt.Sprintf("\n  FREELIST GROUPS %d", i.FreeListGroups.Int64))); err != nil {
			return err
		}
	case i.BufferPool.Valid && i.BufferPool.String != "NULL":
		if _, err := out.Write([]byte(fmt.Sprintf("\n  BUFFER_POOL %s", i.BufferPool.String))); err != nil {
			return err
		}
	case i.FlashCache.Valid:
		switch i.FlashCache.String {
		case "DEFAULT":
			if _, err := out.Write([]byte("\n  FLASH_CACHE DEFAULT")); err != nil {
				return err
			}
		case "KEEP":
			if _, err := out.Write([]byte("\n  FLASH_CACHE KEEP")); err != nil {
				return err
			}
		case "NONE":
			if _, err := out.Write([]byte("\n  FLASH_CACHE NONE")); err != nil {
				return err
			}
		}
	}

	if _, err := out.Write([]byte("\n)")); err != nil {
		return err
	}
	return nil
}

type indexMeta struct {
	IndexName            sql.NullString
	Owner                sql.NullString
	IndexType            sql.NullString
	Status               sql.NullString
	TableOwner           sql.NullString
	TableName            sql.NullString
	TableType            sql.NullString
	Uniqueness           sql.NullString
	Logging              sql.NullString
	TablespaceName       sql.NullString
	NumRows              sql.NullFloat64
	LastAnalyzed         sql.NullTime
	Degree               sql.NullString
	Instances            sql.NullString
	Partitioned          sql.NullString
	Temporary            sql.NullString
	Generated            sql.NullString
	BufferPool           sql.NullString
	IniTrans             sql.NullInt64
	MaxTrans             sql.NullInt64
	InitialExtent        sql.NullInt64
	NextExtent           sql.NullInt64
	MinExtents           sql.NullInt64
	MaxExtents           sql.NullInt64
	PctFree              sql.NullInt64
	PctThreshold         sql.NullInt64
	PctIncrease          sql.NullInt64
	IncludeColumn        sql.NullString
	FreeLists            sql.NullInt64
	FreeListGroups       sql.NullInt64
	BLevel               sql.NullInt64
	LeafBlocks           sql.NullInt64
	DistinctKeys         sql.NullInt64
	AvgLeafBlocksPerKey  sql.NullInt64
	AvgDataBlocksPerKey  sql.NullInt64
	ClusteringFactor     sql.NullInt64
	SampleSize           sql.NullInt64
	Compression          sql.NullString
	PrefixLength         sql.NullInt64
	Secondary            sql.NullString
	UserStats            sql.NullString
	Duration             sql.NullString
	PctDirectAccess      sql.NullInt64
	ColumnExpression     sql.NullString
	Descend              sql.NullString
	IndexTypeOwner       sql.NullString
	IndexTypeName        sql.NullString
	Parameters           sql.NullString
	DomidxStatus         sql.NullString
	DomidxOpstatus       sql.NullString
	FuncidxStatus        sql.NullString
	GlobalStats          sql.NullString
	IotRedundantPkeyElim sql.NullString
	JoinIndex            sql.NullString
	Dropped              sql.NullString
	Visibility           sql.NullString
	DomidxManagement     sql.NullString
	FlashCache           sql.NullString
	ColTabOwner          sql.NullString
	ColTabName           sql.NullString
	ColumnName           sql.NullString
	ConstraintName       sql.NullString
	ConstraintType       sql.NullString
}

const (
	dumpTableSQL = `
SELECT
  T.TABLE_NAME,
  T.OWNER,
  T.TABLESPACE_NAME,
  T.CLUSTER_NAME,
  T.IOT_NAME,
  T.PCT_FREE,
  T.PCT_USED,
  T.INI_TRANS,
  T.MAX_TRANS,
  T.INITIAL_EXTENT,
  T.NEXT_EXTENT,
  T.MIN_EXTENTS,
  T.MAX_EXTENTS,
  T.PCT_INCREASE,
  T.FREELISTS,
  T.FREELIST_GROUPS,
  T.LOGGING,
  T.BACKED_UP,
  T.NUM_ROWS,
  T.BLOCKS,
  T.EMPTY_BLOCKS,
  T.AVG_SPACE,
  T.CHAIN_CNT,
  T.AVG_ROW_LEN,
  T.AVG_SPACE_FREELIST_BLOCKS,
  T.NUM_FREELIST_BLOCKS,
  T.DEGREE,
  T.INSTANCES,
  T.CACHE,
  T.TABLE_LOCK,
  T.SAMPLE_SIZE,
  T.LAST_ANALYZED,
  T.PARTITIONED,
  T.IOT_TYPE,
  T.TEMPORARY,
  T.SECONDARY,
  T.NESTED,
  T.BUFFER_POOL,
  T.MONITORING,
  T.CLUSTER_OWNER,
  TC.COMMENTS,
  T.OBJECT_ID_TYPE,
  T.TABLE_TYPE_OWNER,
  T.TABLE_TYPE,
  T.GLOBAL_STATS,
  T.USER_STATS,
  T.DURATION,
  T.SKIP_CORRUPT,
  T.ROW_MOVEMENT,
  ET.TABLE_NAME EXT_TABLE_NAME,
  T.DEPENDENCIES,
  T.COMPRESSION,
  T.DROPPED,
  T.STATUS DROP_STATUS,
  T.COMPRESS_FOR,
  O.STATUS,
  O.GENERATED
FROM
  SYS.ALL_ALL_TABLES T,
  SYS.ALL_OB_EXTERNAL_TABLE_FILES ET,
  SYS.ALL_OBJECTS O,
  SYS.ALL_TAB_COMMENTS TC
WHERE
  TC.OWNER (+) = T.OWNER
  AND TC.TABLE_NAME (+) = T.TABLE_NAME
  AND ET.TABLE_NAME (+) = T.TABLE_NAME
  AND ET.OWNER (+) = T.OWNER
  AND O.OWNER (+) = T.OWNER
  AND O.OBJECT_NAME (+) = T.TABLE_NAME
  AND O.OBJECT_TYPE = 'TABLE'
  AND T.OWNER = '%s'
  AND T.IOT_NAME IS NULL
  AND COALESCE(T.NESTED, 'NO') = 'NO'
ORDER BY
  T.TABLE_NAME ASC`
	dumpFieldSQL = `
SELECT
	T.IOT_TYPE,
	ET.TABLE_NAME EXT_TABLE_NAME,
	C.TABLE_NAME,
	C.COLUMN_NAME,
	C.DATA_TYPE,
	C.DATA_TYPE_OWNER,
	C.DATA_LENGTH,
	C.DATA_PRECISION,
	C.DATA_SCALE,
	C.NULLABLE,
	C.COLUMN_ID,
	C.DATA_DEFAULT,
	C.CHAR_LENGTH,
	C.CHAR_USED,
	C.COLLATION,
	C.DEFAULT_ON_NULL,
	COM.COLUMN_NAME IS_INVISIBLE,
	COM.COMMENTS
FROM
	"SYS"."ALL_TAB_COLS" C,
	SYS.ALL_ALL_TABLES T,
	SYS.ALL_OB_EXTERNAL_TABLE_FILES ET,
	"SYS"."ALL_COL_COMMENTS" COM
WHERE
	COM.OWNER(+) = C.OWNER
	AND COM.TABLE_NAME(+) = C.TABLE_NAME
	AND COM.COLUMN_NAME(+) = C.COLUMN_NAME
	AND C.USER_GENERATED = 'YES'
	AND T.TABLE_NAME = C.TABLE_NAME
	AND T.OWNER = C.OWNER
	AND ET.TABLE_NAME(+) = T.TABLE_NAME
	AND ET.OWNER(+) = T.OWNER
	AND C.OWNER = '%s'
ORDER BY C.TABLE_NAME, C.COLUMN_ID ASC`
	dumpConstraintSQL = `
SELECT
	T.IOT_TYPE,
	ET.TABLE_NAME EXT_TABLE_NAME,
	CONS.TABLE_NAME,
	CONS.CONSTRAINT_NAME,
	CONS.CONSTRAINT_TYPE,
	CONS.DELETE_RULE,
	CONS.DEFERRABLE,
	CONS.DEFERRED,
	CONS.VALIDATED,
	CONS.RELY,
	CONS.SEARCH_CONDITION,
	CONS.STATUS,
	COLS.COLUMN_NAME,
	CONS.R_OWNER,
	CONS_R.TABLE_NAME R_TABLE_NAME,
	CONS.R_CONSTRAINT_NAME,
	(
		SELECT
			COLS_R.COLUMN_NAME
		FROM
			SYS.ALL_CONS_COLUMNS COLS_R
		WHERE
			COLS_R.OWNER = CONS.R_OWNER
			AND COLS_R.CONSTRAINT_NAME = CONS.R_CONSTRAINT_NAME
			AND COLS_R.POSITION = COLS.POSITION
	) R_COLUMN_NAME
FROM SYS.ALL_CONSTRAINTS CONS,
	SYS.ALL_CONS_COLUMNS COLS,
	SYS.ALL_CONSTRAINTS CONS_R ,
	SYS.ALL_ALL_TABLES T ,
	SYS.ALL_OB_EXTERNAL_TABLE_FILES ET
WHERE
	COLS.OWNER(+) = CONS.OWNER
	AND COLS.TABLE_NAME(+) = CONS.TABLE_NAME
	AND COLS.CONSTRAINT_NAME(+) = CONS.CONSTRAINT_NAME
	AND CONS_R.OWNER(+) = CONS.R_OWNER
	AND CONS_R.CONSTRAINT_NAME(+) = CONS.R_CONSTRAINT_NAME
	AND T.TABLE_NAME = CONS.TABLE_NAME
	AND T.OWNER = CONS.OWNER
	AND ET.TABLE_NAME(+) = T.TABLE_NAME
	AND ET.OWNER(+) = T.OWNER
	AND CONS.OWNER = '%s'
ORDER BY CONS.TABLE_NAME, CONS.CONSTRAINT_NAME, COLS.POSITION
`
	dumpIndexSQL = `
SELECT
	I.INDEX_NAME,
	I.OWNER,
	I.INDEX_TYPE,
	I.STATUS,
	I.TABLE_OWNER,
	I.TABLE_NAME,
	I.TABLE_TYPE,
	I.UNIQUENESS,
	I.LOGGING,
	I.TABLESPACE_NAME,
	I.NUM_ROWS,
	I.LAST_ANALYZED,
	I.DEGREE,
	I.INSTANCES,
	I.PARTITIONED,
	I.TEMPORARY,
	I.GENERATED,
	I.BUFFER_POOL,
	I.INI_TRANS,
	I.MAX_TRANS,
	I.INITIAL_EXTENT,
	I.NEXT_EXTENT,
	I.MIN_EXTENTS,
	I.MAX_EXTENTS,
	I.PCT_FREE,
	I.PCT_THRESHOLD,
	I.PCT_INCREASE,
	I.INCLUDE_COLUMN,
	I.FREELISTS,
	I.FREELIST_GROUPS,
	I.BLEVEL,
	I.LEAF_BLOCKS,
	I.DISTINCT_KEYS,
	I.AVG_LEAF_BLOCKS_PER_KEY,
	I.AVG_DATA_BLOCKS_PER_KEY,
	I.CLUSTERING_FACTOR,
	I.SAMPLE_SIZE,
	I.COMPRESSION,
	I.PREFIX_LENGTH,
	I.SECONDARY,
	I.USER_STATS,
	I.DURATION,
	I.PCT_DIRECT_ACCESS,
	IE.COLUMN_EXPRESSION,
	IC.DESCEND,
	I.ITYP_OWNER,
	I.ITYP_NAME,
	I.PARAMETERS,
	I.DOMIDX_STATUS,
	I.DOMIDX_OPSTATUS,
	I.FUNCIDX_STATUS,
	I.GLOBAL_STATS,
	I.IOT_REDUNDANT_PKEY_ELIM,
	I.JOIN_INDEX,
	I.DROPPED,
	I.VISIBILITY,
	I.DOMIDX_MANAGEMENT,
	I.FLASH_CACHE,
	IC.TABLE_OWNER COL_TAB_OWNER,
	IC.TABLE_NAME COL_TAB_NAME,
	IC.COLUMN_NAME,
	C.CONSTRAINT_NAME,
	C.CONSTRAINT_TYPE
FROM
	SYS.ALL_INDEXES I,
	SYS.ALL_IND_COLUMNS IC,
	SYS.ALL_IND_EXPRESSIONS IE,
	SYS.ALL_CONSTRAINTS C
WHERE
	C.OWNER(+) = I.OWNER
	AND C.TABLE_NAME(+) = I.TABLE_NAME
	AND C.CONSTRAINT_NAME(+) = I.INDEX_NAME
	AND IC.INDEX_OWNER(+) = I.OWNER
	AND IC.INDEX_NAME(+) = I.INDEX_NAME
	AND I.INDEX_TYPE != 'LOB'
	AND I.INDEX_TYPE != 'DOMAIN'
	AND I.INDEX_TYPE != 'CLUSTER'
	AND IE.INDEX_OWNER(+) = IC.INDEX_OWNER
	AND IE.INDEX_NAME(+) = IC.INDEX_NAME
	AND IE.COLUMN_POSITION(+) = IC.COLUMN_POSITION
	AND C.CONSTRAINT_NAME IS NULL
	AND I.OWNER = '%s'
ORDER BY I.INDEX_NAME, I.TABLE_NAME ASC, IC.COLUMN_POSITION ASC
`
)

func dumpTableTxn(ctx context.Context, txn *sql.Tx, schema string, out io.Writer) error {
	tableMap := make(map[string]*tableSchema)
	tableRows, err := txn.QueryContext(ctx, fmt.Sprintf(dumpTableSQL, schema))
	var constraintList []*constraintMeta
	if err != nil {
		return errors.Wrapf(err, "failed to exec dumpTableSQL")
	}
	defer tableRows.Close()

	for tableRows.Next() {
		meta := tableMeta{}
		// (help-wanted) Sadly, go-ora with struct tag does not work.
		if err := tableRows.Scan(
			&meta.TableName,
			&meta.Owner,
			&meta.TableSpaceName,
			&meta.ClusterName,
			&meta.IotName,
			&meta.PctFree,
			&meta.PctUsed,
			&meta.IniTrans,
			&meta.MaxTrans,
			&meta.InitialExtent,
			&meta.NextExtent,
			&meta.MinExtents,
			&meta.MaxExtents,
			&meta.PctIncrease,
			&meta.FreeLists,
			&meta.FreeListGroups,
			&meta.Logging,
			&meta.BackedUp,
			&meta.NumRows,
			&meta.Blocks,
			&meta.EmptyBlocks,
			&meta.AvgSpace,
			&meta.ChainCnt,
			&meta.AvgRowLen,
			&meta.AvgSpaceFreeListBlocks,
			&meta.NumFreeBlocks,
			&meta.Degree,
			&meta.Instances,
			&meta.Cache,
			&meta.TableLock,
			&meta.SampleSize,
			&meta.LastAnalyzed,
			&meta.Partitioned,
			&meta.IotType,
			&meta.Temporary,
			&meta.Secondary,
			&meta.Nested,
			&meta.BufferPool,
			&meta.Monitoring,
			&meta.ClusterOwner,
			&meta.Comments,
			&meta.ObjectIDType,
			&meta.TableTypeOwner,
			&meta.TableType,
			&meta.GlobalStats,
			&meta.UserStats,
			&meta.Duration,
			&meta.SkipCorrupt,
			&meta.RowMovement,
			&meta.ExtTableName,
			&meta.Dependencies,
			&meta.Compression,
			&meta.Dropped,
			&meta.DropStatus,
			&meta.CompressFor,
			&meta.Status,
			&meta.Generated,
		); err != nil {
			return err
		}
		if !meta.TableName.Valid {
			continue
		}
		tableMap[meta.TableName.String] = &tableSchema{
			meta: &meta,
		}
	}
	if err := tableRows.Err(); err != nil {
		return err
	}

	fieldRows, err := txn.QueryContext(ctx, fmt.Sprintf(dumpFieldSQL, schema))
	if err != nil {
		return errors.Wrapf(err, "failed to exec dumpFiledSQL")
	}
	defer fieldRows.Close()
	for fieldRows.Next() {
		field := fieldMeta{}
		// (help-wanted) Sadly, go-ora with struct tag does not work.
		if err := fieldRows.Scan(
			&field.IotType,
			&field.ExtTableName,
			&field.TableName,
			&field.ColumnName,
			&field.DataType,
			&field.DataTypeOwner,
			&field.DataLength,
			&field.DataPrecision,
			&field.DataScale,
			&field.Nullable,
			&field.ColumnID,
			&field.DataDefault,
			&field.CharLength,
			&field.CharUsed,
			&field.Collation,
			&field.DefaultOnNull,
			&field.IsInvisible,
			&field.Comments,
		); err != nil {
			return err
		}
		if !field.TableName.Valid {
			slog.Warn("column table name null", slog.String("schema", schema))
			continue
		}
		if !field.ColumnName.Valid {
			slog.Warn("column name null", slog.String("schema", schema), slog.String("table", field.TableName.String))
			continue
		}
		if _, ok := tableMap[field.TableName.String]; !ok {
			slog.Warn("column table not found", slog.String("schema", schema), slog.String("table", field.TableName.String), slog.String("column", field.ColumnName.String))
			continue
		}
		tableMap[field.TableName.String].fields = append(tableMap[field.TableName.String].fields, &field)
	}
	if err := fieldRows.Err(); err != nil {
		return err
	}

	constraintRows, err := txn.QueryContext(ctx, fmt.Sprintf(dumpConstraintSQL, schema))
	if err != nil {
		return errors.Wrapf(err, "failed to exec dumpConstraintSQL")
	}
	defer constraintRows.Close()
	for constraintRows.Next() {
		constraint := constraintMeta{}
		// (help-wanted) Sadly, go-ora with struct tag does not work.
		if err := constraintRows.Scan(
			&constraint.IotType,
			&constraint.ExtTableName,
			&constraint.TableName,
			&constraint.ConstraintName,
			&constraint.ConstraintType,
			&constraint.DeleteRule,
			&constraint.Deferrable,
			&constraint.Deferred,
			&constraint.Validated,
			&constraint.Rely,
			&constraint.SearchCondition,
			&constraint.Status,
			&constraint.ColumnName,
			&constraint.ROwner,
			&constraint.RTableName,
			&constraint.RConstraintName,
			&constraint.RColumnName,
		); err != nil {
			return err
		}
		if !constraint.TableName.Valid {
			slog.Warn("constraint table name null", slog.String("schema", schema))
			continue
		}
		if !constraint.ConstraintName.Valid {
			slog.Warn("constraint name null", slog.String("schema", schema), slog.String("table", constraint.TableName.String))
			continue
		}
		constraintList = append(constraintList, &constraint)
	}
	if err := constraintRows.Err(); err != nil {
		return err
	}

	var mergedConstraintList []*mergedConstraintMeta
	for _, constraint := range constraintList {
		if len(mergedConstraintList) == 0 || mergedConstraintList[len(mergedConstraintList)-1].ConstraintName != constraint.ConstraintName {
			mergedConstraintList = append(mergedConstraintList, &mergedConstraintMeta{
				IotType:         constraint.IotType,
				ExtTableName:    constraint.ExtTableName,
				TableName:       constraint.TableName,
				ConstraintName:  constraint.ConstraintName,
				ConstraintType:  constraint.ConstraintType,
				DeleteRule:      constraint.DeleteRule,
				Deferrable:      constraint.Deferrable,
				Deferred:        constraint.Deferred,
				Validated:       constraint.Validated,
				Rely:            constraint.Rely,
				SearchCondition: constraint.SearchCondition,
				Status:          constraint.Status,
				ColumnName:      []sql.NullString{constraint.ColumnName},
				ROwner:          constraint.ROwner,
				RTableName:      constraint.RTableName,
				RConstraintName: constraint.RConstraintName,
				RColumnName:     []sql.NullString{constraint.RColumnName},
			})
		} else {
			mergedConstraintList[len(mergedConstraintList)-1].ColumnName = append(mergedConstraintList[len(mergedConstraintList)-1].ColumnName, constraint.ColumnName)
			mergedConstraintList[len(mergedConstraintList)-1].RColumnName = append(mergedConstraintList[len(mergedConstraintList)-1].RColumnName, constraint.RColumnName)
		}
	}

	for _, constraint := range mergedConstraintList {
		if _, ok := tableMap[constraint.TableName.String]; !ok {
			slog.Warn("constraint table not found", slog.String("schema", schema), slog.String("table", constraint.TableName.String), slog.String("constraint", constraint.ConstraintName.String))
			continue
		}
		tableMap[constraint.TableName.String].constraints = append(tableMap[constraint.TableName.String].constraints, constraint)
	}

	return assembleTableStatement(tableMap, out)
}

func dumpIndexTxn(ctx context.Context, txn *sql.Tx, schema string, out io.Writer) error {
	indexes := []*indexMeta{}
	indexRows, err := txn.QueryContext(ctx, fmt.Sprintf(dumpIndexSQL, schema))
	if err != nil {
		return err
	}
	defer indexRows.Close()
	for indexRows.Next() {
		index := indexMeta{}
		// (help-wanted) Sadly, go-ora with struct tag does not work.
		if err := indexRows.Scan(
			&index.IndexName,
			&index.Owner,
			&index.IndexType,
			&index.Status,
			&index.TableOwner,
			&index.TableName,
			&index.TableType,
			&index.Uniqueness,
			&index.Logging,
			&index.TablespaceName,
			&index.NumRows,
			&index.LastAnalyzed,
			&index.Degree,
			&index.Instances,
			&index.Partitioned,
			&index.Temporary,
			&index.Generated,
			&index.BufferPool,
			&index.IniTrans,
			&index.MaxTrans,
			&index.InitialExtent,
			&index.NextExtent,
			&index.MinExtents,
			&index.MaxExtents,
			&index.PctFree,
			&index.PctThreshold,
			&index.PctIncrease,
			&index.IncludeColumn,
			&index.FreeLists,
			&index.FreeListGroups,
			&index.BLevel,
			&index.LeafBlocks,
			&index.DistinctKeys,
			&index.AvgLeafBlocksPerKey,
			&index.AvgDataBlocksPerKey,
			&index.ClusteringFactor,
			&index.SampleSize,
			&index.Compression,
			&index.PrefixLength,
			&index.Secondary,
			&index.UserStats,
			&index.Duration,
			&index.PctDirectAccess,
			&index.ColumnExpression,
			&index.Descend,
			&index.IndexTypeOwner,
			&index.IndexTypeName,
			&index.Parameters,
			&index.DomidxStatus,
			&index.DomidxOpstatus,
			&index.FuncidxStatus,
			&index.GlobalStats,
			&index.IotRedundantPkeyElim,
			&index.JoinIndex,
			&index.Dropped,
			&index.Visibility,
			&index.DomidxManagement,
			&index.FlashCache,
			&index.ColTabOwner,
			&index.ColTabName,
			&index.ColumnName,
			&index.ConstraintName,
			&index.ConstraintType,
		); err != nil {
			return err
		}
		if !index.IndexName.Valid {
			continue
		}
		indexes = append(indexes, &index)
	}
	if err := indexRows.Err(); err != nil {
		return err
	}

	var mergedIndexList []*mergedIndexMeta
	for _, index := range indexes {
		if len(mergedIndexList) == 0 || mergedIndexList[len(mergedIndexList)-1].IndexName != index.IndexName {
			mergedIndexList = append(mergedIndexList, &mergedIndexMeta{
				IndexName:            index.IndexName,
				Owner:                index.Owner,
				IndexType:            index.IndexType,
				Status:               index.Status,
				TableOwner:           index.TableOwner,
				TableName:            index.TableName,
				TableType:            index.TableType,
				Uniqueness:           index.Uniqueness,
				Logging:              index.Logging,
				TablespaceName:       index.TablespaceName,
				NumRows:              index.NumRows,
				LastAnalyzed:         index.LastAnalyzed,
				Degree:               index.Degree,
				Instances:            index.Instances,
				Partitioned:          index.Partitioned,
				Temporary:            index.Temporary,
				Generated:            index.Generated,
				BufferPool:           index.BufferPool,
				IniTrans:             index.IniTrans,
				MaxTrans:             index.MaxTrans,
				InitialExtent:        index.InitialExtent,
				NextExtent:           index.NextExtent,
				MinExtents:           index.MinExtents,
				MaxExtents:           index.MaxExtents,
				PctFree:              index.PctFree,
				PctThreshold:         index.PctThreshold,
				PctIncrease:          index.PctIncrease,
				IncludeColumn:        index.IncludeColumn,
				FreeLists:            index.FreeLists,
				FreeListGroups:       index.FreeListGroups,
				BLevel:               index.BLevel,
				LeafBlocks:           index.LeafBlocks,
				DistinctKeys:         index.DistinctKeys,
				AvgLeafBlocksPerKey:  index.AvgLeafBlocksPerKey,
				AvgDataBlocksPerKey:  index.AvgDataBlocksPerKey,
				ClusteringFactor:     index.ClusteringFactor,
				SampleSize:           index.SampleSize,
				Compression:          index.Compression,
				PrefixLength:         index.PrefixLength,
				Secondary:            index.Secondary,
				UserStats:            index.UserStats,
				Duration:             index.Duration,
				PctDirectAccess:      index.PctDirectAccess,
				ColumnExpression:     []sql.NullString{index.ColumnExpression},
				Descend:              []sql.NullString{index.Descend},
				IndexTypeOwner:       index.IndexTypeOwner,
				IndexTypeName:        index.IndexTypeName,
				Parameters:           index.Parameters,
				DomidxStatus:         index.DomidxStatus,
				DomidxOpstatus:       index.DomidxOpstatus,
				FuncidxStatus:        index.FuncidxStatus,
				GlobalStats:          index.GlobalStats,
				IotRedundantPkeyElim: index.IotRedundantPkeyElim,
				JoinIndex:            index.JoinIndex,
				Dropped:              index.Dropped,
				Visibility:           index.Visibility,
				DomidxManagement:     index.DomidxManagement,
				FlashCache:           index.FlashCache,
				ColTabOwner:          index.ColTabOwner,
				ColTabName:           index.ColTabName,
				ColumnName:           []sql.NullString{index.ColumnName},
				ConstraintName:       index.ConstraintName,
				ConstraintType:       index.ConstraintType,
			})
		} else {
			mergedIndexList[len(mergedIndexList)-1].ColumnName = append(mergedIndexList[len(mergedIndexList)-1].ColumnName, index.ColumnName)
			mergedIndexList[len(mergedIndexList)-1].ColumnExpression = append(mergedIndexList[len(mergedIndexList)-1].ColumnExpression, index.ColumnExpression)
			mergedIndexList[len(mergedIndexList)-1].Descend = append(mergedIndexList[len(mergedIndexList)-1].Descend, index.Descend)
		}
	}

	return assembleIndexes(mergedIndexList, out)
}
