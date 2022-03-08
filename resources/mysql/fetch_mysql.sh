#! /bin/sh

ckmd5() {
    if [[ $(uname) = Darwin ]]; then
        test $(md5 -q $1) = $2
    else
        test $(md5sum $1 | awk '{ print $1 }') = $2
    fi
}

atomic_download() {
    if [[ ! -e $1 ]]; then
        curl -o $1.tmp https://cdn.mysql.com//Downloads/MySQL-8.0/$1
        if $(ckmd5 $1.tmp $2); then
            mv $1.tmp $1
        else
            rm $1.tmp
        fi
    fi
}

mysql_8_0_28_macos11_arm64() {
    atomic_download mysql-8.0.28-macos11-arm64.tar.gz f1943053b12428e4c0e4ed309a636fd0
}

mysql_8_0_28_linux_glibc2_17_x86_64_minimal() {
    atomic_download mysql-8.0.28-linux-glibc2.17-x86_64-minimal.tar.xz 55a7759e25cc527416150c8181ce3f6d
}

mysql() {
    mysql_8_0_28_macos11_arm64
    mysql_8_0_28_linux_glibc2_17_x86_64_minimal
}

mysql
