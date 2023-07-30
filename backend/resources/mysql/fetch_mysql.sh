#!/bin/bash

ckmd5() {
    if [[ $(uname) = Darwin ]]; then
        test $(md5 -q $1) = $2
    else
        test $(md5sum $1 | awk '{ print $1 }') = $2
    fi
}

atomic_download() {
    if [[ ! -e $1 ]]; then
        echo "Downloading $1..."
        curl -o $1.tmp https://cdn.mysql.com/archives/mysql-8.0/$1
        if [ -n "$2" ]; then 
            if $(ckmd5 $1.tmp $2); then
                mv $1.tmp $1
            else
                rm $1.tmp
            fi
        fi
    fi
}

atomic_download $1 $2