package api

import (
	"context"

	"github.com/bytebase/bytebase/db"
)

type SqlConfig struct {
	DBType   db.Type `jsonapi:"attr,dbType"`
	Host     string  `jsonapi:"attr,host"`
	Port     string  `jsonapi:"attr,port"`
	Username string  `jsonapi:"attr,username"`
	Password string  `jsonapi:"attr,password"`
}

type SqlResultSet struct {
	Error string `jsonapi:"attr,error"`
}

type SqlService interface {
	Ping(ctx context.Context, config *SqlConfig) (*SqlResultSet, error)
}
