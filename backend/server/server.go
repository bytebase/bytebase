package server

type Environment struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Order uint   `json:"order"`
}

type EnvironmentService interface {
}

type Server struct {
	EnvironmentService EnvironmentService
}

func NewServer() *Server {
	s := &Server{}

	s.registerEnvironmentRoutes()

	return s
}
