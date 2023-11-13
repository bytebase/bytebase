package mysql

import (
	"strings"

	parser "github.com/bytebase/mysql-parser"
)

// NormalizeMySQLTableName normalizes the given table name.
func NormalizeMySQLTableName(ctx parser.ITableNameContext) (string, string) {
	if ctx.QualifiedIdentifier() != nil {
		return normalizeMySQLQualifiedIdentifier(ctx.QualifiedIdentifier())
	}
	if ctx.DotIdentifier() != nil {
		return "", NormalizeMySQLIdentifier(ctx.DotIdentifier().Identifier())
	}
	return "", ""
}

// NormalizeMySQLTableRef normalizes the given table reference.
func NormalizeMySQLTableRef(ctx parser.ITableRefContext) (string, string) {
	if ctx.QualifiedIdentifier() != nil {
		return normalizeMySQLQualifiedIdentifier(ctx.QualifiedIdentifier())
	}
	if ctx.DotIdentifier() != nil {
		return "", NormalizeMySQLIdentifier(ctx.DotIdentifier().Identifier())
	}
	return "", ""
}

// NormalizeMySQLColumnName normalizes the given column name.
func NormalizeMySQLColumnName(ctx parser.IColumnNameContext) (string, string, string) {
	if ctx.Identifier() != nil {
		return "", "", NormalizeMySQLIdentifier(ctx.Identifier())
	}
	return NormalizeMySQLFieldIdentifier(ctx.FieldIdentifier())
}

// NormalizeMySQLFieldIdentifier normalizes the given field identifier.
func NormalizeMySQLFieldIdentifier(ctx parser.IFieldIdentifierContext) (string, string, string) {
	list := []string{}
	if ctx.QualifiedIdentifier() != nil {
		id1, id2 := normalizeMySQLQualifiedIdentifier(ctx.QualifiedIdentifier())
		list = append(list, id1, id2)
	}

	if ctx.DotIdentifier() != nil {
		list = append(list, NormalizeMySQLIdentifier(ctx.DotIdentifier().Identifier()))
	}

	for len(list) < 3 {
		list = append([]string{""}, list...)
	}

	return list[0], list[1], list[2]
}

func normalizeMySQLQualifiedIdentifier(qualifiedIdentifier parser.IQualifiedIdentifierContext) (string, string) {
	list := []string{NormalizeMySQLIdentifier(qualifiedIdentifier.Identifier())}
	if qualifiedIdentifier.DotIdentifier() != nil {
		list = append(list, NormalizeMySQLIdentifier(qualifiedIdentifier.DotIdentifier().Identifier()))
	}

	if len(list) == 1 {
		list = append([]string{""}, list...)
	}

	return list[0], list[1]
}

// NormalizeMySQLIdentifier normalizes the given identifier.
func NormalizeMySQLIdentifier(identifier parser.IIdentifierContext) string {
	if identifier.PureIdentifier() != nil {
		if identifier.PureIdentifier().IDENTIFIER() != nil {
			return identifier.PureIdentifier().IDENTIFIER().GetText()
		}
		// For back tick quoted identifier, we need to remove the back tick.
		text := identifier.PureIdentifier().BACK_TICK_QUOTED_ID().GetText()
		return text[1 : len(text)-1]
	}
	return identifier.GetText()
}

// NormalizeMySQLTextOrIdentifier normalizes the given TextOrIdentifier.
func NormalizeMySQLTextOrIdentifier(ctx parser.ITextOrIdentifierContext) string {
	if ctx.Identifier() != nil {
		return NormalizeMySQLIdentifier(ctx.Identifier())
	}
	textString := ctx.TextStringLiteral().GetText()
	// remove the quotations.
	return textString[1 : len(textString)-1]
}

// NormalizeMySQLTextStringLiteral normalize the given TextStringLiteral.
func NormalizeMySQLTextLiteral(ctx parser.ITextLiteralContext) string {
	textString := ctx.GetText()
	// remove the quotations.
	return textString[1 : len(textString)-1]
}

// NormalizeMySQLTextStringLiteral normalize the given TextStringLiteral.
func NormalizeMySQLTextStringLiteral(ctx parser.ITextStringLiteralContext) string {
	textString := ctx.GetText()
	// remove the quotations.
	return textString[1 : len(textString)-1]
}

// NormalizeMySQLSelectAlias normalizes the given select alias.
func NormalizeMySQLSelectAlias(selectAlias parser.ISelectAliasContext) string {
	if selectAlias.Identifier() != nil {
		return NormalizeMySQLIdentifier(selectAlias.Identifier())
	}
	textString := selectAlias.TextStringLiteral().GetText()
	return textString[1 : len(textString)-1]
}

// NormalizeMySQLIdentifierList normalizes the given identifier list.
func NormalizeMySQLIdentifierList(ctx parser.IIdentifierListContext) []string {
	var result []string
	for _, identifier := range ctx.AllIdentifier() {
		result = append(result, NormalizeMySQLIdentifier(identifier))
	}
	return result
}

// NormalizeMySQLViewName normalizes the given view name.
func NormalizeMySQLViewName(ctx parser.IViewNameContext) (string, string) {
	if ctx.QualifiedIdentifier() != nil {
		return normalizeMySQLQualifiedIdentifier(ctx.QualifiedIdentifier())
	}
	if ctx.DotIdentifier() != nil {
		return "", NormalizeMySQLIdentifier(ctx.DotIdentifier().Identifier())
	}
	return "", ""
}

// NormalizeMySQLEventName normalizes the given event name.
func NormalizeMySQLEventName(ctx parser.IEventNameContext) (string, string) {
	if ctx.QualifiedIdentifier() != nil {
		return normalizeMySQLQualifiedIdentifier(ctx.QualifiedIdentifier())
	}
	return "", ""
}

// NormalizeMySQLTriggerName normalizes the given trigger name.
func NormalizeMySQLTriggerName(ctx parser.ITriggerNameContext) (string, string) {
	if ctx.QualifiedIdentifier() != nil {
		return normalizeMySQLQualifiedIdentifier(ctx.QualifiedIdentifier())
	}
	return "", ""
}

// NormalizeMySQLFunctionName normalizes the given function name.
func NormalizeMySQLFunctionName(ctx parser.IFunctionNameContext) (string, string) {
	if ctx.QualifiedIdentifier() != nil {
		return normalizeMySQLQualifiedIdentifier(ctx.QualifiedIdentifier())
	}
	return "", ""
}

// NormalizeMySQLProcedureName normalizes the given procedure name.
func NormalizeMySQLProcedureName(ctx parser.IProcedureNameContext) (string, string) {
	if ctx.QualifiedIdentifier() != nil {
		return normalizeMySQLQualifiedIdentifier(ctx.QualifiedIdentifier())
	}
	return "", ""
}

// NormalizeMySQLSchemaRef noamalizes the given schemaRef.
func NormalizeMySQLSchemaRef(ctx parser.ISchemaRefContext) string {
	if ctx.Identifier() != nil {
		return NormalizeMySQLIdentifier(ctx.Identifier())
	}
	return ""
}

// NormalizeMySQLKeyListVariants normalize the given keyListVariants.
func NormalizeKeyListVariants(ctx parser.IKeyListVariantsContext) []string {
	if ctx.KeyList() != nil {
		return NormalizeKeyList(ctx.KeyList())
	}
	if ctx.KeyListWithExpression() != nil {
		return NormalizeKeyListWithExpression(ctx.KeyListWithExpression())
	}
	return nil
}

// NormalizeMySQLKeyList normalize the given keyList.
func NormalizeKeyList(ctx parser.IKeyListContext) []string {
	var result []string
	for _, key := range ctx.AllKeyPart() {
		keyText := NormalizeMySQLIdentifier(key.Identifier())
		result = append(result, keyText)
	}
	return result
}

// NormalizeMySQLKeyListWithExpression normalize the given keyListWithExpression.
func NormalizeKeyListWithExpression(ctx parser.IKeyListWithExpressionContext) []string {
	var result []string
	for _, key := range ctx.AllKeyPartOrExpression() {
		if key.KeyPart() != nil {
			keyText := NormalizeMySQLIdentifier(key.KeyPart().Identifier())
			result = append(result, keyText)
		} else if key.ExprWithParentheses() != nil {
			keyText := key.GetParser().GetTokenStream().GetTextFromRuleContext(key.ExprWithParentheses())
			result = append(result, keyText)
		}
	}
	return result
}

// NormalizeMySQLIndexName normalize the given IndexName.
func NormalizeIndexName(ctx parser.IIndexNameContext) string {
	if ctx.Identifier() != nil {
		return NormalizeMySQLIdentifier(ctx.Identifier())
	}
	return ""
}

// NormalizeMySQLIndexName normalize the given IndeRef.
func NormalizeIndexRef(ctx parser.IIndexRefContext) (string, string, string) {
	if ctx.FieldIdentifier() != nil {
		return NormalizeMySQLFieldIdentifier(ctx.FieldIdentifier())
	}
	return "", "", ""
}

// NormalizeMySQLIdentifierListWithParentheses normalize the given IdentififerListWithparentheses.
func NormalizeIdentifierListWithParentheses(ctx parser.IIdentifierListWithParenthesesContext) []string {
	if ctx.IdentifierList() != nil {
		return NormalizeMySQLIdentifierList(ctx.IdentifierList())
	}
	return nil
}

// NormalizeMySQLConstraintName normalize the given IConstraintName.
func NormalizeConstraintName(ctx parser.IConstraintNameContext) string {
	if ctx.Identifier() != nil {
		return NormalizeMySQLIdentifier(ctx.Identifier())
	}
	return ""
}

// NormalizeMySQLColumnInternalRef noamalizes the given columnInternalRef.
func NormalizeMySQLColumnInternalRef(ctx parser.IColumnInternalRefContext) string {
	if ctx.Identifier() != nil {
		return NormalizeMySQLIdentifier(ctx.Identifier())
	}
	return ""
}

// NormalizeMySQLCharsetName noamalizes the given charset name.
func NormalizeMySQLCharsetName(ctx parser.ICharsetNameContext) string {
	switch {
	case ctx.TextOrIdentifier() != nil:
		return NormalizeMySQLTextOrIdentifier(ctx.TextOrIdentifier())
	case ctx.DEFAULT_SYMBOL() != nil:
		return "DEFAULT"
	case ctx.BINARY_SYMBOL() != nil:
		return "BINARY"
	}
	return ""
}

// NormalizeMySQLDataType noamalizes the given dataType.
// campact for tidb parser compatibility.
// eg: varchar(5).
// compact is true, return varchar.
// compact is false, return varchar(5).
func NormalizeMySQLDataType(ctx parser.IDataTypeContext, compact bool) string {
	if !compact {
		return ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)
	}
	switch ctx.GetType_().GetTokenType() {
	case parser.MySQLParserDOUBLE_SYMBOL:
		if ctx.PRECISION_SYMBOL() != nil {
			return "double precision"
		}
		return "double"
	case parser.MySQLParserCHAR_SYMBOL:
		if ctx.VARYING_SYMBOL() != nil {
			return "char varying"
		}
		return "char"
	default:
		return strings.ToLower(ctx.GetType_().GetText())
	}
}
