package mysql

import (
	"strings"

	"github.com/antlr4-go/antlr/v4"
	mysql "github.com/bytebase/mysql-parser"

	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

const (
	autoIncrementSymbol = "AUTO_INCREMENT"
	autoRandSymbol      = "AUTO_RANDOM"
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

func extractKeyListVariants(ctx mysql.IKeyListVariantsContext) []string {
	if ctx.KeyList() != nil {
		return extractKeyList(ctx.KeyList())
	}
	if ctx.KeyListWithExpression() != nil {
		return extractKeyListWithExpression(ctx.KeyListWithExpression())
	}
	return nil
}

func extractKeyListWithExpression(ctx mysql.IKeyListWithExpressionContext) []string {
	var result []string
	for _, key := range ctx.AllKeyPartOrExpression() {
		if key.KeyPart() != nil {
			keyText := mysqlparser.NormalizeMySQLIdentifier(key.KeyPart().Identifier())
			result = append(result, keyText)
		} else if key.ExprWithParentheses() != nil {
			keyText := key.GetParser().GetTokenStream().GetTextFromRuleContext(key.ExprWithParentheses())
			result = append(result, keyText)
		}
	}
	return result
}

func extractKeyList(ctx mysql.IKeyListContext) []string {
	var result []string
	for _, key := range ctx.AllKeyPart() {
		keyText := mysqlparser.NormalizeMySQLIdentifier(key.Identifier())
		if key.FieldLength() != nil || key.Direction() != nil {
			keyText = key.GetParser().GetTokenStream().GetTextFromRuleContext(key)
		}
		result = append(result, keyText)
	}
	return result
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

func nextDefaultChannelTokenIndex(tokens antlr.TokenStream, currentIndex int) int {
	for i := currentIndex + 1; i < tokens.Size(); i++ {
		if tokens.Get(i).GetChannel() == antlr.TokenDefaultChannel {
			return i
		}
	}
	return 0
}
