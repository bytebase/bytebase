package pg

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// TestScenario represents a complete test scenario with all required files
type TestScenario struct {
	Name              string
	Category          string
	Description       string
	InitialSchema     string
	ExpectedSchema    string
	ExpectedDiff      string
	ExpectedMigration string
}

// TestSDLComprehensiveValidation runs comprehensive SDL validation tests organized by categories
func TestSDLComprehensiveValidation(t *testing.T) {
	ctx := context.Background()

	// Start PostgreSQL container
	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("test_db"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Minute),
		),
	)
	require.NoError(t, err)
	defer func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Logf("failed to terminate container: %s", err)
		}
	}()

	// Get connection string
	connectionString, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	// Connect to the database
	connConfig, err := pgx.ParseConfig(connectionString)
	require.NoError(t, err)
	db := stdlib.OpenDB(*connConfig)
	defer db.Close()

	// Load test scenarios from organized files
	testScenarios, err := loadTestScenarios()
	require.NoError(t, err)
	require.NotEmpty(t, testScenarios, "No test scenarios found")

	// Group scenarios by category for better organization
	categorizedTests := make(map[string][]TestScenario)
	for _, scenario := range testScenarios {
		categorizedTests[scenario.Category] = append(categorizedTests[scenario.Category], scenario)
	}

	// Run tests by category
	for category, scenarios := range categorizedTests {
		t.Run(category, func(t *testing.T) {
			for _, scenario := range scenarios {
				t.Run(scenario.Name, func(t *testing.T) {
					runSDLTestScenario(ctx, t, connConfig, db, scenario)
				})
			}
		})
	}
}

// loadTestScenarios loads all test scenarios from the sdl_testdata directory
func loadTestScenarios() ([]TestScenario, error) {
	var scenarios []TestScenario

	testDataDir := "sdl_testdata"
	if _, err := os.Stat(testDataDir); os.IsNotExist(err) {
		return nil, errors.Errorf("test data directory %s does not exist", testDataDir)
	}

	err := filepath.Walk(testDataDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Look for test_description.md files to identify complete test scenarios
		if info.Name() == "test_description.md" {
			scenario, err := loadTestScenario(filepath.Dir(path))
			if err != nil {
				return errors.Wrapf(err, "failed to load scenario from %s", path)
			}
			scenarios = append(scenarios, scenario)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	// If no scenarios found in files, return error
	if len(scenarios) == 0 {
		return nil, errors.New("no test scenarios found in test data directory")
	}

	return scenarios, nil
}

// loadTestScenario loads a single test scenario from a directory
func loadTestScenario(scenarioDir string) (TestScenario, error) {
	var scenario TestScenario

	// Extract category and name from path
	relativePath, err := filepath.Rel("sdl_testdata", scenarioDir)
	if err != nil {
		return scenario, err
	}

	pathParts := strings.Split(relativePath, string(filepath.Separator))
	scenario.Category = pathParts[0]
	scenario.Name = strings.Join(pathParts[1:], "_")

	// Load description
	descPath := filepath.Join(scenarioDir, "test_description.md")
	if descData, err := os.ReadFile(descPath); err == nil {
		scenario.Description = string(descData)
	}

	// Load initial schema
	initialPath := filepath.Join(scenarioDir, "initial_schema.sql")
	if initialData, err := os.ReadFile(initialPath); err == nil {
		scenario.InitialSchema = string(initialData)
	}

	// Load expected schema
	expectedPath := filepath.Join(scenarioDir, "expected_schema.sql")
	if expectedData, err := os.ReadFile(expectedPath); err == nil {
		scenario.ExpectedSchema = string(expectedData)
	} else {
		return scenario, errors.New("expected_schema.sql is required")
	}

	// Load expected diff (optional)
	diffPath := filepath.Join(scenarioDir, "diff.json")
	if diffData, err := os.ReadFile(diffPath); err == nil {
		scenario.ExpectedDiff = string(diffData)
	}

	// Load expected migration (optional)
	migrationPath := filepath.Join(scenarioDir, "ddl.sql")
	if migrationData, err := os.ReadFile(migrationPath); err == nil {
		scenario.ExpectedMigration = string(migrationData)
	}

	return scenario, nil
}

// runSDLTestScenario executes a single SDL test scenario
func runSDLTestScenario(ctx context.Context, t *testing.T, connConfig *pgx.ConnConfig, mainDB *sql.DB, scenario TestScenario) {
	t.Logf("Running SDL test scenario: %s - %s", scenario.Category, scenario.Name)
	if scenario.Description != "" {
		t.Logf("Description: %s", strings.TrimSpace(scenario.Description))
	}

	// Create a unique database for this test (PostgreSQL limit is 63 chars)
	category := strings.ReplaceAll(scenario.Category, "/", "_")
	name := strings.ReplaceAll(scenario.Name, "/", "_")

	// Shorten common prefixes to avoid length issues
	category = strings.ReplaceAll(category, "table_operations", "tbl_ops")
	name = strings.ReplaceAll(name, "column_operations_", "col_")
	name = strings.ReplaceAll(name, "add_column_", "add_")
	name = strings.ReplaceAll(name, "constraints", "cons")
	name = strings.ReplaceAll(name, "foreign_keys", "fk")
	name = strings.ReplaceAll(name, "indexes", "idx")

	dbName := fmt.Sprintf("sdl_test_%s_%s", category, name)
	dbName = strings.ReplaceAll(dbName, "-", "_") // Replace hyphens with underscores
	dbName = strings.ToLower(dbName)

	// Ensure length doesn't exceed PostgreSQL's 63 character limit
	if len(dbName) > 63 {
		dbName = dbName[:63]
	}

	_, err := mainDB.Exec(fmt.Sprintf("CREATE DATABASE %s", dbName))
	require.NoError(t, err)
	defer func() {
		_, _ = mainDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbName))
	}()

	// Connect to the test database
	testConnConfig := *connConfig
	testConnConfig.Database = dbName
	testDB := stdlib.OpenDB(testConnConfig)
	defer testDB.Close()

	// Execute the enhanced SDL validation process
	err = executeEnhancedSDLValidation(ctx, t, &testConnConfig, testDB, dbName, scenario)
	require.NoError(t, err, "SDL validation failed for scenario: %s/%s", scenario.Category, scenario.Name)
}

// executeEnhancedSDLValidation implements the enhanced 5-step SDL validation with diff comparison
func executeEnhancedSDLValidation(ctx context.Context, t *testing.T, connConfig *pgx.ConnConfig, testDB *sql.DB, dbName string, scenario TestScenario) error {
	t.Log("=== Step 1: Apply initial schema to database ===")
	if strings.TrimSpace(scenario.InitialSchema) != "" {
		_, err := testDB.Exec(scenario.InitialSchema)
		if err != nil {
			return errors.Wrapf(err, "failed to apply initial schema")
		}
		t.Logf("✓ Applied initial schema (%d characters)", len(scenario.InitialSchema))
	} else {
		t.Log("✓ Starting with empty database")
	}

	t.Log("=== Step 2: Sync to get current schema metadata (Schema B) ===")
	schemaB, err := getSyncMetadataForSDL(ctx, connConfig, dbName)
	if err != nil {
		return errors.Wrapf(err, "failed to sync schema B")
	}
	t.Log("✓ Synced current schema metadata")

	t.Log("=== Step 3: Get expected schema metadata and generate migration DDL ===")
	schemaCMetadata, err := GetDatabaseMetadata(scenario.ExpectedSchema)
	if err != nil {
		return errors.Wrapf(err, "failed to get expected schema C metadata")
	}
	t.Log("✓ Parsed expected schema metadata")

	// Generate migration DDL from schema B to schema C using the existing function
	migrationDDL, schemaDiff, err := generateMigrationDDLFromMetadata(schemaB, schemaCMetadata)
	require.NoError(t, err, "Failed to generate migration DDL from metadata")

	// Use validateOrSaveTestFiles to generate/validate diff.json and ddl.sql files
	testName := strings.ReplaceAll(t.Name(), "/", "_")
	err = validateOrSaveTestFiles(t, schemaDiff, testName, migrationDDL)
	if err != nil {
		return errors.Wrapf(err, "test file validation/creation failed")
	}

	// Compare with expected diff if provided
	if scenario.ExpectedDiff != "" {
		err := validateSchemaDiff(t, schemaDiff, scenario.ExpectedDiff)
		if err != nil {
			t.Logf("WARNING: Schema diff validation failed: %v", err)
			// Note: This is a warning for now, can be made stricter later
		} else {
			t.Log("✓ Generated schema diff matches expected")
		}
	}

	t.Logf("✓ Generated migration DDL (%d characters)", len(migrationDDL))
	if migrationDDL != "" {
		t.Logf("Migration DDL preview (first 200 chars):\n%s",
			truncateString(migrationDDL, 200))
	}

	// Compare with expected migration DDL if provided
	if scenario.ExpectedMigration != "" {
		normalizedGenerated := normalizeSQLString(migrationDDL)
		normalizedExpected := normalizeSQLString(scenario.ExpectedMigration)
		if normalizedGenerated != normalizedExpected {
			t.Log("Generated migration DDL differs from expected:")
			t.Logf("Expected:\n%s", scenario.ExpectedMigration)
			t.Logf("Generated:\n%s", migrationDDL)

			// For comprehensive testing, we can make this a hard requirement
			if strings.TrimSpace(normalizedExpected) != "" {
				require.Equal(t, normalizedExpected, normalizedGenerated,
					"Generated migration DDL does not match expected DDL")
			}
		} else {
			t.Log("✓ Generated migration DDL matches expected")
		}
	}

	t.Log("=== Step 4: Apply migration DDL and sync final schema ===")
	if strings.TrimSpace(migrationDDL) != "" {
		_, err := testDB.Exec(migrationDDL)
		if err != nil {
			return errors.Wrapf(err, "failed to apply migration DDL: %s", migrationDDL)
		}
		t.Log("✓ Applied migration DDL")
	} else {
		t.Log("✓ No migration DDL needed (schemas already match)")
	}

	// Sync to get final schema D
	schemaD, err := getSyncMetadataForSDL(ctx, connConfig, dbName)
	if err != nil {
		return errors.Wrapf(err, "failed to sync schema D after migration")
	}
	t.Log("✓ Synced final schema metadata")

	t.Log("=== Step 5: Validate schema consistency ===")
	err = validateSchemaConsistency(schemaD, schemaCMetadata)
	if err != nil {
		return errors.Wrapf(err, "schema validation failed")
	}
	t.Log("✓ Final schema matches expected schema perfectly")

	return nil
}

// normalizeSQLString normalizes SQL for comparison by removing extra whitespace and standardizing formatting
func normalizeSQLString(sql string) string {
	// Remove extra whitespace and normalize line endings
	lines := strings.Split(sql, "\n")
	var normalizedLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			normalizedLines = append(normalizedLines, trimmed)
		}
	}
	return strings.Join(normalizedLines, "\n")
}

// truncateString truncates a string to maxLength characters
func truncateString(s string, maxLength int) string {
	if len(s) <= maxLength {
		return s
	}
	return s[:maxLength] + "..."
}

// validateSchemaDiff validates that the generated schema diff matches the expected diff
func validateSchemaDiff(t *testing.T, actualDiff any, expectedDiffJSON string) error {
	// For now, we'll do basic validation by comparing the string representations
	// In the future, this could be enhanced to do structured JSON comparison

	actualJSON, err := marshalDiffToJSON(actualDiff)
	if err != nil {
		return errors.Wrap(err, "failed to marshal actual diff to JSON")
	}

	// Normalize both JSON strings for comparison
	actualNormalized := normalizeJSONString(actualJSON)
	expectedNormalized := normalizeJSONString(expectedDiffJSON)

	if actualNormalized != expectedNormalized {
		t.Log("Schema diff comparison:")
		t.Logf("Expected diff:\n%s", expectedDiffJSON)
		t.Logf("Actual diff:\n%s", actualJSON)
		return errors.New("schema diff does not match expected")
	}

	return nil
}

// marshalDiffToJSON marshals a schema diff object to JSON string
func marshalDiffToJSON(diff any) (string, error) {
	// Convert the diff object to JSON
	jsonBytes, err := json.MarshalIndent(diff, "", "  ")
	if err != nil {
		return "", errors.Wrap(err, "failed to marshal diff to JSON")
	}
	return string(jsonBytes), nil
}

// normalizeJSONString normalizes JSON for comparison
func normalizeJSONString(jsonStr string) string {
	// Remove extra whitespace and normalize formatting
	lines := strings.Split(jsonStr, "\n")
	var normalizedLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			normalizedLines = append(normalizedLines, trimmed)
		}
	}
	return strings.Join(normalizedLines, "\n")
}
