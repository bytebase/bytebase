package api

type Principal struct {
	ID    uint   `jsonapi:"primary,principal"`
	Name  string `jsonapi:"attr,name"`
	Email string `jsonapi:"attr,email"`
}
