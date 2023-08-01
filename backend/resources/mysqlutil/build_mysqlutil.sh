#!/bin/bash
set -e

OUT_PREFIX="mysqlutil-8.0.33"
pack(){
    echo $1
    tar xvf $1
    FILENAME=$(basename $1)
    FILENAME_NOEXT="${FILENAME%.*.*}"
    cd ${FILENAME_NOEXT} 
    case $2 in
        linux)
            tar cvzf ../${OUT_PREFIX}-$2-$3.tar.gz bin/mysqlbinlog bin/mysqldump bin/mysql lib/private/libcrypto* lib/private/libssl*
            ;;
        darwin)
            tar cvzf ../${OUT_PREFIX}-$2-$3.tar.gz bin/mysqlbinlog bin/mysqldump bin/mysql lib/libcrypto* lib/libssl*
            ;;
    esac
    rm -rf $(pwd)
    cd ..
}

GOOS=darwin GOARCH=amd64 go generate --tags mysql ../mysql/...
GOOS=darwin GOARCH=arm64 go generate --tags mysql ../mysql/...
GOOS=linux GOARCH=amd64 go generate --tags mysql ../mysql/...
GOOS=linux GOARCH=arm64 go generate --tags mysql ../mysql/...

pack ../mysql/mysql-8.0.33-macos13-x86_64.tar.gz darwin amd64
pack ../mysql/mysql-8.0.33-macos13-arm64.tar.gz darwin arm64
pack ../mysql/mysql-8.0.33-linux-glibc2.17-x86_64-minimal.tar.xz linux amd64
pack ../mysql/mysql-8.0.33-linux-glibc2.17-aarch64.tar.gz linux arm64
