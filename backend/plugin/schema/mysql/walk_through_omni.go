package mysql

import (
	"fmt"
	"slices"
	"strings"

	"github.com/bytebase/omni/mysql/catalog"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	schema.RegisterWalkThroughWithContext(storepb.Engine_MYSQL, WalkThroughOmni)
	schema.RegisterWalkThroughWithContext(storepb.Engine_MARIADB, WalkThroughOmni)
	schema.RegisterWalkThroughWithContext(storepb.Engine_OCEANBASE, WalkThroughOmni)
}

// WalkThroughOmni performs DDL simulation using omni catalog.Exec().
// Flow:
//  1. GetDatabaseDefinition(metadata) → schemaDDL
//  2. catalog.Exec(schemaDDL) → load existing schema state
//  3. catalog.Exec(userSQL) → execute user DDL
//  4. Map errors → *storepb.Advice
//  5. Convert updated catalog → DatabaseMetadata (for downstream rules)
func WalkThroughOmni(ctx schema.WalkThroughContext, d *model.DatabaseMetadata, _ []base.AST) *storepb.Advice {
	if ctx.RawSQL == "" {
		return nil
	}

	dbName := d.GetProto().GetName()

	// Step 1: Generate DDL from current schema state.
	schemaDDL, err := schema.GetDatabaseDefinition(
		storepb.Engine_MYSQL,
		schema.GetDefinitionContext{},
		d.GetProto(),
	)
	if err != nil {
		return &storepb.Advice{
			Status:        storepb.Advice_ERROR,
			Code:          code.DDLSimulationFailed.Int32(),
			Title:         "Failed to generate schema DDL",
			Content:       err.Error(),
			StartPosition: &storepb.Position{Line: 0},
		}
	}

	// Step 2: Create catalog, create database, and load existing schema.
	c := catalog.New()
	// Disable FK checks so that tables can reference not-yet-loaded tables,
	// matching standard MySQL behavior for schema loading.
	initSQL := fmt.Sprintf("SET foreign_key_checks = 0;\nCREATE DATABASE IF NOT EXISTS `%s`;\nUSE `%s`;", dbName, dbName)
	if schemaDDL != "" {
		initSQL += "\n" + schemaDDL
	}
	if _, err := c.Exec(initSQL, &catalog.ExecOptions{ContinueOnError: true}); err != nil {
		return &storepb.Advice{
			Status:        storepb.Advice_ERROR,
			Code:          code.DDLSimulationFailed.Int32(),
			Title:         "Failed to load schema DDL",
			Content:       err.Error(),
			StartPosition: &storepb.Position{Line: 0},
		}
	}

	// Step 3: Execute user SQL.
	results, execErr := c.Exec(ctx.RawSQL, &catalog.ExecOptions{ContinueOnError: true})
	if execErr != nil {
		return &storepb.Advice{
			Status:        storepb.Advice_ERROR,
			Code:          code.DDLSimulationFailed.Int32(),
			Title:         "DDL simulation failed",
			Content:       execErr.Error(),
			StartPosition: &storepb.Position{Line: 0},
		}
	}

	// Step 4: Report the first error from the simulation.
	for _, r := range results {
		if r.Error == nil {
			continue
		}
		errCode := mapMySQLErrorToCode(r.Error)
		content := r.Error.Error()
		if catErr, ok := r.Error.(*catalog.Error); ok {
			content = catErr.Message
		}
		return &storepb.Advice{
			Status:  storepb.Advice_ERROR,
			Code:    errCode.Int32(),
			Title:   "DDL simulation failed",
			Content: content,
			StartPosition: &storepb.Position{
				Line: int32(r.Line),
			},
		}
	}

	// Step 5: Convert catalog state back to DatabaseMetadata for downstream rules.
	newProto := catalogToProto(c, dbName)
	newMetadata := model.NewDatabaseMetadata(newProto, nil, d.GetConfig(), storepb.Engine_MYSQL, true)
	d.ReplaceFrom(newMetadata)

	return nil
}

// mapMySQLErrorToCode converts an omni MySQL catalog error to a bytebase error code.
func mapMySQLErrorToCode(err error) code.Code {
	catErr, ok := err.(*catalog.Error)
	if !ok {
		return code.DDLSimulationFailed
	}

	switch catErr.Code {
	case catalog.ErrDupDatabase:
		return code.DDLSimulationFailed
	case catalog.ErrUnknownDatabase:
		return code.DatabaseNotExists
	case catalog.ErrDupTable:
		return code.TableExists
	case catalog.ErrUnknownTable, catalog.ErrNoSuchTable:
		return code.TableNotExists
	case catalog.ErrDupColumn:
		return code.ColumnExists
	case catalog.ErrNoSuchColumn:
		return code.ColumnNotExists
	case catalog.ErrDupKeyName, catalog.ErrDupIndex:
		return code.IndexExists
	case catalog.ErrMultiplePriKey:
		return code.PrimaryKeyExists
	case catalog.ErrDupEntry:
		return code.IndexExists
	case catalog.ErrCantDropKey:
		return code.IndexNotExists
	case catalog.ErrFKNoRefTable:
		return code.TableHasFK
	case catalog.ErrFKCannotDropParent:
		return code.TableHasFK
	case catalog.ErrFKMissingIndex, catalog.ErrFKIncompatibleColumns:
		return code.TableHasFK
	case catalog.ErrCheckConstraintViolated:
		return code.DDLSimulationFailed
	default:
		return code.DDLSimulationFailed
	}
}

// catalogToProto converts the omni catalog state to a storepb.DatabaseSchemaMetadata.
func catalogToProto(c *catalog.Catalog, dbName string) *storepb.DatabaseSchemaMetadata {
	dbMeta := &storepb.DatabaseSchemaMetadata{
		Name: dbName,
	}

	db := c.GetDatabase(dbName)
	if db == nil {
		return dbMeta
	}

	dbMeta.CharacterSet = db.Charset
	dbMeta.Collation = db.Collation

	// MySQL uses a single empty-name schema.
	schemaMeta := &storepb.SchemaMetadata{
		Name: "",
	}

	// Tables.
	tableNames := make([]string, 0, len(db.Tables))
	for name := range db.Tables {
		tableNames = append(tableNames, name)
	}
	slices.Sort(tableNames)

	for _, tName := range tableNames {
		t := db.Tables[tName]
		schemaMeta.Tables = append(schemaMeta.Tables, tableToProto(t))
	}

	// Views.
	viewNames := make([]string, 0, len(db.Views))
	for name := range db.Views {
		viewNames = append(viewNames, name)
	}
	slices.Sort(viewNames)

	for _, vName := range viewNames {
		v := db.Views[vName]
		schemaMeta.Views = append(schemaMeta.Views, &storepb.ViewMetadata{
			Name:       v.Name,
			Definition: v.Definition,
		})
	}

	// Functions.
	funcNames := make([]string, 0, len(db.Functions))
	for name := range db.Functions {
		funcNames = append(funcNames, name)
	}
	slices.Sort(funcNames)

	for _, fName := range funcNames {
		f := db.Functions[fName]
		schemaMeta.Functions = append(schemaMeta.Functions, routineToFunctionProto(f))
	}

	// Procedures.
	procNames := make([]string, 0, len(db.Procedures))
	for name := range db.Procedures {
		procNames = append(procNames, name)
	}
	slices.Sort(procNames)

	for _, pName := range procNames {
		p := db.Procedures[pName]
		schemaMeta.Procedures = append(schemaMeta.Procedures, routineToProcedureProto(p))
	}

	dbMeta.Schemas = append(dbMeta.Schemas, schemaMeta)
	return dbMeta
}

func tableToProto(t *catalog.Table) *storepb.TableMetadata {
	table := &storepb.TableMetadata{
		Name:      t.Name,
		Engine:    t.Engine,
		Collation: t.Collation,
		Comment:   t.Comment,
	}
	if t.Charset != "" {
		table.Charset = t.Charset
	}

	// Columns.
	for _, col := range t.Columns {
		colMeta := &storepb.ColumnMetadata{
			Name:     col.Name,
			Position: int32(col.Position),
			Type:     col.ColumnType,
			Nullable: col.Nullable,
			Comment:  col.Comment,
		}
		if col.Default != nil {
			colMeta.Default = *col.Default
		}
		if col.OnUpdate != "" {
			colMeta.OnUpdate = col.OnUpdate
		}
		if col.Generated != nil {
			genType := storepb.GenerationMetadata_TYPE_VIRTUAL
			if col.Generated.Stored {
				genType = storepb.GenerationMetadata_TYPE_STORED
			}
			colMeta.Generation = &storepb.GenerationMetadata{
				Type:       genType,
				Expression: col.Generated.Expr,
			}
		}
		if col.Charset != "" {
			colMeta.CharacterSet = col.Charset
		}
		if col.Collation != "" {
			colMeta.Collation = col.Collation
		}
		table.Columns = append(table.Columns, colMeta)
	}

	// Indexes.
	for _, idx := range t.Indexes {
		idxMeta := &storepb.IndexMetadata{
			Name:    idx.Name,
			Type:    strings.ToUpper(idx.IndexType),
			Unique:  idx.Unique,
			Primary: idx.Primary,
			Visible: idx.Visible,
			Comment: idx.Comment,
		}
		for _, col := range idx.Columns {
			if col.Expr != "" {
				idxMeta.Expressions = append(idxMeta.Expressions, col.Expr)
			} else {
				idxMeta.Expressions = append(idxMeta.Expressions, col.Name)
			}
			idxMeta.KeyLength = append(idxMeta.KeyLength, int64(col.Length))
			idxMeta.Descending = append(idxMeta.Descending, col.Descending)
		}
		table.Indexes = append(table.Indexes, idxMeta)
	}

	// Foreign keys from constraints.
	for _, con := range t.Constraints {
		switch con.Type {
		case catalog.ConForeignKey:
			fk := &storepb.ForeignKeyMetadata{
				Name:              con.Name,
				Columns:           con.Columns,
				ReferencedTable:   con.RefTable,
				ReferencedColumns: con.RefColumns,
				OnUpdate:          con.OnUpdate,
				OnDelete:          con.OnDelete,
				MatchType:         con.MatchType,
			}
			if con.RefDatabase != "" {
				fk.ReferencedSchema = con.RefDatabase
			}
			table.ForeignKeys = append(table.ForeignKeys, fk)
		case catalog.ConCheck:
			table.CheckConstraints = append(table.CheckConstraints, &storepb.CheckConstraintMetadata{
				Name:       con.Name,
				Expression: con.CheckExpr,
			})
		default:
		}
	}

	// Partition info.
	if t.Partitioning != nil {
		table.Partitions = partitionsToProto(t.Partitioning)
	}

	return table
}

func partitionsToProto(p *catalog.PartitionInfo) []*storepb.TablePartitionMetadata {
	var result []*storepb.TablePartitionMetadata
	for _, pd := range p.Partitions {
		result = append(result, &storepb.TablePartitionMetadata{
			Name:       pd.Name,
			Type:       partitionTypeToProto(p.Type),
			Expression: p.Expr,
			Value:      pd.ValueExpr,
		})
	}
	return result
}

func partitionTypeToProto(t string) storepb.TablePartitionMetadata_Type {
	switch strings.ToUpper(t) {
	case "RANGE":
		return storepb.TablePartitionMetadata_RANGE
	case "RANGE COLUMNS":
		return storepb.TablePartitionMetadata_RANGE_COLUMNS
	case "LIST":
		return storepb.TablePartitionMetadata_LIST
	case "LIST COLUMNS":
		return storepb.TablePartitionMetadata_LIST_COLUMNS
	case "HASH":
		return storepb.TablePartitionMetadata_HASH
	case "LINEAR HASH":
		return storepb.TablePartitionMetadata_LINEAR_HASH
	case "KEY":
		return storepb.TablePartitionMetadata_KEY
	case "LINEAR KEY":
		return storepb.TablePartitionMetadata_LINEAR_KEY
	default:
		return storepb.TablePartitionMetadata_TYPE_UNSPECIFIED
	}
}

func routineToFunctionProto(r *catalog.Routine) *storepb.FunctionMetadata {
	return &storepb.FunctionMetadata{
		Name:       r.Name,
		Definition: r.Body,
	}
}

func routineToProcedureProto(r *catalog.Routine) *storepb.ProcedureMetadata {
	return &storepb.ProcedureMetadata{
		Name:       r.Name,
		Definition: r.Body,
	}
}
