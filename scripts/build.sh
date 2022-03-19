#!/bin/sh

# cd to the root directory and run
# ./scripts/build.sh [outdir]

# exit when any command fails
set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color

# Version function used for version string comparison
function version { echo "$@" | awk -F. '{ printf("%d%03d%03d%03d\n", $1,$2,$3,$4); }'; }

if [ `dirname "${BASH_SOURCE[0]}"` != "./scripts" ] && [ `dirname "${BASH_SOURCE[0]}"` != "scripts" ]
then
  echo "${RED}Precheck failed.${NC} Build script must run from root directory: scripts/build.sh"; exit 1;
fi

if [ -z "$1" ];
then
  mkdir -p bytebase-build
  OUTPUT_DIR=$( cd bytebase-build &> /dev/null && pwd )
else
  OUTPUT_DIR="$1"
fi

OUTPUT_BINARY=$OUTPUT_DIR/bytebase

GO_VERSION=`go version | { read _ _ v _; echo ${v#go}; }`
if [ "$(version ${GO_VERSION})" -lt "$(version 1.16)" ];
then
   echo "${RED}Precheck failed.${NC} Require go version >= 1.16. Current version ${GO_VERSION}."; exit 1;
fi

if ! command -v npm &> /dev/null
then
   echo "${RED}Precheck failed.${NC} npm is not installed."; exit 1;
fi

# Step 1 - Build the frontend release version into the backend/server/dist folder
# Step 2 - Build the monolithic app by building backend release version together with the backend/server/dist (leveraing embed introduced in Golang 1.16).
VERSION=`cat ./scripts/VERSION`
echo "Start building Bytebase monolithic ${VERSION}..."

echo ""
echo "Step 1 - building bytebase frontend..."

if command -v pnpm &> /dev/null
then
    pnpm --cwd ./frontend i && pnpm --cwd ./frontend release
else
    npm --prefix ./frontend run release
fi

echo "Completed building bytebase frontend."

echo ""
echo "Step 2 - building bytebase backend..."

flags="-X 'github.com/bytebase/bytebase/bin/server/cmd.version=${VERSION}'
-X 'github.com/bytebase/bytebase/bin/server/cmd.goversion=$(go version)'
-X 'github.com/bytebase/bytebase/bin/server/cmd.gitcommit=$(git rev-parse HEAD)'
-X 'github.com/bytebase/bytebase/bin/server/cmd.buildtime=$(date -u +"%Y-%m-%dT%H:%M:%SZ")'
-X 'github.com/bytebase/bytebase/bin/server/cmd.builduser=$(id -u -n)'"

# -ldflags="-w -s" means omit DWARF symbol table and the symbol table and debug information
go build --tags "release" -ldflags "-w -s $flags" -o ${OUTPUT_BINARY} ./bin/server/main.go

echo "Completed building bytebase backend."

echo ""
echo "Step 3 - printing version..."

${OUTPUT_BINARY} version

echo ""
echo "${GREEN}Completed building Bytebase monolithic ${VERSION} at ${OUTPUT_BINARY}.${NC}"
echo ""
echo "Command to start Bytebase on http://localhost:8080"
echo ""
echo "$ ${OUTPUT_BINARY} --host http://localhost --port 8080${NC}"
echo ""
