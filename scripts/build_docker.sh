#!/bin/sh

# cd to the root directory and run
# ./scripts/build_docker.sh

# exit when any command fails
set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color

if [ `dirname "${BASH_SOURCE[0]}"` != "./scripts" ] && [ `dirname "${BASH_SOURCE[0]}"` != "scripts" ]
then
  echo "${RED}Precheck failed.${NC} Build script must run from root directory: scripts/build_docker.sh"; exit 1;
fi

VERSION=`cat ./scripts/VERSION`
echo "Start building Bytebase docker image ${VERSION}..."

docker build \
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
echo "Command to start Bytebase on http://localhost:8080"
echo ""
echo "$ docker run --init --name bytebase --restart always --publish 8080:8080 --volume ~/.bytebase/data:/var/opt/bytebase bytebase/bytebase:${VERSION} --data /var/opt/bytebase --host http://localhost --port 8080"
echo ""
echo "Command to start Bytebase in demo mode on http://example.com"
echo ""
echo "$ docker run --init --name bytebase --restart always --publish 80:80 --volume ~/.bytebase/data:/var/opt/bytebase bytebase/bytebase:${VERSION} --data /var/opt/bytebase --host http://example.com --port 80 --demo"
echo ""
echo "Command to start Bytebase in readonly mode on http://example.com"
echo ""
echo "$ docker run --init --name bytebase --restart always --publish 80:80 --volume ~/.bytebase/data:/var/opt/bytebase bytebase/bytebase:${VERSION} --data /var/opt/bytebase --host http://example.com --port 80 --readonly"
echo ""