// Package mysql implements the SQL advisor rules for MySQL.
package mysql

import (
	"github.com/bytebase/omni/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

// columnFindingPosition is where a finding's advice starts: the statement's first
// line plus the line of the finding's anchor within it.
func columnFindingPosition(stmt base.ParsedStatement, f v1review.Finding) *storepb.Position {
	return base.ConvertANTLRLineToPosition(stmt.BaseLine() + int(findingLine(stmt.Text, f)))
}

// statementPosition is where advice about a whole statement starts: the line
// of the statement's first non-blank character.
func statementPosition(stmt base.ParsedStatement) *storepb.Position {
	return columnFindingPosition(stmt, v1review.Finding{})
}

// v1reviewSchema returns the snapshot md holds for an omni rule, an empty
// one when md is nil.
func v1reviewSchema(md *model.DatabaseMetadata) v1review.Schema {
	if md == nil {
		return v1review.Schema{}
	}
	return v1review.Schema{Metadata: md.GetProto(), ObjectCaseSensitive: md.GetIsObjectCaseSensitive()}
}
