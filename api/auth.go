package api

import "github.com/bytebase/bytebase/plugin/vcs"

// AuthProvider is the authentication provider which only supports GitLab for now.
type AuthProvider struct {
	ID            int      `jsonapi:"attr,id"`
	Type          vcs.Type `jsonapi:"attr,type"`
	Name          string   `jsonapi:"attr,name"`
	InstanceURL   string   `jsonapi:"attr,instanceUrl"`
	ApplicationID string   `jsonapi:"attr,applicationId"`
	// Secret will be used for OAuth on the client side when user choose to login via Gitlab
	Secret string `jsonapi:"attr,secret"`
}

// GitlabLogin is the API message for logins via Gitlab.
type GitlabLogin struct {
	// VCSID is the ID of the chosen role provider(a VCS also).
	VCSID int `jsonapi:"attr,vcsId"`
	// Name is the user configured name of the role provider.
	Name string `jsonapi:"attr,name"`
	// Code is the authentication code granted by GitLab server,
	// we will use this code to exchange the OAuth token.
	Code string `jsonapi:"attr,code"`
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
