#!/bin/sh

cd "$(dirname "$0")/../"

sh ./proto/generate_go.sh

sh ./proto/generate_ts.sh
