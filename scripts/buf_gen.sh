#!/bin/sh

set -e

current_dir=$(dirname "$0")
cd "$current_dir/../proto"

buf generate --template buf.gen.go.yaml
buf generate --template buf.gen.ts.yaml --path "./v1"
