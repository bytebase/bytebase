#!/bin/sh

# exit when any command fails
set -e

RED='\033[0;31m'
NC='\033[0m' # No Color

goversion=`go version | { read _ _ v _; echo ${v#go}; }`

echo ${goversion}

if [[ "${goversion}" < "1.16" ]];
then
   echo "${RED}Precheck failed.${NC} Require go version >= 1.16. Current version ${goversion}."; exit 1;
fi

if ! command -v npm &> /dev/null
then
   echo "${RED}Precheck failed.${NC} npm is not installed."; exit 1;
fi

version=`cat ./VERSION`

# Step 1 - Build the frontend release version into the backend/server/dist folder
# Step 2 - Build the monolithic app by building backend release version together with the backend/server/dist (leveraing embed introduced in Golang 1.16).
echo "Start building Bytebase monolithic ${version}..."

echo "Step 1 - building bytebase frontend..."

if command -v yarn &> /dev/null
then
    yarn --cwd ../frontend release
else
    npm --prefix ../frontend run release
fi

echo "Completed building bytebase frontend."


echo "Step 2 - building bytebase backend..."

flags="-X 'github.com/bytebase/bytebase/bin/server/cmd.version=${version}'
-X 'github.com/bytebase/bytebase/bin/server/cmd.goversion=$(go version)'
-X 'github.com/bytebase/bytebase/bin/server/cmd.gitcommit=$(git rev-parse HEAD)'
-X 'github.com/bytebase/bytebase/bin/server/cmd.buildtime=$(date -u +"%Y-%m-%dT%H:%M:%SZ")'
-X 'github.com/bytebase/bytebase/bin/server/cmd.builduser=$(id -u -n)'"

# -ldflags="-w -s" means omit DWARF symbol table and the symbol table and debug information
go build -ldflags "-w -s $flags" -o ./bytebase-build/bytebase ./bin/server/main.go

echo "Completed building bytebase backend."

echo "Completed building Bytebase monolithic ${version}."