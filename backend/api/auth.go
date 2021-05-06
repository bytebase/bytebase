package api

type Login struct {
	Email    string `jsonapi:"attr,email"`
	Password string `jsonapi:"attr,password"`
}

type AuthService interface {
	FindPrincipalByEmail(email string) (*Principal, error)
}
