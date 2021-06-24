#!/bin/sh

version=`cat ./VERSION`

echo "Start building Bytebase docker image ${version}..."

docker build \
    --build-arg VERSION=${version} \
    --build-arg GO_VERSION="$(go version)" \
    --build-arg GIT_COMMIT="$(git rev-parse HEAD)"\
    --build-arg BUILD_TIME="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"  \
    --build-arg BUILD_USER="$(id -u -n)" \
    -t bytebase/bytebase .

echo "Completed building Bytebase docker image ${version}."