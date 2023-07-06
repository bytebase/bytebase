#!/bin/bash

OS=$2
ARCH=$3
OUT_PREFIX=$4

tar xvf $1
FILENAME=$(basename $1)
FILENAME_NOEXT="${FILENAME%.*.*}"
cd ${FILENAME_NOEXT} 
case "${OS}" in
    linux)
        tar cvzf ../${OUT_PREFIX}-${OS}-${ARCH}.tar.gz bin/mysqlbinlog bin/mysqldump bin/mysql lib/private/libcrypto* lib/private/libssl*
        ;;
    darwin)
        tar cvzf ../${OUT_PREFIX}-${OS}-${ARCH}.tar.gz bin/mysqlbinlog bin/mysqldump bin/mysql lib/libcrypto* lib/libssl*
        ;;
esac
