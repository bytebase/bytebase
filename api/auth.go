package api

// Login is the API message for logins.
type Login struct {
	// Domain specific fields
	Email    string `jsonapi:"attr,email"`
	Password string `jsonapi:"attr,password"`
}

// Signup is the API message for sign-ups.
type Signup struct {
	// Domain specific fields
	Name     string `jsonapi:"attr,name"`
	Email    string `jsonapi:"attr,email"`
	Password string `jsonapi:"attr,password"`
}
