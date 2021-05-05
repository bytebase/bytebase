package api

type Environment struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Order uint   `json:"order"`
}

type EnvironmentService interface {
}
