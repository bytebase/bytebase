#!/bin/sh
# ===========================================================================
# File: build_init.sh
# Description: common variables & functions for the build scripts.
# ===========================================================================

set -e

# Global variables
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color

# Invoke from project root
VERSION='local'
GIT_COMMIT=$(git rev-parse HEAD)

# Version function used for version string comparison
version() { echo "$@" | awk -F. '{ printf("%d%03d%03d%03d\n", $1,$2,$3,$4); }'; }

# Ensure output directory existed
mkdir_output() {
    if [ -z "$1" ]; then
        mkdir -p bytebase-build
        OUTPUT_DIR=$(cd bytebase-build > /dev/null && pwd)
    else
        OUTPUT_DIR="$1"
    fi
    echo "$OUTPUT_DIR"
}

# Go and node version checks.
TARGET_GO_VERSION="1.24.4"
GO_VERSION=`go version | { read _ _ v _; echo ${v#go}; }`
if [ "$(version ${GO_VERSION})" -lt "$(version $TARGET_GO_VERSION)" ];
then
   echo "${RED}Precheck failed.${NC} Require go version >= $TARGET_GO_VERSION. Current version ${GO_VERSION}."; exit 1;
fi

NODE_VERSION=`node -v | { read v; echo ${v#v}; }`
if [ "$(version ${NODE_VERSION})" -lt "$(version 23.11.0)" ];
then
   echo "${RED}Precheck failed.${NC} Require node.js version >= 23.11.0. Current version ${NODE_VERSION}."; exit 1;
fi

if ! command -v npm > /dev/null
then
   echo "${RED}Precheck failed.${NC} npm is not installed."; exit 1;
fi