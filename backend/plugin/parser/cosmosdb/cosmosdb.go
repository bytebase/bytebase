package cosmosdb

import (
	omnicosmosdb "github.com/bytebase/omni/cosmosdb"
	"github.com/bytebase/omni/cosmosdb/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// AST wraps an omni CosmosDB AST node and implements the base.AST interface.
type AST struct {
	Node          ast.Node
	Text          string
	StartPosition *storepb.Position
}

// ASTStartPosition implements base.AST.
func (a *AST) ASTStartPosition() *storepb.Position {
	return a.StartPosition
}

// GetNode extracts the omni AST node from a base.AST interface.
func GetNode(a base.AST) (ast.Node, bool) {
	if a == nil {
		return nil, false
	}
	wrapped, ok := a.(*AST)
	if !ok {
		return nil, false
	}
	return wrapped.Node, true
}

// ParseCosmosDB parses the given CosmosDB SQL statement using omni's parser.
func ParseCosmosDB(statement string) ([]*AST, error) {
	stmts, err := omnicosmosdb.Parse(statement)
	if err != nil {
		return nil, err
	}

	var result []*AST
	positionMapper := base.NewByteOffsetPositionMapper(statement)
	for _, stmt := range stmts {
		if stmt.Empty() {
			continue
		}
		result = append(result, &AST{
			Node:          stmt.AST,
			Text:          stmt.Text,
			StartPosition: positionMapper.Position(stmt.ByteStart),
		})
	}
	return result, nil
}
