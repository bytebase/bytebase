package doris

import parser "github.com/bytebase/doris-parser"

func NormalizeQualifiedName(ctx parser.IQualifiedNameContext) []string {
	if ctx == nil {
		return nil
	}
	var result []string
	for _, id := range ctx.AllIdentifier() {
		if id == nil {
			continue
		}
		result = append(result, NormalizeIdentifier(id))
	}
	return result
}

func NormalizeIdentifier(ctx parser.IIdentifierContext) string {
	if ctx == nil {
		return ""
	}

	switch ctx.(type) {
	case *parser.UnquotedIdentifierContext:
		return ctx.GetText()
	case *parser.DigitIdentifierContext:
		return ctx.GetText()
	case *parser.BackQuotedIdentifierContext:
		return ctx.GetText()[1 : len(ctx.GetText())-1]
	}

	return ctx.GetText()
}
