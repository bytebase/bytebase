package api

type Principal struct {
	ID    uint   `jsonapi:"primary,principal"`
	Email string `jsonapi:"attr,email"`
}
