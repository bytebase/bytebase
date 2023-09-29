package parser

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	tidbast "github.com/pingcap/tidb/parser/ast"
	"github.com/pingcap/tidb/parser/mysql"
	"github.com/pkg/errors"

	plsql "github.com/bytebase/plsql-parser"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
	plsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/plsql"
	snowparser "github.com/bytebase/bytebase/backend/plugin/parser/snowflake"
	"github.com/bytebase/bytebase/backend/plugin/parser/sql/ast"
	tidbparser "github.com/bytebase/bytebase/backend/plugin/parser/tidb"
	"github.com/bytebase/bytebase/backend/plugin/parser/tokenizer"
	tsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/tsql"
)

// ExtractChangedResources extracts the changed resources from the SQL.
func ExtractChangedResources(engineType EngineType, currentDatabase string, currentSchema string, sql string) ([]base.SchemaResource, error) {
	switch engineType {
	case MySQL, MariaDB, OceanBase:
		return mysqlparser.ExtractMySQLChangedResources(currentDatabase, sql)
	case Oracle:
		return plsqlparser.ExtractOracleChangedResources(currentDatabase, currentSchema, sql)
	default:
		if currentDatabase == "" {
			return nil, errors.Errorf("database must be specified for engine type: %s", engineType)
		}
		return nil, errors.Errorf("engine type %q is not supported", engineType)
	}
}

// ExtractResourceList extracts the resource list from the SQL.
func ExtractResourceList(engineType EngineType, currentDatabase string, currentSchema string, sql string) ([]base.SchemaResource, error) {
	switch engineType {
	case TiDB:
		return tidbparser.ExtractTiDBResourceList(currentDatabase, sql)
	case MySQL, MariaDB, OceanBase:
		// The resource list for MySQL may contains table, view and temporary table.
		return mysqlparser.ExtractMySQLResourceList(currentDatabase, sql)
	case Oracle:
		// The resource list for Oracle may contains table, view and temporary table.
		return plsqlparser.ExtractOracleResourceList(currentDatabase, currentSchema, sql)
	case Postgres, RisingWave:
		// The resource list for Postgres may contains table, view and temporary table.
		return pgparser.ExtractPostgresResourceList(currentDatabase, currentSchema, sql)
	case Snowflake:
		return snowparser.ExtractSnowflakeNormalizeResourceListFromSelectStatement(currentDatabase, "PUBLIC", sql)
	case MSSQL:
		return tsqlparser.ExtractMSSQLNormalizedResourceListFromSelectStatement(currentDatabase, "dbo", sql)
	default:
		if currentDatabase == "" {
			return nil, errors.Errorf("database must be specified for engine type: %s", engineType)
		}
		return []base.SchemaResource{{Database: currentDatabase}}, nil
	}
}

// SplitMultiSQL splits statement into a slice of the single SQL.
func SplitMultiSQL(engineType EngineType, statement string) ([]base.SingleSQL, error) {
	var list []base.SingleSQL
	var err error
	switch engineType {
	case Oracle:
		tree, tokens, err := plsqlparser.ParsePLSQL(statement)
		if err != nil {
			return nil, err
		}

		var result []base.SingleSQL
		for _, item := range tree.GetChildren() {
			if stmt, ok := item.(plsql.IUnit_statementContext); ok {
				stopIndex := stmt.GetStop().GetTokenIndex()
				if stmt.GetStop().GetTokenType() == plsql.PlSqlParserSEMICOLON {
					stopIndex--
				}
				lastToken := tokens.Get(stopIndex)
				result = append(result, base.SingleSQL{
					Text:     tokens.GetTextFromTokens(stmt.GetStart(), lastToken),
					LastLine: lastToken.GetLine(),
					Empty:    false,
				})
			}
		}
		return result, nil
	case MSSQL:
		t := tokenizer.NewTokenizer(statement)
		list, err = t.SplitStandardMultiSQL()
	case Postgres, Redshift, RisingWave:
		t := tokenizer.NewTokenizer(statement)
		list, err = t.SplitPostgreSQLMultiSQL()
	case MySQL, MariaDB, OceanBase:
		return mysqlparser.SplitMySQL(statement)
	case TiDB:
		t := tokenizer.NewTokenizer(statement)
		list, err = t.SplitTiDBMultiSQL()
	default:
		err = applyMultiStatements(strings.NewReader(statement), func(sql string) error {
			list = append(list, base.SingleSQL{
				Text:     sql,
				LastLine: 0,
				Empty:    false,
			})
			return nil
		})
	}

	if err != nil {
		return nil, err
	}

	var result []base.SingleSQL
	for _, sql := range list {
		if sql.Empty {
			continue
		}
		if engineType == Oracle {
			sql.Text = strings.TrimRight(sql.Text, " \n\t;")
		}
		result = append(result, sql)
	}
	return result, nil
}

// applyMultiStatements will apply the split statements from scanner.
// This function only used for SQLite, snowflake.
// For MySQL and PostgreSQL, use parser.SplitMultiSQLStream instead.
// Copy from plugin/db/util/driverutil.go.
func applyMultiStatements(sc io.Reader, f func(string) error) error {
	// TODO(rebelice): use parser/tokenizer to split SQL statements.
	reader := bufio.NewReader(sc)
	var sb strings.Builder
	delimiter := false
	comment := false
	done := false
	for !done {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				done = true
			} else {
				return err
			}
		}
		line = strings.TrimRight(line, "\r\n")

		execute := false
		switch {
		case strings.HasPrefix(line, "/*"):
			if strings.Contains(line, "*/") {
				if !strings.HasSuffix(line, "*/") {
					return errors.Errorf("`*/` must be the end of the line; new statement should start as a new line")
				}
			} else {
				comment = true
			}
			continue
		case comment && !strings.Contains(line, "*/"):
			// Skip the line when in comment mode.
			continue
		case comment && strings.Contains(line, "*/"):
			if !strings.HasSuffix(line, "*/") {
				return errors.Errorf("`*/` must be the end of the line; new statement should start as a new line")
			}
			comment = false
			continue
		case sb.Len() == 0 && line == "":
			continue
		case strings.HasPrefix(line, "--"):
			continue
		case line == "DELIMITER ;;":
			delimiter = true
			continue
		case line == "DELIMITER ;" && delimiter:
			delimiter = false
			execute = true
		case strings.HasSuffix(line, ";"):
			_, _ = sb.WriteString(line)
			_, _ = sb.WriteString("\n")
			if !delimiter {
				execute = true
			}
		default:
			_, _ = sb.WriteString(line)
			_, _ = sb.WriteString("\n")
			continue
		}
		if execute {
			s := sb.String()
			s = strings.Trim(s, "\n\t ")
			if s != "" {
				if err := f(s); err != nil {
					return errors.Wrapf(err, "execute query %q failed", s)
				}
			}
			sb.Reset()
		}
	}
	// Apply the remaining content.
	s := sb.String()
	s = strings.Trim(s, "\n\t ")
	if s != "" {
		if err := f(s); err != nil {
			return errors.Wrapf(err, "execute query %q failed", s)
		}
	}

	return nil
}

// SplitMultiSQLStream splits statement stream into a slice of the single SQL.
func SplitMultiSQLStream(engineType EngineType, src io.Reader, f func(string) error) ([]base.SingleSQL, error) {
	var list []base.SingleSQL
	var err error
	switch engineType {
	case Oracle:
		text := antlr.NewIoStream(src).String()
		sqls, err := SplitMultiSQL(engineType, text)
		if err != nil {
			return nil, err
		}
		for _, sql := range sqls {
			if f != nil {
				if err := f(sql.Text); err != nil {
					return nil, err
				}
			}
		}
		return sqls, nil
	case MSSQL:
		t := tokenizer.NewStreamTokenizer(src, f)
		list, err = t.SplitStandardMultiSQL()
	case Postgres, Redshift, RisingWave:
		t := tokenizer.NewStreamTokenizer(src, f)
		list, err = t.SplitPostgreSQLMultiSQL()
	case MySQL, MariaDB, OceanBase:
		return mysqlparser.SplitMySQLMultiSQLStream(src, f)
	case TiDB:
		t := tokenizer.NewStreamTokenizer(src, f)
		list, err = t.SplitTiDBMultiSQL()
	default:
		return nil, errors.Errorf("engine type is not supported: %s", engineType)
	}

	if err != nil {
		return nil, err
	}

	var result []base.SingleSQL
	for _, sql := range list {
		if sql.Empty {
			continue
		}
		if engineType == Oracle {
			sql.Text = strings.TrimRight(sql.Text, " \n\t;")
		}
		result = append(result, sql)
	}

	return result, nil
}

// SetLineForCreateTableStmt sets the line for columns and table constraints in CREATE TABLE statements.
func SetLineForCreateTableStmt(engineType EngineType, node *ast.CreateTableStmt) error {
	switch engineType {
	case Postgres:
		t := tokenizer.NewTokenizer(node.Text())
		firstLine := node.LastLine() - strings.Count(node.Text(), "\n")
		return t.SetLineForPGCreateTableStmt(node, firstLine)
	default:
		return errors.Errorf("engine type is not supported: %s", engineType)
	}
}

// SetLineForMySQLCreateTableStmt sets the line for columns and table constraints in MySQL CREATE TABLE statments.
// This is a temporary function. Because we do not convert tidb AST to our AST. So we have to implement this.
// TODO(rebelice): remove it.
func SetLineForMySQLCreateTableStmt(node *tidbast.CreateTableStmt) error {
	// exclude CREATE TABLE ... AS and CREATE TABLE ... LIKE statement.
	if len(node.Cols) == 0 {
		return nil
	}
	firstLine := node.OriginTextPosition() - strings.Count(node.Text(), "\n")
	return tokenizer.NewTokenizer(node.Text()).SetLineForMySQLCreateTableStmt(node, firstLine)
}

// ExtractTiDBUnsupportedStmts returns a list of unsupported statements in TiDB extracted from the `stmts`,
// and returns the remaining statements supported by TiDB from `stmts`.
func ExtractTiDBUnsupportedStmts(stmts string) ([]string, string, error) {
	var unsupportStmts []string
	var supportedStmts bytes.Buffer
	// We use our bb tokenizer to help us split the multi-statements into statement list.
	singleSQLs, err := SplitMultiSQL(MySQL, stmts)
	if err != nil {
		return nil, "", errors.Wrapf(err, "cannot split multi sql %q via bytebase parser", stmts)
	}
	for _, sql := range singleSQLs {
		content := sql.Text
		if isTiDBUnsupportStmt(content) {
			unsupportStmts = append(unsupportStmts, content)
			continue
		}
		_, _ = supportedStmts.WriteString(content)
		_, _ = supportedStmts.WriteString("\n")
	}
	return unsupportStmts, supportedStmts.String(), nil
}

// isTiDBUnsupportStmt returns true if this statement is unsupported in TiDB.
func isTiDBUnsupportStmt(stmt string) bool {
	if _, err := tidbparser.ParseTiDB(stmt, "", ""); err != nil {
		return true
	}
	return false
}

// IsTiDBUnsupportDDLStmt checks whether the `stmt` is unsupported DDL statement in TiDB, the following statements are unsupported:
// 1. `CREATE TRIGGER`
// 2. `CREATE EVENT`
// 3. `CREATE FUNCTION`
// 4. `CREATE PROCEDURE`.
func IsTiDBUnsupportDDLStmt(stmt string) bool {
	objects := []string{
		"TRIGGER",
		"EVENT",
		"FUNCTION",
		"PROCEDURE",
	}
	createRegexFmt := "(?i)^\\s*CREATE\\s+(DEFINER=(`(.)+`|(.)+)@(`(.)+`|(.)+)(\\s)+)?%s\\s+"
	dropRegexFmt := "(?i)^\\s*DROP\\s+%s\\s+"
	for _, obj := range objects {
		createRegexp := fmt.Sprintf(createRegexFmt, obj)
		re := regexp.MustCompile(createRegexp)
		if re.MatchString(stmt) {
			return true
		}
		dropRegexp := fmt.Sprintf(dropRegexFmt, obj)
		re = regexp.MustCompile(dropRegexp)
		if re.MatchString(stmt) {
			return true
		}
	}
	return false
}

// TypeString returns the string representation of the type for MySQL.
func TypeString(tp byte) string {
	switch tp {
	case mysql.TypeTiny:
		return "tinyint"
	case mysql.TypeShort:
		return "smallint"
	case mysql.TypeInt24:
		return "mediumint"
	case mysql.TypeLong:
		return "int"
	case mysql.TypeLonglong:
		return "bigint"
	case mysql.TypeFloat:
		return "float"
	case mysql.TypeDouble:
		return "double"
	case mysql.TypeNewDecimal:
		return "decimal"
	case mysql.TypeVarchar:
		return "varchar"
	case mysql.TypeBit:
		return "bit"
	case mysql.TypeTimestamp:
		return "timestamp"
	case mysql.TypeDatetime:
		return "datetime"
	case mysql.TypeDate:
		return "date"
	case mysql.TypeDuration:
		return "time"
	case mysql.TypeJSON:
		return "json"
	case mysql.TypeEnum:
		return "enum"
	case mysql.TypeSet:
		return "set"
	case mysql.TypeTinyBlob:
		return "tinyblob"
	case mysql.TypeMediumBlob:
		return "mediumblob"
	case mysql.TypeLongBlob:
		return "longblob"
	case mysql.TypeBlob:
		return "blob"
	case mysql.TypeVarString:
		return "varbinary"
	case mysql.TypeString:
		return "binary"
	case mysql.TypeGeometry:
		return "geometry"
	}
	return "unknown"
}

// ExtractDatabaseList extracts all databases from statement.
func ExtractDatabaseList(engineType EngineType, statement string, fallbackNormalizedDatabaseName string) ([]string, error) {
	switch engineType {
	case MySQL, TiDB, MariaDB, OceanBase:
		return tidbparser.ExtractDatabaseList(statement)
	case Snowflake:
		return extractSnowSQLNormalizedDatabaseList(statement, fallbackNormalizedDatabaseName)
	case MSSQL:
		return extractMSSQLNormalizedDatabaseList(statement, fallbackNormalizedDatabaseName)
	default:
		return nil, errors.Errorf("engine type is not supported: %s", engineType)
	}
}

func extractMSSQLNormalizedDatabaseList(statement string, normalizedDatabaseName string) ([]string, error) {
	schemaPlaceholder := "dbo"
	schemaResource, err := tsqlparser.ExtractMSSQLNormalizedResourceListFromSelectStatement(normalizedDatabaseName, schemaPlaceholder, statement)
	if err != nil {
		return nil, err
	}

	var result []string
	for _, resource := range schemaResource {
		result = append(result, resource.Database)
	}
	return result, nil
}

// extractSnowSQLNormalizedDatabaseList extracts all databases from statement, and normalizes the database name.
// If the database name is not specified, it will fallback to the normalizedDatabaseName.
func extractSnowSQLNormalizedDatabaseList(statement string, normalizedDatabaseName string) ([]string, error) {
	schemaPlaceholder := "schema_placeholder"
	schemaResource, err := snowparser.ExtractSnowflakeNormalizeResourceListFromSelectStatement(normalizedDatabaseName, schemaPlaceholder, statement)
	if err != nil {
		return nil, err
	}

	var result []string
	for _, resource := range schemaResource {
		result = append(result, resource.Database)
	}
	return result, nil
}
