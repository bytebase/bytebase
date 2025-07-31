package model

import (
	"sync"
	"testing"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func TestDatabaseSchema_DeepCopy_ConcurrentAccess(t *testing.T) {
	// Create a database schema with nested structures
	dbSchema := &DatabaseSchema{
		metadata: &storepb.DatabaseSchemaMetadata{
			Name: "test_db",
		},
		config: &storepb.DatabaseConfig{
			Name: "test_db",
		},
		schema:                []byte("CREATE TABLE test (id INT);"),
		isObjectCaseSensitive: false,
		isDetailCaseSensitive: false,
		configInternal: &DatabaseConfig{
			name: "test_db",
			internal: map[string]*SchemaConfig{
				"public": {
					internal: map[string]*TableConfig{
						"test_table": {
							Classification: "test",
							internal: map[string]*storepb.ColumnCatalog{
								"id": {
									Name:         "id",
									SemanticType: "ID",
								},
							},
						},
					},
				},
			},
		},
	}

	const numGoroutines = 100
	const numOperations = 1000

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 2)

	// Create a channel to collect copies
	copies := make(chan *DatabaseSchema, numGoroutines)

	// Goroutines that create deep copies and modify them
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				// Create a deep copy
				copiedSchema := dbSchema.DeepCopy()

				// Modify the copy
				schemaConfig := copiedSchema.configInternal.CreateOrGetSchemaConfig("schema_" + string(rune(id)))
				tableConfig := schemaConfig.CreateOrGetTableConfig("table_" + string(rune(j%10)))
				tableConfig.CreateOrGetColumnConfig("column_" + string(rune(j%20)))

				// Send one copy per goroutine
				if j == 0 {
					copies <- copiedSchema
				}
			}
		}(i)
	}

	// Goroutines that read from the original
	for i := 0; i < numGoroutines; i++ {
		go func(_ int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				// Read from the original (should not be affected by copies)
				if dbSchema.configInternal != nil {
					if schemaConfig := dbSchema.configInternal.internal["public"]; schemaConfig != nil {
						if tableConfig := schemaConfig.internal["test_table"]; tableConfig != nil {
							_ = tableConfig.internal["id"]
						}
					}
				}
			}
		}(i)
	}

	// Close the copies channel once all goroutines are done
	go func() {
		wg.Wait()
		close(copies)
	}()

	// Verify that all copies are independent
	var collectedCopies []*DatabaseSchema
	for copy := range copies {
		collectedCopies = append(collectedCopies, copy)
	}

	// Verify original is unchanged
	if len(dbSchema.configInternal.internal) != 1 {
		t.Errorf("Original schema count changed: expected 1, got %d", len(dbSchema.configInternal.internal))
	}
	if schemaConfig := dbSchema.configInternal.internal["public"]; schemaConfig != nil {
		if len(schemaConfig.internal) != 1 {
			t.Errorf("Original table count changed: expected 1, got %d", len(schemaConfig.internal))
		}
		if tableConfig := schemaConfig.internal["test_table"]; tableConfig != nil {
			if len(tableConfig.internal) != 1 {
				t.Errorf("Original column count changed: expected 1, got %d", len(tableConfig.internal))
			}
		}
	}

	// Verify copies are different from original
	for i, copy := range collectedCopies {
		if copy == dbSchema {
			t.Errorf("Copy %d is the same object as original", i)
		}
		if copy.configInternal == dbSchema.configInternal {
			t.Errorf("Copy %d configInternal is the same object as original", i)
		}
		// Verify the copy has been modified
		if len(copy.configInternal.internal) < 2 {
			t.Errorf("Copy %d was not properly modified", i)
		}
	}
}

func TestTableConfig_DeepCopy_ConcurrentAccess(t *testing.T) {
	original := &TableConfig{
		Classification: "sensitive",
		internal: map[string]*storepb.ColumnCatalog{
			"id": {
				Name:         "id",
				SemanticType: "ID",
			},
			"email": {
				Name:           "email",
				SemanticType:   "EMAIL",
				Classification: "PII",
			},
		},
	}

	const numGoroutines = 50
	const numOperations = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 3)

	// Writers on copies
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			copiedTable := original.DeepCopy()
			for j := 0; j < numOperations; j++ {
				columnName := "column_" + string(rune(id)) + "_" + string(rune(j))
				copiedTable.CreateOrGetColumnConfig(columnName)
			}
		}(i)
	}

	// Readers on original
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				_ = original.internal["id"]
				_ = original.internal["email"]
			}
		}()
	}

	// Deleters on copies
	for i := 0; i < numGoroutines; i++ {
		go func(_ int) {
			defer wg.Done()
			copiedTable := original.DeepCopy()
			for j := 0; j < numOperations; j++ {
				if j%2 == 0 {
					copiedTable.RemoveColumnConfig("id")
				} else {
					copiedTable.RemoveColumnConfig("email")
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify original is unchanged
	if len(original.internal) != 2 {
		t.Errorf("Original column count changed: expected 2, got %d", len(original.internal))
	}
	if _, ok := original.internal["id"]; !ok {
		t.Error("Original missing 'id' column")
	}
	if _, ok := original.internal["email"]; !ok {
		t.Error("Original missing 'email' column")
	}
}
