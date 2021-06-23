#!/bin/sh

version=0.1.0
flags="-X 'github.com/bytebase/bytebase/bin/server/cmd.version=${version}' -X 'github.com/bytebase/bytebase/bin/server/cmd.goversion=$(go version)' -X 'github.com/bytebase/bytebase/bin/server/cmd.gitcommit=$(git rev-parse HEAD)' -X 'github.com/bytebase/bytebase/bin/server/cmd.buildtime=$(date) ($(date -u))' -X 'github.com/bytebase/bytebase/bin/server/cmd.builduser=$(id -u -n)'"
go build -ldflags "$flags" -o ./bytebase-build/bytebase ./bin/server/main.go