#!/bin/bash
# ===========================================================================
# File: build_docker.sh
# Description: usage: ./build_docker.sh
# ===========================================================================

# exit when any command fails
set -e

cd "$(dirname "$0")/../"
. ./scripts/build_init.sh

echo "Start building Bytebase docker image ${VERSION}..."

BUILDX_ENABLED=${BUILDX_ENABLED:-}
if [ -z "${BUILDX_ENABLED}" ]; then
    BUILDER_STATUS=$(docker buildx inspect 2>/dev/null | awk '/Status/ { print $2 }')
    if [ "${BUILDER_STATUS}" == "running" ]; then
		BUILDX_ENABLED=true
	else
		BUILDX_ENABLED=false
	fi
fi

BUILD_PLATFORMS=${BUILD_PLATFORMS:-linux/amd64,linux/arm64}
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

if [ "${BUILDX_ENABLED}" == "true" ]; then
    DOCKER_BUILD_CMD="buildx build --platform ${BUILD_PLATFORMS}"
    DOCKER_PUSH_CMD_HELP="$ docker ${DOCKER_BUILD_CMD} -f ./scripts/Dockerfile \
    --build-arg VERSION=${VERSION} \
    --build-arg GO_VERSION="$(go version)" \
    --build-arg GIT_COMMIT="$(git rev-parse HEAD)"\
    --build-arg BUILD_TIME="${BUILD_TIME}" \
    --build-arg BUILD_USER="$(id -u -n)" \
    --tag bytebase/bytebase:${VERSION} \
    --push ."
else
    DOCKER_BUILD_CMD="build"
    DOCKER_PUSH_CMD_HELP="$$ docker tag bytebase/bytebase bytebase/bytebase:${VERSION}; docker push bytebase/bytebase:${VERSION}"
fi

DOCKER_BUILDKIT=1 docker ${DOCKER_BUILD_CMD} -f ./scripts/Dockerfile \
    --build-arg VERSION=${VERSION} \
    --build-arg GO_VERSION="$(go version)" \
    --build-arg GIT_COMMIT="$(git rev-parse HEAD)"\
    --build-arg BUILD_TIME="${BUILD_TIME}" \
    --build-arg BUILD_USER="$(id -u -n)" \
    --tag bytebase/bytebase .

echo "${GREEN}Completed building Bytebase docker image ${VERSION}.${NC}"
echo ""
echo "Command to tag and push the image"
echo ""
echo ${DOCKER_PUSH_CMD_HELP} 
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
