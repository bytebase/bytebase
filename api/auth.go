package api

import "github.com/bytebase/bytebase/plugin/vcs"

type AuthProvider struct {
	Type          *vcs.Type `jsonapi:"attr,type"`
	InstanceURL   string    `jsonapi:"attr,instanceUrl"`
	ApplicationID string    `jsonapi:"attr,applicationId"`
	Secret        string    `jsonapi:"attr,secret"`
}

// TODO(zilong): if the number of auth provider adds up, we should use dynamic shcema to avoid repetitive work
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

// Signup is the API message for sign-ups.
type Signup struct {
	// Domain specific fields
	Name     string `jsonapi:"attr,name"`
	Email    string `jsonapi:"attr,email"`
	Password string `jsonapi:"attr,password"`
}
