#!/bin/bash
set -e

OUT_PREFIX="mongoutil-1.6.1"

pack(){
    if [[ ! -e $1 ]]; then
        echo "Downloading $1..."
        curl -o $1.tmp https://downloads.mongodb.com/compass/$1
        mv $1.tmp $1
    fi

    path=$1
    dir=${path%.*}
    case $2 in
    linux)
        tar -xzvf $1
        cd $dir
        tar -cJf ../${OUT_PREFIX}-$2-$3.txz bin/mongosh bin/mongosh_crypt_v1.so
        ;;
    darwin)
        unzip -o $1
        cd $dir
        tar -cJf ../${OUT_PREFIX}-$2-$3.txz bin/mongosh bin/mongosh_crypt_v1.dylib
        ;;
    esac
    cd ..
    rm -rf $dir
    rm $1
}

pack mongosh-1.6.1-darwin-x64.zip darwin amd64
pack mongosh-1.6.1-darwin-arm64.zip darwin arm64
pack mongosh-1.6.1-linux-x64.tgz linux amd64
pack mongosh-1.6.1-linux-arm64.tgz linux arm64
