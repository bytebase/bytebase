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
}
