package api

import "github.com/bytebase/bytebase/plugin/vcs"

// AuthProvider is the authentication provider which only supports GitLab for now.
type AuthProvider struct {
	Type          vcs.Type `jsonapi:"attr,type"`
	Name          string   `jsonapi:"attr,name"`
	InstanceURL   string   `jsonapi:"attr,instanceUrl"`
	ApplicationID string   `jsonapi:"attr,applicationId"`
	// Secret will be used for OAuth on the client side when user choose to login via Gitlab
	Secret string `jsonapi:"attr,secret"`
}

// GitlabLogin is the API message for logins via Gitlab.
type GitlabLogin struct {
	InstanceURL   string `jsonapi:"attr,instanceUrl"`
	ApplicationID string `jsonapi:"attr,applicationId"`
	Secret        string `jsonapi:"attr,secret"`
	AccessToken   string `jsonapi:"attr,accessToken"`
}

// Login is the API message for logins.
type Login struct {
	// Domain specific fields
	Email    string `jsonapi:"attr,email"`
	Password string `jsonapi:"attr,password"`
}

// SignUp is the API message for sign-ups.
type SignUp struct {
	// Domain specific fields
	Name     string `jsonapi:"attr,name"`
	Email    string `jsonapi:"attr,email"`
	Password string `jsonapi:"attr,password"`
}
