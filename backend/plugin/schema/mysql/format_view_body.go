package mysql

import (
	"strings"
)

// formatViewBodySDL pretty-prints a MySQL stored view body (the one-line SELECT text
// SHOW CREATE VIEW emits) for the SDL dump. It is a WHITESPACE-ONLY transformation: the
// token stream is copied verbatim and only the whitespace BETWEEN tokens is rewritten
// (runs collapse to a single space, a newline, or a newline+indent). That keeps the
// omni no-op invariant intact — the SDL loader routes view bodies through parse→deparse
// (deparseViewSelect), so any two whitespace-equivalent bodies land on the same
// canonical Definition — while making the dumped statement readable and git-diffable.
//
// The scanner mirrors stripViewBodyDatabaseQualifier's string-literal awareness and
// extends it to backtick identifiers and parentheses:
//   - '…' / "…" literals are opaque tokens (backslash and doubled-delimiter escapes),
//     so a literal containing " from " or "union all" is never split;
//   - `…` identifiers are opaque tokens (doubled-backtick escape, NO backslash escape),
//     so an alias like `CONCAT('INDEX (', INDEX_TYPE, ')')` is never split;
//   - '(' frames are classified as subquery (next word is select/with) or plain (join
//     trees, expression grouping), so breaks never land inside a subquery or a
//     function/expression argument list.
//
// Break rules (MySQL's stored grammar is a fixed shape — select … from … where …
// group by … having … order by … limit …, unions between members, parenthesized
// left-deep join trees, an optional trailing WITH … CHECK OPTION):
//
//	token                                     condition                      break
//	------------------------------------------------------------------------------------
//	select                                    top level (no open paren)      newline before
//	from/where/having/limit/union             top level                      newline before
//	group by / order by                       top level                      newline before
//	with [cascaded|local] check option        top level, at end of body      newline before
//	first select-list item                    after select [+modifiers]      newline+indent
//	comma between select-list items           top level select list          newline+indent after
//	[natural|left|right|inner|cross…] join    in from clause, not inside     newline+indent before
//	                                          any subquery
//
// Everything else keeps its single-space (or no-space) joining, so long expressions,
// on(…) conditions, scalar subqueries, and group-by lists stay on their line. The
// function is deterministic and idempotent (formatViewBodySDL(f(x)) == f(x)): the A6/A8
// dump-stability guards rely on same-body-in → same-bytes-out.
//
// FAIL-SAFE: the scanner assumes the backslash-escaped canonical literal form that
// SHOW CREATE VIEW / information_schema.VIEWS always re-print (live-verified on 5.7.25
// and 8.0.32: the server re-serializes literals with backslash escapes even for views
// CREATED under NO_BACKSLASH_ESCAPES). Should a body from some other origin still
// mis-tokenize, whitespace INSIDE a mis-scanned literal would be rewritten — data
// corruption, not just bad formatting. Two runtime guards convert every such present or
// future scanner gap into "not pretty" instead of "corrupted":
//  1. the body must tokenize cleanly — a literal/identifier that never closes (the
//     smoking-gun signature of an escape-mode mismatch swallowing a quote, or of a
//     malformed body) refuses formatting;
//  2. the formatted output's token stream must byte-match the input's (the
//     whitespace-only property the tests assert, re-checked at runtime).
//
// On any mismatch the original stored body is returned unformatted, which is still
// canonical for the omni diff (whitespace washes out through parse→deparse).
func formatViewBodySDL(body string) string {
	inTokens, clean := viewBodyTokenStream(body)
	if !clean {
		return body
	}
	formatted := formatViewBodyUnchecked(body)
	outTokens, clean := viewBodyTokenStream(formatted)
	if !clean || !viewBodyTokensEqual(inTokens, outTokens) {
		return body
	}
	return formatted
}

// viewBodyTokenStream splits body into its token stream (whitespace between tokens
// discarded). clean is false when any '…'/"…" literal or `…` identifier runs to the end
// of the body without its closing delimiter.
func viewBodyTokenStream(body string) (tokens []string, clean bool) {
	for i := 0; i < len(body); {
		if isViewBodySpaceByte(body[i]) {
			i++
			continue
		}
		end, closed := scanViewBodyTokenClosed(body, i)
		if !closed {
			return nil, false
		}
		tokens = append(tokens, body[i:end])
		i = end
	}
	return tokens, true
}

// viewBodyTokensEqual reports whether two token streams are element-wise identical.
func viewBodyTokensEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// formatViewBodyUnchecked is the formatting core; formatViewBodySDL wraps it with the
// fail-safe guards above.
func formatViewBodyUnchecked(body string) string {
	var out strings.Builder
	out.Grow(len(body) + 64)

	// stack tracks open parens; true marks a subquery frame ("(" directly followed by
	// select/with). subq counts open subquery frames for O(1) "inside a subquery" tests.
	var stack []bool
	subq := 0
	// selectHead is the region between a top-level `select` and its first list item
	// (modifiers like distinct/straight_join stay attached to the select line).
	// selectList is the top-level select list (comma breaks); fromRegion is the
	// top-level from clause (join breaks).
	selectHead, selectList, fromRegion := false, false, false
	// forced, when non-empty, is the joiner the NEXT token must use (set after a
	// select-list comma, where the input has no whitespace to rewrite).
	forced := ""
	// glue counts the remaining words of a multi-word join operator ("left outer
	// join") after its head broke, so the phrase stays on one line.
	glue := 0
	pendingWS := false

	for i := 0; i < len(body); {
		if isViewBodySpaceByte(body[i]) {
			pendingWS = true
			i++
			continue
		}
		end := scanViewBodyToken(body, i)
		tok := body[i:end]

		joiner := ""
		switch {
		case out.Len() == 0:
			// Never emit whitespace before the first token.
		case forced != "":
			joiner = forced
		case glue > 0:
			// Token is a continuation word of a broken join operator: keep it glued.
			if pendingWS {
				joiner = " "
			}
			glue--
		case pendingWS:
			var joinWords int
			joiner, joinWords = viewBodyJoiner(body, tok, end, stack, subq, selectHead, fromRegion)
			if joiner == "\n"+viewSDLIndent && selectHead {
				selectHead = false
				selectList = true
			}
			glue = joinWords
		default:
		}
		forced = ""
		pendingWS = false
		out.WriteString(joiner)
		out.WriteString(tok)

		// State updates AFTER emitting the token.
		switch {
		case tok == "(":
			sub := viewBodyParenIsSubquery(body, end)
			stack = append(stack, sub)
			if sub {
				subq++
			}
		case tok == ")":
			if n := len(stack); n > 0 {
				if stack[n-1] {
					subq--
				}
				stack = stack[:n-1]
			}
		case tok == ",":
			if len(stack) == 0 && selectList {
				forced = "\n" + viewSDLIndent
			}
		case isViewBodyWord(tok):
			if len(stack) == 0 {
				switch strings.ToLower(tok) {
				case "select":
					selectHead, selectList, fromRegion = true, false, false
				case "from":
					selectHead, selectList, fromRegion = false, false, true
				case "where", "having", "limit", "union":
					selectHead, selectList, fromRegion = false, false, false
				case "group", "order":
					if w, _ := peekViewBodyWord(body, end); w == "by" {
						selectHead, selectList, fromRegion = false, false, false
					}
				default:
				}
			}
		default:
		}
		i = end
	}
	return out.String()
}

// viewSDLIndent is the continuation indent for select-list items and join lines.
const viewSDLIndent = "  "

// viewBodyJoiner decides what a whitespace run before tok becomes: a clause-keyword
// newline, an item/join newline+indent, or a single space. The second result is the
// number of FOLLOWING words that belong to a broken join operator (e.g. 2 for
// "left outer join"), which the caller keeps glued to the head with plain spaces.
func viewBodyJoiner(body, tok string, end int, stack []bool, subq int, selectHead, fromRegion bool) (string, int) {
	if selectHead {
		if isViewBodyWord(tok) && viewSelectModifiers[strings.ToLower(tok)] {
			return " ", 0
		}
		// First select-list item (any token kind: word, literal, `(`, backtick, …).
		return "\n" + viewSDLIndent, 0
	}
	if !isViewBodyWord(tok) {
		return " ", 0
	}
	w := strings.ToLower(tok)
	if len(stack) == 0 && isViewClauseBreak(body, end, w) {
		return "\n", 0
	}
	if fromRegion && subq == 0 {
		if ok, rest := viewJoinPhraseRest(body, end, w); ok {
			return "\n" + viewSDLIndent, rest
		}
	}
	return " ", 0
}

// viewSelectModifiers are the tokens that may sit between SELECT and its first list
// item in the stored form; they stay attached to the select line.
var viewSelectModifiers = map[string]bool{
	"all":                 true,
	"distinct":            true,
	"distinctrow":         true,
	"high_priority":       true,
	"straight_join":       true,
	"sql_small_result":    true,
	"sql_big_result":      true,
	"sql_buffer_result":   true,
	"sql_cache":           true,
	"sql_no_cache":        true,
	"sql_calc_found_rows": true,
}

// isViewClauseBreak reports whether the lower-cased word w, whose token ends at end,
// starts a top-level clause that gets its own line.
func isViewClauseBreak(body string, end int, w string) bool {
	switch w {
	case "select", "from", "where", "having", "limit", "union":
		return true
	case "group", "order":
		next, _ := peekViewBodyWord(body, end)
		return next == "by"
	case "with":
		// Only the trailing WITH [CASCADED|LOCAL] CHECK OPTION breaks; a leading CTE
		// `with` is body-initial and never reaches a joiner decision.
		next, rest := peekViewBodyWord(body, end)
		if next == "cascaded" || next == "local" {
			next, rest = peekViewBodyWord(body, rest)
		}
		if next != "check" {
			return false
		}
		next, rest = peekViewBodyWord(body, rest)
		if next != "option" {
			return false
		}
		return strings.TrimSpace(body[rest:]) == ""
	default:
		return false
	}
}

// viewJoinPhraseRest reports whether the lower-cased word w, whose token ends at end,
// heads a join operator ([natural] [left|right|inner|cross] [outer] join /
// straight_join) — and, when it does, how many words FOLLOW w in the phrase (they must
// stay glued to the broken head).
func viewJoinPhraseRest(body string, end int, w string) (bool, int) {
	switch w {
	case "join", "straight_join":
		return true, 0
	case "inner", "cross":
		next, _ := peekViewBodyWord(body, end)
		if next == "join" {
			return true, 1
		}
		return false, 0
	case "left", "right":
		rest := 1
		next, pos := peekViewBodyWord(body, end)
		if next == "outer" {
			next, _ = peekViewBodyWord(body, pos)
			rest = 2
		}
		if next == "join" {
			return true, rest
		}
		return false, 0
	case "natural":
		rest := 1
		next, pos := peekViewBodyWord(body, end)
		if next == "left" || next == "right" || next == "inner" {
			next, pos = peekViewBodyWord(body, pos)
			rest = 2
		}
		if next == "outer" {
			next, _ = peekViewBodyWord(body, pos)
			rest++
		}
		if next == "join" {
			return true, rest
		}
		return false, 0
	default:
		return false, 0
	}
}

// viewBodyParenIsSubquery classifies the "(" whose token ends at end: true when the
// next token is the select (or CTE with) keyword, i.e. the paren opens a subquery whose
// interior must never be broken.
func viewBodyParenIsSubquery(body string, end int) bool {
	w, _ := peekViewBodyWord(body, end)
	return w == "select" || w == "with"
}

// peekViewBodyWord returns the lower-cased next word token after pos (skipping
// whitespace) and the offset just past it. It returns "" when the next token is not a
// bare word (a literal, an identifier, punctuation, or end of body).
func peekViewBodyWord(body string, pos int) (string, int) {
	for pos < len(body) && isViewBodySpaceByte(body[pos]) {
		pos++
	}
	if pos >= len(body) || !isViewBodyWordByte(body[pos]) {
		return "", pos
	}
	end := scanViewBodyToken(body, pos)
	return strings.ToLower(body[pos:end]), end
}

// isViewBodySpaceByte reports whether b is whitespace between tokens.
func isViewBodySpaceByte(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}

// scanViewBodyToken returns the end offset of the token starting at i (a non-space
// byte): a whole '…'/"…" string literal, a whole `…` identifier, a word run, or a
// single punctuation byte. Literals and identifiers are opaque — breaks can never land
// inside them. String literals honor backslash escapes — the canonical form SHOW CREATE
// VIEW always re-prints (see the formatViewBodySDL fail-safe).
func scanViewBodyToken(body string, i int) int {
	end, _ := scanViewBodyTokenClosed(body, i)
	return end
}

// scanViewBodyTokenClosed is scanViewBodyToken plus a closed flag: false when the token
// is a string literal or backtick identifier that reaches the end of the body without
// its closing delimiter (a malformed or mis-tokenized body).
func scanViewBodyTokenClosed(body string, i int) (int, bool) {
	switch c := body[i]; c {
	case '\'', '"':
		// String literal: backslash escapes the next byte; a doubled delimiter stays
		// inside. Mirrors stripViewBodyDatabaseQualifier.
		for j := i + 1; j < len(body); {
			switch {
			case body[j] == '\\' && j+1 < len(body):
				j += 2
			case body[j] == c:
				if j+1 < len(body) && body[j+1] == c {
					j += 2
					continue
				}
				return j + 1, true
			default:
				j++
			}
		}
		return len(body), false
	case '`':
		// Backtick identifier: only a doubled backtick escapes (no backslash escape).
		for j := i + 1; j < len(body); {
			if body[j] == '`' {
				if j+1 < len(body) && body[j+1] == '`' {
					j += 2
					continue
				}
				return j + 1, true
			}
			j++
		}
		return len(body), false
	default:
		if !isViewBodyWordByte(body[i]) {
			return i + 1, true
		}
		j := i + 1
		for j < len(body) && isViewBodyWordByte(body[j]) {
			j++
		}
		return j, true
	}
}

// isViewBodyWord reports whether tok is a bare word token (unquoted keyword,
// function name, or number). Literal and identifier tokens start with their quote
// byte, which is never a word byte.
func isViewBodyWord(tok string) bool {
	return tok != "" && isViewBodyWordByte(tok[0])
}

// isViewBodyWordByte reports whether b can be part of a bare word. Multibyte (>= 0x80)
// bytes are treated as word bytes so an unquoted multibyte identifier stays one token.
func isViewBodyWordByte(b byte) bool {
	switch {
	case b >= 'a' && b <= 'z', b >= 'A' && b <= 'Z', b >= '0' && b <= '9':
		return true
	case b == '_' || b == '$':
		return true
	case b >= 0x80:
		return true
	default:
		return false
	}
}
