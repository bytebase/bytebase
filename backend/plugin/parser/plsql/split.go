package plsql

import (
	"strings"

	oracleparser "github.com/bytebase/omni/oracle/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterSplitterFunc(storepb.Engine_ORACLE, SplitSQL)
}

// SplitSQL splits the given SQL statement into multiple SQL statements.
//
// It uses omni's lexical Oracle splitter, which handles strings, comments,
// SQL*Plus separators, and PL/SQL blocks without requiring valid SQL.
func SplitSQL(statement string) ([]base.Statement, error) {
	segments := oracleparser.Split(statement)

	result := make([]base.Statement, 0, len(segments))
	positionMapper := base.NewByteOffsetPositionMapper(statement)
	for i := 0; i < len(segments); i++ {
		seg := segments[i]
		if seg.Kind == oracleparser.SegmentSQLPlusCommand {
			continue
		}
		byteStart := seg.ByteStart
		byteEnd := seg.ByteEnd
		if end, ok := matchRecognizeDefineMergeEnd(statement, segments, i); ok {
			byteEnd = end
			i += 2
		}
		byteEnd = byteStart + trimOracleSegmentTrailingHidden(statement[byteStart:byteEnd])
		text := statement[byteStart:byteEnd]
		result = append(result, base.Statement{
			Text:  text,
			Start: positionMapper.Position(byteStart),
			End:   positionMapper.Position(byteEnd),
			Empty: seg.Empty(),
			Range: &storepb.Range{
				Start: int32(byteStart),
				End:   int32(byteEnd),
			},
		})
	}
	return result, nil
}

func matchRecognizeDefineMergeEnd(statement string, segments []oracleparser.Segment, index int) (int, bool) {
	if index+2 >= len(segments) {
		return 0, false
	}
	current := segments[index]
	define := segments[index+1]
	suffix := segments[index+2]
	if current.Kind != oracleparser.SegmentSQL || define.Kind != oracleparser.SegmentSQLPlusCommand || suffix.Kind != oracleparser.SegmentSQL {
		return 0, false
	}
	if !strings.Contains(strings.ToUpper(current.Text), "MATCH_RECOGNIZE") {
		return 0, false
	}
	if !strings.HasPrefix(strings.ToUpper(strings.TrimSpace(define.Text)), "DEFINE ") {
		return 0, false
	}
	merged := statement[current.ByteStart:suffix.ByteEnd]
	if _, err := oracleparser.Parse(merged); err != nil {
		return 0, false
	}
	return suffix.ByteEnd, true
}

func trimOracleSegmentTrailingHidden(text string) int {
	lexer := oracleparser.NewLexer(text)
	lastTokenEnd := 0
	for {
		token := lexer.NextToken()
		if token.Type == 0 {
			break
		}
		lastTokenEnd = token.End
	}
	if lexer.Err != nil || lastTokenEnd == 0 {
		return len(text)
	}
	return lastTokenEnd
}

func SplitSQLForCompletion(statement string) ([]base.Statement, error) {
	segments := oracleparser.Split(statement)

	result := make([]base.Statement, 0, len(segments))
	positionMapper := base.NewByteOffsetPositionMapper(statement)
	for i := 0; i < len(segments); i++ {
		seg := segments[i]
		if seg.Kind == oracleparser.SegmentSQLPlusCommand {
			continue
		}
		byteStart := seg.ByteStart
		byteEnd := seg.ByteEnd
		if end, ok := matchRecognizeDefineMergeEnd(statement, segments, i); ok {
			byteEnd = end
			i += 2
		}
		result = append(result, base.Statement{
			Text:  statement[byteStart:byteEnd],
			Start: positionMapper.Position(byteStart),
			End:   positionMapper.Position(byteEnd),
			Empty: seg.Empty(),
			Range: &storepb.Range{
				Start: int32(byteStart),
				End:   int32(byteEnd),
			},
		})
	}
	return result, nil
}
