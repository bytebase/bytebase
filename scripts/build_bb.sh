#!/bin/sh
# ===========================================================================
# File: build_bb.sh
# Description: usage: ./build_bb.sh [outdir]
# ===========================================================================

# exit when any command fails
set -e

cd "$(dirname "$0")/../"
. ./scripts/init.sh

OUTPUT_DIR=$(mkdir_output "$1")
OUTPUT_BINARY=$OUTPUT_DIR/bb

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
