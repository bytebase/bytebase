#!/bin/sh
# ===========================================================================
# File: build_sql_service_docker.sh
# Description: usage: ./build_sql_service_docker.sh
# ===========================================================================

# exit when any command fails
set -e

cd "$(dirname "$0")/../"
. ./scripts/build_init.sh

echo "Start building SQL Service docker image ${VERSION}..."

docker build -f ./Dockerfile.sql-service\
    --build-arg VERSION=${VERSION} \
    --build-arg GO_VERSION="$(go version)" \
    --build-arg GIT_COMMIT="$(git rev-parse HEAD)"\
    --build-arg BUILD_TIME="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"  \
    --build-arg BUILD_USER="$(id -u -n)" \
    -t bytebase/sql .

echo "${GREEN}Completed building Bytebase SQL Service docker image ${VERSION}.${NC}"
echo ""
echo "Command to tag and push the image"
echo ""
echo "$ docker tag bytebase/sql bytebase/sql:${VERSION}; docker push bytebase/sql:${VERSION}"
echo ""
echo "Command to start Bytebase SQL Service on http://localhost:8081"
echo ""
echo "$ docker run --init --name sql-service --restart always --publish 8081:8081 --volume ~/.sql-service/data:/var/opt/sql-service bytebase/sql:${VERSION} --host http://localhost --port 8081"
echo ""
