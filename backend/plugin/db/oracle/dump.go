package oracle

import (
	"context"
	"database/sql"
	"fmt"
	"io"

	_ "github.com/sijms/go-ora/v2"
)

// Dump dumps the database.
func (driver *Driver) Dump(ctx context.Context, out io.Writer, _ bool) (string, error) {
	txn, err := driver.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return "", err
	}
	defer txn.Rollback()

	schemas, err := getSchemas(txn)
	if err != nil {
		return "", err
	}

	if len(schemas) == 0 {
		return "", nil
	}

	var list []string
	if driver.schemaTenantMode {
		list = append(list, driver.databaseName)
	} else {
		list = append(list, schemas...)
	}
	if err := dumpTxn(ctx, txn, list, out); err != nil {
		return "", err
	}

	if err := txn.Commit(); err != nil {
		return "", err
	}
	return "", nil
}

func dumpTxn(ctx context.Context, txn *sql.Tx, schemas []string, out io.Writer) error {
	for _, schema := range schemas {
		if err := dumpSchemaTxn(ctx, txn, schema, out); err != nil {
			return err
		}
	}
	return nil
}

func dumpSchemaTxn(ctx context.Context, txn *sql.Tx, schema string, out io.Writer) error {
	if err := dumpTableTxn(ctx, txn, schema, out); err != nil {
		return err
	}
	if err := dumpViewTxn(ctx, txn, schema, out); err != nil {
		return err
	}
	if err := dumpFunctionTxn(ctx, txn, schema, out); err != nil {
		return err
	}

	return nil
}

type tableSchema struct {
	meta        *tableMeta
	fields      []*fieldMeta
	constraints []*constraintMeta
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
	NumRows                sql.NullInt64
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
	ObjectIdType           sql.NullString
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
	ColumnId      sql.NullInt64
	DataDefault   sql.NullString
	CharLength    sql.NullInt64
	CharUsed      sql.NullString
	IsInvisible   sql.NullString
	Comments      sql.NullString
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

type viewMeta struct {
	ViewName       sql.NullString
	TextLength     sql.NullInt64
	Text           sql.NullString
	TypeTextLength sql.NullInt64
	TypeText       sql.NullString
	OidTextLength  sql.NullInt64
	OidText        sql.NullString
	ViewTypeOwner  sql.NullString
	ViewType       sql.NullString
	SuperViewName  sql.NullString
	EditioningView sql.NullString
	ReadOnly       sql.NullString
	Status         sql.NullString
	Comments       sql.NullString
	ConstraintName sql.NullString
	ConstraintType sql.NullString
}

type functionMeta struct {
	ObjectName    sql.NullString
	Owner         sql.NullString
	DataObjectId  sql.NullInt64
	ObjectType    sql.NullString
	Status        sql.NullString
	Created       sql.NullTime
	LastDdlTime   sql.NullTime
	Aggregate     sql.NullString
	Pipelined     sql.NullString
	ImplTypeOwner sql.NullString
	ImplTypeName  sql.NullString
	Parallel      sql.NullString
	Interface     sql.NullString
	Deterministic sql.NullString
	AuthId        sql.NullString
	ParamValue    sql.NullString
	ObjectId      sql.NullInt64
	SubProgramId  sql.NullInt64
	Overload      sql.NullInt64
	Timestamp     sql.NullString
	Line          sql.NullInt64
	Text          sql.NullString
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
	NumRows              sql.NullInt64
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

type sequenceMeta struct {
	SequenceName sql.NullString
	MinValue     sql.NullInt64
	MaxValue     sql.NullInt64
	IncrementBy  sql.NullInt64
	CycleFlag    sql.NullString
	OrderFlag    sql.NullString
	CacheSize    sql.NullInt64
	LastNumber   sql.NullInt64
	KeepValue    sql.NullString
	SessionFlag  sql.NullString
}

type triggerOrderingMeta struct {
	TriggerOwner      sql.NullString
	TriggerName       sql.NullString
	ReferencedSchema  sql.NullString
	ReferencedTrigger sql.NullString
	OrderingType      sql.NullString
}

type triggerMeta struct {
	Owner            sql.NullString
	TriggerName      sql.NullString
	TriggerType      sql.NullString
	TriggerEvent     sql.NullString
	BaseObjectType   sql.NullString
	TableName        sql.NullString
	NestedColumn     sql.NullString
	ReferencingNames sql.NullString
	WhenClause       sql.NullString
	IsEnable         sql.NullString
	Description      sql.NullString
	TriggerBody      sql.NullString
	ActionType       sql.NullString
	Edition          sql.NullString
	ColumnName       sql.NullString
	IotType          sql.NullString
	Debug            sql.NullString
	ObjectStatus     sql.NullString
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
  SYS.ALL_EXTERNAL_TABLES ET,
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
  AND T.NESTED = 'NO'
  AND T.SECONDARY = 'N'
  AND NOT EXISTS (
    SELECT
      1
    FROM
      SYS.ALL_MVIEWS MV
    WHERE
      MV.OWNER = T.OWNER
      AND MV.MVIEW_NAME = T.TABLE_NAME
  )
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
	COM.COLUMN_NAME IS_INVISIBLE,
	COM.COMMENTS
FROM
	"SYS"."ALL_TAB_COLS" C,
	SYS.ALL_ALL_TABLES T,
	SYS.ALL_EXTERNAL_TABLES ET,
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
	SYS.ALL_EXTERNAL_TABLES ET
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
	dumpViewSQL = `
SELECT
	V.VIEW_NAME,
	V.TEXT_LENGTH,
	V.TEXT,
	V.TYPE_TEXT_LENGTH,
	V.TYPE_TEXT,
	V.OID_TEXT_LENGTH,
	V.OID_TEXT,
	V.VIEW_TYPE_OWNER,
	V.VIEW_TYPE,
	V.SUPERVIEW_NAME,
	V.EDITIONING_VIEW,
	V.READ_ONLY,
	(
		SELECT
			STATUS
		FROM
			SYS.ALL_OBJECTS
		WHERE
			OWNER = V.OWNER
			AND OBJECT_NAME = V.VIEW_NAME
			AND OBJECT_TYPE = 'VIEW'
			AND SUBOBJECT_NAME IS NULL
	) STATUS,
	(
		SELECT
			COMMENTS
		FROM
			SYS.ALL_TAB_COMMENTS
		WHERE
			OWNER = V.OWNER
			AND TABLE_NAME = V.VIEW_NAME
	) COMMENTS,
	C.CONSTRAINT_NAME,
	C.CONSTRAINT_TYPE
FROM
	SYS.ALL_VIEWS V,
	SYS.ALL_CONSTRAINTS C
WHERE
	C.OWNER(+) = V.OWNER
	AND C.TABLE_NAME(+) = V.VIEW_NAME
	AND V.OWNER = '%s'
ORDER BY V.OWNER, V.VIEW_NAME ASC
`
	dumpFunctionSQL = `
SELECT
	O.OBJECT_NAME,
	O.OWNER,
	O.DATA_OBJECT_ID,
	O.OBJECT_TYPE,
	O.STATUS,
	O.CREATED,
	O.LAST_DDL_TIME,
	P.AGGREGATE,
	P.PIPELINED,
	P.IMPLTYPEOWNER,
	P.IMPLTYPENAME,
	P.PARALLEL,
	P.INTERFACE,
	P.DETERMINISTIC,
	P.AUTHID,
	SS.PARAM_VALUE,
	P.OBJECT_ID,
	P.SUBPROGRAM_ID,
	P.OVERLOAD,
	O.TIMESTAMP,
	S.LINE,
	S.TEXT
FROM
	SYS.ALL_OBJECTS O,
	SYS.ALL_PROCEDURES P,
	SYS.ALL_STORED_SETTINGS SS,
	SYS.ALL_SOURCE S
WHERE
	O.OBJECT_TYPE IN ('PROCEDURE', 'FUNCTION')
	AND O.OWNER = '%s'
	AND O.OBJECT_ID NOT IN (SELECT PURGE_OBJECT FROM RECYCLEBIN)
	AND P.OWNER(+) = O.OWNER
	AND P.OBJECT_NAME(+) = O.OBJECT_NAME
	AND SS.OWNER(+) = O.OWNER
	AND SS.OBJECT_NAME(+) = O.OBJECT_NAME
	AND SS.PARAM_NAME(+) = 'plsql_debug'
	AND O.OWNER(+) = S.OWNER
	AND O.OBJECT_NAME(+) = S.NAME
	AND O.OBJECT_TYPE(+) = S.TYPE
ORDER BY O.OBJECT_NAME ASC, S.LINE
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
	SYS.DBA_INDEXES I,
	SYS.DBA_IND_COLUMNS IC,
	SYS.DBA_IND_EXPRESSIONS IE,
	SYS.DBA_CONSTRAINTS C
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
	dumpSequenceSQL = `
SELECT
	SEQUENCE_NAME,
	MIN_VALUE,
	MAX_VALUE,
	INCREMENT_BY,
	CYCLE_FLAG,
	ORDER_FLAG,
	CACHE_SIZE,
	LAST_NUMBER,
	KEEP_VALUE,
	SESSION_FLAG
FROM
	SYS.ALL_SEQUENCES
WHERE
	SEQUENCE_OWNER = '%s'
ORDER BY SEQUENCE_NAME ASC
`
	dumpTriggerOrderingSQL = `
SELECT
	ATO.TRIGGER_OWNER,
	ATO.TRIGGER_NAME,
	ATO.REFERENCED_TRIGGER_OWNER AS REFERENCED_SCHEMA,
	ATO.REFERENCED_TRIGGER_NAME AS REFERENCED_TRIGGER,
	ATO.ORDERING_TYPE
FROM
	ALL_TRIGGER_ORDERING ATO
WHERE
	ATO.TRIGGER_OWNER = '%s'
`
	dumpTriggerSQL = `
SELECT
	AT.OWNER,
	AT.TRIGGER_NAME,
	AT.TRIGGER_TYPE,
	AT.TRIGGERING_EVENT,
	AT.TABLE_OWNER,
	AT.BASE_OBJECT_TYPE,
	AT.TABLE_NAME,
	AT.COLUMN_NAME AS NESTED_COLUMN,
	AT.REFERENCING_NAMES,
	AT.WHEN_CLAUSE,
	AT.STATUS AS IS_ENABLE,
	AT.DESCRIPTION,
	AT.TRIGGER_BODY,
	AT.ACTION_TYPE,
	AT.CROSSEDITION AS EDITION,
	ATC.COLUMN_NAME,
	(
		SELECT
			T.IOT_TYPE
		FROM
			SYS.ALL_TABLES T
		WHERE
			AT.TABLE_OWNER = T.OWNER
			AND AT.TABLE_NAME = T.TABLE_NAME
	) AS IOT_TYPE,
	(
		SELECT
			S.PARAM_VALUE
		FROM
			SYS.ALL_STORED_SETTINGS S
		WHERE
			S.OBJECT_TYPE = 'TRIGGER'
			AND S.PARAM_NAME = 'plsql_debug'
			AND S.OWNER = AT.OWNER
			AND S.OBJECT_NAME = AT.TRIGGER_NAME
	) AS DEBUG,
	(
		SELECT
			O.STATUS
		FROM
			SYS.ALL_OBJECTS O
		WHERE
			O.OWNER = AT.OWNER
			AND O.OBJECT_NAME = AT.TRIGGER_NAME
			AND O.OBJECT_TYPE = 'TRIGGER'
			AND O.SUBOBJECT_NAME IS NULL
	) AS OBJECT_STATUS
FROM
	SYS.ALL_TRIGGERS AT,
	SYS.ALL_TRIGGER_COLS ATC
WHERE
	AT.OWNER = ATC.TRIGGER_OWNER(+)
	AND AT.TRIGGER_NAME = ATC.TRIGGER_NAME(+)
	AND AT.TABLE_OWNER = ATC.TABLE_OWNER(+)
	AND AT.TABLE_NAME =  ATC.TABLE_NAME(+)
	AND ATC.COLUMN_LIST(+) = 'YES'
	AND AT.OWNER = '%s'
ORDER BY AT.TABLE_OWNER, AT.TABLE_NAME, AT.TRIGGER_NAME, ATC.COLUMN_NAME ASC
`
)

func dumpTableTxn(ctx context.Context, txn *sql.Tx, schema string, out io.Writer) error {
	tableMap := make(map[string]*tableSchema)
	tableRows, err := txn.QueryContext(ctx, fmt.Sprintf(dumpTableSQL, schema))
	if err != nil {
		return err
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
			&meta.ObjectIdType,
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
		return err
	}
	defer fieldRows.Close()
	for fieldRows.Next() {
		fields := fieldMeta{}
		// (help-wanted) Sadly, go-ora with struct tag does not work.
		if err := fieldRows.Scan(
			&fields.IotType,
			&fields.ExtTableName,
			&fields.TableName,
			&fields.ColumnName,
			&fields.DataType,
			&fields.DataTypeOwner,
			&fields.DataLength,
			&fields.DataPrecision,
			&fields.DataScale,
			&fields.Nullable,
			&fields.ColumnId,
			&fields.DataDefault,
			&fields.CharLength,
			&fields.CharUsed,
			&fields.IsInvisible,
			&fields.Comments,
		); err != nil {
			return err
		}
		if !fields.TableName.Valid {
			continue
		}
		tableMap[fields.TableName.String].fields = append(tableMap[fields.TableName.String].fields, &fields)
	}
	if err := fieldRows.Err(); err != nil {
		return err
	}

	constraintRows, err := txn.QueryContext(ctx, fmt.Sprintf(dumpConstraintSQL, schema))
	if err != nil {
		return err
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
			continue
		}
		tableMap[constraint.TableName.String].constraints = append(tableMap[constraint.TableName.String].constraints, &constraint)
	}
	if err := constraintRows.Err(); err != nil {
		return err
	}

	// TODO: assemble CREATE TABLE
	return nil
}

func dumpViewTxn(ctx context.Context, txn *sql.Tx, schema string, out io.Writer) error {
	viewList := []*viewMeta{}
	viewRows, err := txn.QueryContext(ctx, fmt.Sprintf(dumpViewSQL, schema))
	if err != nil {
		return err
	}
	defer viewRows.Close()
	for viewRows.Next() {
		view := viewMeta{}
		// (help-wanted) Sadly, go-ora with struct tag does not work.
		if err := viewRows.Scan(
			&view.ViewName,
			&view.TextLength,
			&view.Text,
			&view.TypeTextLength,
			&view.TypeText,
			&view.OidTextLength,
			&view.OidText,
			&view.ViewTypeOwner,
			&view.ViewType,
			&view.SuperViewName,
			&view.EditioningView,
			&view.ReadOnly,
			&view.Status,
			&view.Comments,
			&view.ConstraintName,
			&view.ConstraintType,
		); err != nil {
			return err
		}
		if !view.ViewName.Valid {
			continue
		}
		viewList = append(viewList, &view)
	}
	if err := viewRows.Err(); err != nil {
		return err
	}

	// TODO: assemble CREATE VIEW
	return nil
}

func dumpFunctionTxn(ctx context.Context, txn *sql.Tx, schema string, out io.Writer) error {
	functionList := []*functionMeta{}
	functionRows, err := txn.QueryContext(ctx, fmt.Sprintf(dumpFunctionSQL, schema))
	if err != nil {
		return err
	}
	defer functionRows.Close()
	for functionRows.Next() {
		function := functionMeta{}
		// (help-wanted) Sadly, go-ora with struct tag does not work.
		if err := functionRows.Scan(
			&function.ObjectName,
			&function.Owner,
			&function.DataObjectId,
			&function.ObjectType,
			&function.Status,
			&function.Created,
			&function.LastDdlTime,
			&function.Aggregate,
			&function.Pipelined,
			&function.ImplTypeOwner,
			&function.ImplTypeName,
			&function.Parallel,
			&function.Interface,
			&function.Deterministic,
			&function.AuthId,
			&function.ParamValue,
			&function.ObjectId,
			&function.SubProgramId,
			&function.Overload,
			&function.Timestamp,
			&function.Line,
			&function.Text,
		); err != nil {
			return err
		}
		if !function.ObjectName.Valid {
			continue
		}
		functionList = append(functionList, &function)
	}
	if err := functionRows.Err(); err != nil {
		return err
	}

	// TODO: assemble CREATE FUNCTION
	return nil
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

	// TODO: assemble CREATE INDEX
	return nil
}

func dumpSequenceTxn(ctx context.Context, txn *sql.Tx, schema string, out io.Writer) error {
	sequences := []*sequenceMeta{}
	sequenceRows, err := txn.QueryContext(ctx, fmt.Sprintf(dumpSequenceSQL, schema))
	if err != nil {
		return err
	}
	defer sequenceRows.Close()
	for sequenceRows.Next() {
		sequence := sequenceMeta{}
		// (help-wanted) Sadly, go-ora with struct tag does not work.
		if err := sequenceRows.Scan(
			&sequence.SequenceName,
			&sequence.MinValue,
			&sequence.MaxValue,
			&sequence.IncrementBy,
			&sequence.CycleFlag,
			&sequence.OrderFlag,
			&sequence.CacheSize,
			&sequence.LastNumber,
			&sequence.KeepValue,
			&sequence.SessionFlag,
		); err != nil {
			return err
		}
		if !sequence.SequenceName.Valid {
			continue
		}
		sequences = append(sequences, &sequence)
	}
	if err := sequenceRows.Err(); err != nil {
		return err
	}

	// TODO: assemble CREATE SEQUENCE
	return nil
}

func dumpTriggerOrderingTxn(ctx context.Context, txn *sql.Tx, schema string, out io.Writer) error {
	triggerOrderingMap := make(map[string]*triggerOrderingMeta)
	triggerOrderingRows, err := txn.QueryContext(ctx, fmt.Sprintf(dumpTriggerOrderingSQL, schema))
	if err != nil {
		return err
	}
	defer triggerOrderingRows.Close()
	for triggerOrderingRows.Next() {
		triggerOrdering := triggerOrderingMeta{}
		if err := triggerOrderingRows.Scan(
			&triggerOrdering.TriggerOwner,
			&triggerOrdering.TriggerName,
			&triggerOrdering.ReferencedSchema,
			&triggerOrdering.ReferencedTrigger,
			&triggerOrdering.OrderingType,
		); err != nil {
			return err
		}
		if !triggerOrdering.TriggerName.Valid {
			continue
		}
		triggerOrderingMap[triggerOrdering.TriggerName.String] = &triggerOrdering
	}
	if err := triggerOrderingRows.Err(); err != nil {
		return err
	}

	triggers := []*triggerMeta{}
	triggerRows, err := txn.QueryContext(ctx, fmt.Sprintf(dumpTriggerSQL, schema))
	if err != nil {
		return err
	}
	defer triggerRows.Close()
	for triggerRows.Next() {
		trigger := triggerMeta{}
		if err := triggerRows.Scan(
			&trigger.Owner,
			&trigger.TriggerName,
			&trigger.TriggerType,
			&trigger.TriggerEvent,
			&trigger.BaseObjectType,
			&trigger.TableName,
			&trigger.NestedColumn,
			&trigger.ReferencingNames,
			&trigger.WhenClause,
			&trigger.IsEnable,
			&trigger.Description,
			&trigger.TriggerBody,
			&trigger.ActionType,
			&trigger.Edition,
			&trigger.ColumnName,
			&trigger.IotType,
			&trigger.Debug,
			&trigger.ObjectStatus,
		); err != nil {
			return err
		}
		if !trigger.TriggerName.Valid {
			continue
		}
		triggers = append(triggers, &trigger)
	}
	if err := triggerRows.Err(); err != nil {
		return err
	}

	// TODO: assemble CREATE TRIGGER
	return nil
}

// Restore restores a database.
func (*Driver) Restore(_ context.Context, _ io.Reader) (err error) {
	// TODO(d): implement it.
	return nil
}
