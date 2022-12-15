#!/bin/sh

cd "$(dirname "$0")/../"

protoc --plugin=./frontend/node_modules/.bin/protoc-gen-ts_proto --ts_proto_out=./frontend/src/types/proto --ts_proto_opt=esModuleInterop=true -I=proto/store database.proto
