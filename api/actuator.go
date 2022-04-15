package api

// ServerInfo is the API message for server info.
// Actuator concept is similar to the Spring Boot Actuator
type ServerInfo struct {
	Version        string `json:"version"`
	Readonly       bool   `json:"readonly"`
	Demo           bool   `json:"demo"`
	Host           string `json:"host"`
	Port           string `json:"port"`
	NeedAdminSetup bool   `json:"needAdminSetup"`
	// Rand may be based on the server start time, thus exposing startedTs to the client may cause security issues (e.g. jwt key is based on Rand).
	// StartedTs   int64  `json:"startedTs"`
}
