package redshift

import (
	"strings"

	parser "github.com/bytebase/parser/redshift"
)

// Normalization functions for legacy Redshift ANTLR query-span paths.
func normalizeRedshiftName(ctx parser.INameContext) string {
	if ctx == nil {
		return ""
	}
	return normalizeRedshiftIdentifierText(ctx.GetText())
}

func normalizeRedshiftColid(ctx parser.IColidContext) string {
	if ctx == nil {
		return ""
	}
	return normalizeRedshiftIdentifierText(ctx.GetText())
}

func normalizeRedshiftIdentifier(ctx parser.IIdentifierContext) string {
	if ctx == nil {
		return ""
	}
	return normalizeRedshiftIdentifierText(ctx.GetText())
}

func normalizeRedshiftCollabel(ctx parser.ICollabelContext) string {
	if ctx == nil {
		return ""
	}
	return normalizeRedshiftIdentifierText(ctx.GetText())
}

func normalizeRedshiftAttrName(ctx parser.IAttr_nameContext) string {
	if ctx == nil {
		return ""
	}
	return normalizeRedshiftIdentifierText(ctx.GetText())
}

func normalizeRedshiftTableAlias(ctx parser.ITable_aliasContext) string {
	if ctx == nil {
		return ""
	}
	return normalizeRedshiftIdentifierText(ctx.GetText())
}

func normalizeRedshiftNameList(ctx parser.IName_listContext) []string {
	if ctx == nil {
		return nil
	}
	var result []string
	for _, name := range ctx.AllName() {
		result = append(result, normalizeRedshiftName(name))
	}
	return result
}

func normalizeRedshiftQuotedIdentifier(text string) string {
	if len(text) < 2 {
		return text
	}
	return strings.ReplaceAll(text[1:len(text)-1], `""`, `"`)
}

func normalizeRedshiftIdentifierText(text string) string {
	if len(text) >= 2 && text[0] == '"' && text[len(text)-1] == '"' {
		return normalizeRedshiftQuotedIdentifier(text)
	}
	return strings.ToLower(text)
}

func NormalizeRedshiftQualifiedName(ctx parser.IQualified_nameContext) []string {
	if ctx == nil {
		return nil
	}
	var result []string
	if ctx.Colid() != nil {
		result = append(result, normalizeRedshiftColid(ctx.Colid()))
	}
	if ctx.Indirection() != nil {
		for _, el := range ctx.Indirection().AllIndirection_el() {
			if el.Attr_name() != nil {
				result = append(result, normalizeRedshiftAttrName(el.Attr_name()))
			}
		}
	}
	return result
}
