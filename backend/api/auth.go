package api

type Login struct {
	// Domain specific fields
	Email    string `jsonapi:"attr,email"`
	Password string `jsonapi:"attr,password"`
}

type Signup struct {
	// Domain specific fields
	Name     string `jsonapi:"attr,name"`
	Email    string `jsonapi:"attr,email"`
	Password string `jsonapi:"attr,password"`
}
