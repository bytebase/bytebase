package server

const (
	// secretLength is the length for the secret used to sign the JWT auto token
	secretLength = 32
)

// retrieved via the SettingService upon startup
type config struct {
	// secret used to sign the JWT auth token
	secret string
	// workspaceID used to initial the identify for a new workspace.
	workspaceID string
}
