package pg

import (
	"fmt"
	"slices"

	"github.com/bytebase/omni/pg/catalog"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

// WalkThroughOmni performs DDL simulation using omni catalog.Exec().
// It replaces the ANTLR-based walkthrough with the following flow:
//  1. GetDatabaseDefinition(metadata) → schemaDDL
//  2. catalog.Exec(schemaDDL) → load existing schema state
//  3. catalog.Exec(userSQL) → execute user DDL
//  4. Map errors → *storepb.Advice
//  5. Convert updated catalog → DatabaseMetadata (for downstream rules)
func WalkThroughOmni(ctx schema.WalkThroughContext, d *model.DatabaseMetadata, _ []base.AST) *storepb.Advice {
	if ctx.RawSQL == "" {
		return nil
	}

	// Step 1: Generate DDL from current schema state.
	schemaDDL, err := schema.GetDatabaseDefinition(
		storepb.Engine_POSTGRES,
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

	// Step 2: Create catalog and load existing schema.
	c := catalog.New()
	if ctx.SessionUser != "" {
		c.SetSessionUser(ctx.SessionUser)
	}

	// Set initial search path from metadata config.
	if searchPath := getConfiguredSearchPath(d); len(searchPath) > 0 {
		c.SetSearchPath(searchPath)
	}

	if schemaDDL != "" {
		// Load existing schema. Use ContinueOnError so partial schemas
		// (e.g. columns without types in metadata) still load what they can.
		if _, err := c.Exec(schemaDDL, &catalog.ExecOptions{ContinueOnError: true}); err != nil {
			return &storepb.Advice{
				Status:        storepb.Advice_ERROR,
				Code:          code.DDLSimulationFailed.Int32(),
				Title:         "Failed to load schema DDL",
				Content:       err.Error(),
				StartPosition: &storepb.Position{Line: 0},
			}
		}
	}

	// Step 3: Execute user SQL with ContinueOnError so that downstream
	// rules can check FinalMetadata even when some statements fail.
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
		errCode := mapSQLSTATEToCode(r.Error)
		return &storepb.Advice{
			Status:  storepb.Advice_ERROR,
			Code:    errCode.Int32(),
			Title:   "DDL simulation failed",
			Content: r.Error.Error(),
			StartPosition: &storepb.Position{
				Line: int32(r.Line),
			},
		}
	}

	// Step 5: Convert catalog state back to DatabaseMetadata for downstream rules.
	newProto := catalogToProto(c)
	extractViewDependencies(newProto)
	newMetadata := model.NewDatabaseMetadata(newProto, nil, d.GetConfig(), storepb.Engine_POSTGRES, true)
	d.ReplaceFrom(newMetadata)

	return nil
}

// mapSQLSTATEToCode converts an omni catalog error to a bytebase error code.
func mapSQLSTATEToCode(err error) code.Code {
	catErr, ok := err.(*catalog.Error)
	if !ok {
		return code.DDLSimulationFailed
	}

	switch catErr.Code {
	// Table/relation errors
	case catalog.CodeDuplicateTable:
		return code.TableExists
	case catalog.CodeUndefinedTable:
		return code.TableNotExists

	// Column errors
	case catalog.CodeDuplicateColumn:
		return code.ColumnExists
	case catalog.CodeUndefinedColumn, catalog.CodeInvalidColumnDefinition:
		return code.ColumnNotExists

	// Schema errors
	case catalog.CodeDuplicateSchema:
		return code.RelationExists
	case catalog.CodeUndefinedSchema:
		return code.SchemaNotExists

	// Index/constraint errors
	case catalog.CodeDuplicateObject, catalog.CodeUniqueViolation:
		return code.IndexExists
	case catalog.CodeUndefinedObject:
		return code.IndexNotExists

	// FK errors
	case catalog.CodeInvalidFK, catalog.CodeForeignKeyViolation:
		return code.TableHasFK

	// PK/table definition errors
	case catalog.CodeDuplicatePKey:
		return code.PrimaryKeyExists

	// Type/expression errors
	case catalog.CodeDatatypeMismatch:
		return code.ChangeColumnType
	case catalog.CodeNotNullViolation:
		return code.ColumnCannotNull

	// Syntax
	case catalog.CodeSyntaxError:
		return code.StatementSyntaxError

	// Naming
	case catalog.CodeReservedName:
		return code.NameIsKeywordIdentifier

	default:
		return code.DDLSimulationFailed
	}
}

func getConfiguredSearchPath(d *model.DatabaseMetadata) []string {
	configured := d.GetConfiguredSearchPath()
	if len(configured) == 0 {
		return nil
	}

	searchPath := make([]string, 0, len(configured))
	for _, item := range configured {
		if item.CurrentUser {
			searchPath = append(searchPath, "$user")
			continue
		}
		if item.Schema == "" {
			continue
		}
		searchPath = append(searchPath, item.Schema)
	}
	return searchPath
}

// catalogToProto converts the omni catalog state to a storepb.DatabaseSchemaMetadata proto.
func catalogToProto(c *catalog.Catalog) *storepb.DatabaseSchemaMetadata {
	dbMeta := &storepb.DatabaseSchemaMetadata{
		Name: "postgres",
	}

	for _, s := range c.UserSchemas() {
		schemaMeta := &storepb.SchemaMetadata{
			Name: s.Name,
		}

		// Convert tables, views, materialized views.
		// Sort relations for deterministic output.
		relNames := make([]string, 0, len(s.Relations))
		for name := range s.Relations {
			relNames = append(relNames, name)
		}
		slices.Sort(relNames)

		for _, relName := range relNames {
			rel := s.Relations[relName]
			switch rel.RelKind {
			case 'r', 'p', 'f': // table, partitioned table, foreign table
				schemaMeta.Tables = append(schemaMeta.Tables, relationToTableProto(c, rel))
			case 'v':
				schemaMeta.Views = append(schemaMeta.Views, relationToViewProto(c, rel))
			case 'm':
				schemaMeta.MaterializedViews = append(schemaMeta.MaterializedViews, relationToMatViewProto(c, rel))
			default:
			}
		}

		// Convert sequences.
		seqNames := make([]string, 0, len(s.Sequences))
		for name := range s.Sequences {
			seqNames = append(seqNames, name)
		}
		slices.Sort(seqNames)

		for _, seqName := range seqNames {
			seq := s.Sequences[seqName]
			schemaMeta.Sequences = append(schemaMeta.Sequences, &storepb.SequenceMetadata{
				Name:      seq.Name,
				DataType:  c.FormatType(seq.TypeOID, -1),
				Start:     fmt.Sprintf("%d", seq.Start),
				MinValue:  fmt.Sprintf("%d", seq.MinValue),
				MaxValue:  fmt.Sprintf("%d", seq.MaxValue),
				Increment: fmt.Sprintf("%d", seq.Increment),
				Cycle:     seq.Cycle,
				CacheSize: fmt.Sprintf("%d", seq.CacheValue),
			})
		}

		dbMeta.Schemas = append(dbMeta.Schemas, schemaMeta)
	}

	return dbMeta
}

// relationToTableProto converts an omni Relation to a storepb.TableMetadata.
func relationToTableProto(c *catalog.Catalog, rel *catalog.Relation) *storepb.TableMetadata {
	table := &storepb.TableMetadata{
		Name: rel.Name,
	}

	// Columns.
	for _, col := range rel.Columns {
		colMeta := &storepb.ColumnMetadata{
			Name:     col.Name,
			Position: int32(col.AttNum),
			Type:     c.FormatType(col.TypeOID, col.TypeMod),
			Nullable: !col.NotNull,
			Default:  col.Default,
		}
		if col.Generated == 's' {
			colMeta.Generation = &storepb.GenerationMetadata{
				Type:       storepb.GenerationMetadata_TYPE_STORED,
				Expression: col.GenerationExpr,
			}
		}
		if col.Identity != 0 {
			colMeta.IsIdentity = true
			switch col.Identity {
			case 'a':
				colMeta.IdentityGeneration = storepb.ColumnMetadata_ALWAYS
			case 'd':
				colMeta.IdentityGeneration = storepb.ColumnMetadata_BY_DEFAULT
			default:
			}
		}
		table.Columns = append(table.Columns, colMeta)
	}

	// Indexes.
	for _, idx := range c.IndexesOf(rel.OID) {
		idxMeta := &storepb.IndexMetadata{
			Name:    idx.Name,
			Type:    idx.AccessMethod,
			Unique:  idx.IsUnique,
			Primary: idx.IsPrimary,
		}

		// Build expressions list.
		exprIdx := 0
		for i, attnum := range idx.Columns {
			if attnum == 0 {
				// Expression column.
				if exprIdx < len(idx.Exprs) {
					idxMeta.Expressions = append(idxMeta.Expressions, idx.Exprs[exprIdx])
					exprIdx++
				}
			} else {
				// Regular column — find name.
				colName := ""
				for _, col := range rel.Columns {
					if col.AttNum == attnum {
						colName = col.Name
						break
					}
				}
				idxMeta.Expressions = append(idxMeta.Expressions, colName)
			}

			// Descending flag.
			if i < len(idx.IndOption) {
				idxMeta.Descending = append(idxMeta.Descending, idx.IndOption[i]&1 != 0)
			}
		}

		// IsConstraint: index backs a constraint if ConstraintOID != 0.
		if idx.ConstraintOID != 0 {
			idxMeta.IsConstraint = true
		}

		table.Indexes = append(table.Indexes, idxMeta)
	}

	// Foreign keys and check constraints from constraints.
	for _, con := range c.ConstraintsOf(rel.OID) {
		switch con.Type {
		case 'f': // FK
			fk := &storepb.ForeignKeyMetadata{
				Name: con.Name,
			}
			// Columns.
			for _, attnum := range con.Columns {
				for _, col := range rel.Columns {
					if col.AttNum == attnum {
						fk.Columns = append(fk.Columns, col.Name)
						break
					}
				}
			}
			// Referenced table.
			fRel := c.GetRelationByOID(con.FRelOID)
			if fRel != nil {
				if fRel.Schema != nil {
					fk.ReferencedSchema = fRel.Schema.Name
				}
				fk.ReferencedTable = fRel.Name
				for _, attnum := range con.FColumns {
					for _, col := range fRel.Columns {
						if col.AttNum == attnum {
							fk.ReferencedColumns = append(fk.ReferencedColumns, col.Name)
							break
						}
					}
				}
			}
			// Actions.
			fk.OnUpdate = fkActionToString(con.FKUpdAction)
			fk.OnDelete = fkActionToString(con.FKDelAction)
			fk.MatchType = fkMatchToString(con.FKMatchType)
			table.ForeignKeys = append(table.ForeignKeys, fk)
		case 'c': // CHECK
			table.CheckConstraints = append(table.CheckConstraints, &storepb.CheckConstraintMetadata{
				Name:       con.Name,
				Expression: con.CheckExpr,
			})
		case 'x': // EXCLUDE
			table.ExcludeConstraints = append(table.ExcludeConstraints, &storepb.ExcludeConstraintMetadata{
				Name:       con.Name,
				Expression: con.CheckExpr,
			})
		default:
		}
	}

	return table
}

// relationToViewProto converts an omni Relation (view) to a storepb.ViewMetadata.
func relationToViewProto(c *catalog.Catalog, rel *catalog.Relation) *storepb.ViewMetadata {
	view := &storepb.ViewMetadata{
		Name: rel.Name,
	}
	if rel.Schema != nil {
		if def, err := c.GetViewDefinition(rel.Schema.Name, rel.Name); err == nil {
			view.Definition = def
		}
	}
	return view
}

// relationToMatViewProto converts an omni Relation (materialized view) to a storepb.MaterializedViewMetadata.
func relationToMatViewProto(c *catalog.Catalog, rel *catalog.Relation) *storepb.MaterializedViewMetadata {
	mv := &storepb.MaterializedViewMetadata{
		Name: rel.Name,
	}
	if rel.Schema != nil {
		if def, err := c.GetMatViewDefinition(rel.Schema.Name, rel.Name); err == nil {
			mv.Definition = def
		}
	}

	// Materialized view indexes.
	for _, idx := range c.IndexesOf(rel.OID) {
		idxMeta := &storepb.IndexMetadata{
			Name:    idx.Name,
			Type:    idx.AccessMethod,
			Unique:  idx.IsUnique,
			Primary: idx.IsPrimary,
		}
		exprIdx := 0
		for i, attnum := range idx.Columns {
			if attnum == 0 {
				if exprIdx < len(idx.Exprs) {
					idxMeta.Expressions = append(idxMeta.Expressions, idx.Exprs[exprIdx])
					exprIdx++
				}
			} else {
				for _, col := range rel.Columns {
					if col.AttNum == attnum {
						idxMeta.Expressions = append(idxMeta.Expressions, col.Name)
						break
					}
				}
			}

			// Descending flag.
			if i < len(idx.IndOption) {
				idxMeta.Descending = append(idxMeta.Descending, idx.IndOption[i]&1 != 0)
			}
		}

		if idx.ConstraintOID != 0 {
			idxMeta.IsConstraint = true
		}

		mv.Indexes = append(mv.Indexes, idxMeta)
	}

	return mv
}

func fkActionToString(action byte) string {
	switch action {
	case 'a':
		return "NO ACTION"
	case 'r':
		return "RESTRICT"
	case 'c':
		return "CASCADE"
	case 'n':
		return "SET NULL"
	case 'd':
		return "SET DEFAULT"
	default:
		return "NO ACTION"
	}
}

func fkMatchToString(match byte) string {
	switch match {
	case 'f':
		return "FULL"
	case 'p':
		return "PARTIAL"
	default:
		return "SIMPLE"
	}
}
