package api

// OpenAPIUserLogin is the API message for user login through OpenAPI.
type OpenAPIUserLogin struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Token string `json:"token"`
}
