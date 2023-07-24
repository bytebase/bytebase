#!/bin/bash
# ===========================================================================
# File: build_bb_docker.sh
# Description: usage: ./build_bb_docker.sh
# ===========================================================================

## Uncomment following mirrors, for China mainland developers.
##
# NVM_MIRROR=${NVM_MIRROR:-https://mirrors.ustc.edu.cn/node/}
# NODE_MIRROR=${NODE_MIRROR:-https://mirrors.ustc.edu.cn/node/}
# NPM_REGISTRY=${NPM_REGISTRY:-https://npmreg.proxy.ustclug.org/}

# exit when any command fails
set -e

cd "$(dirname "$0")/../"
. ./scripts/build_init.sh

echo "Start building bb docker image ${VERSION}..."

docker build -f ./scripts/Dockerfile.bb \
    --build-arg VERSION=${VERSION} \
    --build-arg GO_VERSION="$(go version)" \
    --build-arg GIT_COMMIT="$(git rev-parse HEAD)"\
    --build-arg BUILD_TIME="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"  \
    --build-arg BUILD_USER="$(id -u -n)" \
    --build-arg NVM_MIRROR="${NVM_MIRROR}" \
    --build-arg NODE_MIRROR="${NODE_MIRROR}" \
    --build-arg NPM_REGISTRY="${NPM_REGISTRY}" \
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
