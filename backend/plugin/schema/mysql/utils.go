package mysql

import (
	"strconv"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	mysql "github.com/bytebase/mysql-parser"

	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

const (
	autoIncrementSymbol = "AUTO_INCREMENT"
	autoRandSymbol      = "AUTO_RANDOM"
)

var (
	// https://dev.mysql.com/doc/refman/8.0/en/data-type-defaults.html
	// expressionDefaultOnlyTypes is a list of types that only accept expression as default
	// value. While we restore the following types, we should not restore the default null.
	// +-------+--------------------------------------------------------------------+
	// | Table | Create Table                                                       |
	// +-------+--------------------------------------------------------------------+
	// | u     | CREATE TABLE `u` (                                                 |
	// |       |   `b` blob,                                                        |
	// |       |   `t` text,                                                        |
	// |       |   `g` geometry DEFAULT NULL,                                       |
	// |       |   `j` json DEFAULT NULL                                            |
	// |       | ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci |
	// +-------+--------------------------------------------------------------------+.
	expressionDefaultOnlyTypes = map[string]bool{
		// BLOB & TEXT
		// https://dev.mysql.com/doc/refman/8.0/en/blob.html
		"TINYBLOB":   true,
		"BLOB":       true,
		"MEIDUMBLOB": true,
		"LONGBLOB":   true,
		"TINYTEXT":   true,
		"TEXT":       true,
		"MEDIUMTEXT": true,
		"LONGTEXT":   true,

		// In practice, the following types restore the default null by mysqldump.
		// // GEOMETRY
		// "GEOMETRY": true,
		// // JSON
		// // https://dev.mysql.com/doc/refman/8.0/en/json.html
		// "JSON": true,
	}
)

func extractReference(ctx mysql.IReferencesContext) (string, []string) {
	_, table := mysqlparser.NormalizeMySQLTableRef(ctx.TableRef())
	if ctx.IdentifierListWithParentheses() != nil {
		columns := extractIdentifierList(ctx.IdentifierListWithParentheses().IdentifierList())
		return table, columns
	}
	return table, nil
}

func extractIdentifierList(ctx mysql.IIdentifierListContext) []string {
	var result []string
	for _, identifier := range ctx.AllIdentifier() {
		result = append(result, mysqlparser.NormalizeMySQLIdentifier(identifier))
	}
	return result
}

func extractKeyListVariants(ctx mysql.IKeyListVariantsContext) ([]string, []int64) {
	if ctx.KeyList() != nil {
		return extractKeyList(ctx.KeyList())
	}
	if ctx.KeyListWithExpression() != nil {
		return extractKeyListWithExpression(ctx.KeyListWithExpression())
	}
	return nil, nil
}

func extractKeyListWithExpression(ctx mysql.IKeyListWithExpressionContext) ([]string, []int64) {
	var result []string
	var keyLengths []int64
	for _, key := range ctx.AllKeyPartOrExpression() {
		if key.KeyPart() != nil {
			keyText, keyLength := getKeyExpressionAndLengthFromKeyPart(key.KeyPart())
			result = append(result, keyText)
			keyLengths = append(keyLengths, keyLength)
		} else if key.ExprWithParentheses() != nil {
			keyText := key.GetParser().GetTokenStream().GetTextFromRuleContext(key.ExprWithParentheses())
			result = append(result, keyText)
			keyLengths = append(keyLengths, -1)
		}
	}
	return result, keyLengths
}

func extractKeyList(ctx mysql.IKeyListContext) ([]string, []int64) {
	var result []string
	var keyLengths []int64
	for _, key := range ctx.AllKeyPart() {
		keyText, keyLength := getKeyExpressionAndLengthFromKeyPart(key)
		result = append(result, keyText)
		keyLengths = append(keyLengths, keyLength)
	}
	return result, keyLengths
}

func getKeyExpressionAndLengthFromKeyPart(ctx mysql.IKeyPartContext) (string, int64) {
	keyText := mysqlparser.NormalizeMySQLIdentifier(ctx.Identifier())
	keyLength := int64(-1)
	if ctx.FieldLength() != nil {
		l := ctx.FieldLength().GetText()
		if len(l) > 2 && l[0] == '(' && l[len(l)-1] == ')' {
			l = l[1 : len(l)-1]
		}
		length, err := strconv.ParseInt(l, 10, 64)
		if err != nil {
			length = -1
		}
		keyLength = length
	}
	return keyText, keyLength
}

type columnAttr struct {
	text  string
	order int
}

var columnAttrOrder = map[string]int{
	"NULL":           1,
	"DEFAULT":        2,
	"VISIBLE":        3,
	"AUTO_INCREMENT": 4,
	"AUTO_RAND":      4,
	"UNIQUE":         5,
	"KEY":            6,
	"COMMENT":        7,
	"COLLATE":        8,
	"COLUMN_FORMAT":  9,
	"SECONDARY":      10,
	"STORAGE":        11,
	"SERIAL":         12,
	"SRID":           13,
	"ON":             14,
	"CHECK":          15,
	"ENFORCED":       16,
}

func extractNewAttrs(column *columnState, attrs []mysql.IColumnAttributeContext) []columnAttr {
	var result []columnAttr
	nullExists := false
	defaultExists := false
	commentExists := false
	for _, attr := range attrs {
		if attr.GetValue() != nil {
			switch strings.ToUpper(attr.GetValue().GetText()) {
			case "DEFAULT":
				defaultExists = true
			case "COMMENT":
				commentExists = true
			}
		} else if attr.NullLiteral() != nil {
			nullExists = true
		}
	}

	if !nullExists && !column.nullable {
		result = append(result, columnAttr{
			text:  "NOT NULL",
			order: columnAttrOrder["NULL"],
		})
	}
	if !defaultExists && column.hasDefault {
		// todo(zp): refactor column attribute.
		if strings.EqualFold(column.defaultValue.toString(), "AUTO_INCREMENT") {
			result = append(result, columnAttr{
				text:  column.defaultValue.toString(),
				order: columnAttrOrder["DEFAULT"],
			})
		} else {
			result = append(result, columnAttr{
				text:  "DEFAULT " + column.defaultValue.toString(),
				order: columnAttrOrder["DEFAULT"],
			})
		}
	}
	if !commentExists && column.comment != "" {
		result = append(result, columnAttr{
			text:  "COMMENT '" + column.comment + "'",
			order: columnAttrOrder["COMMENT"],
		})
	}
	return result
}

func getAttrOrder(attr mysql.IColumnAttributeContext) int {
	if attr.GetValue() != nil {
		switch strings.ToUpper(attr.GetValue().GetText()) {
		case "DEFAULT", "ON", "AUTO_INCREMENT", "SERIAL", "KEY", "UNIQUE", "COMMENT", "COLUMN_FORMAT", "STORAGE", "SRID":
			return columnAttrOrder[attr.GetValue().GetText()]
		}
	}
	if attr.NullLiteral() != nil {
		return columnAttrOrder["NULL"]
	}
	if attr.SECONDARY_SYMBOL() != nil {
		return columnAttrOrder["SECONDARY"]
	}
	if attr.Collate() != nil {
		return columnAttrOrder["COLLATE"]
	}
	if attr.CheckConstraint() != nil {
		return columnAttrOrder["CHECK"]
	}
	if attr.ConstraintEnforcement() != nil {
		return columnAttrOrder["ENFORCED"]
	}
	return len(columnAttrOrder) + 1
}

func equalKeys(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, key := range a {
		if key != b[i] {
			return false
		}
	}
	return true
}

func equalKeyLengths(a, b []int64) bool {
	if len(a) != len(b) {
		return false
	}
	for i, length := range a {
		if length != b[i] {
			return false
		}
	}
	return true
}

func nextDefaultChannelTokenIndex(tokens antlr.TokenStream, currentIndex int) int {
	for i := currentIndex + 1; i < tokens.Size(); i++ {
		if tokens.Get(i).GetChannel() == antlr.TokenDefaultChannel {
			return i
		}
	}
	return 0
}
