VERSION=`cat ./scripts/VERSION`
MODE="release"
GO_VERSION="unknown"
GIT_COMMIT=$1
BUILD_TIME="unknown"
BUILD_USER="unknown"

echo "Start building Bytebase staging docker image version: ${VERSION}, commit: ${GIT_COMMIT}..."

# we append 5 digits commit hash to the version for stating env (e.g. v0.10.0-abcde)
STAGING_VERSION=${VERSION}-${GIT_COMMIT:0:5}

CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build \
    --tags ${MODE} \
    -ldflags="-w -s -X 'github.com/bytebase/bytebase/bin/server/cmd.version=${STAGING_VERSION}' -X 'github.com/bytebase/bytebase/bin/server/cmd.goversion=${GO_VERSION}' -X 'github.com/bytebase/bytebase/bin/server/cmd.gitcommit=${GIT_COMMIT}' -X 'github.com/bytebase/bytebase/bin/server/cmd.buildtime=${BUILD_TIME}' -X 'github.com/bytebase/bytebase/bin/server/cmd.builduser=${BUILD_USER}'" \
    -o bytebase \
    ./bin/server/main.go
