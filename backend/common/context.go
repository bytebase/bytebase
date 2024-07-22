package common

import "context"

// ContextKey is the key type of context value.
type ContextKey int

const (
	// PrincipalIDContextKey is the key name used to store principal id in the context.
	PrincipalIDContextKey ContextKey = iota
	// RoleContextKey is the key name used to store principal role in the context.
	RoleContextKey
	// LoopbackContextKey is the key name used for loopback interface in the context.
	LoopbackContextKey
	// UserContextKey is the key name used to store user message in the context.
	UserContextKey
	ProjectIDsContextKey
	AuthContextKey
)

func WithProjectIDs(ctx context.Context, projectIDs []string) context.Context {
	return context.WithValue(ctx, ProjectIDsContextKey, projectIDs)
}

func GetProjectIDsFromContext(ctx context.Context) ([]string, bool) {
	v, ok := ctx.Value(ProjectIDsContextKey).([]string)
	return v, ok
}

type AuthMethod int

const (
	AuthMethodUnspecified AuthMethod = iota
	AuthMethodIAM
	AuthMethodCustom
)

type AuthContext struct {
	AllowWithoutCredential bool
	Permission             string
	AuthMethod             AuthMethod
}
