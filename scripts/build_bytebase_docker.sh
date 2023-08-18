#!/bin/sh
# ===========================================================================
# File: build_docker.sh
# Description: usage: ./build_docker.sh
# ===========================================================================

# exit when any command fails
set -e

cd "$(dirname "$0")/../"
. ./scripts/build_init.sh

echo "Start building Bytebase docker image ${VERSION}..."

docker build -f ./scripts/Dockerfile \
    --build-arg VERSION=${VERSION} \
    --build-arg GO_VERSION="$(go version)" \
    --build-arg GIT_COMMIT="$(git rev-parse HEAD)"\
    --build-arg BUILD_TIME="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"  \
    --build-arg BUILD_USER="$(id -u -n)" \
    -t bytebase/bytebase .

echo "${GREEN}Completed building Bytebase docker image ${VERSION}.${NC}"
echo ""
echo "Command to tag and push the image"
echo ""
echo "$ docker tag bytebase/bytebase bytebase/bytebase:${VERSION}; docker push bytebase/bytebase:${VERSION}"
echo ""
echo "Command to start Bytebase on port 8080"
echo ""
echo "$ docker run --init --name bytebase --restart always --publish 8080:8080 --volume ~/.bytebase/data:/var/opt/bytebase bytebase/bytebase:${VERSION} --data /var/opt/bytebase --port 8080"
echo ""
