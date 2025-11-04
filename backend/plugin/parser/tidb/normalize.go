package tidb

import (
	parser "github.com/bytebase/parser/tidb"
)

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
