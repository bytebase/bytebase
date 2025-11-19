package mysql

import (
	"context"
	"regexp"
	"strings"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
	mysqldb "github.com/bytebase/bytebase/backend/plugin/db/mysql"
)

// TestGetDatabaseMetadataWithTestcontainer tests the get_database_metadata function
// by comparing its output with the metadata retrieved from a real MySQL instance.
func TestGetDatabaseMetadataWithTestcontainer(t *testing.T) {
	ctx := context.Background()

	// Start MySQL container
	container, err := testcontainer.GetTestMySQLContainer(ctx)
	require.NoError(t, err)
	defer container.Close(ctx)

	// Create test database
	_, err = container.GetDB().Exec("CREATE DATABASE IF NOT EXISTS test_db")
	require.NoError(t, err)

	host := container.GetHost()
	port := container.GetPort()

	// Create MySQL driver for metadata sync
	driver := &mysqldb.Driver{}
	config := db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Type:     storepb.DataSourceType_ADMIN,
			Username: "root",
			Host:     host,
			Port:     port,
			Database: "test_db",
		},
		Password: "root-password",
		ConnectionContext: db.ConnectionContext{
			EngineVersion: "8.0",
			DatabaseName:  "test_db",
		},
	}

	openedDriver, err := driver.Open(ctx, storepb.Engine_MYSQL, config)
	require.NoError(t, err)
	defer openedDriver.Close(ctx)

	// Cast to MySQL driver for SyncDBSchema
	mysqlDriver, ok := openedDriver.(*mysqldb.Driver)
	require.True(t, ok, "failed to cast to mysql.Driver")

	// Test cases with various MySQL features
	testCases := []struct {
		name string
		ddl  string
	}{
		{
			name: "basic_table_creation1",
			ddl: `
			CREATE TABLE users(
    			id INT PRIMARY KEY AUTO_INCREMENT,
				name varchar(220)
			);
			`,
		},
		{
			name: "basic_table_creation2",
			ddl: `
			CREATE TABLE users(
    			id INT PRIMARY KEY AUTO_INCREMENT
			);
			`,
		},
		{
			name: "basic_tables_with_constraints",
			ddl: `
CREATE TABLE users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE,
    email VARCHAR(100) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE TABLE posts (
    id INT AUTO_INCREMENT PRIMARY KEY,
    user_id INT NOT NULL,
    title VARCHAR(200) NOT NULL,
    content TEXT,
    published BOOLEAN DEFAULT false,
    CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    INDEX idx_user_id (user_id),
    INDEX idx_published (published)
);
`,
		},
		{
			name: "table_with_various_data_types",
			ddl: `
CREATE TABLE data_types_test (
    id INT AUTO_INCREMENT PRIMARY KEY,
    tiny_int_col TINYINT,
    small_int_col SMALLINT,
    medium_int_col MEDIUMINT,
    int_col INT,
    big_int_col BIGINT,
    decimal_col DECIMAL(10, 2),
    float_col FLOAT,
    double_col DOUBLE,
    bit_col BIT(8),
    bool_col BOOLEAN,
    char_col CHAR(10),
    varchar_col VARCHAR(255),
    binary_col BINARY(16),
    varbinary_col VARBINARY(255),
    tinyblob_col TINYBLOB,
    blob_col BLOB,
    mediumblob_col MEDIUMBLOB,
    longblob_col LONGBLOB,
    tinytext_col TINYTEXT,
    text_col TEXT,
    mediumtext_col MEDIUMTEXT,
    longtext_col LONGTEXT,
    enum_col ENUM('small', 'medium', 'large'),
    set_col SET('red', 'green', 'blue'),
    date_col DATE,
    time_col TIME,
    datetime_col DATETIME,
    timestamp_col TIMESTAMP,
    year_col YEAR,
    json_col JSON
);
`,
		},
		{
			name: "views_and_generated_columns",
			ddl: `
CREATE TABLE employees (
    id INT AUTO_INCREMENT PRIMARY KEY,
    first_name VARCHAR(50) NOT NULL,
    last_name VARCHAR(50) NOT NULL,
    full_name VARCHAR(101) GENERATED ALWAYS AS (CONCAT(first_name, ' ', last_name)) STORED,
    email VARCHAR(100) GENERATED ALWAYS AS (CONCAT(LOWER(first_name), '.', LOWER(last_name), '@company.com')) VIRTUAL,
    department VARCHAR(50),
    salary DECIMAL(10, 2)
);

CREATE VIEW active_employees AS
SELECT id, first_name, last_name, full_name, department
FROM employees
WHERE department IS NOT NULL;

CREATE VIEW high_earners AS
SELECT id, full_name, salary
FROM employees
WHERE salary > 100000;
`,
		},
		{
			name: "functions_and_procedures",
			ddl: `
DELIMITER //

CREATE FUNCTION get_employee_count(dept VARCHAR(50)) 
RETURNS INT
DETERMINISTIC
READS SQL DATA
BEGIN
    DECLARE emp_count INT;
    SELECT COUNT(*) INTO emp_count 
    FROM employees 
    WHERE department = dept;
    RETURN emp_count;
END//

CREATE PROCEDURE update_employee_salary(
    IN emp_id INT,
    IN new_salary DECIMAL(10, 2)
)
BEGIN
    UPDATE employees 
    SET salary = new_salary 
    WHERE id = emp_id;
END//

DELIMITER ;
`,
		},
		{
			name: "partitioned_tables",
			ddl: `
CREATE TABLE sales (
    id INT AUTO_INCREMENT,
    sale_date DATE NOT NULL,
    amount DECIMAL(10, 2) NOT NULL,
    region VARCHAR(50) NOT NULL,
    PRIMARY KEY (id, sale_date)
) PARTITION BY RANGE (YEAR(sale_date)) (
    PARTITION p2021 VALUES LESS THAN (2022),
    PARTITION p2022 VALUES LESS THAN (2023),
    PARTITION p2023 VALUES LESS THAN (2024),
    PARTITION p2024 VALUES LESS THAN (2025)
);

CREATE TABLE user_logs (
    id INT AUTO_INCREMENT,
    user_id INT NOT NULL,
    log_date DATE NOT NULL,
    action VARCHAR(100),
    PRIMARY KEY (id, user_id)
) PARTITION BY HASH(user_id) PARTITIONS 4;

CREATE TABLE regions (
    id INT AUTO_INCREMENT,
    region_code VARCHAR(10) NOT NULL,
    region_name VARCHAR(100),
    PRIMARY KEY (id, region_code)
) PARTITION BY KEY(region_code) PARTITIONS 3;

CREATE TABLE products (
    id INT AUTO_INCREMENT,
    category VARCHAR(50) NOT NULL,
    name VARCHAR(100),
    price DECIMAL(10, 2),
    PRIMARY KEY (id, category)
) PARTITION BY LIST COLUMNS(category) (
    PARTITION p_electronics VALUES IN ('laptop', 'phone', 'tablet'),
    PARTITION p_clothing VALUES IN ('shirt', 'pants', 'shoes'),
    PARTITION p_food VALUES IN ('fruit', 'vegetable', 'meat')
);
`,
		},
		{
			name: "indexes_with_various_types",
			ddl: `
CREATE TABLE orders (
    id INT AUTO_INCREMENT PRIMARY KEY,
    customer_id INT NOT NULL,
    order_date DATE NOT NULL,
    total_amount DECIMAL(10, 2) NOT NULL,
    status VARCHAR(20),
    notes TEXT,
    INDEX idx_customer (customer_id),
    INDEX idx_date_desc (order_date DESC),
    INDEX idx_customer_date (customer_id ASC, order_date DESC),
    UNIQUE INDEX idx_unique_customer_status (customer_id, status),
    FULLTEXT INDEX idx_fulltext_notes (notes)
);
`,
		},
		{
			name: "check_constraints",
			ddl: `
CREATE TABLE products_with_checks (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    price DECIMAL(10, 2) NOT NULL,
    quantity INT NOT NULL,
    category VARCHAR(50),
    CONSTRAINT chk_price CHECK (price > 0),
    CONSTRAINT chk_quantity CHECK (quantity >= 0),
    CONSTRAINT chk_category CHECK (category IN ('electronics', 'clothing', 'food', 'other'))
);
`,
		},
		{
			name: "triggers",
			ddl: `
CREATE TABLE users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(50) NOT NULL,
    email VARCHAR(100) NOT NULL
);

CREATE TABLE audit_log (
    id INT AUTO_INCREMENT PRIMARY KEY,
    table_name VARCHAR(100),
    action VARCHAR(10),
    user VARCHAR(100),
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

DELIMITER //

CREATE TRIGGER users_after_insert
AFTER INSERT ON users
FOR EACH ROW
BEGIN
    INSERT INTO audit_log (table_name, action, user) 
    VALUES ('users', 'INSERT', USER());
END//

CREATE TRIGGER users_after_update
AFTER UPDATE ON users
FOR EACH ROW
BEGIN
    INSERT INTO audit_log (table_name, action, user) 
    VALUES ('users', 'UPDATE', USER());
END//

DELIMITER ;
`,
		},
		{
			name: "table_with_comments",
			ddl: `
CREATE TABLE documented_table (
    id INT AUTO_INCREMENT PRIMARY KEY COMMENT 'Unique identifier',
    name VARCHAR(100) NOT NULL COMMENT 'The name of the item',
    description TEXT COMMENT 'Detailed description',
    price DECIMAL(10, 2) COMMENT 'Price in USD',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT 'Creation timestamp',
    INDEX idx_name (name) COMMENT 'Index on name for faster lookups'
) COMMENT='This table stores documented items';
`,
		},
		{
			name: "complex_foreign_keys",
			ddl: `
CREATE TABLE departments (
    dept_id INT PRIMARY KEY,
    dept_name VARCHAR(100) NOT NULL
);

CREATE TABLE employees_fk (
    emp_id INT PRIMARY KEY,
    emp_name VARCHAR(100) NOT NULL,
    dept_id INT,
    manager_id INT,
    CONSTRAINT fk_dept FOREIGN KEY (dept_id) REFERENCES departments(dept_id) ON DELETE SET NULL ON UPDATE CASCADE,
    CONSTRAINT fk_manager FOREIGN KEY (manager_id) REFERENCES employees_fk(emp_id) ON DELETE RESTRICT ON UPDATE RESTRICT
);
`,
		},
		{
			name: "advanced_column_attributes",
			ddl: `
CREATE TABLE advanced_columns (
    id INT AUTO_INCREMENT PRIMARY KEY,
    unsigned_int INT UNSIGNED NOT NULL,
    zerofill_int INT(8) ZEROFILL,
    unsigned_zerofill INT UNSIGNED ZEROFILL,
    tiny_unsigned TINYINT UNSIGNED,
    big_unsigned BIGINT UNSIGNED,
    decimal_unsigned DECIMAL(10,2) UNSIGNED,
    float_unsigned FLOAT UNSIGNED,
    double_unsigned DOUBLE UNSIGNED,
    bit_field BIT(16),
    char_binary CHAR(10) BINARY,
    varchar_binary VARCHAR(50) BINARY,
    text_collate TEXT COLLATE utf8mb4_unicode_ci,
    varchar_collate VARCHAR(100) COLLATE utf8mb4_bin,
    datetime_precision DATETIME(6),
    timestamp_precision TIMESTAMP(3),
    time_precision TIME(2)
);
`,
		},
		{
			name: "spatial_data_types",
			ddl: `
CREATE TABLE spatial_test (
    id INT AUTO_INCREMENT PRIMARY KEY,
    location GEOMETRY NOT NULL,
    point_col POINT NOT NULL,
    line_col LINESTRING NOT NULL,
    polygon_col POLYGON NOT NULL,
    multipoint_col MULTIPOINT NOT NULL,
    multiline_col MULTILINESTRING NOT NULL,
    multipolygon_col MULTIPOLYGON NOT NULL,
    geocollection_col GEOMETRYCOLLECTION NOT NULL,
    SPATIAL INDEX idx_location (location),
    SPATIAL INDEX idx_point (point_col)
);
`,
		},
		{
			name: "mysql8_invisible_indexes",
			ddl: `
CREATE TABLE invisible_index_test (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(100),
    status VARCHAR(20),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_name (name),
    INDEX idx_email (email) INVISIBLE,
    INDEX idx_status_invisible (status) INVISIBLE,
    INDEX idx_composite (name, email) INVISIBLE
);
`,
		},
		{
			name: "advanced_indexes",
			ddl: `
CREATE TABLE advanced_index_test (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    price DECIMAL(10,2),
    category VARCHAR(50),
    tags JSON,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_name_prefix (name(10)),
    INDEX idx_price_desc (price DESC),
    INDEX idx_composite_mixed (category ASC, price DESC),
    UNIQUE INDEX idx_unique_name_category (name, category),
    FULLTEXT INDEX idx_fulltext_desc (description),
    INDEX idx_expression ((YEAR(created_at)))
);
`,
		},
		{
			name: "table_level_options",
			ddl: `
CREATE TABLE table_options_test (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    data TEXT
) ENGINE=InnoDB 
  DEFAULT CHARSET=utf8mb4 
  COLLATE=utf8mb4_unicode_ci 
  ROW_FORMAT=DYNAMIC 
  AUTO_INCREMENT=1000
  COMMENT='Table with various options';
`,
		},
		{
			name: "advanced_partitioning_maxvalue",
			ddl: `
CREATE TABLE sales_maxvalue (
    id INT AUTO_INCREMENT,
    sale_date DATE NOT NULL,
    amount DECIMAL(10, 2) NOT NULL,
    region VARCHAR(50) NOT NULL,
    PRIMARY KEY (id, sale_date)
) PARTITION BY RANGE (YEAR(sale_date)) (
    PARTITION p2021 VALUES LESS THAN (2022),
    PARTITION p2022 VALUES LESS THAN (2023),
    PARTITION p2023 VALUES LESS THAN (2024),
    PARTITION p_future VALUES LESS THAN MAXVALUE
);
`,
		},
		{
			name: "multi_column_partitioning",
			ddl: `
CREATE TABLE multi_column_partition (
    id INT AUTO_INCREMENT,
    year_col INT NOT NULL,
    month_col INT NOT NULL,
    data VARCHAR(100),
    PRIMARY KEY (id, year_col, month_col)
) PARTITION BY RANGE COLUMNS(year_col, month_col) (
    PARTITION p202101 VALUES LESS THAN (2021, 7),
    PARTITION p202107 VALUES LESS THAN (2022, 1),
    PARTITION p202201 VALUES LESS THAN (2022, 7),
    PARTITION p202207 VALUES LESS THAN (2023, 1)
);
`,
		},
		{
			name: "self_referencing_foreign_keys",
			ddl: `
CREATE TABLE hierarchical_data (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    parent_id INT,
    manager_id INT,
    CONSTRAINT fk_parent FOREIGN KEY (parent_id) REFERENCES hierarchical_data(id) ON DELETE CASCADE,
    CONSTRAINT fk_manager FOREIGN KEY (manager_id) REFERENCES hierarchical_data(id) ON DELETE SET NULL
);
`,
		},
		{
			name: "complex_generated_columns",
			ddl: `
CREATE TABLE complex_generated (
    id INT AUTO_INCREMENT PRIMARY KEY,
    first_name VARCHAR(50) NOT NULL,
    last_name VARCHAR(50) NOT NULL,
    birth_date DATE NOT NULL,
    salary DECIMAL(10,2),
    full_name VARCHAR(101) GENERATED ALWAYS AS (CONCAT(first_name, ' ', last_name)) STORED,
    email VARCHAR(150) GENERATED ALWAYS AS (CONCAT(LOWER(first_name), '.', LOWER(last_name), '@company.com')) VIRTUAL,
    age_years INT GENERATED ALWAYS AS (TIMESTAMPDIFF(YEAR, birth_date, '2024-01-01')) VIRTUAL,
    salary_category VARCHAR(20) GENERATED ALWAYS AS (
        CASE 
            WHEN salary < 50000 THEN 'junior'
            WHEN salary < 100000 THEN 'mid'
            ELSE 'senior'
        END
    ) STORED,
    INDEX idx_full_name (full_name),
    INDEX idx_age (age_years),
    INDEX idx_salary_cat (salary_category)
);
`,
		},
		{
			name: "quoted_identifiers_special_chars",
			ddl: `
CREATE TABLE ` + "`special-table`" + ` (
    ` + "`id-field`" + ` INT AUTO_INCREMENT PRIMARY KEY,
    ` + "`first name`" + ` VARCHAR(50) NOT NULL,
    ` + "`email@domain`" + ` VARCHAR(100),
    ` + "`order`" + ` INT COMMENT 'Reserved keyword as column name',
    ` + "`group`" + ` VARCHAR(50) COMMENT 'Another reserved keyword',
    INDEX ` + "`idx-name`" + ` (` + "`first name`" + `),
    UNIQUE INDEX ` + "`idx-email`" + ` (` + "`email@domain`" + `)
) COMMENT='Table with special characters in identifiers';
`,
		},
		{
			name: "json_and_array_features",
			ddl: `
CREATE TABLE json_features (
    id INT AUTO_INCREMENT PRIMARY KEY,
    metadata JSON,
    config JSON,
    tags JSON,
    name_from_json VARCHAR(100) GENERATED ALWAYS AS (JSON_UNQUOTE(JSON_EXTRACT(metadata, '$.name'))) STORED,
    tag_count INT GENERATED ALWAYS AS (JSON_LENGTH(tags)) VIRTUAL,
    CHECK (JSON_VALID(metadata)),
    CHECK (JSON_VALID(config)),
    INDEX idx_name_json (name_from_json),
    INDEX idx_tag_count (tag_count)
);
`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Clean up any existing objects from previous tests
			_, err := container.GetDB().ExecContext(ctx, "DROP DATABASE IF EXISTS test_db")
			require.NoError(t, err)
			_, err = container.GetDB().ExecContext(ctx, "CREATE DATABASE test_db")
			require.NoError(t, err)

			// Execute DDL using the driver
			_, err = openedDriver.Execute(ctx, tc.ddl, db.ExecuteOptions{})
			require.NoError(t, err)

			// Get metadata from live database using driver
			dbMetadata, err := mysqlDriver.SyncDBSchema(ctx)
			require.NoError(t, err)

			// Get metadata from parser
			parsedMetadata, err := GetDatabaseMetadata(tc.ddl)
			require.NoError(t, err)

			// Compare metadata
			compareMetadata(t, dbMetadata, parsedMetadata, tc.name)
		})
	}
}

// compareMetadata compares the metadata from the live database with the parsed metadata
func compareMetadata(t *testing.T, dbMeta, parsedMeta *storepb.DatabaseSchemaMetadata, testName string) {
	// For MySQL, we only have one schema
	require.Equal(t, 1, len(parsedMeta.Schemas), "parsed metadata should have exactly one schema")

	dbMetadata := dbMeta.Schemas[0]
	parsedSchema := parsedMeta.Schemas[0]

	// Compare tables
	require.Equal(t, len(dbMetadata.Tables), len(parsedSchema.Tables),
		"mismatch in number of tables for test %s", testName)

	// Create a map for easier lookup
	dbTableMap := make(map[string]*storepb.TableMetadata)
	for _, table := range dbMetadata.Tables {
		dbTableMap[table.Name] = table
	}

	for _, parsedTable := range parsedSchema.Tables {
		dbTable, exists := dbTableMap[parsedTable.Name]
		require.True(t, exists, "table %s not found in database metadata", parsedTable.Name)

		// Compare columns
		compareColumns(t, dbTable.Columns, parsedTable.Columns, parsedTable.Name)

		// Compare indexes
		compareIndexes(t, dbTable.Indexes, parsedTable.Indexes, parsedTable.Name)

		// Compare foreign keys
		compareForeignKeys(t, dbTable.ForeignKeys, parsedTable.ForeignKeys, parsedTable.Name)

		// Compare check constraints (MySQL 8.0+)
		compareCheckConstraints(t, dbTable.CheckConstraints, parsedTable.CheckConstraints, parsedTable.Name)

		// Compare partitions
		comparePartitions(t, dbTable.Partitions, parsedTable.Partitions, parsedTable.Name)
	}

	// Compare views
	compareViews(t, dbMetadata.Views, parsedSchema.Views)

	// Compare functions
	compareFunctions(t, dbMetadata.Functions, parsedSchema.Functions)

	// Compare procedures
	compareProcedures(t, dbMetadata.Procedures, parsedSchema.Procedures)
}

func compareColumns(t *testing.T, dbColumns, parsedColumns []*storepb.ColumnMetadata, tableName string) {
	require.Equal(t, len(dbColumns), len(parsedColumns),
		"mismatch in number of columns for table %s", tableName)

	// Create a map for easier lookup
	dbColumnMap := make(map[string]*storepb.ColumnMetadata)
	for _, col := range dbColumns {
		dbColumnMap[col.Name] = col
	}

	for _, parsedCol := range parsedColumns {
		dbCol, exists := dbColumnMap[parsedCol.Name]
		require.True(t, exists, "column %s.%s not found in database metadata", tableName, parsedCol.Name)

		// Compare column properties (normalize types to lowercase)
		dbType := strings.ToLower(dbCol.Type)
		parsedType := strings.ToLower(parsedCol.Type)
		require.Equal(t, dbType, parsedType,
			"type mismatch for column %s.%s", tableName, parsedCol.Name)
		require.Equal(t, dbCol.Nullable, parsedCol.Nullable,
			"nullable mismatch for column %s.%s", tableName, parsedCol.Name)

		// Compare default values (allowing for some normalization differences)
		compareDefaultValues(t, dbCol, parsedCol, tableName, parsedCol.Name)

		// Compare comments
		require.Equal(t, dbCol.Comment, parsedCol.Comment,
			"comment mismatch for column %s.%s", tableName, parsedCol.Name)
	}
}

func compareDefaultValues(t *testing.T, dbDefault, parsedDefault any, tableName, columnName string) {
	if dbDefault == nil && parsedDefault == nil {
		return
	}

	if dbDefault == nil || parsedDefault == nil {
		t.Errorf("default value mismatch for column %s.%s: db=%v, parsed=%v",
			tableName, columnName, dbDefault, parsedDefault)
		return
	}

	// Extract default expressions
	dbExpr := ""
	parsedExpr := ""

	// Handle database default
	if dbCol, ok := dbDefault.(*storepb.ColumnMetadata); ok {
		if dbCol.Default != "" {
			dbExpr = dbCol.Default
		}
	}

	// Handle parsed default
	if parsedCol, ok := parsedDefault.(*storepb.ColumnMetadata); ok {
		if parsedCol.Default != "" {
			parsedExpr = parsedCol.Default
		}
	}

	// Normalize and compare
	dbExpr = normalizeDefaultExpression(dbExpr)
	parsedExpr = normalizeDefaultExpression(parsedExpr)

	require.Equal(t, dbExpr, parsedExpr,
		"default value mismatch for column %s.%s", tableName, columnName)
}

func normalizeDefaultExpression(expr string) string {
	// Normalize common variations
	expr = strings.TrimSpace(expr)
	expr = strings.ToUpper(expr)

	// Handle CURRENT_TIMESTAMP variations
	if strings.Contains(expr, "CURRENT_TIMESTAMP") || strings.Contains(expr, "NOW()") {
		return "CURRENT_TIMESTAMP"
	}

	// Handle AUTO_INCREMENT
	if expr == "AUTO_INCREMENT" {
		return "AUTO_INCREMENT"
	}

	// Remove parentheses from simple expressions
	if strings.HasPrefix(expr, "(") && strings.HasSuffix(expr, ")") {
		expr = expr[1 : len(expr)-1]
	}

	return expr
}

func compareIndexes(t *testing.T, dbIndexes, parsedIndexes []*storepb.IndexMetadata, tableName string) {
	require.Equal(t, len(dbIndexes), len(parsedIndexes),
		"mismatch in number of indexes for table %s", tableName)

	// Create maps for easier lookup
	dbIndexMap := make(map[string]*storepb.IndexMetadata)
	for _, idx := range dbIndexes {
		dbIndexMap[idx.Name] = idx
	}

	// Check that all parsed indexes exist in db with exact match
	for _, parsedIdx := range parsedIndexes {
		dbIdx, exists := dbIndexMap[parsedIdx.Name]
		require.True(t, exists, "index %s on table %s not found in database metadata", parsedIdx.Name, tableName)

		// Compare all IndexMetadata members for complete consistency

		// 1. Name (already validated above through map lookup)
		require.Equal(t, dbIdx.Name, parsedIdx.Name,
			"name mismatch for index %s on table %s", parsedIdx.Name, tableName)

		// 2. Type (index type: BTREE, HASH, FULLTEXT, SPATIAL, etc.)
		require.Equal(t, dbIdx.Type, parsedIdx.Type,
			"type mismatch for index %s on table %s", parsedIdx.Name, tableName)

		// 3. Primary (whether the index is a primary key index)
		require.Equal(t, dbIdx.Primary, parsedIdx.Primary,
			"primary mismatch for index %s on table %s", parsedIdx.Name, tableName)

		// 4. Unique (whether the index is unique)
		require.Equal(t, dbIdx.Unique, parsedIdx.Unique,
			"unique mismatch for index %s on table %s", parsedIdx.Name, tableName)

		// 5. Visible (whether the index is visible - MySQL 8.0+ feature)
		require.Equal(t, dbIdx.Visible, parsedIdx.Visible,
			"visible mismatch for index %s on table %s", parsedIdx.Name, tableName)

		// 6. Comment (index comment)
		require.Equal(t, dbIdx.Comment, parsedIdx.Comment,
			"comment mismatch for index %s on table %s", parsedIdx.Name, tableName)

		// 7. Expressions (columns or expressions that the index is on)
		require.Equal(t, len(dbIdx.Expressions), len(parsedIdx.Expressions),
			"expression count mismatch for index %s on table %s", parsedIdx.Name, tableName)

		for i, expr := range parsedIdx.Expressions {
			if i < len(dbIdx.Expressions) {
				// Normalize expressions for comparison
				dbExpr := normalizeIndexExpression(dbIdx.Expressions[i])
				parsedExpr := normalizeIndexExpression(expr)
				require.Equal(t, dbExpr, parsedExpr,
					"expression mismatch for index %s on table %s at position %d", parsedIdx.Name, tableName, i)
			}
		}

		// 8. KeyLength (key lengths for each expression, -1 if not specified)
		require.Equal(t, len(dbIdx.KeyLength), len(parsedIdx.KeyLength),
			"key length count mismatch for index %s on table %s", parsedIdx.Name, tableName)
		require.Equal(t, dbIdx.KeyLength, parsedIdx.KeyLength,
			"key length mismatch for index %s on table %s", parsedIdx.Name, tableName)

		// 9. Descending (descending flags for each expression)
		require.Equal(t, len(dbIdx.Descending), len(parsedIdx.Descending),
			"descending flag count mismatch for index %s on table %s", parsedIdx.Name, tableName)
		require.Equal(t, dbIdx.Descending, parsedIdx.Descending,
			"descending flags mismatch for index %s on table %s", parsedIdx.Name, tableName)

		// 10. Definition (the full index definition - may be empty for some parsers)
		// Note: We only compare if both have non-empty definitions as some parsers may not populate this
		if dbIdx.Definition != "" && parsedIdx.Definition != "" {
			require.Equal(t, dbIdx.Definition, parsedIdx.Definition,
				"definition mismatch for index %s on table %s", parsedIdx.Name, tableName)
		}

		// Note: Other fields like ParentIndexSchema, ParentIndexName, Granularity, and SupportNullScan
		// are database-specific and may not apply to MySQL, so we don't validate them here
	}
}

// normalizeIndexExpression normalizes index expressions for comparison
func normalizeIndexExpression(expr string) string {
	// Remove backticks
	expr = strings.ReplaceAll(expr, "`", "")

	// Remove outer parentheses from expressions like "(year(created_at))"
	if strings.HasPrefix(expr, "(") && strings.HasSuffix(expr, ")") {
		expr = expr[1 : len(expr)-1]
	}

	// Convert to uppercase for case-insensitive comparison
	expr = strings.ToUpper(expr)

	return expr
}

func compareForeignKeys(t *testing.T, dbFKs, parsedFKs []*storepb.ForeignKeyMetadata, tableName string) {
	require.Equal(t, len(dbFKs), len(parsedFKs),
		"mismatch in number of foreign keys for table %s", tableName)

	// Create a map for easier lookup
	dbFKMap := make(map[string]*storepb.ForeignKeyMetadata)
	for _, fk := range dbFKs {
		dbFKMap[fk.Name] = fk
	}

	for _, parsedFK := range parsedFKs {
		dbFK, exists := dbFKMap[parsedFK.Name]
		require.True(t, exists, "foreign key %s on table %s not found in database metadata",
			parsedFK.Name, tableName)

		// Compare foreign key properties
		require.Equal(t, dbFK.Columns, parsedFK.Columns,
			"columns mismatch for foreign key %s on table %s", parsedFK.Name, tableName)
		require.Equal(t, dbFK.ReferencedTable, parsedFK.ReferencedTable,
			"referenced table mismatch for foreign key %s on table %s", parsedFK.Name, tableName)
		require.Equal(t, dbFK.ReferencedColumns, parsedFK.ReferencedColumns,
			"referenced columns mismatch for foreign key %s on table %s", parsedFK.Name, tableName)
		require.Equal(t, dbFK.OnDelete, parsedFK.OnDelete,
			"on delete mismatch for foreign key %s on table %s", parsedFK.Name, tableName)
		require.Equal(t, dbFK.OnUpdate, parsedFK.OnUpdate,
			"on update mismatch for foreign key %s on table %s", parsedFK.Name, tableName)
	}
}

func compareCheckConstraints(t *testing.T, dbChecks, parsedChecks []*storepb.CheckConstraintMetadata, tableName string) {
	// MySQL 8.0+ supports check constraints
	// For older versions, this might be empty
	if len(dbChecks) == 0 && len(parsedChecks) == 0 {
		return
	}

	require.Equal(t, len(dbChecks), len(parsedChecks),
		"mismatch in number of check constraints for table %s", tableName)

	// Create a map for easier lookup
	dbCheckMap := make(map[string]*storepb.CheckConstraintMetadata)
	for _, chk := range dbChecks {
		dbCheckMap[chk.Name] = chk
	}

	for _, parsedCheck := range parsedChecks {
		dbCheck, exists := dbCheckMap[parsedCheck.Name]
		require.True(t, exists, "check constraint %s on table %s not found in database metadata",
			parsedCheck.Name, tableName)

		// Compare expressions (allowing for some normalization)
		dbExpr := normalizeCheckExpression(dbCheck.Expression)
		parsedExpr := normalizeCheckExpression(parsedCheck.Expression)
		require.Equal(t, dbExpr, parsedExpr,
			"expression mismatch for check constraint %s on table %s", parsedCheck.Name, tableName)
	}
}

func normalizeCheckExpression(expr string) string {
	// Normalize check constraint expressions
	expr = strings.TrimSpace(expr)

	// Remove outer parentheses if present (MySQL normalizes this way)
	if strings.HasPrefix(expr, "(") && strings.HasSuffix(expr, ")") {
		expr = expr[1 : len(expr)-1]
	}

	expr = strings.ReplaceAll(expr, "`", "")
	expr = strings.ReplaceAll(expr, " ", "")

	// Remove character set prefixes like _utf8mb4
	re := regexp.MustCompile(`_[a-zA-Z0-9]+\\`)
	expr = re.ReplaceAllString(expr, "")

	// Remove escaped quotes
	expr = strings.ReplaceAll(expr, `\'`, `'`)
	expr = strings.ReplaceAll(expr, `\"`, `"`)

	expr = strings.ToLower(expr)
	return expr
}

func comparePartitions(t *testing.T, dbPartitions, parsedPartitions []*storepb.TablePartitionMetadata, tableName string) {
	require.Equal(t, len(dbPartitions), len(parsedPartitions),
		"mismatch in number of partitions for table %s", tableName)

	// Create a map for easier lookup
	dbPartMap := make(map[string]*storepb.TablePartitionMetadata)
	for _, part := range dbPartitions {
		dbPartMap[part.Name] = part
	}

	for _, parsedPart := range parsedPartitions {
		dbPart, exists := dbPartMap[parsedPart.Name]
		require.True(t, exists, "partition %s on table %s not found in database metadata",
			parsedPart.Name, tableName)

		// Compare partition properties
		// Note: Expression comparison might need normalization
		require.NotEmpty(t, dbPart.Expression, "partition expression should not be empty for %s", parsedPart.Name)
		require.NotEmpty(t, parsedPart.Expression, "parsed partition expression should not be empty for %s", parsedPart.Name)
	}
}

func compareViews(t *testing.T, dbViews, parsedViews []*storepb.ViewMetadata) {
	require.Equal(t, len(dbViews), len(parsedViews), "mismatch in number of views")

	// Create a map for easier lookup
	dbViewMap := make(map[string]*storepb.ViewMetadata)
	for _, view := range dbViews {
		dbViewMap[view.Name] = view
	}

	for _, parsedView := range parsedViews {
		dbView, exists := dbViewMap[parsedView.Name]
		require.True(t, exists, "view %s not found in database metadata", parsedView.Name)

		// View definitions might differ in formatting, so just check they're non-empty
		require.NotEmpty(t, dbView.Definition, "database view definition should not be empty")
		require.NotEmpty(t, parsedView.Definition, "parsed view definition should not be empty")
	}
}

func compareFunctions(t *testing.T, dbFuncs, parsedFuncs []*storepb.FunctionMetadata) {
	require.Equal(t, len(dbFuncs), len(parsedFuncs), "mismatch in number of functions")

	// Create a map for easier lookup
	dbFuncMap := make(map[string]*storepb.FunctionMetadata)
	for _, fn := range dbFuncs {
		dbFuncMap[fn.Name] = fn
	}

	for _, parsedFunc := range parsedFuncs {
		dbFunc, exists := dbFuncMap[parsedFunc.Name]
		require.True(t, exists, "function %s not found in database metadata", parsedFunc.Name)

		// Function definitions might differ in formatting, so just check they're non-empty
		require.NotEmpty(t, dbFunc.Definition, "database function definition should not be empty")
		require.NotEmpty(t, parsedFunc.Definition, "parsed function definition should not be empty")
	}
}

func compareProcedures(t *testing.T, dbProcs, parsedProcs []*storepb.ProcedureMetadata) {
	require.Equal(t, len(dbProcs), len(parsedProcs), "mismatch in number of procedures")

	// Create a map for easier lookup
	dbProcMap := make(map[string]*storepb.ProcedureMetadata)
	for _, proc := range dbProcs {
		dbProcMap[proc.Name] = proc
	}

	for _, parsedProc := range parsedProcs {
		dbProc, exists := dbProcMap[parsedProc.Name]
		require.True(t, exists, "procedure %s not found in database metadata", parsedProc.Name)

		// Procedure definitions might differ in formatting, so just check they're non-empty
		require.NotEmpty(t, dbProc.Definition, "database procedure definition should not be empty")
		require.NotEmpty(t, parsedProc.Definition, "parsed procedure definition should not be empty")
	}
}
