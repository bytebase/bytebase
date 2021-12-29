VERSION=`cat ./scripts/VERSION`
MODE="release"
GO_VERSION="unknown"
GIT_COMMIT="unknown"
BUILD_TIME="unknown"
BUILD_USER="unknown"

CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build \
    --tags ${MODE} \
    -ldflags="-w -s -X 'github.com/bytebase/bytebase/bin/server/cmd.version=${VERSION}' -X 'github.com/bytebase/bytebase/bin/server/cmd.goversion=${GO_VERSION}' -X 'github.com/bytebase/bytebase/bin/server/cmd.gitcommit=${GIT_COMMIT}' -X 'github.com/bytebase/bytebase/bin/server/cmd.buildtime=${BUILD_TIME}' -X 'github.com/bytebase/bytebase/bin/server/cmd.builduser=${BUILD_USER}'" \
    -o bytebase \
    ./bin/server/main.go
