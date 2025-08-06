# SQL Review Rules Documentation

This document provides comprehensive documentation for all SQL Review Rule types available in Bytebase, including their payload structures and examples.

## Table of Contents

1. [Rule Categories](#rule-categories)
2. [Payload Structure Types](#payload-structure-types)
3. [Template Tokens](#template-tokens)
4. [Engine Support](#engine-support)

## Rule Categories

### 1. Engine Rules

#### `engine.mysql.use-innodb`
**Description**: Require InnoDB as storage engine (MySQL/MariaDB only)  
**Engines**: MySQL, MariaDB  
**Payload**: None (empty)

---

### 2. Naming Rules

#### `naming.fully-qualified`
**Description**: Enforce using fully qualified object names  
**Engines**: PostgreSQL  
**Payload**: None (empty)

#### `naming.table`
**Description**: Enforce table name format  
**Engines**: All  
**Payload Structure**: `NamingRulePayload`
```json
{
  "format": "^[a-z]+(_[a-z]+)*$",
  "maxLength": 64
}
```

#### `naming.column`
**Description**: Enforce column name format  
**Engines**: All  
**Payload Structure**: `NamingRulePayload`
```json
{
  "format": "^[a-z]+(_[a-z]+)*$",
  "maxLength": 64
}
```

#### `naming.index.pk`
**Description**: Enforce primary key name format  
**Engines**: All  
**Payload Structure**: `NamingRulePayload`
```json
{
  "format": "^pk_{{table}}_{{column_list}}$",
  "maxLength": 64
}
```

#### `naming.index.uk`
**Description**: Enforce unique key name format  
**Engines**: All  
**Payload Structure**: `NamingRulePayload`
```json
{
  "format": "^uk_{{table}}_{{column_list}}$",
  "maxLength": 64
}
```

#### `naming.index.fk`
**Description**: Enforce foreign key name format  
**Engines**: All  
**Payload Structure**: `NamingRulePayload`
```json
{
  "format": "^fk_{{referencing_table}}_{{referencing_column}}_{{referenced_table}}_{{referenced_column}}$",
  "maxLength": 64
}
```

#### `naming.index.idx`
**Description**: Enforce index name format  
**Engines**: All  
**Payload Structure**: `NamingRulePayload`
```json
{
  "format": "^idx_{{table}}_{{column_list}}$",
  "maxLength": 64
}
```

#### `naming.column.auto-increment`
**Description**: Enforce auto-increment column name format  
**Engines**: MySQL, MariaDB  
**Payload Structure**: `NamingRulePayload`
```json
{
  "format": "^id$",
  "maxLength": 64
}
```

#### `naming.table.no-keyword`
**Description**: Enforce table name not to use keywords  
**Engines**: All  
**Payload**: None (empty)

#### `naming.identifier.no-keyword`
**Description**: Enforce identifier not to use keywords  
**Engines**: All  
**Payload**: None (empty)

#### `naming.identifier.case`
**Description**: Enforce identifier case convention  
**Engines**: All  
**Payload Structure**: `NamingCaseRulePayload`
```json
{
  "upper": true
}
```

---

### 3. Statement Rules

#### Basic Statement Rules

#### `statement.select.no-select-all`
**Description**: Disallow 'SELECT *' statements  
**Engines**: All  
**Payload**: None (empty)

#### `statement.where.require.select`
**Description**: Require 'WHERE' clause for SELECT statements  
**Engines**: All  
**Payload**: None (empty)

#### `statement.where.require.update-delete`
**Description**: Require 'WHERE' clause for UPDATE/DELETE statements  
**Engines**: All  
**Payload**: None (empty)

#### `statement.where.no-leading-wildcard-like`
**Description**: Disallow leading '%' in LIKE conditions  
**Engines**: All  
**Payload**: None (empty)

#### DDL/DML Control Rules

#### `statement.disallow-commit`
**Description**: Disallow using COMMIT statements  
**Engines**: All  
**Payload**: None (empty)

#### `statement.disallow-limit`
**Description**: Disallow LIMIT in INSERT/DELETE/UPDATE statements  
**Engines**: All  
**Payload**: None (empty)

#### `statement.disallow-order-by`
**Description**: Disallow ORDER BY in DELETE/UPDATE statements  
**Engines**: All  
**Payload**: None (empty)

#### `statement.merge-alter-table`
**Description**: Disallow redundant ALTER TABLE statements  
**Engines**: All  
**Payload**: None (empty)

#### `statement.disallow-mix-in-ddl`
**Description**: Disallow DML statements in DDL transactions  
**Engines**: All  
**Payload**: None (empty)

#### `statement.disallow-mix-in-dml`
**Description**: Disallow DDL statements in DML transactions  
**Engines**: All  
**Payload**: None (empty)

#### Insert Rules

#### `statement.insert.row-limit`
**Description**: Enforce insert row limit  
**Engines**: All  
**Payload Structure**: `NumberTypeRulePayload`
```json
{
  "number": 1000
}
```

#### `statement.insert.must-specify-column`
**Description**: Enforce column specification in INSERT statements  
**Engines**: All  
**Payload**: None (empty)

#### `statement.insert.disallow-order-by-rand`
**Description**: Disallow ORDER BY RAND in INSERT statements  
**Engines**: MySQL, MariaDB  
**Payload**: None (empty)

#### Performance Rules

#### `statement.affected-row-limit`
**Description**: Enforce UPDATE/DELETE affected row limit  
**Engines**: All  
**Payload Structure**: `NumberTypeRulePayload`
```json
{
  "number": 1000
}
```

#### `statement.maximum-limit-value`
**Description**: Enforce maximum LIMIT value  
**Engines**: All  
**Payload Structure**: `NumberTypeRulePayload`
```json
{
  "number": 1000
}
```

#### `statement.maximum-join-table-count`
**Description**: Enforce maximum join table count  
**Engines**: All  
**Payload Structure**: `NumberTypeRulePayload`
```json
{
  "number": 2
}
```

#### `statement.where.maximum-logical-operator-count`
**Description**: Enforce maximum logical operators in WHERE clause  
**Engines**: All  
**Payload Structure**: `NumberTypeRulePayload`
```json
{
  "number": 10
}
```

#### Advanced Statement Rules

#### `statement.dml-dry-run`
**Description**: Dry run DML statements before execution  
**Engines**: All  
**Payload**: None (empty)

#### `statement.where.no-equal-null`
**Description**: Check WHERE clause does not use equality with NULL  
**Engines**: All  
**Payload**: None (empty)

#### `statement.where.disallow-functions-and-calculations`
**Description**: Disallow functions and calculations in WHERE clause  
**Engines**: All  
**Payload**: None (empty)

#### `statement.query.minimum-plan-level`
**Description**: Enforce minimum query execution plan level  
**Engines**: All  
**Payload Structure**: `StringTypeRulePayload`
```json
{
  "string": "INDEX"
}
```

#### `statement.max-execution-time`
**Description**: Enforce maximum execution time  
**Engines**: MySQL, MariaDB  
**Payload**: None (empty)

#### `statement.require-algorithm-option`
**Description**: Require ALGORITHM option in ALTER TABLE  
**Engines**: MySQL, MariaDB  
**Payload**: None (empty)

#### `statement.require-lock-option`
**Description**: Require LOCK option in ALTER TABLE  
**Engines**: MySQL, MariaDB  
**Payload**: None (empty)

#### PostgreSQL-specific Statement Rules

#### `statement.disallow-on-del-cascade`
**Description**: Disallow ON DELETE CASCADE  
**Engines**: PostgreSQL  
**Payload**: None (empty)

#### `statement.disallow-rm-tbl-cascade`
**Description**: Disallow CASCADE when removing table  
**Engines**: PostgreSQL  
**Payload**: None (empty)

#### `statement.disallow-add-column-with-default`
**Description**: Disallow adding column with DEFAULT value  
**Engines**: PostgreSQL  
**Payload**: None (empty)

#### `statement.add-check-not-valid`
**Description**: Require adding check constraints as NOT VALID  
**Engines**: PostgreSQL  
**Payload**: None (empty)

#### `statement.add-foreign-key-not-valid`
**Description**: Require adding foreign key as NOT VALID  
**Engines**: PostgreSQL  
**Payload**: None (empty)

#### `statement.disallow-add-not-null`
**Description**: Disallow adding NOT NULL constraint  
**Engines**: PostgreSQL  
**Payload**: None (empty)

#### `statement.create-specify-schema`
**Description**: Disallow creating table without schema specification  
**Engines**: PostgreSQL  
**Payload**: None (empty)

#### `statement.check-set-role-variable`
**Description**: Require check for SET ROLE variable  
**Engines**: PostgreSQL  
**Payload**: None (empty)

#### `statement.object-owner-check`
**Description**: Check object ownership  
**Engines**: PostgreSQL  
**Payload**: None (empty)

#### `statement.non-transactional`
**Description**: Check for non-transactional statements  
**Engines**: PostgreSQL  
**Payload**: None (empty)

---

### 4. Table Rules

#### `table.require-pk`
**Description**: Require table to have primary key  
**Engines**: All  
**Payload**: None (empty)

#### `table.no-foreign-key`
**Description**: Disallow foreign keys  
**Engines**: All  
**Payload**: None (empty)

#### `table.drop-naming-convention`
**Description**: Only allow dropping tables that match naming convention  
**Engines**: All  
**Payload Structure**: `NamingRulePayload`
```json
{
  "format": "_del$"
}
```

#### `table.comment`
**Description**: Enforce table comment convention  
**Engines**: All  
**Payload Structure**: `CommentConventionRulePayload`
```json
{
  "required": true,
  "requiredClassification": false,
  "maxLength": 64
}
```

#### `table.disallow-partition`
**Description**: Disallow table partitioning  
**Engines**: All  
**Payload**: None (empty)

#### `table.disallow-trigger`
**Description**: Disallow table triggers  
**Engines**: All  
**Payload**: None (empty)

#### `table.no-duplicate-index`
**Description**: Require no duplicate indexes  
**Engines**: All  
**Payload**: None (empty)

#### `table.text-fields-total-length`
**Description**: Enforce total length limit of text fields  
**Engines**: All  
**Payload Structure**: `NumberTypeRulePayload`
```json
{
  "number": 1000
}
```

#### `table.disallow-set-charset`
**Description**: Disallow setting table charset  
**Engines**: MySQL, MariaDB  
**Payload**: None (empty)

#### `table.disallow-ddl`
**Description**: Disallow DDL operations on specific tables  
**Engines**: All  
**Payload Structure**: `StringArrayTypeRulePayload`
```json
{
  "list": ["table1", "table2", "sensitive_table"]
}
```

#### `table.disallow-dml`
**Description**: Disallow DML operations on specific tables  
**Engines**: All  
**Payload Structure**: `StringArrayTypeRulePayload`
```json
{
  "list": ["readonly_table", "archive_table"]
}
```

#### `table.limit-size`
**Description**: Restrict access to tables based on size  
**Engines**: All  
**Payload Structure**: `NumberTypeRulePayload`
```json
{
  "number": 10000000
}
```

#### `table.require-charset`
**Description**: Enforce table charset requirement  
**Engines**: MySQL, MariaDB  
**Payload**: None (empty)

#### `table.require-collation`
**Description**: Enforce table collation requirement  
**Engines**: MySQL, MariaDB  
**Payload**: None (empty)

---

### 5. Column Rules

#### `column.required`
**Description**: Enforce required columns in tables  
**Engines**: All  
**Payload Structure**: `StringArrayTypeRulePayload`
```json
{
  "list": ["id", "created_ts", "updated_ts", "creator_id", "updater_id"]
}
```

#### `column.no-null`
**Description**: Enforce columns cannot have NULL values  
**Engines**: All  
**Payload**: None (empty)

#### `column.disallow-change-type`
**Description**: Disallow changing column data type  
**Engines**: All  
**Payload**: None (empty)

#### `column.set-default-for-not-null`
**Description**: Require default value for NOT NULL columns  
**Engines**: All  
**Payload**: None (empty)

#### `column.disallow-change`
**Description**: Disallow CHANGE COLUMN operations  
**Engines**: MySQL, MariaDB  
**Payload**: None (empty)

#### `column.disallow-changing-order`
**Description**: Disallow changing column order  
**Engines**: All  
**Payload**: None (empty)

#### `column.disallow-drop`
**Description**: Disallow dropping columns  
**Engines**: All  
**Payload**: None (empty)

#### `column.disallow-drop-in-index`
**Description**: Disallow dropping columns that are part of an index  
**Engines**: All  
**Payload**: None (empty)

#### `column.comment`
**Description**: Enforce column comment convention  
**Engines**: All  
**Payload Structure**: `CommentConventionRulePayload`
```json
{
  "required": true,
  "requiredClassification": false,
  "maxLength": 64
}
```

#### `column.auto-increment-must-integer`
**Description**: Require auto-increment columns to be integer type  
**Engines**: MySQL, MariaDB  
**Payload**: None (empty)

#### `column.type-disallow-list`
**Description**: Enforce column type disallow list  
**Engines**: All  
**Payload Structure**: `StringArrayTypeRulePayload`
```json
{
  "list": ["JSON", "BINARY_FLOAT", "BLOB", "LONGTEXT"]
}
```

#### `column.disallow-set-charset`
**Description**: Disallow setting column charset  
**Engines**: MySQL, MariaDB  
**Payload**: None (empty)

#### `column.maximum-character-length`
**Description**: Enforce maximum character length for string columns  
**Engines**: All  
**Payload Structure**: `NumberTypeRulePayload`
```json
{
  "number": 20
}
```

#### `column.maximum-varchar-length`
**Description**: Enforce maximum varchar length  
**Engines**: All  
**Payload Structure**: `NumberTypeRulePayload`
```json
{
  "number": 2560
}
```

#### `column.auto-increment-initial-value`
**Description**: Enforce auto-increment initial value  
**Engines**: MySQL, MariaDB  
**Payload Structure**: `NumberTypeRulePayload`
```json
{
  "number": 1
}
```

#### `column.auto-increment-must-unsigned`
**Description**: Require auto-increment columns to be unsigned  
**Engines**: MySQL, MariaDB  
**Payload**: None (empty)

#### `column.current-time-count-limit`
**Description**: Enforce current time column count limit  
**Engines**: MySQL, MariaDB  
**Payload**: None (empty)

#### `column.require-default`
**Description**: Enforce column default value requirement  
**Engines**: All  
**Payload**: None (empty)

#### `column.default-disallow-volatile`
**Description**: Disallow volatile functions in column defaults  
**Engines**: PostgreSQL  
**Payload**: None (empty)

#### `column.require-charset`
**Description**: Enforce column charset requirement  
**Engines**: MySQL, MariaDB  
**Payload**: None (empty)

#### `column.require-collation`
**Description**: Enforce column collation requirement  
**Engines**: MySQL, MariaDB  
**Payload**: None (empty)

---

### 6. Index Rules

#### `index.no-duplicate-column`
**Description**: Require no duplicate columns in index  
**Engines**: All  
**Payload**: None (empty)

#### `index.key-number-limit`
**Description**: Enforce index key number limit  
**Engines**: All  
**Payload Structure**: `NumberTypeRulePayload`
```json
{
  "number": 5
}
```

#### `index.pk-type-limit`
**Description**: Enforce primary key type restriction  
**Engines**: All  
**Payload**: None (empty)

#### `index.type-no-blob`
**Description**: Enforce no BLOB columns in index  
**Engines**: All  
**Payload**: None (empty)

#### `index.total-number-limit`
**Description**: Enforce total index number limit per table  
**Engines**: All  
**Payload Structure**: `NumberTypeRulePayload`
```json
{
  "number": 5
}
```

#### `index.primary-key-type-allowlist`
**Description**: Enforce primary key type allowlist  
**Engines**: All  
**Payload Structure**: `StringArrayTypeRulePayload`
```json
{
  "list": ["serial", "bigserial", "int", "bigint", "uuid"]
}
```

#### `index.create-concurrently`
**Description**: Require creating indexes concurrently  
**Engines**: PostgreSQL  
**Payload**: None (empty)

#### `index.type-allow-list`
**Description**: Enforce index type allowlist  
**Engines**: All  
**Payload Structure**: `StringArrayTypeRulePayload`
```json
{
  "list": ["BTREE", "HASH", "GIN", "GIST"]
}
```

#### `index.not-redundant`
**Description**: Prohibit redundant indices  
**Engines**: All  
**Payload**: None (empty)

---

### 7. System Rules

#### `system.charset.allowlist`
**Description**: Enforce charset allowlist  
**Engines**: MySQL, MariaDB  
**Payload Structure**: `StringArrayTypeRulePayload`
```json
{
  "list": ["utf8mb4", "UTF8"]
}
```

#### `system.collation.allowlist`
**Description**: Enforce collation allowlist  
**Engines**: MySQL, MariaDB  
**Payload Structure**: `StringArrayTypeRulePayload`
```json
{
  "list": ["utf8mb4_0900_ai_ci", "utf8mb4_unicode_ci"]
}
```

#### `system.comment.length`
**Description**: Limit comment length  
**Engines**: All  
**Payload Structure**: `NumberTypeRulePayload`
```json
{
  "number": 64
}
```

#### `system.procedure.disallow-create`
**Description**: Disallow creating stored procedures  
**Engines**: All  
**Payload**: None (empty)

#### `system.event.disallow-create`
**Description**: Disallow creating events  
**Engines**: MySQL, MariaDB  
**Payload**: None (empty)

#### `system.view.disallow-create`
**Description**: Disallow creating views  
**Engines**: All  
**Payload**: None (empty)

#### `system.function.disallow-create`
**Description**: Disallow creating functions  
**Engines**: All  
**Payload**: None (empty)

#### `system.function.disallowed-list`
**Description**: Enforce disallowed function list  
**Engines**: All  
**Payload Structure**: `StringArrayTypeRulePayload`
```json
{
  "list": ["rand", "uuid", "sleep", "now"]
}
```

---

### 8. Schema Rules

#### `schema.backward-compatibility`
**Description**: Enforce backward compatibility in schema changes  
**Engines**: All  
**Payload**: None (empty)

---

### 9. Database Rules

#### `database.drop-empty-database`
**Description**: Check if database is empty before allowing drop  
**Engines**: All  
**Payload**: None (empty)

---

### 10. Advice Rules

#### `advice.online-migration`
**Description**: Advise using online migration for large tables  
**Engines**: MySQL, MariaDB  
**Payload Structure**: `NumberTypeRulePayload`
```json
{
  "number": 100000000
}
```

---

### 11. Builtin Rules

#### `builtin.prior-backup-check`
**Description**: Check for prior backup before executing destructive operations  
**Engines**: All  
**Payload**: None (empty)

---

## Payload Structure Types

### 1. NamingRulePayload
Used for naming convention rules that require format patterns and length limits.
```json
{
  "format": "string (regex pattern)",
  "maxLength": "number"
}
```

**Example:**
```json
{
  "format": "^[a-z]+(_[a-z]+)*$",
  "maxLength": 64
}
```

### 2. StringArrayTypeRulePayload
Used for rules that work with lists of strings (allowlists, disallow lists, required items).
```json
{
  "list": ["string1", "string2", "string3"]
}
```

**Example:**
```json
{
  "list": ["id", "created_ts", "updated_ts", "creator_id", "updater_id"]
}
```

### 3. NumberTypeRulePayload
Used for rules that enforce numeric limits or thresholds.
```json
{
  "number": "number"
}
```

**Example:**
```json
{
  "number": 1000
}
```

### 4. StringTypeRulePayload
Used for rules that require a single string value.
```json
{
  "string": "string value"
}
```

**Example:**
```json
{
  "string": "INDEX"
}
```

### 5. CommentConventionRulePayload
Used for comment-related rules that define requirements and constraints.
```json
{
  "required": "boolean",
  "requiredClassification": "boolean",
  "maxLength": "number"
}
```

**Example:**
```json
{
  "required": true,
  "requiredClassification": false,
  "maxLength": 64
}
```

### 6. NamingCaseRulePayload
Used for case convention rules.
```json
{
  "upper": "boolean"
}
```

**Example:**
```json
{
  "upper": true
}
```

---

## Template Tokens

For naming rules that support templates, the following tokens are available:

| Token | Description | Usage Example |
|-------|-------------|---------------|
| `{{table}}` | Current table name | `idx_{{table}}_name` |
| `{{column_list}}` | List of column names | `pk_{{table}}_{{column_list}}` |
| `{{referencing_table}}` | Table that references another | `fk_{{referencing_table}}_{{referenced_table}}` |
| `{{referencing_column}}` | Column that references another | `fk_{{referencing_table}}_{{referencing_column}}` |
| `{{referenced_table}}` | Table being referenced | `fk_{{referencing_table}}_{{referenced_table}}` |
| `{{referenced_column}}` | Column being referenced | `fk_{{referenced_table}}_{{referenced_column}}` |

**Template Example:**
```json
{
  "format": "^fk_{{referencing_table}}_{{referencing_column}}_{{referenced_table}}_{{referenced_column}}$",
  "maxLength": 64
}
```

---

## Engine Support

Based on actual implementation in the codebase, the following database engines support SQL review rules:

### Fully Supported Engines (100+ rules)
- **MySQL** (`storepb.Engine_MYSQL`) - ~100+ rules with comprehensive coverage
- **TiDB** (`storepb.Engine_TIDB`) - ~85+ rules with TiDB-specific optimizations

### Well-Supported Engines (50+ rules)
- **PostgreSQL** (`storepb.Engine_POSTGRES`) - ~65+ rules including PostgreSQL-specific features

### Good Support (25-50 rules)
- **Oracle** (`storepb.Engine_ORACLE`) - ~35+ rules covering fundamental areas
- **Microsoft SQL Server** (`storepb.Engine_MSSQL`) - ~35+ rules with MSSQL-specific features
- **Snowflake** (`storepb.Engine_SNOWFLAKE`) - ~27+ rules focused on key areas

### Limited Support (< 10 rules)
- **MariaDB** (`storepb.Engine_MARIADB`) - Inherits MySQL rule implementations
- **OceanBase** (`storepb.Engine_OCEANBASE`) - ~3 rules with basic OceanBase optimizations

### No SQL Review Support
The following engines do **NOT** support SQL review rules:
- ClickHouse, SQLite, MongoDB, Redis, Spanner, Redshift, StarRocks, Doris, Hive, Elasticsearch, BigQuery, DynamoDB, Databricks, CockroachDB, CosmosDB, Trino, Cassandra

### Engine-Specific Rules

#### MySQL/TiDB/MariaDB Specific
- `engine.mysql.use-innodb`
- `column.auto-increment-*` rules
- `statement.require-algorithm-option`
- `statement.require-lock-option`
- `table.require-charset`
- `system.charset.allowlist`
- `system.collation.allowlist`

#### PostgreSQL Specific
- `naming.fully-qualified`
- `statement.disallow-on-del-cascade`
- `statement.disallow-rm-tbl-cascade`
- `statement.add-check-not-valid`
- `statement.add-foreign-key-not-valid`
- `statement.disallow-add-column-with-default`
- `statement.disallow-add-not-null`
- `statement.create-specify-schema`
- `index.create-concurrently`
- `column.default-disallow-volatile`

#### Oracle Specific
- Oracle-compatible naming and type rules

#### SQL Server Specific  
- MSSQL-compatible syntax and feature rules

#### Snowflake Specific
- Snowflake-compatible data types and syntax rules

#### Cross-Engine Rules
Most naming, table, column, and basic statement rules work across all supported engines with engine-appropriate implementations.

---

## Usage Notes

1. **Empty Payload**: Rules with no configuration options use an empty payload (`""` or `null`).

2. **Default Values**: When creating rules programmatically, use the `SetDefaultSQLReviewRulePayload` function to generate appropriate default payloads.

3. **Validation**: All payloads are validated against their expected structure when rules are created or updated.

4. **Engine Compatibility**: Always check engine support before applying rules to ensure compatibility.

5. **Template Expansion**: Template tokens are expanded at rule evaluation time using the actual schema object names.

6. **Regular Expressions**: Format patterns in naming rules use standard regular expression syntax.

---

This documentation covers all 150+ SQL review rule types available in Bytebase across all supported database engines. Each rule is designed to enforce specific database standards and best practices to maintain code quality, performance, and consistency.