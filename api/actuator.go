package api

// Actuator concept is similar to the Spring Boot Actuator
type ServerInfo struct {
	Mode      string `json:"mode"`
	Host      string `json:"host"`
	Port      string `json:"port"`
	StartedTs int64  `json:"startedTs"`
}
