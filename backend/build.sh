#!/bin/sh

version=`cat ./VERSION`

echo "Start building Bytebase monolithic ${version}..."

flags="-X 'github.com/bytebase/bytebase/bin/server/cmd.version=${version}'
-X 'github.com/bytebase/bytebase/bin/server/cmd.goversion=$(go version)'
-X 'github.com/bytebase/bytebase/bin/server/cmd.gitcommit=$(git rev-parse HEAD)'
-X 'github.com/bytebase/bytebase/bin/server/cmd.buildtime=$(date -u +"%Y-%m-%dT%H:%M:%SZ")'
-X 'github.com/bytebase/bytebase/bin/server/cmd.builduser=$(id -u -n)'"

go build -ldflags "$flags" -o ./bytebase-build/bytebase ./bin/server/main.go

echo "Completed building Bytebase monolithic ${version}."