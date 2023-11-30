#!/bin/sh

./scripts/build_bytebase.sh
./scripts/build_bb.sh

tmp_dir=$(mktemp -d) || abort "cannot create temp directory"
echo $tmp_dir
cp ./bytebase-build/bytebase $tmp_dir
cp ./bytebase-build/bb $tmp_dir
cp ./LICENSE $tmp_dir
cp ./LICENSE.enterprise $tmp_dir
cp scripts/VERSION $tmp_dir

tar -czvf ./bytebase-build/bytebase_darwin_arm64.tar.gz -C $tmp_dir .
