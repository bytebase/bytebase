package api

// Actuator concept is similar to the Spring Boot Actuator
type ServerInfo struct {
	Demo      bool   `json:"demo"`
	Host      string `json:"host"`
	Port      string `json:"port"`
	StartedTs int64  `json:"startedTs"`
}
