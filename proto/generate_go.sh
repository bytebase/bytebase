#!/bin/sh

cd "$(dirname "$0")/../"

# https://developers.google.com/protocol-buffers/docs/gotutorial
# store package belongs to storage related proto's.
protoc --proto_path=proto --go_out=./proto proto/store/database.proto proto/v1/database_role.proto proto/v1/issue.proto
