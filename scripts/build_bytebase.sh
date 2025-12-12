#!/bin/sh
# ===========================================================================
# File: build_bytebase
# Description: usage: ./build_bytebase [outdir]
# ===========================================================================

# exit when any command fails
set -e

cd "$(dirname "$0")/../"
. ./scripts/build_init.sh

OUTPUT_DIR=$(mkdir_output "$1")

# Step 1 - Build the frontend release version into the backend/server/dist folder
# Step 2 - Build the monolithic app by building backend release version together with the backend/server/dist.
echo "Start building Bytebase monolithic ${VERSION}..."

echo ""
echo "Step 1 - building Bytebase frontend..."

rm -rf ./backend/server/dist

pnpm --dir ./frontend i && pnpm --dir ./frontend release

echo "Completed building Bytebase frontend."

echo ""
echo "Step 2 - building Bytebase backend..."

OUTPUT_BINARY=$OUTPUT_DIR/bytebase
go build -p=8 --tags "release,embed_frontend" -ldflags "-w -s -X 'github.com/bytebase/bytebase/backend/args.Version=${VERSION}' -X 'github.com/bytebase/bytebase/backend/args.GitCommit=${GIT_COMMIT}'" -o ${OUTPUT_BINARY} ./backend/bin/server/main.go

echo ""
echo "${GREEN}Completed building Bytebase ${VERSION}.${NC}"
echo ""
echo "Command to start Bytebase on port 8080"
echo ""
echo "$ ${OUTPUT_BINARY} --port 8080${NC}"
echo ""
