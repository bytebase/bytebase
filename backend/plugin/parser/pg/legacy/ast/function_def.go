package ast

// FunctionParameterMode is the definition for function parameter mode.
type FunctionParameterMode int

const (
	// FunctionParameterModeUndefined is the undefined parameter mode.
	FunctionParameterModeUndefined FunctionParameterMode = iota
	// FunctionParameterModeIn is mode for the IN parameters. It's the default value.
	FunctionParameterModeIn
	// FunctionParameterModeOut is mode for the OUT parameters.
	FunctionParameterModeOut
	// FunctionParameterModeInOut is mode for the OUT parameters.
	FunctionParameterModeInOut
	// FunctionParameterModeVariadic is mode for the VARIADIC parameters.
	FunctionParameterModeVariadic
	// FunctionParameterModeTable is mode for the TABLE parameters.
	FunctionParameterModeTable
	// FunctionParameterModeDefault is mode for the DEFAULT parameters.
	FunctionParameterModeDefault
)

// FunctionParameterDef is the struct for function parameter definition.
type FunctionParameterDef struct {
	Name string
	Type DataType
	Mode FunctionParameterMode
}

// FunctionDef is the struct for function definition.
type FunctionDef struct {
	node

	Schema        string
	Name          string
	ParameterList []*FunctionParameterDef
}
