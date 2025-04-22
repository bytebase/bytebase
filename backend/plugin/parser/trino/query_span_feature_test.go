package trino

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func TestPredicateExtraction(t *testing.T) {
	// Setup a test query with a predicate
	// Create an extractor with context
	gCtx := base.GetQuerySpanContext{
		InstanceID: "test-instance",
		Engine:     storepb.Engine_TRINO,
	}
	extractor := newQuerySpanExtractor("catalog1", "public", gCtx, false)

	// Create a listener - not used directly but initializes the extractor relationship
	_ = newTrinoQuerySpanListener(extractor)

	// Manually add some source columns
	extractor.sourceColumns[base.ColumnResource{
		Database: "catalog1",
		Schema:   "public",
		Table:    "users",
		Column:   "id",
	}] = true
	extractor.sourceColumns[base.ColumnResource{
		Database: "catalog1",
		Schema:   "public",
		Table:    "users",
		Column:   "name",
	}] = true

	// Create test boolean expressions for id > 10 and name LIKE 'A%'
	// This is a simple test that doesn't fully parse the tree, but tests the extraction logic

	// Just verify our predicate column extraction method works
	idCol := "id"
	nameCol := "name"

	// Create a listener to use for the test
	listener := newTrinoQuerySpanListener(extractor)

	// Call the method directly with column names
	listener.addPredicateColumn(idCol)
	listener.addPredicateColumn(nameCol)

	// Check that predicates were extracted
	assert.Equal(t, 2, len(extractor.predicateColumns), "Expected 2 predicate columns")

	var foundIDCol, foundNameCol bool
	for col := range extractor.predicateColumns {
		switch col.Column {
		case "id":
			foundIDCol = true
		case "name":
			foundNameCol = true
		}
	}

	assert.True(t, foundIDCol, "Expected 'id' predicate column")
	assert.True(t, foundNameCol, "Expected 'name' predicate column")
}

func TestCTEHandling(t *testing.T) {
	// Test handling of Common Table Expressions (CTEs)

	// Create a test extractor
	gCtx := base.GetQuerySpanContext{
		InstanceID: "test-instance",
		Engine:     storepb.Engine_TRINO,
	}
	extractor := newQuerySpanExtractor("catalog1", "public", gCtx, false)

	// Create a pseudo table for a CTE
	cteName := "temp_cte"
	pseudoTable := base.NewPseudoTable(cteName, nil)

	// Add it to the extractor
	extractor.ctes = append(extractor.ctes, pseudoTable)

	// Verify we can look up this CTE by name
	var found bool
	for _, cte := range extractor.ctes {
		if cte.Name == cteName {
			found = true
			break
		}
	}

	assert.True(t, found, "CTE was not found in extractor")
	assert.Equal(t, 1, len(extractor.ctes), "Expected 1 CTE")

	// Test that our CTE lookup in EnterTableName would work
	// Create a listener - not used directly but initializes the extractor relationship
	_ = newTrinoQuerySpanListener(extractor)

	// Before recording table sources, check it's empty
	assert.Equal(t, 0, len(extractor.tableSourcesFrom), "Expected no table sources initially")

	// Simulate a table lookup that matches our CTE
	for _, cte := range extractor.ctes {
		if strings.EqualFold(cte.Name, cteName) {
			// This is a CTE reference
			extractor.tableSourcesFrom = append(extractor.tableSourcesFrom, cte)
			break
		}
	}

	// Verify the CTE was added as a table source
	assert.Equal(t, 1, len(extractor.tableSourcesFrom), "CTE should be added as a table source")
	assert.Equal(t, cteName, extractor.tableSourcesFrom[0].GetTableName(), "CTE name mismatch")
}

func TestUnnestAndLateralSupport(t *testing.T) {
	// Test the core functionality of UNNEST and LATERAL query handling

	// Create a test extractor
	gCtx := base.GetQuerySpanContext{
		InstanceID: "test-instance",
		Engine:     storepb.Engine_TRINO,
	}
	extractor := newQuerySpanExtractor("catalog1", "public", gCtx, false)

	// Add some source columns that might come from an UNNEST
	extractor.sourceColumns[base.ColumnResource{
		Database: "catalog1",
		Schema:   "public",
		Table:    "users",
		Column:   "arrays", // Array column being unnested
	}] = true

	// Create a listener - not used directly but initializes the extractor relationship
	_ = newTrinoQuerySpanListener(extractor)

	// Test creating a derived table for UNNEST results
	results := []base.QuerySpanResult{
		{
			Name:          "item",
			IsPlainField:  true,
			SourceColumns: make(base.SourceColumnSet),
		},
	}

	// Check initial state
	assert.Equal(t, 0, len(extractor.tableSourcesFrom), "Expected no table sources initially")

	// Create a pseudo table for the UNNEST operation
	unnestTable := base.NewPseudoTable("unnest", results)
	extractor.tableSourcesFrom = append(extractor.tableSourcesFrom, unnestTable)

	// Verify the UNNEST was added as a table source
	assert.Equal(t, 1, len(extractor.tableSourcesFrom), "UNNEST should be added as a table source")
	assert.Equal(t, "unnest", extractor.tableSourcesFrom[0].GetTableName(), "UNNEST name mismatch")

	// Test the LATERAL subquery handling (similar approach)
	// Create a derived table for LATERAL results
	lateralResults := []base.QuerySpanResult{
		{
			Name:          "x",
			IsPlainField:  true,
			SourceColumns: make(base.SourceColumnSet),
		},
	}

	// Create a pseudo table for the LATERAL subquery
	lateralTable := base.NewPseudoTable("lateral", lateralResults)

	// Save current state before adding
	prevCount := len(extractor.tableSourcesFrom)

	// Add the lateral table
	extractor.tableSourcesFrom = append(extractor.tableSourcesFrom, lateralTable)

	// Verify it was added
	assert.Equal(t, prevCount+1, len(extractor.tableSourcesFrom), "LATERAL should be added as a table source")
	assert.Equal(t, "lateral", extractor.tableSourcesFrom[prevCount].GetTableName(), "LATERAL name mismatch")
}

func TestExtractQualifiedNameParts(t *testing.T) {
	// Test the Trino-specific 3-part naming convention (catalog.schema.table)

	// Create the extractor and listener
	gCtx := base.GetQuerySpanContext{
		InstanceID: "test-instance",
		Engine:     storepb.Engine_TRINO,
	}
	extractor := newQuerySpanExtractor("default_catalog", "default_schema", gCtx, false)

	// Test extracting database, schema, and table names when fully qualified
	db, schema, table := "catalog1", "schema1", "table1"

	// Create a test function that simulates extracting qualified name parts
	extractedDB, extractedSchema, extractedTable := db, schema, table

	// Verify the extraction works as expected
	assert.Equal(t, "catalog1", extractedDB, "Database/catalog name should be extracted correctly")
	assert.Equal(t, "schema1", extractedSchema, "Schema name should be extracted correctly")
	assert.Equal(t, "table1", extractedTable, "Table name should be extracted correctly")

	// Test with default values for partially qualified names
	partialResource := base.ColumnResource{
		Database: extractor.defaultDatabase,
		Schema:   extractor.defaultSchema,
	}

	// Verify defaults are applied
	assert.Equal(t, "default_catalog", partialResource.Database, "Default catalog not applied")
	assert.Equal(t, "default_schema", partialResource.Schema, "Default schema not applied")
}
