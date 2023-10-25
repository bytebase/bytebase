package mysql

import parser "github.com/bytebase/mysql-parser"

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
