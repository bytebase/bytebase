#!/bin/sh
# ===========================================================================
# File: build_bytebase_docker.sh
# Description: usage: ./build_bytebase_docker.sh
# ===========================================================================

# exit when any command fails
set -e

cd "$(dirname "$0")/../"
. ./scripts/build_init.sh

echo "Start building Bytebase docker image ${VERSION}..."

rm -rf ./backend/resources/tmp
mkdir -p ./backend/resources/tmp/mongoutil-1.6.1-linux-amd64/
mkdir -p ./backend/resources/tmp/mysqlutil-8.0.33-linux-amd64/
mkdir -p ./backend/resources/tmp/postgres-linux-amd64-16/
tar -Jxf ./backend/resources/mongoutil/mongoutil-1.6.1-linux-amd64.txz -C ./backend/resources/tmp/mongoutil-1.6.1-linux-amd64/
tar -zxf ./backend/resources/mysqlutil/mysqlutil-8.0.33-linux-amd64.tar.gz -C ./backend/resources/tmp/mysqlutil-8.0.33-linux-amd64/
tar -Jxf ./backend/resources/postgres/postgres-linux-amd64.txz -C ./backend/resources/tmp/postgres-linux-amd64-16/

docker build -f ./scripts/Dockerfile \
    --build-arg VERSION="${VERSION}" \
    --build-arg GO_VERSION="$(go version)" \
    --build-arg GIT_COMMIT="$(git rev-parse HEAD)"\
    --build-arg BUILD_TIME="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"  \
    --build-arg BUILD_USER="$(id -u -n)" \
    --network=host \
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

rm -rf ./backend/resources/tmp
