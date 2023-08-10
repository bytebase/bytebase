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
# need docker buildx
docker buildx build -f ./scripts/Dockerfile \
    --platform=linux/amd64,linux/arm64 \
    --build-arg VERSION=${VERSION} \
    --build-arg GO_VERSION="$(go version)" \
    --build-arg GIT_COMMIT="$(git rev-parse HEAD)"\
    --build-arg BUILD_TIME="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"  \
    --build-arg BUILD_USER="$(id -u -n)" \
    --sbom=false \
    --push \
    -t bytebase/bytebase:${VERSION} .

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
echo "Command to start Bytebase on port 8080 and exposed at http://example.com via a separate gateway"
echo ""
echo "$ docker run --init --name bytebase --restart always --publish 8080:8080 --volume ~/.bytebase/data:/var/opt/bytebase bytebase/bytebase:${VERSION} --data /var/opt/bytebase --port 8080 --external-url http://example.com"
echo ""
echo "Command to start Bytebase in readonly and use default demo on port 8080"
echo ""
echo "$ docker run --init --name bytebase --restart always --publish 8080:8080 --volume ~/.bytebase/data:/var/opt/bytebase bytebase/bytebase:${VERSION} --data /var/opt/bytebase --port 8080 --demo default --readonly"
echo ""
