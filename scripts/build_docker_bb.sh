#!/bin/sh
# ===========================================================================
# File: build_docker_bb.sh
# Description: usage: ./build_docker_bb.sh
# ===========================================================================

# exit when any command fails
set -e

cd "$(dirname "$0")/../"
. ./scripts/init.sh

echo "Start building bb docker image ${VERSION}..."

docker build -f ./Dockerfile.bb\
    --build-arg VERSION=${VERSION} \
    --build-arg GO_VERSION="$(go version)" \
    --build-arg GIT_COMMIT="$(git rev-parse HEAD)"\
    --build-arg BUILD_TIME="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"  \
    --build-arg BUILD_USER="$(id -u -n)" \
    -t bytebase/bb .

echo "${GREEN}Completed building bb docker image ${VERSION}.${NC}"
echo ""
echo "Command to tag and push the image"
echo ""
echo "$ docker tag bytebase/bb bytebase/bb:${VERSION}; docker push bytebase/bb:${VERSION}"
echo ""
echo "Command to run bb"
echo ""
echo "$ docker run --rm --name bb bytebase/bb"
echo ""
