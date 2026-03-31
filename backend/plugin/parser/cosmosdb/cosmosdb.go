package cosmosdb

import (
	"unicode/utf8"

	omnicosmosdb "github.com/bytebase/omni/cosmosdb"
	"github.com/bytebase/omni/cosmosdb/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// OmniAST wraps an omni CosmosDB AST node and implements the base.AST interface.
type OmniAST struct {
	Node          ast.Node
	Text          string
	StartPosition *storepb.Position
}

// ASTStartPosition implements base.AST.
func (a *OmniAST) ASTStartPosition() *storepb.Position {
	return a.StartPosition
}

// GetOmniNode extracts the omni AST node from a base.AST interface.
func GetOmniNode(a base.AST) (ast.Node, bool) {
	if a == nil {
		return nil, false
	}
	omniAST, ok := a.(*OmniAST)
	if !ok {
		return nil, false
	}
	return omniAST.Node, true
}

// ParseCosmosDB parses the given CosmosDB SQL statement using omni's parser.
func ParseCosmosDB(statement string) ([]*OmniAST, error) {
	stmts, err := omnicosmosdb.Parse(statement)
	if err != nil {
		return nil, err
	}

	var result []*OmniAST
	for _, stmt := range stmts {
		if stmt.Empty() {
			continue
		}
		result = append(result, &OmniAST{
			Node:          stmt.AST,
			Text:          stmt.Text,
			StartPosition: byteOffsetToRunePosition(statement, stmt.ByteStart),
		})
	}
	return result, nil
}

// byteOffsetToRunePosition converts a byte offset to a 1-based line:column position
// where column is measured in Unicode code points.
func byteOffsetToRunePosition(sql string, byteOffset int) *storepb.Position {
	if byteOffset > len(sql) {
		byteOffset = len(sql)
	}

	line := int32(1)
	runeCol := int32(0)
	i := 0
	for i < byteOffset {
		r, size := utf8.DecodeRuneInString(sql[i:])
		if r == '\n' {
			line++
			runeCol = 0
		} else {
			runeCol++
		}
		i += size
	}

	return &storepb.Position{
		Line:   line,
		Column: runeCol + 1,
	}
}
