package pg

import (
	parser "github.com/bytebase/parser/postgresql"

	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

// ForeignKeyDefWithAST holds foreign key constraint definition with its AST node for text comparison
type ForeignKeyDefWithAST struct {
	Name    string
	ASTNode parser.ITableconstraintContext
}

// CheckConstraintDefWithAST holds check constraint definition with its AST node for text comparison
type CheckConstraintDefWithAST struct {
	Name    string
	ASTNode parser.ITableconstraintContext
}

// ExcludeConstraintDefWithAST holds EXCLUDE constraint definition with its AST node for text comparison
type ExcludeConstraintDefWithAST struct {
	Name    string
	ASTNode parser.ITableconstraintContext
}

// IndexDefWithAST holds index/unique constraint definition with its AST node for text comparison
type IndexDefWithAST struct {
	Name    string
	ASTNode parser.ITableconstraintContext
}

// processForeignKeyChanges analyzes foreign key constraint changes between old and new table definitions
// Following the text-first comparison pattern for performance optimization
func processForeignKeyChanges(oldTable, newTable *parser.CreatestmtContext, currentDBSDLChunks *currentDatabaseSDLChunks, tableIdentifier string) []*schema.ForeignKeyDiff {
	if oldTable == nil || newTable == nil {
		return []*schema.ForeignKeyDiff{}
	}

	// Step 1: Extract all foreign key definitions with their AST nodes for text comparison
	oldFKList := extractForeignKeyDefinitionsInOrder(oldTable)
	newFKList := extractForeignKeyDefinitionsInOrder(newTable)

	// Create maps for quick lookup
	oldFKMap := make(map[string]*ForeignKeyDefWithAST)
	for _, def := range oldFKList {
		oldFKMap[def.Name] = def
	}
	newFKMap := make(map[string]*ForeignKeyDefWithAST)
	for _, def := range newFKList {
		newFKMap[def.Name] = def
	}

	var fkDiffs []*schema.ForeignKeyDiff

	// Step 2: Process current foreign keys to find created and modified foreign keys
	for _, newFKDef := range newFKList {
		if oldFKDef, exists := oldFKMap[newFKDef.Name]; exists {
			// FK exists in both - check if modified by comparing text first
			currentText := getForeignKeyText(newFKDef.ASTNode)
			previousText := getForeignKeyText(oldFKDef.ASTNode)
			if currentText != previousText {
				// Apply constraint-level usability check: skip diff if current constraint matches database metadata SDL
				schemaName, tableName := parseIdentifier(tableIdentifier)
				if currentDBSDLChunks != nil && currentDBSDLChunks.shouldSkipConstraintDiffForUsability(currentText, schemaName, tableName, newFKDef.Name) {
					continue
				}
				// FK was modified - drop and recreate (PostgreSQL pattern)
				fkDiffs = append(fkDiffs, &schema.ForeignKeyDiff{
					Action:     schema.MetadataDiffActionDrop,
					OldASTNode: oldFKDef.ASTNode,
				})
				fkDiffs = append(fkDiffs, &schema.ForeignKeyDiff{
					Action:     schema.MetadataDiffActionCreate,
					NewASTNode: newFKDef.ASTNode,
				})
			}
		} else {
			// New foreign key - store AST node only
			fkDiffs = append(fkDiffs, &schema.ForeignKeyDiff{
				Action:     schema.MetadataDiffActionCreate,
				NewASTNode: newFKDef.ASTNode,
			})
		}
	}

	// Step 3: Process old foreign keys to find dropped ones
	for _, oldFKDef := range oldFKList {
		if _, exists := newFKMap[oldFKDef.Name]; !exists {
			// Foreign key was dropped - store AST node only
			fkDiffs = append(fkDiffs, &schema.ForeignKeyDiff{
				Action:     schema.MetadataDiffActionDrop,
				OldASTNode: oldFKDef.ASTNode,
			})
		}
	}

	return fkDiffs
}

// processCheckConstraintChanges analyzes check constraint changes between old and new table definitions
// Following the text-first comparison pattern for performance optimization
func processCheckConstraintChanges(oldTable, newTable *parser.CreatestmtContext, currentDBSDLChunks *currentDatabaseSDLChunks, tableIdentifier string) []*schema.CheckConstraintDiff {
	if oldTable == nil || newTable == nil {
		return []*schema.CheckConstraintDiff{}
	}

	// Step 1: Extract all check constraint definitions with their AST nodes for text comparison
	oldCheckList := extractCheckConstraintDefinitionsInOrder(oldTable)
	newCheckList := extractCheckConstraintDefinitionsInOrder(newTable)

	// Create maps for quick lookup
	oldCheckMap := make(map[string]*CheckConstraintDefWithAST)
	for _, def := range oldCheckList {
		oldCheckMap[def.Name] = def
	}
	newCheckMap := make(map[string]*CheckConstraintDefWithAST)
	for _, def := range newCheckList {
		newCheckMap[def.Name] = def
	}

	var checkDiffs []*schema.CheckConstraintDiff

	// Step 2: Process current check constraints to find created and modified check constraints
	for _, newCheckDef := range newCheckList {
		if oldCheckDef, exists := oldCheckMap[newCheckDef.Name]; exists {
			// Check constraint exists in both - check if modified by comparing text first
			currentText := getCheckConstraintText(newCheckDef.ASTNode)
			previousText := getCheckConstraintText(oldCheckDef.ASTNode)
			if currentText != previousText {
				// Apply constraint-level usability check: skip diff if current constraint matches database metadata SDL
				schemaName, tableName := parseIdentifier(tableIdentifier)
				if currentDBSDLChunks != nil && currentDBSDLChunks.shouldSkipConstraintDiffForUsability(currentText, schemaName, tableName, newCheckDef.Name) {
					continue
				}
				// Check constraint was modified - drop and recreate (PostgreSQL pattern)
				checkDiffs = append(checkDiffs, &schema.CheckConstraintDiff{
					Action:     schema.MetadataDiffActionDrop,
					OldASTNode: oldCheckDef.ASTNode,
				})
				checkDiffs = append(checkDiffs, &schema.CheckConstraintDiff{
					Action:     schema.MetadataDiffActionCreate,
					NewASTNode: newCheckDef.ASTNode,
				})
			}
		} else {
			// New check constraint - store AST node only
			checkDiffs = append(checkDiffs, &schema.CheckConstraintDiff{
				Action:     schema.MetadataDiffActionCreate,
				NewASTNode: newCheckDef.ASTNode,
			})
		}
	}

	// Step 3: Process old check constraints to find dropped ones
	for _, oldCheckDef := range oldCheckList {
		if _, exists := newCheckMap[oldCheckDef.Name]; !exists {
			// Check constraint was dropped - store AST node only
			checkDiffs = append(checkDiffs, &schema.CheckConstraintDiff{
				Action:     schema.MetadataDiffActionDrop,
				OldASTNode: oldCheckDef.ASTNode,
			})
		}
	}

	return checkDiffs
}

// processExcludeConstraintChanges analyzes EXCLUDE constraint changes between old and new table definitions
// Following the text-first comparison pattern for performance optimization
func processExcludeConstraintChanges(oldTable, newTable *parser.CreatestmtContext, currentDBSDLChunks *currentDatabaseSDLChunks, tableIdentifier string) []*schema.ExcludeConstraintDiff {
	if oldTable == nil || newTable == nil {
		return []*schema.ExcludeConstraintDiff{}
	}

	// Step 1: Extract all EXCLUDE constraint definitions with their AST nodes for text comparison
	oldExcludeList := extractExcludeConstraintDefinitionsInOrder(oldTable)
	newExcludeList := extractExcludeConstraintDefinitionsInOrder(newTable)

	// Create maps for quick lookup
	oldExcludeMap := make(map[string]*ExcludeConstraintDefWithAST)
	for _, def := range oldExcludeList {
		oldExcludeMap[def.Name] = def
	}
	newExcludeMap := make(map[string]*ExcludeConstraintDefWithAST)
	for _, def := range newExcludeList {
		newExcludeMap[def.Name] = def
	}

	var excludeDiffs []*schema.ExcludeConstraintDiff

	// Step 2: Process current EXCLUDE constraints to find created and modified EXCLUDE constraints
	for _, newExcludeDef := range newExcludeList {
		if oldExcludeDef, exists := oldExcludeMap[newExcludeDef.Name]; exists {
			// EXCLUDE constraint exists in both - check if modified by comparing text first
			currentText := getExcludeConstraintText(newExcludeDef.ASTNode)
			previousText := getExcludeConstraintText(oldExcludeDef.ASTNode)
			if currentText != previousText {
				// Apply constraint-level usability check: skip diff if current constraint matches database metadata SDL
				schemaName, tableName := parseIdentifier(tableIdentifier)
				if currentDBSDLChunks != nil && currentDBSDLChunks.shouldSkipConstraintDiffForUsability(currentText, schemaName, tableName, newExcludeDef.Name) {
					continue
				}
				// EXCLUDE constraint was modified - drop and recreate (PostgreSQL pattern)
				excludeDiffs = append(excludeDiffs, &schema.ExcludeConstraintDiff{
					Action:     schema.MetadataDiffActionDrop,
					OldASTNode: oldExcludeDef.ASTNode,
				})
				excludeDiffs = append(excludeDiffs, &schema.ExcludeConstraintDiff{
					Action:     schema.MetadataDiffActionCreate,
					NewASTNode: newExcludeDef.ASTNode,
				})
			}
		} else {
			// New EXCLUDE constraint - store AST node only
			excludeDiffs = append(excludeDiffs, &schema.ExcludeConstraintDiff{
				Action:     schema.MetadataDiffActionCreate,
				NewASTNode: newExcludeDef.ASTNode,
			})
		}
	}

	// Step 3: Process old EXCLUDE constraints to find dropped ones
	for _, oldExcludeDef := range oldExcludeList {
		if _, exists := newExcludeMap[oldExcludeDef.Name]; !exists {
			// EXCLUDE constraint was dropped - store AST node only
			excludeDiffs = append(excludeDiffs, &schema.ExcludeConstraintDiff{
				Action:     schema.MetadataDiffActionDrop,
				OldASTNode: oldExcludeDef.ASTNode,
			})
		}
	}

	return excludeDiffs
}

// processPrimaryKeyChanges analyzes primary key constraint changes between old and new table definitions
func processPrimaryKeyChanges(oldTable, newTable *parser.CreatestmtContext, currentDBSDLChunks *currentDatabaseSDLChunks, tableIdentifier string) []*schema.PrimaryKeyDiff {
	if oldTable == nil || newTable == nil {
		return []*schema.PrimaryKeyDiff{}
	}

	// Step 1: Extract all primary key constraint definitions with their AST nodes for text comparison
	oldPKMap := extractPrimaryKeyDefinitionsWithAST(oldTable)
	newPKMap := extractPrimaryKeyDefinitionsWithAST(newTable)

	var pkDiffs []*schema.PrimaryKeyDiff

	// Step 2: Process current primary keys to find created and modified primary keys
	for pkName, newPKDef := range newPKMap {
		if oldPKDef, exists := oldPKMap[pkName]; exists {
			// PK exists in both - check if modified by comparing text first
			currentText := getIndexText(newPKDef.ASTNode)
			previousText := getIndexText(oldPKDef.ASTNode)
			if currentText != previousText {
				// Apply constraint-level usability check: skip diff if current constraint matches database metadata SDL
				schemaName, tableName := parseIdentifier(tableIdentifier)
				if currentDBSDLChunks != nil && currentDBSDLChunks.shouldSkipConstraintDiffForUsability(currentText, schemaName, tableName, pkName) {
					continue
				}
				// PK was modified - store AST nodes only
				pkDiffs = append(pkDiffs, &schema.PrimaryKeyDiff{
					Action:     schema.MetadataDiffActionDrop,
					OldASTNode: oldPKDef.ASTNode,
				})
				pkDiffs = append(pkDiffs, &schema.PrimaryKeyDiff{
					Action:     schema.MetadataDiffActionCreate,
					NewASTNode: newPKDef.ASTNode,
				})
			}
		} else {
			// New PK - store AST node only
			pkDiffs = append(pkDiffs, &schema.PrimaryKeyDiff{
				Action:     schema.MetadataDiffActionCreate,
				NewASTNode: newPKDef.ASTNode,
			})
		}
	}

	// Step 3: Process old primary keys to find dropped ones
	for pkName, oldPKDef := range oldPKMap {
		if _, exists := newPKMap[pkName]; !exists {
			// PK was dropped - store AST node only
			pkDiffs = append(pkDiffs, &schema.PrimaryKeyDiff{
				Action:     schema.MetadataDiffActionDrop,
				OldASTNode: oldPKDef.ASTNode,
			})
		}
	}

	return pkDiffs
}

// processUniqueConstraintChanges analyzes unique constraint changes between old and new table definitions
func processUniqueConstraintChanges(oldTable, newTable *parser.CreatestmtContext, currentDBSDLChunks *currentDatabaseSDLChunks, tableIdentifier string) []*schema.UniqueConstraintDiff {
	if oldTable == nil || newTable == nil {
		return []*schema.UniqueConstraintDiff{}
	}

	// Step 1: Extract all unique constraint definitions with their AST nodes for text comparison
	oldUKList := extractUniqueConstraintDefinitionsInOrder(oldTable)
	newUKList := extractUniqueConstraintDefinitionsInOrder(newTable)

	// Create maps for quick lookup
	oldUKMap := make(map[string]*IndexDefWithAST)
	for _, def := range oldUKList {
		oldUKMap[def.Name] = def
	}
	newUKMap := make(map[string]*IndexDefWithAST)
	for _, def := range newUKList {
		newUKMap[def.Name] = def
	}

	var ukDiffs []*schema.UniqueConstraintDiff

	// Step 2: Process current unique constraints to find created and modified unique constraints
	for _, newUKDef := range newUKList {
		if oldUKDef, exists := oldUKMap[newUKDef.Name]; exists {
			// UK exists in both - check if modified by comparing text first
			currentText := getIndexText(newUKDef.ASTNode)
			previousText := getIndexText(oldUKDef.ASTNode)
			if currentText != previousText {
				// Apply constraint-level usability check: skip diff if current constraint matches database metadata SDL
				schemaName, tableName := parseIdentifier(tableIdentifier)
				if currentDBSDLChunks != nil && currentDBSDLChunks.shouldSkipConstraintDiffForUsability(currentText, schemaName, tableName, newUKDef.Name) {
					continue
				}
				// UK was modified - drop and recreate (PostgreSQL pattern)
				ukDiffs = append(ukDiffs, &schema.UniqueConstraintDiff{
					Action:     schema.MetadataDiffActionDrop,
					OldASTNode: oldUKDef.ASTNode,
				})
				ukDiffs = append(ukDiffs, &schema.UniqueConstraintDiff{
					Action:     schema.MetadataDiffActionCreate,
					NewASTNode: newUKDef.ASTNode,
				})
			}
		} else {
			// New UK - store AST node only
			ukDiffs = append(ukDiffs, &schema.UniqueConstraintDiff{
				Action:     schema.MetadataDiffActionCreate,
				NewASTNode: newUKDef.ASTNode,
			})
		}
	}

	// Step 3: Process old unique constraints to find dropped ones
	for _, oldUKDef := range oldUKList {
		if _, exists := newUKMap[oldUKDef.Name]; !exists {
			// UK was dropped - store AST node only
			ukDiffs = append(ukDiffs, &schema.UniqueConstraintDiff{
				Action:     schema.MetadataDiffActionDrop,
				OldASTNode: oldUKDef.ASTNode,
			})
		}
	}

	return ukDiffs
}

// extractUniqueConstraintDefinitionsInOrder extracts unique constraints with their AST nodes in SQL order
func extractUniqueConstraintDefinitionsInOrder(createStmt *parser.CreatestmtContext) []*IndexDefWithAST {
	var ukList []*IndexDefWithAST

	if createStmt == nil || createStmt.Opttableelementlist() == nil {
		return ukList
	}

	tableElementList := createStmt.Opttableelementlist().Tableelementlist()
	if tableElementList == nil {
		return ukList
	}

	for _, element := range tableElementList.AllTableelement() {
		if element.Tableconstraint() != nil {
			constraint := element.Tableconstraint()
			if constraint.Constraintelem() != nil {
				elem := constraint.Constraintelem()
				// Check for UNIQUE constraints (but not PRIMARY KEY)
				isUnique := elem.UNIQUE() != nil && (elem.PRIMARY() == nil || elem.KEY() == nil)

				if isUnique {
					// This is a unique constraint
					name := ""
					if constraint.Name() != nil {
						name = pgparser.NormalizePostgreSQLName(constraint.Name())
					}
					// Use constraint definition text as fallback key if name is empty
					if name == "" {
						// Get the full original text from tokens
						if parser := constraint.GetParser(); parser != nil {
							if tokenStream := parser.GetTokenStream(); tokenStream != nil {
								name = tokenStream.GetTextFromRuleContext(constraint)
							}
						}
						if name == "" {
							name = constraint.GetText() // Final fallback
						}
					}
					ukList = append(ukList, &IndexDefWithAST{
						Name:    name,
						ASTNode: constraint,
					})
				}
			}
		}
	}

	return ukList
}

// extractForeignKeyDefinitionsInOrder extracts foreign key constraints with their AST nodes in SQL order
func extractForeignKeyDefinitionsInOrder(createStmt *parser.CreatestmtContext) []*ForeignKeyDefWithAST {
	var fkList []*ForeignKeyDefWithAST

	if createStmt == nil || createStmt.Opttableelementlist() == nil {
		return fkList
	}

	tableElementList := createStmt.Opttableelementlist().Tableelementlist()
	if tableElementList == nil {
		return fkList
	}

	for _, element := range tableElementList.AllTableelement() {
		if element.Tableconstraint() != nil {
			constraint := element.Tableconstraint()
			if constraint.Constraintelem() != nil {
				elem := constraint.Constraintelem()
				if elem.FOREIGN() != nil && elem.KEY() != nil {
					// This is a foreign key constraint
					name := ""
					if constraint.Name() != nil {
						name = pgparser.NormalizePostgreSQLName(constraint.Name())
					}
					// Use constraint definition text as fallback key if name is empty
					if name == "" {
						// Get the full original text from tokens
						if parser := constraint.GetParser(); parser != nil {
							if tokenStream := parser.GetTokenStream(); tokenStream != nil {
								name = tokenStream.GetTextFromRuleContext(constraint)
							}
						}
						if name == "" {
							name = constraint.GetText() // Final fallback
						}
					}
					fkList = append(fkList, &ForeignKeyDefWithAST{
						Name:    name,
						ASTNode: constraint,
					})
				}
			}
		}
	}

	return fkList
}

// extractCheckConstraintDefinitionsInOrder extracts check constraints with their AST nodes in SQL order
func extractCheckConstraintDefinitionsInOrder(createStmt *parser.CreatestmtContext) []*CheckConstraintDefWithAST {
	var checkList []*CheckConstraintDefWithAST

	if createStmt == nil || createStmt.Opttableelementlist() == nil {
		return checkList
	}

	tableElementList := createStmt.Opttableelementlist().Tableelementlist()
	if tableElementList == nil {
		return checkList
	}

	for _, element := range tableElementList.AllTableelement() {
		if element.Tableconstraint() != nil {
			constraint := element.Tableconstraint()
			if constraint.Constraintelem() != nil {
				elem := constraint.Constraintelem()
				if elem.CHECK() != nil {
					// This is a check constraint
					name := ""
					if constraint.Name() != nil {
						name = pgparser.NormalizePostgreSQLName(constraint.Name())
					}
					// Use constraint definition text as fallback key if name is empty
					if name == "" {
						// Get the full original text from tokens
						if parser := constraint.GetParser(); parser != nil {
							if tokenStream := parser.GetTokenStream(); tokenStream != nil {
								name = tokenStream.GetTextFromRuleContext(constraint)
							}
						}
						if name == "" {
							name = constraint.GetText() // Final fallback
						}
					}
					checkList = append(checkList, &CheckConstraintDefWithAST{
						Name:    name,
						ASTNode: constraint,
					})
				}
			}
		}
	}

	return checkList
}

// extractExcludeConstraintDefinitionsInOrder extracts EXCLUDE constraints in their original order with AST nodes
func extractExcludeConstraintDefinitionsInOrder(createStmt *parser.CreatestmtContext) []*ExcludeConstraintDefWithAST {
	var excludeList []*ExcludeConstraintDefWithAST

	if createStmt == nil || createStmt.Opttableelementlist() == nil {
		return excludeList
	}

	tableElementList := createStmt.Opttableelementlist().Tableelementlist()
	if tableElementList == nil {
		return excludeList
	}

	for _, element := range tableElementList.AllTableelement() {
		if element.Tableconstraint() != nil {
			constraint := element.Tableconstraint()
			if constraint.Constraintelem() != nil {
				elem := constraint.Constraintelem()
				if elem.EXCLUDE() != nil {
					// This is an EXCLUDE constraint
					name := ""
					if constraint.Name() != nil {
						name = pgparser.NormalizePostgreSQLName(constraint.Name())
					}
					// Use constraint definition text as fallback key if name is empty
					if name == "" {
						// Get the full original text from tokens
						if parser := constraint.GetParser(); parser != nil {
							if tokenStream := parser.GetTokenStream(); tokenStream != nil {
								name = tokenStream.GetTextFromRuleContext(constraint)
							}
						}
						if name == "" {
							name = constraint.GetText() // Final fallback
						}
					}
					excludeList = append(excludeList, &ExcludeConstraintDefWithAST{
						Name:    name,
						ASTNode: constraint,
					})
				}
			}
		}
	}

	return excludeList
}

// extractPrimaryKeyDefinitionsWithAST extracts primary key constraints with their AST nodes
func extractPrimaryKeyDefinitionsWithAST(createStmt *parser.CreatestmtContext) map[string]*IndexDefWithAST {
	pkMap := make(map[string]*IndexDefWithAST)

	if createStmt == nil || createStmt.Opttableelementlist() == nil {
		return pkMap
	}

	tableElementList := createStmt.Opttableelementlist().Tableelementlist()
	if tableElementList == nil {
		return pkMap
	}

	for _, element := range tableElementList.AllTableelement() {
		if element.Tableconstraint() != nil {
			constraint := element.Tableconstraint()
			if constraint.Constraintelem() != nil {
				elem := constraint.Constraintelem()
				// Check for PRIMARY KEY constraints
				isPrimary := elem.PRIMARY() != nil && elem.KEY() != nil

				if isPrimary {
					// This is a primary key constraint
					name := ""
					if constraint.Name() != nil {
						name = pgparser.NormalizePostgreSQLName(constraint.Name())
					}
					// Use constraint definition text as fallback key if name is empty
					if name == "" {
						// Get the full original text from tokens
						if parser := constraint.GetParser(); parser != nil {
							if tokenStream := parser.GetTokenStream(); tokenStream != nil {
								name = tokenStream.GetTextFromRuleContext(constraint)
							}
						}
						if name == "" {
							name = constraint.GetText() // Final fallback
						}
					}
					pkMap[name] = &IndexDefWithAST{
						Name:    name,
						ASTNode: constraint,
					}
				}
			}
		}
	}

	return pkMap
}

// getForeignKeyText returns the text representation of a foreign key constraint for comparison
func getForeignKeyText(constraintAST parser.ITableconstraintContext) string {
	if constraintAST == nil {
		return ""
	}

	// Get tokens from the parser for precise text extraction
	if parser := constraintAST.GetParser(); parser != nil {
		if tokenStream := parser.GetTokenStream(); tokenStream != nil {
			start := constraintAST.GetStart()
			stop := constraintAST.GetStop()
			if start != nil && stop != nil {
				return tokenStream.GetTextFromTokens(start, stop)
			}
		}
	}

	// Fallback to GetText() if tokens are not available
	return constraintAST.GetText()
}

// getCheckConstraintText returns the text representation of a check constraint for comparison
func getCheckConstraintText(constraintAST parser.ITableconstraintContext) string {
	if constraintAST == nil {
		return ""
	}

	// Get tokens from the parser for precise text extraction
	if parser := constraintAST.GetParser(); parser != nil {
		if tokenStream := parser.GetTokenStream(); tokenStream != nil {
			start := constraintAST.GetStart()
			stop := constraintAST.GetStop()
			if start != nil && stop != nil {
				return tokenStream.GetTextFromTokens(start, stop)
			}
		}
	}

	// Fallback to GetText() if tokens are not available
	return constraintAST.GetText()
}

// getExcludeConstraintText returns the text representation of an EXCLUDE constraint for comparison
func getExcludeConstraintText(constraintAST parser.ITableconstraintContext) string {
	if constraintAST == nil {
		return ""
	}

	// Get tokens from the parser for precise text extraction
	if parser := constraintAST.GetParser(); parser != nil {
		if tokenStream := parser.GetTokenStream(); tokenStream != nil {
			start := constraintAST.GetStart()
			stop := constraintAST.GetStop()
			if start != nil && stop != nil {
				return tokenStream.GetTextFromTokens(start, stop)
			}
		}
	}

	// Fallback to GetText() if tokens are not available
	return constraintAST.GetText()
}

// getIndexText returns the text representation of an index/unique constraint for comparison
func getIndexText(constraintAST parser.ITableconstraintContext) string {
	if constraintAST == nil {
		return ""
	}

	// Get tokens from the parser for precise text extraction
	if parser := constraintAST.GetParser(); parser != nil {
		if tokenStream := parser.GetTokenStream(); tokenStream != nil {
			start := constraintAST.GetStart()
			stop := constraintAST.GetStop()
			if start != nil && stop != nil {
				return tokenStream.GetTextFromTokens(start, stop)
			}
		}
	}

	// Fallback to GetText() if tokens are not available
	return constraintAST.GetText()
}
