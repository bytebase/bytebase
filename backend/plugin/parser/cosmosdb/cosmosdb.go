package cosmosdb

import (
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
	positionMapper := base.NewByteOffsetPositionMapper(statement)
	for _, stmt := range stmts {
		if stmt.Empty() {
			continue
		}
		result = append(result, &OmniAST{
			Node:          stmt.AST,
			Text:          stmt.Text,
			StartPosition: positionMapper.Position(stmt.ByteStart),
		})
	}
	return result, nil
}
