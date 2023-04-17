package mail

import (
	"errors"
	"net/smtp"
)

type loginAuth struct {
	username string
	password string
}

// LoginAuth returns an Auth that implements the LOGIN authentication.
func LoginAuth(username, password string) smtp.Auth {
	return &loginAuth{username, password}
}

func (*loginAuth) Start(*smtp.ServerInfo) (string, []byte, error) {
	return "LOGIN", []byte{}, nil
}

func (la *loginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if more {
		switch string(fromServer) {
		case "Username:":
			return []byte(la.username), nil
		case "Password:":
			return []byte(la.password), nil
		default:
			return nil, errors.New("Unknown fromServer")
		}
	}
	return nil, nil
}
