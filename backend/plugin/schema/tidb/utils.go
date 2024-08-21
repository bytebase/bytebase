package tidb

import (
	"strconv"

	"github.com/antlr4-go/antlr/v4"
	tidb "github.com/bytebase/tidb-parser"

	tidbparser "github.com/bytebase/bytebase/backend/plugin/parser/tidb"
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

func extractReference(ctx tidb.IReferencesContext) (string, []string) {
	_, table := tidbparser.NormalizeTiDBTableRef(ctx.TableRef())
	if ctx.IdentifierListWithParentheses() != nil {
		columns := extractIdentifierList(ctx.IdentifierListWithParentheses().IdentifierList())
		return table, columns
	}
	return table, nil
}

func extractIdentifierList(ctx tidb.IIdentifierListContext) []string {
	var result []string
	for _, identifier := range ctx.AllIdentifier() {
		result = append(result, tidbparser.NormalizeTiDBIdentifier(identifier))
	}
	return result
}

func extractKeyListVariants(ctx tidb.IKeyListVariantsContext) ([]string, []int64) {
	if ctx.KeyList() != nil {
		return extractKeyList(ctx.KeyList())
	}
	if ctx.KeyListWithExpression() != nil {
		return extractKeyListWithExpression(ctx.KeyListWithExpression())
	}
	return nil, nil
}

func extractKeyListWithExpression(ctx tidb.IKeyListWithExpressionContext) ([]string, []int64) {
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

func extractKeyList(ctx tidb.IKeyListContext) ([]string, []int64) {
	var result []string
	var keyLengths []int64
	for _, key := range ctx.AllKeyPart() {
		keyText, keyLength := getKeyExpressionAndLengthFromKeyPart(key)
		result = append(result, keyText)
		keyLengths = append(keyLengths, keyLength)
	}
	return result, keyLengths
}

func getKeyExpressionAndLengthFromKeyPart(ctx tidb.IKeyPartContext) (string, int64) {
	keyText := tidbparser.NormalizeTiDBIdentifier(ctx.Identifier())
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

func nextDefaultChannelTokenIndex(tokens antlr.TokenStream, currentIndex int) int {
	for i := currentIndex + 1; i < tokens.Size(); i++ {
		if tokens.Get(i).GetChannel() == antlr.TokenDefaultChannel {
			return i
		}
	}
	return 0
}
