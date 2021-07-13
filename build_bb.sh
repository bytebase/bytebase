#!/bin/sh

# cd to the root directory and run
# ./build_bb.sh [outdir]

# exit when any command fails
set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color

if [ -z "$1" ];
then
  OUTPUT_DIR=$( cd bytebase-build &> /dev/null && pwd )
else
  OUTPUT_DIR="$1"
fi

OUTPUT_BINARY=$OUTPUT_DIR/bb

if [[ `dirname "${BASH_SOURCE[0]}"` != "." ]]
then
  echo "${RED}Precheck failed.${NC} Build script must run from Bytebase root directory ${SCRIPT_DIR}"; exit 1;
fi

VERSION=`cat ./VERSION`
echo "Start building bb ${VERSION}..."

flags="-X 'github.com/bytebase/bytebase/bin/bb/cmd.version=${VERSION}'
-X 'github.com/bytebase/bytebase/bin/bb/cmd.goversion=$(go version)'
-X 'github.com/bytebase/bytebase/bin/bb/cmd.gitcommit=$(git rev-parse HEAD)'
-X 'github.com/bytebase/bytebase/bin/bb/cmd.buildtime=$(date -u +"%Y-%m-%dT%H:%M:%SZ")'
-X 'github.com/bytebase/bytebase/bin/bb/cmd.builduser=$(id -u -n)'"

# -ldflags="-w -s" means omit DWARF symbol table and the symbol table and debug information
go build --tags "release" -ldflags "-w -s $flags" -o ${OUTPUT_BINARY} ./bin/bb/main.go

echo "Completed building bb."

echo ""
echo "Printing version..."

${OUTPUT_BINARY} version