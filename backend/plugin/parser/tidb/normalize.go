package tidb

import (
	"strings"

	parser "github.com/bytebase/tidb-parser"
)

// NormalizeTiDBTableName normalizes the given table name.
func NormalizeTiDBTableName(ctx parser.ITableNameContext) (string, string) {
	if ctx.QualifiedIdentifier() != nil {
		return normalizeTiDBQualifiedIdentifier(ctx.QualifiedIdentifier())
	}
	if ctx.DotIdentifier() != nil {
		return "", NormalizeTiDBIdentifier(ctx.DotIdentifier().Identifier())
	}
	return "", ""
}

// NormalizeTiDBTableRef normalizes the given table reference.
func NormalizeTiDBTableRef(ctx parser.ITableRefContext) (string, string) {
	if ctx.QualifiedIdentifier() != nil {
		return normalizeTiDBQualifiedIdentifier(ctx.QualifiedIdentifier())
	}
	if ctx.DotIdentifier() != nil {
		return "", NormalizeTiDBIdentifier(ctx.DotIdentifier().Identifier())
	}
	return "", ""
}

// NormalizeTiDBColumnName normalizes the given column name.
func NormalizeTiDBColumnName(ctx parser.IColumnNameContext) (string, string, string) {
	if ctx.Identifier() != nil {
		return "", "", NormalizeTiDBIdentifier(ctx.Identifier())
	}
	return NormalizeTiDBFieldIdentifier(ctx.FieldIdentifier())
}

// NormalizeTiDBFieldIdentifier normalizes the given field identifier.
func NormalizeTiDBFieldIdentifier(ctx parser.IFieldIdentifierContext) (string, string, string) {
	list := []string{}
	if ctx.QualifiedIdentifier() != nil {
		id1, id2 := normalizeTiDBQualifiedIdentifier(ctx.QualifiedIdentifier())
		list = append(list, id1, id2)
	}

	if ctx.DotIdentifier() != nil {
		list = append(list, NormalizeTiDBIdentifier(ctx.DotIdentifier().Identifier()))
	}

	for len(list) < 3 {
		list = append([]string{""}, list...)
	}

	return list[0], list[1], list[2]
}

func normalizeTiDBQualifiedIdentifier(qualifiedIdentifier parser.IQualifiedIdentifierContext) (string, string) {
	list := []string{NormalizeTiDBIdentifier(qualifiedIdentifier.Identifier())}
	if qualifiedIdentifier.DotIdentifier() != nil {
		list = append(list, NormalizeTiDBIdentifier(qualifiedIdentifier.DotIdentifier().Identifier()))
	}

	if len(list) == 1 {
		list = append([]string{""}, list...)
	}

	return list[0], list[1]
}

// NormalizeTiDBIdentifier normalizes the given identifier.
func NormalizeTiDBIdentifier(identifier parser.IIdentifierContext) string {
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

// NormalizeTiDBTextOrIdentifier normalizes the given TextOrIdentifier.
func NormalizeTiDBTextOrIdentifier(ctx parser.ITextOrIdentifierContext) string {
	if ctx.Identifier() != nil {
		return NormalizeTiDBIdentifier(ctx.Identifier())
	}
	textString := ctx.TextStringLiteral().GetText()
	// remove the quotations.
	return textString[1 : len(textString)-1]
}

// NormalizeTiDBTextStringLiteral normalize the given TextStringLiteral.
func NormalizeTiDBTextLiteral(ctx parser.ITextLiteralContext) string {
	textString := ctx.GetText()
	// remove the quotations.
	return textString[1 : len(textString)-1]
}

// NormalizeTiDBTextStringLiteral normalize the given TextStringLiteral.
func NormalizeTiDBTextStringLiteral(ctx parser.ITextStringLiteralContext) string {
	textString := ctx.GetText()
	// remove the quotations.
	return textString[1 : len(textString)-1]
}

// NormalizeTiDBSignedStringLiteral normalize the given SignedLiteral.
func NormalizeTiDBSignedLiteral(ctx parser.ISignedLiteralContext) string {
	textString := ctx.GetText()
	if (strings.HasPrefix(textString, "'") && strings.HasSuffix(textString, "'")) || (strings.HasPrefix(textString, "\"") && strings.HasSuffix(textString, "\"")) {
		textString = textString[1 : len(textString)-1]
	}
	return textString
}

// NormalizeTiDBSelectAlias normalizes the given select alias.
func NormalizeTiDBSelectAlias(selectAlias parser.ISelectAliasContext) string {
	if selectAlias.Identifier() != nil {
		return NormalizeTiDBIdentifier(selectAlias.Identifier())
	}
	textString := selectAlias.TextStringLiteral().GetText()
	return textString[1 : len(textString)-1]
}

// NormalizeTiDBIdentifierList normalizes the given identifier list.
func NormalizeTiDBIdentifierList(ctx parser.IIdentifierListContext) []string {
	var result []string
	for _, identifier := range ctx.AllIdentifier() {
		result = append(result, NormalizeTiDBIdentifier(identifier))
	}
	return result
}

// NormalizeTiDBViewName normalizes the given view name.
func NormalizeTiDBViewName(ctx parser.IViewNameContext) (string, string) {
	if ctx.QualifiedIdentifier() != nil {
		return normalizeTiDBQualifiedIdentifier(ctx.QualifiedIdentifier())
	}
	if ctx.DotIdentifier() != nil {
		return "", NormalizeTiDBIdentifier(ctx.DotIdentifier().Identifier())
	}
	return "", ""
}

// NormalizeTiDBEventName normalizes the given event name.
func NormalizeTiDBEventName(ctx parser.IEventNameContext) (string, string) {
	if ctx.QualifiedIdentifier() != nil {
		return normalizeTiDBQualifiedIdentifier(ctx.QualifiedIdentifier())
	}
	return "", ""
}

// NormalizeTiDBTriggerName normalizes the given trigger name.
func NormalizeTiDBTriggerName(ctx parser.ITriggerNameContext) (string, string) {
	if ctx.QualifiedIdentifier() != nil {
		return normalizeTiDBQualifiedIdentifier(ctx.QualifiedIdentifier())
	}
	return "", ""
}

// NormalizeTiDBFunctionName normalizes the given function name.
func NormalizeTiDBFunctionName(ctx parser.IFunctionNameContext) (string, string) {
	if ctx.QualifiedIdentifier() != nil {
		return normalizeTiDBQualifiedIdentifier(ctx.QualifiedIdentifier())
	}
	return "", ""
}

// NormalizeTiDBProcedureName normalizes the given procedure name.
func NormalizeTiDBProcedureName(ctx parser.IProcedureNameContext) (string, string) {
	if ctx.QualifiedIdentifier() != nil {
		return normalizeTiDBQualifiedIdentifier(ctx.QualifiedIdentifier())
	}
	return "", ""
}

// NormalizeTiDBSchemaRef normalize the given schemaRef.
func NormalizeTiDBSchemaRef(ctx parser.ISchemaRefContext) string {
	if ctx.Identifier() != nil {
		return NormalizeTiDBIdentifier(ctx.Identifier())
	}
	return ""
}

// NormalizeTiDBSchemaRef normalize the given schemaName.
func NormalizeTiDBSchemaName(ctx parser.ISchemaNameContext) string {
	if ctx.Identifier() != nil {
		return NormalizeTiDBIdentifier(ctx.Identifier())
	}
	return ""
}

// NormalizeTiDBKeyListVariants normalize the given keyListVariants.
func NormalizeKeyListVariants(ctx parser.IKeyListVariantsContext) []string {
	if ctx.KeyList() != nil {
		return NormalizeKeyList(ctx.KeyList())
	}
	if ctx.KeyListWithExpression() != nil {
		return NormalizeKeyListWithExpression(ctx.KeyListWithExpression())
	}
	return nil
}

// NormalizeTiDBKeyList normalize the given keyList.
func NormalizeKeyList(ctx parser.IKeyListContext) []string {
	var result []string
	for _, key := range ctx.AllKeyPart() {
		keyText := NormalizeTiDBIdentifier(key.Identifier())
		result = append(result, keyText)
	}
	return result
}

// NormalizeTiDBKeyListWithExpression normalize the given keyListWithExpression.
func NormalizeKeyListWithExpression(ctx parser.IKeyListWithExpressionContext) []string {
	var result []string
	for _, key := range ctx.AllKeyPartOrExpression() {
		if key.KeyPart() != nil {
			keyText := NormalizeTiDBIdentifier(key.KeyPart().Identifier())
			result = append(result, keyText)
		} else if key.ExprWithParentheses() != nil {
			keyText := key.GetParser().GetTokenStream().GetTextFromRuleContext(key.ExprWithParentheses())
			result = append(result, keyText)
		}
	}
	return result
}

// NormalizeTiDBIndexName normalize the given IndexName.
func NormalizeIndexName(ctx parser.IIndexNameContext) string {
	if ctx.Identifier() != nil {
		return NormalizeTiDBIdentifier(ctx.Identifier())
	}
	return ""
}

// NormalizeTiDBIndexName normalize the given IndeRef.
func NormalizeIndexRef(ctx parser.IIndexRefContext) (string, string, string) {
	if ctx.FieldIdentifier() != nil {
		return NormalizeTiDBFieldIdentifier(ctx.FieldIdentifier())
	}
	return "", "", ""
}

// NormalizeTiDBIdentifierListWithParentheses normalize the given IdentififerListWithparentheses.
func NormalizeIdentifierListWithParentheses(ctx parser.IIdentifierListWithParenthesesContext) []string {
	if ctx.IdentifierList() != nil {
		return NormalizeTiDBIdentifierList(ctx.IdentifierList())
	}
	return nil
}

// NormalizeTiDBConstraintName normalize the given IConstraintName.
func NormalizeConstraintName(ctx parser.IConstraintNameContext) string {
	if ctx.Identifier() != nil {
		return NormalizeTiDBIdentifier(ctx.Identifier())
	}
	return ""
}

// NormalizeTiDBColumnInternalRef noamalizes the given columnInternalRef.
func NormalizeTiDBColumnInternalRef(ctx parser.IColumnInternalRefContext) string {
	if ctx.Identifier() != nil {
		return NormalizeTiDBIdentifier(ctx.Identifier())
	}
	return ""
}

func normalizeTiDBColumnRef(ctx parser.IColumnRefContext) (string, string, string) {
	return NormalizeTiDBFieldIdentifier(ctx.FieldIdentifier())
}

// NormalizeTiDBCharsetName noamalizes the given charset name.
func NormalizeTiDBCharsetName(ctx parser.ICharsetNameContext) string {
	switch {
	case ctx.TextOrIdentifier() != nil:
		return NormalizeTiDBTextOrIdentifier(ctx.TextOrIdentifier())
	case ctx.DEFAULT_SYMBOL() != nil:
		return "DEFAULT"
	case ctx.BINARY_SYMBOL() != nil:
		return "BINARY"
	}
	return ""
}

// NormalizeTiDBCollationName noamalizes the given collation name.
func NormalizeTiDBCollationName(ctx parser.ICollationNameContext) string {
	switch {
	case ctx.TextOrIdentifier() != nil:
		return NormalizeTiDBTextOrIdentifier(ctx.TextOrIdentifier())
	case ctx.DEFAULT_SYMBOL() != nil:
		return "DEFAULT"
	case ctx.BINARY_SYMBOL() != nil:
		return "BINARY"
	}
	return ""
}

// NormalizeTiDBDataType noamalizes the given dataType.
// campact for tidb parser compatibility.
// eg: varchar(5).
// compact is true, return varchar.
// compact is false, return varchar(5).
func NormalizeTiDBDataType(ctx parser.IDataTypeContext, compact bool) string {
	if !compact {
		return ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)
	}
	switch ctx.GetType_().GetTokenType() {
	case parser.TiDBParserDOUBLE_SYMBOL:
		if ctx.PRECISION_SYMBOL() != nil {
			return "double precision"
		}
		return "double"
	case parser.TiDBParserCHAR_SYMBOL:
		if ctx.VARYING_SYMBOL() != nil {
			return "char varying"
		}
		return "char"
	default:
		return strings.ToLower(ctx.GetType_().GetText())
	}
}

// GetCharSetName get charset name from DataTypeContext.
func GetCharSetName(ctx parser.IDataTypeContext) string {
	if ctx.CharsetWithOptBinary() == nil {
		return ""
	}
	charset := NormalizeTiDBCharsetName(ctx.CharsetWithOptBinary().CharsetName())
	return charset
}

// IsTypeType check if the dataType is time type.
func IsTimeType(ctx parser.IDataTypeContext) bool {
	if ctx.GetType_() == nil {
		return false
	}

	switch ctx.GetType_().GetTokenType() {
	case parser.TiDBParserDATETIME_SYMBOL, parser.TiDBParserTIMESTAMP_SYMBOL:
		return true
	default:
		return false
	}
}
