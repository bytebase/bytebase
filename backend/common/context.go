package common

// ContextKey is the key type of context value.
type ContextKey int

const (
	// PrincipalIDContextKey is the key name used to store principal id in the context.
	PrincipalIDContextKey ContextKey = iota
	// RoleContextKey is the key name used to store principal role in the context.
	RoleContextKey
	// LoopbackContextKey is the key name used for loopback interface in the context.
	LoopbackContextKey
)
