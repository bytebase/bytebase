#!/bin/bash
set -e

if [[ ! -e $1 ]]; then
    echo "Downloading $1..."
    curl -o $1.tmp https://cdn.mysql.com/archives/mysql-8.0/$1
    mv $1.tmp $1
fi
