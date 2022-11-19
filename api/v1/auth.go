package v1

// Login is the API message for logins.
type Login struct {
	// Domain specific fields
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse is the API message for user login response.
type LoginResponse struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Token string `json:"token"`
}
