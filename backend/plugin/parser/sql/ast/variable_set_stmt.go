package ast

type VariableSetStmt struct {
	node

	Name    string
	Args    []ExpressionNode
	IsLocal bool
}

func (s *VariableSetStmt) GetRoleName() string {
	if len(s.Args) != 1 {
		return ""
	}

	if v, ok := s.Args[0].(*StringDef); ok {
		return v.Value
	}

	return ""
}
