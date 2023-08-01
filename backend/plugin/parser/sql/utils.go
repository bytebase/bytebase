package parser

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	pgquery "github.com/pganalyze/pg_query_go/v2"
	tidbparser "github.com/pingcap/tidb/parser"
	tidbast "github.com/pingcap/tidb/parser/ast"
	"github.com/pingcap/tidb/parser/model"
	"github.com/pingcap/tidb/parser/mysql"
	"github.com/pkg/errors"

	plsql "github.com/bytebase/plsql-parser"

	"github.com/bytebase/bytebase/backend/plugin/parser/sql/ast"
)

// SingleSQL is a separate SQL split from multi-SQL.
type SingleSQL struct {
	Text     string
	BaseLine int
	LastLine int
	// The sql is empty, such as `/* comments */;` or just `;`.
	Empty bool
}

// SchemaResource is the resource of the schema.
type SchemaResource struct {
	Database string
	Schema   string
	Table    string
}

// String implements fmt.Stringer interface.
func (r SchemaResource) String() string {
	return fmt.Sprintf("%s.%s.%s", r.Database, r.Schema, r.Table)
}

// Pretty returns the pretty string of the resource.
func (r SchemaResource) Pretty() string {
	list := make([]string, 0, 3)
	if r.Database != "" {
		list = append(list, r.Database)
	}
	if r.Schema != "" {
		list = append(list, r.Schema)
	}
	if r.Table != "" {
		list = append(list, r.Table)
	}
	return strings.Join(list, ".")
}

// ExtractChangedResources extracts the changed resources from the SQL.
func ExtractChangedResources(engineType EngineType, currentDatabase string, _ string, sql string) ([]SchemaResource, error) {
	switch engineType {
	case MySQL, MariaDB, OceanBase:
		return extractMySQLChangedResources(currentDatabase, sql)
	default:
		if currentDatabase == "" {
			return nil, errors.Errorf("database must be specified for engine type: %s", engineType)
		}
		return nil, errors.Errorf("engine type %q is not supported", engineType)
	}
}

// ExtractResourceList extracts the resource list from the SQL.
func ExtractResourceList(engineType EngineType, currentDatabase string, currentSchema string, sql string) ([]SchemaResource, error) {
	switch engineType {
	case TiDB:
		return extractTiDBResourceList(currentDatabase, sql)
	case MySQL, MariaDB, OceanBase:
		// The resource list for MySQL may contains table, view and temporary table.
		return extractMySQLResourceList(currentDatabase, sql)
	case Oracle:
		// The resource list for Oracle may contains table, view and temporary table.
		return extractOracleResourceList(currentDatabase, currentSchema, sql)
	case Postgres:
		// The resource list for Postgres may contains table, view and temporary table.
		return extractPostgresResourceList(currentDatabase, "public", sql)
	case Snowflake:
		return extractSnowflakeNormalizeResourceListFromSelectStatement(currentDatabase, "PUBLIC", sql)
	default:
		if currentDatabase == "" {
			return nil, errors.Errorf("database must be specified for engine type: %s", engineType)
		}
		return []SchemaResource{{Database: currentDatabase}}, nil
	}
}

func extractPostgresResourceList(currentDatabase string, currentSchema string, sql string) ([]SchemaResource, error) {
	jsonText, err := pgquery.ParseToJSON(sql)
	if err != nil {
		return nil, err
	}

	var jsonData map[string]any

	if err := json.Unmarshal([]byte(jsonText), &jsonData); err != nil {
		return nil, err
	}

	resourceMap := make(map[string]SchemaResource)
	list := extractRangeVarFromJSON(currentDatabase, currentSchema, jsonData)
	for _, resource := range list {
		resourceMap[resource.String()] = resource
	}
	list = []SchemaResource{}
	for _, resource := range resourceMap {
		list = append(list, resource)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].String() < list[j].String()
	})
	return list, nil
}

func extractRangeVarFromJSON(currentDatabase string, currentSchema string, jsonData map[string]any) []SchemaResource {
	var result []SchemaResource
	if jsonData["RangeVar"] != nil {
		resource := SchemaResource{
			Database: currentDatabase,
			Schema:   currentSchema,
		}
		rangeVar := jsonData["RangeVar"].(map[string]any)
		if rangeVar["schemaname"] != nil {
			resource.Schema = rangeVar["schemaname"].(string)
		}
		if rangeVar["relname"] != nil {
			resource.Table = rangeVar["relname"].(string)
		}
		result = append(result, resource)
	}

	for _, value := range jsonData {
		switch v := value.(type) {
		case map[string]any:
			result = append(result, extractRangeVarFromJSON(currentDatabase, currentSchema, v)...)
		case []any:
			for _, item := range v {
				if m, ok := item.(map[string]any); ok {
					result = append(result, extractRangeVarFromJSON(currentDatabase, currentSchema, m)...)
				}
			}
		}
	}

	return result
}

func extractTiDBResourceList(currentDatabase string, sql string) ([]SchemaResource, error) {
	nodes, err := ParseTiDB(sql, "", "")
	if err != nil {
		return nil, err
	}

	resourceMap := make(map[string]SchemaResource)

	for _, node := range nodes {
		tableList := ExtractMySQLTableList(node, false /* asName */)
		for _, table := range tableList {
			resource := SchemaResource{
				Database: table.Schema.O,
				Schema:   "",
				Table:    table.Name.O,
			}
			if resource.Database == "" {
				resource.Database = currentDatabase
			}
			if _, ok := resourceMap[resource.String()]; !ok {
				resourceMap[resource.String()] = resource
			}
		}
	}

	resourceList := make([]SchemaResource, 0, len(resourceMap))
	for _, resource := range resourceMap {
		resourceList = append(resourceList, resource)
	}
	sort.Slice(resourceList, func(i, j int) bool {
		return resourceList[i].String() < resourceList[j].String()
	})

	return resourceList, nil
}

// GetSQLFingerprint returns the fingerprint of the SQL.
func GetSQLFingerprint(engineType EngineType, sql string) (string, error) {
	switch engineType {
	case MySQL, TiDB, MariaDB:
		return getMySQLFingerprint(sql)
	default:
		return "", errors.Errorf("engine type is not supported: %s", engineType)
	}
}

// From https://github.com/percona/percona-toolkit/blob/af686fe186d1fca4c4392c8fa75c31a00c8fb273/bin/pt-query-digest#L2930
func getMySQLFingerprint(query string) (string, error) {
	// Match SQL queries generated by mysqldump command.
	if matched, _ := regexp.MatchString(`\ASELECT /\*!40001 SQL_NO_CACHE \*/ \* FROM `, query); matched {
		return "mysqldump", nil
	}
	// Match SQL queries generated by Percona Toolkit.
	if matched, _ := regexp.MatchString(`/\*\w+\.\w+:[0-9]/[0-9]\*/`, query); matched {
		return "percona-toolkit", nil
	}
	// Match administrator commands.
	if matched, _ := regexp.MatchString(`\Aadministrator command: `, query); matched {
		return query, nil
	}
	// Match stored procedure call statements.
	if matched, _ := regexp.MatchString(`\A\s*(call\s+\S+)\(`, query); matched {
		return strings.ToLower(regexp.MustCompile(`\A\s*(call\s+\S+)\(`).FindStringSubmatch(query)[1]), nil
	}
	// Match INSERT INTO or REPLACE INTO statements.
	if beginning := regexp.MustCompile(`(?i)((?:INSERT|REPLACE)(?: IGNORE)?\s+INTO.+?VALUES\s*\(.*?\))\s*,\s*\(`).FindStringSubmatch(query); len(beginning) > 0 {
		query = beginning[1]
	}

	// Match multi-line comments and single-line comments, and remove them.
	mlcRe := regexp.MustCompile(`(?s)/\*.*?\*/`)
	olcRe := regexp.MustCompile(`(?m)--.*$`)
	query = mlcRe.ReplaceAllString(query, "")
	query = olcRe.ReplaceAllString(query, "")

	// Replace the database name in USE statements with a question mark (?).
	query = regexp.MustCompile(`(?i)\Ause \S+\z`).ReplaceAllString(query, "use ?")

	// Replace escape characters and special characters in SQL queries with a question mark (?).
	query = regexp.MustCompile(`([^\\])(\\')`).ReplaceAllString(query, "$1")
	query = regexp.MustCompile(`([^\\])(\\")`).ReplaceAllString(query, "$1")
	query = regexp.MustCompile(`\\\\`).ReplaceAllString(query, "")
	query = regexp.MustCompile(`\\'`).ReplaceAllString(query, "")
	query = regexp.MustCompile(`\\"`).ReplaceAllString(query, "")
	query = regexp.MustCompile(`([^\\])(".*?[^\\]?")`).ReplaceAllString(query, "$1?")
	query = regexp.MustCompile(`([^\\])('.*?[^\\]?')`).ReplaceAllString(query, "$1?")

	// Replace boolean values in SQL queries with a question mark (?).
	query = regexp.MustCompile(`\bfalse\b|\btrue\b`).ReplaceAllString(query, "?")

	// Replace MD5 values in SQL queries with a question mark (?).
	if matched, _ := regexp.MatchString(`([._-])[a-f0-9]{32}`, query); matched {
		query = regexp.MustCompile(`([._-])[a-f0-9]{32}`).ReplaceAllString(query, "$1?")
	}

	// Replace numbers in SQL queries with a question mark (?).
	if matched, _ := regexp.MatchString(`\b[0-9+-][0-9a-f.xb+-]*`, query); matched {
		query = regexp.MustCompile(`\b[0-9+-][0-9a-f.xb+-]*`).ReplaceAllString(query, "?")
	}

	// Replace special characters in SQL queries with a question mark (?).
	if matched, _ := regexp.MatchString(`[xb+-]\?`, query); matched {
		query = regexp.MustCompile(`[xb+-]\?`).ReplaceAllString(query, "?")
	} else {
		query = regexp.MustCompile(`[xb.+-]\?`).ReplaceAllString(query, "?")
	}

	// Remove spaces and line breaks in SQL queries.
	query = strings.TrimSpace(query)
	query = strings.TrimRight(query, "\n\r\f ")
	query = regexp.MustCompile(`\s+`).ReplaceAllString(query, " ")
	query = strings.ToLower(query)

	// Replace NULL values in SQL queries with a question mark (?).
	query = regexp.MustCompile(`\bnull\b`).ReplaceAllString(query, "?")

	// Replace IN and VALUES clauses in SQL queries with a question mark (?).
	query = regexp.MustCompile(`\b(in|values?)(?:[\s,]*\([\s?,]*\))+`).ReplaceAllString(query, "$1(?+)")

	var err error
	query, err = collapseUnion(query)
	if err != nil {
		return "", err
	}

	// Replace numbers in the LIMIT clause of SQL queries with a question mark (?).
	query = regexp.MustCompile(`\blimit \?(?:, ?\?| offset \?)?`).ReplaceAllString(query, "limit ?")

	// Remove ASC sorting in SQL queries.
	if matched, _ := regexp.MatchString(`\border by `, query); matched {
		ascRegexp := regexp.MustCompile(`(.+?)\s+asc`)
		for {
			if matched := ascRegexp.MatchString(query); matched {
				query = ascRegexp.ReplaceAllString(query, "$1")
			} else {
				break
			}
		}
	}

	return query, nil
}

func collapseUnion(query string) (string, error) {
	// The origin perl code is:
	//   $query =~ s{                          # Collapse UNION
	//     \b(select\s.*?)(?:(\sunion(?:\sall)?)\s\1)+
	//	  }
	//	  {$1 /*repeat$2*/}xg;
	// But Golang doesn't support \1(back-reference).
	// So we use the following code to replace it.
	unionRegexp := regexp.MustCompile(`\s(union all|union)\s`)
	parts := unionRegexp.Split(query, -1)
	if len(parts) == 1 {
		return query, nil
	}
	// Add a sentinel node to the end of the slice.
	// Because we remove all comments before, so all parts are different from sentinel node.
	parts = append(parts, "/*Sentinel Node*/")
	separators := unionRegexp.FindAllString(query, -1)
	if len(parts) != len(separators)+2 {
		return "", errors.Errorf("find %d parts, but %d separators", len(parts)-1, len(separators))
	}
	start := 0
	var buf bytes.Buffer
	if _, err := buf.WriteString(parts[start]); err != nil {
		return "", err
	}
	for i, part := range parts {
		if i == 0 {
			continue
		}
		if part == parts[start] {
			continue
		}
		if i == start+1 {
			// The i-th part is not equal to the front part.
			if _, err := buf.WriteString(separators[i-1]); err != nil {
				return "", err
			}
		} else {
			// deal with the same parts[start, i-1] and start < i-1.
			if _, err := buf.WriteString(" /*repeat"); err != nil {
				return "", err
			}
			// Write the last separator between the same parts[start, i-1].
			// In other words, the last separator is the separator between the i-th part and the (i-1)-th part.
			// So the index of the last separator is (i-1)-1.
			if _, err := buf.WriteString(separators[(i-1)-1]); err != nil {
				return "", err
			}
			if _, err := buf.WriteString("*/"); err != nil {
				return "", err
			}
		}
		start = i
		// Don't write the sentinel node.
		if i != len(parts)-1 {
			if _, err := buf.WriteString(parts[start]); err != nil {
				return "", err
			}
		}
	}
	return buf.String(), nil
}

// SplitMultiSQLAndNormalize split multiple SQLs and normalize them.
// For MySQL, filter DELIMITER statements and replace all non-semicolon delimiters with semicolons.
func SplitMultiSQLAndNormalize(engineType EngineType, statement string) ([]SingleSQL, error) {
	switch engineType {
	case MySQL:
		has, list, err := hasDelimiter(statement)
		if err != nil {
			return nil, err
		}
		if has {
			var result []SingleSQL
			delimiter := `;`
			for _, sql := range list {
				if IsDelimiter(sql.Text) {
					delimiter, err = ExtractDelimiter(sql.Text)
					if err != nil {
						return nil, err
					}
					continue
				}
				if delimiter != ";" {
					result = append(result, SingleSQL{
						Text:     fmt.Sprintf("%s;", strings.TrimSuffix(sql.Text, delimiter)),
						LastLine: sql.LastLine,
						Empty:    sql.Empty,
					})
				} else {
					result = append(result, sql)
				}
			}
			return result, nil
		}

		return SplitMultiSQL(MySQL, statement)
	default:
		return SplitMultiSQL(engineType, statement)
	}
}

// SplitMultiSQL splits statement into a slice of the single SQL.
func SplitMultiSQL(engineType EngineType, statement string) ([]SingleSQL, error) {
	var list []SingleSQL
	var err error
	switch engineType {
	case Oracle:
		tree, tokens, err := ParsePLSQL(statement)
		if err != nil {
			return nil, err
		}

		var result []SingleSQL
		for _, item := range tree.GetChildren() {
			if stmt, ok := item.(plsql.IUnit_statementContext); ok {
				stopIndex := stmt.GetStop().GetTokenIndex()
				if stmt.GetStop().GetTokenType() == plsql.PlSqlParserSEMICOLON {
					stopIndex--
				}
				lastToken := tokens.Get(stopIndex)
				result = append(result, SingleSQL{
					Text:     tokens.GetTextFromTokens(stmt.GetStart(), lastToken),
					LastLine: lastToken.GetLine(),
					Empty:    false,
				})
			}
		}
		return result, nil
	case MSSQL:
		t := newTokenizer(statement)
		list, err = t.splitStandardMultiSQL()
	case Postgres, Redshift:
		t := newTokenizer(statement)
		list, err = t.splitPostgreSQLMultiSQL()
	case MySQL, MariaDB, OceanBase:
		return SplitMySQL(statement)
	case TiDB:
		t := newTokenizer(statement)
		list, err = t.splitTiDBMultiSQL()
	default:
		err = applyMultiStatements(strings.NewReader(statement), func(sql string) error {
			list = append(list, SingleSQL{
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

	var result []SingleSQL
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

// Note that the reader is read completely into memory and so it must actually
// have a stopping point - you cannot pass in a reader on an open-ended source such
// as a socket for instance.
func splitMySQLMultiSQLStream(src io.Reader, f func(string) error) ([]SingleSQL, error) {
	result, err := SplitMySQLStream(src)
	if err != nil {
		return nil, err
	}

	for _, sql := range result {
		if f != nil {
			if err := f(sql.Text); err != nil {
				return nil, err
			}
		}
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
func SplitMultiSQLStream(engineType EngineType, src io.Reader, f func(string) error) ([]SingleSQL, error) {
	var list []SingleSQL
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
		t := newStreamTokenizer(src, f)
		list, err = t.splitStandardMultiSQL()
	case Postgres, Redshift:
		t := newStreamTokenizer(src, f)
		list, err = t.splitPostgreSQLMultiSQL()
	case MySQL, MariaDB, OceanBase:
		return splitMySQLMultiSQLStream(src, f)
	case TiDB:
		t := newStreamTokenizer(src, f)
		list, err = t.splitTiDBMultiSQL()
	default:
		return nil, errors.Errorf("engine type is not supported: %s", engineType)
	}

	if err != nil {
		return nil, err
	}

	var result []SingleSQL
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
		t := newTokenizer(node.Text())
		firstLine := node.LastLine() - strings.Count(node.Text(), "\n")
		return t.setLineForPGCreateTableStmt(node, firstLine)
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
	return newTokenizer(node.Text()).setLineForMySQLCreateTableStmt(node, firstLine)
}

// ExtractTiDBUnsupportStmts returns a list of unsupported statements in TiDB extracted from the `stmts`,
// and returns the remaining statements supported by TiDB from `stmts`.
func ExtractTiDBUnsupportStmts(stmts string) ([]string, string, error) {
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
		_, _ = supportedStmts.Write([]byte(content))
		_, _ = supportedStmts.Write([]byte("\n"))
	}
	return unsupportStmts, supportedStmts.String(), nil
}

// isTiDBUnsupportStmt returns true if this statement is unsupported in TiDB.
func isTiDBUnsupportStmt(stmt string) bool {
	if _, err := ParseTiDB(stmt, "", ""); err != nil {
		return true
	}
	return false
}

func hasDelimiter(statement string) (bool, []SingleSQL, error) {
	// use splitTiDBMultiSQL to check if the statement has delimiter
	list, err := SplitMultiSQL(TiDB, statement)
	if err != nil {
		return false, nil, errors.Errorf("failed to split multi sql: %v", err)
	}

	for _, sql := range list {
		if IsDelimiter(sql.Text) {
			return true, list, nil
		}
	}

	return false, list, nil
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

// IsDelimiter returns true if the statement is a delimiter statement.
func IsDelimiter(stmt string) bool {
	delimiterRegex := `(?i)^\s*DELIMITER\s+`
	re := regexp.MustCompile(delimiterRegex)
	return re.MatchString(stmt)
}

// ExtractDelimiter extracts the delimiter from the delimiter statement.
func ExtractDelimiter(stmt string) (string, error) {
	delimiterRegex := `(?i)^\s*DELIMITER\s+(?P<DELIMITER>[^\s\\]+)\s*`
	re := regexp.MustCompile(delimiterRegex)
	matchList := re.FindStringSubmatch(stmt)
	index := re.SubexpIndex("DELIMITER")
	if index >= 0 && index < len(matchList) {
		return matchList[index], nil
	}
	return "", errors.Errorf("cannot extract delimiter from %q", stmt)
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
		return extractMySQLDatabaseList(statement)
	case Snowflake:
		return extractSnowSQLNormalizedDatabaseList(statement, fallbackNormalizedDatabaseName)
	default:
		return nil, errors.Errorf("engine type is not supported: %s", engineType)
	}
}

// extractSnowSQLNormalizedDatabaseList extracts all databases from statement, and normalizes the database name.
// If the database name is not specified, it will fallback to the normalizedDatabaseName.
func extractSnowSQLNormalizedDatabaseList(statement string, normalizedDatabaseName string) ([]string, error) {
	schemaPlaceholder := "schema_placeholder"
	schemaResource, err := extractSnowflakeNormalizeResourceListFromSelectStatement(normalizedDatabaseName, schemaPlaceholder, statement)
	if err != nil {
		return nil, err
	}

	var result []string
	for _, resource := range schemaResource {
		result = append(result, resource.Database)
	}
	return result, nil
}

func newMySQLParser() *tidbparser.Parser {
	p := tidbparser.New()

	// To support MySQL8 window function syntax.
	// See https://github.com/bytebase/bytebase/issues/175.
	p.EnableWindowFunc(true)

	return p
}

func extractMySQLDatabaseList(statement string) ([]string, error) {
	databaseMap := make(map[string]bool)

	p := newMySQLParser()
	nodeList, _, err := p.Parse(statement, "", "")
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parser statement %q", statement)
	}

	for _, node := range nodeList {
		databaseList := extractMySQLDatabaseListFromNode(node)
		for _, database := range databaseList {
			databaseMap[database] = true
		}
	}

	var databaseList []string
	for database := range databaseMap {
		databaseList = append(databaseList, database)
	}
	sort.Slice(databaseList, func(i, j int) bool {
		return databaseList[i] < databaseList[j]
	})
	return databaseList, nil
}

// extractMySQLDatabaseListFromNode extracts all the database from node.
func extractMySQLDatabaseListFromNode(in tidbast.Node) []string {
	tableNameList := ExtractMySQLTableList(in, false /* asName */)

	databaseMap := make(map[string]bool)
	for _, tableName := range tableNameList {
		databaseMap[tableName.Schema.O] = true
	}

	var databaseList []string
	for databaseName := range databaseMap {
		databaseList = append(databaseList, databaseName)
	}

	sort.Strings(databaseList)
	return databaseList
}

// ExtractMySQLTableList extracts all the TableNames from node.
// If asName is true, extract AsName prior to OrigName.
func ExtractMySQLTableList(in tidbast.Node, asName bool) []*tidbast.TableName {
	input := []*tidbast.TableName{}
	return extractTableList(in, input, asName)
}

// -------------------------------------------- DO NOT TOUCH --------------------------------------------

// extractTableList extracts all the TableNames from node.
// If asName is true, extract AsName prior to OrigName.
// Privilege check should use OrigName, while expression may use AsName.
// WARNING: copy from TiDB core code, do NOT touch!
func extractTableList(node tidbast.Node, input []*tidbast.TableName, asName bool) []*tidbast.TableName {
	switch x := node.(type) {
	case *tidbast.SelectStmt:
		if x.From != nil {
			input = extractTableList(x.From.TableRefs, input, asName)
		}
		if x.Where != nil {
			input = extractTableList(x.Where, input, asName)
		}
		if x.With != nil {
			for _, cte := range x.With.CTEs {
				input = extractTableList(cte.Query, input, asName)
			}
		}
		for _, f := range x.Fields.Fields {
			if s, ok := f.Expr.(*tidbast.SubqueryExpr); ok {
				input = extractTableList(s, input, asName)
			}
		}
	case *tidbast.DeleteStmt:
		input = extractTableList(x.TableRefs.TableRefs, input, asName)
		if x.IsMultiTable {
			for _, t := range x.Tables.Tables {
				input = extractTableList(t, input, asName)
			}
		}
		if x.Where != nil {
			input = extractTableList(x.Where, input, asName)
		}
		if x.With != nil {
			for _, cte := range x.With.CTEs {
				input = extractTableList(cte.Query, input, asName)
			}
		}
	case *tidbast.UpdateStmt:
		input = extractTableList(x.TableRefs.TableRefs, input, asName)
		for _, e := range x.List {
			input = extractTableList(e.Expr, input, asName)
		}
		if x.Where != nil {
			input = extractTableList(x.Where, input, asName)
		}
		if x.With != nil {
			for _, cte := range x.With.CTEs {
				input = extractTableList(cte.Query, input, asName)
			}
		}
	case *tidbast.InsertStmt:
		input = extractTableList(x.Table.TableRefs, input, asName)
		input = extractTableList(x.Select, input, asName)
	case *tidbast.SetOprStmt:
		l := &tidbast.SetOprSelectList{}
		unfoldSelectList(x.SelectList, l)
		for _, s := range l.Selects {
			input = extractTableList(s.(tidbast.ResultSetNode), input, asName)
		}
	case *tidbast.PatternInExpr:
		if s, ok := x.Sel.(*tidbast.SubqueryExpr); ok {
			input = extractTableList(s, input, asName)
		}
	case *tidbast.ExistsSubqueryExpr:
		if s, ok := x.Sel.(*tidbast.SubqueryExpr); ok {
			input = extractTableList(s, input, asName)
		}
	case *tidbast.BinaryOperationExpr:
		if s, ok := x.R.(*tidbast.SubqueryExpr); ok {
			input = extractTableList(s, input, asName)
		}
	case *tidbast.SubqueryExpr:
		input = extractTableList(x.Query, input, asName)
	case *tidbast.Join:
		input = extractTableList(x.Left, input, asName)
		input = extractTableList(x.Right, input, asName)
	case *tidbast.TableSource:
		if s, ok := x.Source.(*tidbast.TableName); ok {
			if x.AsName.L != "" && asName {
				newTableName := *s
				newTableName.Name = x.AsName
				newTableName.Schema = model.NewCIStr("")
				input = append(input, &newTableName)
			} else {
				input = append(input, s)
			}
		} else if s, ok := x.Source.(*tidbast.SelectStmt); ok {
			if s.From != nil {
				var innerList []*tidbast.TableName
				innerList = extractTableList(s.From.TableRefs, innerList, asName)
				if len(innerList) > 0 {
					innerTableName := innerList[0]
					if x.AsName.L != "" && asName {
						newTableName := *innerList[0]
						newTableName.Name = x.AsName
						newTableName.Schema = model.NewCIStr("")
						innerTableName = &newTableName
					}
					input = append(input, innerTableName)
				}
			}
		}
	}
	return input
}

// WARNING: copy from TiDB core code, do NOT touch!
func unfoldSelectList(list *tidbast.SetOprSelectList, unfoldList *tidbast.SetOprSelectList) {
	for _, sel := range list.Selects {
		switch s := sel.(type) {
		case *tidbast.SelectStmt:
			unfoldList.Selects = append(unfoldList.Selects, s)
		case *tidbast.SetOprSelectList:
			unfoldSelectList(s, unfoldList)
		}
	}
}
